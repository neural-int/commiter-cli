import Foundation
import MLX
import MLXLMCommon
import MLXGuidedGeneration

// One generation owns a matcher. The lock protects the terminal read after
// the asynchronous generation stream has drained.
public final class GrammarSamplingState: @unchecked Sendable {
    private let lock = NSLock()
    private let constraint: GrammarConstraint?
    private let factory: @Sendable () throws -> GrammarConstraint
    private var committedTokens: [Int] = []
    private let vocabSize: Int
    private let eos: Int
    private let whitespace: Set<Int>
    private var failed: Bool
    private var acceptedStop = false
    private var firstFailure: String?
    private let runtimeStopIDs: Set<Int>
    private let unknownID: Int?
    private let alignRuntimeStops: Bool
    private var stopCounts = [Int](repeating: 0, count: 6)
    public init(constraint: GrammarConstraint?, vocabSize: Int, eos: Int, whitespace: Set<Int> = [], runtimeStopIDs: Set<Int> = [], unknownID: Int? = nil, alignRuntimeStops: Bool = false, factory: @escaping @Sendable () throws -> GrammarConstraint) {
        self.factory = factory
        self.runtimeStopIDs = runtimeStopIDs; self.unknownID = unknownID; self.alignRuntimeStops = alignRuntimeStops
        self.constraint = constraint; self.vocabSize = vocabSize
        self.eos = eos; self.whitespace = whitespace; self.failed = constraint == nil
        self.firstFailure = constraint == nil ? "constraint_unavailable" : nil
    }
    public var succeeded: Bool {
        lock.lock(); defer { lock.unlock() }
        return !failed && acceptedStop
    }
    // Only fixed content-free codes leave this state. No token or exception data.
    public var diagnostic: String {
        lock.lock(); defer { lock.unlock() }
        return firstFailure ?? (acceptedStop && !failed ? "none" : "terminal_no_accepted_stop")
    }
    private func fail(_ code: String) {
        failed = true
        if firstFailure == nil { firstFailure = code }
    }
    public func maskValues(count: Int) -> [Float] {
        lock.lock(); defer { lock.unlock() }
        var values = [Float](repeating: -.infinity, count: count)
        if acceptedStop && !failed {
            // TokenIterator evaluates one token ahead before yielding EOS.
            // Preserve the accepted terminal state during this prefetch.
            if values.indices.contains(eos) { values[eos] = 0 }
            return values
        }
        do {
            guard !failed, let constraint else { throw StateError.failed }
            let mask = try constraint.computeMask()
            for i in 0..<min(vocabSize, count) {
                let word = i / 32
                guard word < mask.mask.count else { throw StateError.maskWordCount }
                if (UInt32(bitPattern: mask.mask[word]) >> UInt32(i % 32)) & 1 == 1 {
                    values[i] = whitespace.contains(i) ? -200 : 0
                }
            }
            if alignRuntimeStops {
                var excluded = runtimeStopIDs
                if let unknownID { excluded.insert(unknownID) }
                excluded.remove(eos)
                for token in excluded where values.indices.contains(token) { values[token] = -.infinity }
            }
            guard values.contains(where: { $0.isFinite }) else { throw StateError.noFiniteToken }
        } catch {
            switch error {
            case StateError.maskWordCount: fail("mask_word_count")
            case StateError.noFiniteToken: fail("mask_no_finite_token")
            case StateError.failed: fail("constraint_unavailable")
            default: fail("mask_compute_error")
            }
            values = [Float](repeating: -.infinity, count: count)
            // Force a bounded stop. The caller rejects this stream as failure.
            if values.indices.contains(eos) { values[eos] = 0 }
        }
        return values
    }
    // Counts only: EOS / other runtime stop / unknown, and each before acceptance.
    public func runtimeStopCounts() -> [Int] {
        lock.lock(); defer { lock.unlock() }; return stopCounts
    }
    public func commit(_ token: Int) {
        lock.lock(); defer { lock.unlock() }
        let stopIndex: Int? = token == unknownID ? 2 : (token == eos ? 0 : (runtimeStopIDs.contains(token) ? 1 : nil))
        if let stopIndex {
            stopCounts[stopIndex] += 1
            if !acceptedStop { stopCounts[stopIndex + 3] += 1 }
        }
        if acceptedStop && !failed {
            if token != eos { fail("unexpected_prefetch_token") }
            return
        }
        guard !failed else { return }
        guard let constraint else { fail("constraint_unavailable"); return }
        guard token >= 0, token < vocabSize else { fail("token_bounds"); return }
        do {
            let result = try constraint.commitToken(Int32(token))
            guard result.tokens.isEmpty else { fail("unexpected_fast_forward"); return }
            acceptedStop = result.isTerminated
            committedTokens.append(token)
        } catch { fail("token_commit_error") }
    }
    public func independentCopy() -> GrammarSamplingState {
        lock.lock(); defer { lock.unlock() }
        let clone = failed ? nil : try? factory()
        let result = GrammarSamplingState(constraint: clone, vocabSize: vocabSize, eos: eos, whitespace: whitespace, runtimeStopIDs: runtimeStopIDs, unknownID: unknownID, alignRuntimeStops: alignRuntimeStops, factory: factory)
        // xgrammar v0.1.30 has no Fork(). Replay transient token IDs into a
        // separate matcher, never share mutable state between copies.
        for token in committedTokens { result.commit(token) }
        if failed { result.firstFailure = firstFailure }
        result.stopCounts = stopCounts
        return result
    }
    private enum StateError: Error { case failed, maskWordCount, noFiniteToken }
}
public struct GrammarSamplingProcessor: LogitProcessor {
    private let state: GrammarSamplingState
    public init(state: GrammarSamplingState) { self.state = state }
    public mutating func prompt(_ prompt: MLXArray) {}
    public func process(logits: MLXArray) -> MLXArray {
        logits + MLXArray(state.maskValues(count: logits.dim(-1)))
    }
    public mutating func didSample(token: MLXArray) { state.commit(token.item(Int.self)) }
    public func copy() -> Self { Self(state: state.independentCopy()) }
}
