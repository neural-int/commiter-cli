import Testing
import MLX
import MLXLMCommon
import MLXGuidedGeneration
@testable import BoundedGeneration

@Suite("Overlapped token state updates")
struct OverlappedTokenProcessorTests {
    func state(limit: Int = 2) throws -> NativeThoughtBudgetState {
        let tokenizer = try GrammarTokenizer(vocab: ["<think>", "thought", "</think>", "{}", "<eos>"], vocabType: .raw, eosTokenId: 4)
        let factory: @Sendable () throws -> GrammarConstraint = {
            try GrammarConstraint(tokenizer: tokenizer, jsonSchema: "{\"type\":\"object\",\"additionalProperties\":false}", fastForward: false)
        }
        let response = GrammarSamplingState(constraint: try factory(), vocabSize: 5, eos: 4, factory: factory)
        return NativeThoughtBudgetState(limit: limit, open: 0, close: 2, eos: 4, response: response, promptHasOpen: true)
    }
    func logits() -> MLXArray { MLXArray.zeros([5]) }
    func sample(_ token: Int, into processor: inout OverlappedTokenProcessor) {
        processor.didSample(token: MLXArray(Int32(token)))
    }
    @Test func budgetBoundaryAndGrammarAreUpdatedBeforeNextSampling() throws {
        let state = try state()
        var processor = OverlappedTokenProcessor(NativeThoughtBudgetProcessor(state: state))
        _ = processor.process(logits: logits())
        sample(1, into: &processor)
        #expect(state.stats().prefixTokens == 0)
        _ = processor.process(logits: logits())
        #expect(state.stats().prefixTokens == 1)
        sample(1, into: &processor)
        let boundary = processor.process(logits: logits()).asArray(Float.self)
        #expect(state.stats().prefixTokens == 2)
        #expect(boundary[2].isFinite && !boundary[1].isFinite && !boundary[4].isFinite)
        sample(2, into: &processor)
        let response = processor.process(logits: logits()).asArray(Float.self)
        #expect(state.stats().forcedClose && response[3].isFinite && !response[1].isFinite)
        sample(3, into: &processor)
        _ = processor.process(logits: logits())
        sample(4, into: &processor)
        _ = processor.process(logits: logits()) // Iterator commits EOS before returning it.
        #expect(state.succeeded && state.stats().responseSamples == 2)
        sample(4, into: &processor) // Unreturned lookahead stays pending.
        #expect(state.succeeded && state.stats().responseSamples == 2)
    }
    @Test func naturalCloseAndEarlyEOSRemainDistinct() throws {
        let natural = try state()
        var p = OverlappedTokenProcessor(NativeThoughtBudgetProcessor(state: natural))
        _ = p.process(logits: logits()); sample(2, into: &p)
        _ = p.process(logits: logits())
        #expect(!natural.stats().forcedClose && natural.stats().prefixTokens == 0)
        sample(3, into: &p); _ = p.process(logits: logits())
        sample(4, into: &p); _ = p.process(logits: logits())
        #expect(natural.succeeded)
        let early = try state()
        var e = OverlappedTokenProcessor(NativeThoughtBudgetProcessor(state: early))
        _ = e.process(logits: logits()); sample(4, into: &e)
        let mask = e.process(logits: logits()).asArray(Float.self)
        #expect(early.diagnostic == "native_eos_before_close" && !early.succeeded)
        #expect(mask[4].isFinite && !mask[3].isFinite)
    }
    @Test func cancellationDoesNotCommitPendingLookahead() throws {
        let state = try state()
        do {
            var p = OverlappedTokenProcessor(NativeThoughtBudgetProcessor(state: state))
            _ = p.process(logits: logits()); sample(1, into: &p)
        }
        #expect(state.stats().prefixTokens == 0 && !state.succeeded)
    }
    @Test func copiesOwnIndependentPendingSlotsAndMatchers() throws {
        let original = try state()
        var p = OverlappedTokenProcessor(NativeThoughtBudgetProcessor(state: original))
        _ = p.process(logits: logits()); sample(1, into: &p)
        var copy = p.copy()
        _ = copy.process(logits: logits()); sample(2, into: &copy)
        _ = copy.process(logits: logits()); sample(3, into: &copy)
        _ = copy.process(logits: logits()); sample(4, into: &copy)
        _ = copy.process(logits: logits())
        #expect(original.stats().prefixTokens == 0 && !original.succeeded)
        _ = p.process(logits: logits())
        #expect(original.stats().prefixTokens == 1 && !original.succeeded)
    }
}
