import Foundation
import MLX
import MLXLMCommon

public struct NativeThoughtBudgetStats: Sendable {
    public let prefixTokens: Int
    public let forcedClose: Bool
    public let responseSamples: Int
}

// Bounded transition for explicitly verified model markers. Stores counts, never thought tokens.
public final class NativeThoughtBudgetState: @unchecked Sendable {
    private let lock = NSLock()
    private let limit: Int
    private let open: Int
    private let close: Int
    private let eos: Int
    private let forceOpen: Bool
    private let response: GrammarSamplingState
    private var prefixTokens = 0
    private var seenOpen = false
    private var inResponse = false
    private var closeScheduled = false
    private var forcedClose = false
    private var failed = false
    private var responseSamples = 0
    private var firstFailure: String?

    public init(limit: Int, open: Int, close: Int, eos: Int, response: GrammarSamplingState, forceOpen: Bool = false) {
        self.limit = limit; self.open = open; self.close = close
        self.eos = eos; self.response = response; self.forceOpen = forceOpen
        self.failed = limit < 1
        self.firstFailure = limit < 1 ? "native_invalid_budget" : nil
    }
    public var succeeded: Bool {
        lock.lock(); defer { lock.unlock() }
        return !failed && inResponse && response.succeeded
    }
    public var diagnostic: String {
        lock.lock(); defer { lock.unlock() }
        if let firstFailure { return firstFailure }
        return inResponse ? response.diagnostic : "native_prefix_not_closed"
    }
    private func fail(_ code: String) {
        failed = true
        if firstFailure == nil { firstFailure = code }
    }
    public func stats() -> NativeThoughtBudgetStats {
        lock.lock(); defer { lock.unlock() }
        return .init(prefixTokens: prefixTokens, forcedClose: forcedClose, responseSamples: responseSamples)
    }
    private func only(_ token: Int, count: Int) -> [Float] {
        var mask = [Float](repeating: -.infinity, count: count)
        if mask.indices.contains(token) { mask[token] = 0 }
        return mask
    }
    public func maskValues(count: Int) -> [Float]? {
        lock.lock(); defer { lock.unlock() }
        if failed { return only(eos, count: count) }
        if inResponse { return response.maskValues(count: count) }
        if forceOpen && prefixTokens == 0 { return only(open, count: count) }
        if prefixTokens >= limit {
            guard seenOpen else { fail("native_budget_before_open"); return only(eos, count: count) }
            closeScheduled = true
            return only(close, count: count)
        }
        return nil
    }
    public func commit(_ token: Int) {
        lock.lock(); defer { lock.unlock() }
        guard !failed else { return }
        if inResponse {
            if !response.succeeded { responseSamples += 1 }
            response.commit(token)
            return
        }
        if token == close {
            guard seenOpen else { fail("native_close_before_open"); return }
            forcedClose = closeScheduled
            inResponse = true
            return
        }
        if token == eos { fail("native_eos_before_close"); return }
        prefixTokens += 1
        if token == open { seenOpen = true }
    }
    public func independentCopy() -> NativeThoughtBudgetState {
        lock.lock(); defer { lock.unlock() }
        let result = NativeThoughtBudgetState(limit: limit, open: open, close: close, eos: eos, response: response.independentCopy(), forceOpen: forceOpen)
        result.prefixTokens = prefixTokens; result.seenOpen = seenOpen
        result.inResponse = inResponse; result.closeScheduled = closeScheduled
        result.forcedClose = forcedClose; result.failed = failed
        result.responseSamples = responseSamples; result.firstFailure = firstFailure
        return result
    }
}
public struct NativeThoughtBudgetProcessor: LogitProcessor {
    private let state: NativeThoughtBudgetState
    public init(state: NativeThoughtBudgetState) { self.state = state }
    public mutating func prompt(_ prompt: MLXArray) {}
    public func process(logits: MLXArray) -> MLXArray {
        if let mask = state.maskValues(count: logits.dim(-1)) { return logits + MLXArray(mask) }
        return logits
    }
    public mutating func didSample(token: MLXArray) { state.commit(token.item(Int.self)) }
    public func copy() -> Self { Self(state: state.independentCopy()) }
}
