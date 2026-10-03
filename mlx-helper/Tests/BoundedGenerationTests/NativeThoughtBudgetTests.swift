import Testing
import MLXGuidedGeneration
@testable import BoundedGeneration

@Suite("Bounded native thought transition")
struct NativeThoughtBudgetTests {
    func state(forceOpen: Bool = false) throws -> NativeThoughtBudgetState {
        let tokenizer = try GrammarTokenizer(vocab: ["<|channel>", "thought ", "<channel|>", "{\"candidate_id\":\"C001\"}", "<eos>"], vocabType: .raw, eosTokenId: 4)
        let schema = "{\"type\":\"object\",\"properties\":{\"candidate_id\":{\"type\":\"string\",\"enum\":[\"C001\"]}},\"required\":[\"candidate_id\"],\"additionalProperties\":false}"
        let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
        let response = GrammarSamplingState(constraint: try factory(), vocabSize: 5, eos: 4, factory: factory)
        return NativeThoughtBudgetState(limit: 2, open: 0, close: 2, eos: 4, response: response, forceOpen: forceOpen)
    }
    @Test func forcesBoundaryThenValidatesResponse() throws {
        let s = try state()
        #expect(s.maskValues(count: 5) == nil)
        s.commit(0); s.commit(1)
        let mask = try #require(s.maskValues(count: 5))
        #expect(mask[2].isFinite && !mask[4].isFinite)
        s.commit(2)
        #expect(s.stats().prefixTokens == 2 && s.stats().forcedClose)
        #expect(try #require(s.maskValues(count: 5))[3].isFinite)
        s.commit(3); s.commit(4)
        #expect(s.succeeded)
        s.commit(4) // Terminal prefetch is absorbed by response grammar.
        #expect(s.succeeded && s.stats().responseSamples == 2)
    }
    @Test func naturalBoundaryIsNotForced() throws {
        let s = try state(); s.commit(0); s.commit(2)
        #expect(!s.stats().forcedClose && s.stats().prefixTokens == 1)
        s.commit(3); s.commit(4); #expect(s.succeeded)
    }
    @Test func earlyEOSAndMissingChannelFailClosed() throws {
        let early = try state(); early.commit(4); #expect(!early.succeeded)
        let missing = try state(); missing.commit(1); missing.commit(1)
        #expect(try #require(missing.maskValues(count: 5))[4].isFinite)
        missing.commit(4); #expect(!missing.succeeded)
    }
    @Test func copiesDoNotSharePhaseState() throws {
        let original = try state(); original.commit(0)
        let copy = original.independentCopy(); copy.commit(1)
        _ = copy.maskValues(count: 5); copy.commit(2); copy.commit(3); copy.commit(4)
        #expect(copy.succeeded && copy.stats().forcedClose)
        #expect(!original.succeeded && original.stats().prefixTokens == 1)
        #expect(original.maskValues(count: 5) == nil)
    }
    @Test func ministralMarkersCloseBeforeStrictBooleanResponse() throws {
        var vocab = (0..<37).map { "unused\($0)" }
        vocab[32] = "true"; vocab[33] = "thought"; vocab[34] = "[THINK]"; vocab[35] = "[/THINK]"; vocab[36] = "<eos>"
        let tokenizer = try GrammarTokenizer(vocab:vocab, vocabType:.raw, eosTokenId:36)
        let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer:tokenizer,jsonSchema:"{\"type\":\"boolean\"}",fastForward:false) }
        let response = GrammarSamplingState(constraint:try factory(),vocabSize:37,eos:36,factory:factory)
        let state = NativeThoughtBudgetState(limit:2,open:34,close:35,eos:36,response:response)
        state.commit(34);state.commit(33)
        #expect(try #require(state.maskValues(count:37))[35].isFinite)
        state.commit(35)
        #expect(!state.succeeded)
        #expect(!((try #require(state.maskValues(count:37)))[34].isFinite))
        state.commit(32);state.commit(36)
        #expect(state.succeeded && state.stats().forcedClose)
    }

    @Test func forcedOpenAndCopiedBoundaryRemainIndependent() throws {
        let original = try state(forceOpen:true)
        let mask = try #require(original.maskValues(count:5))
        #expect(mask[0].isFinite && !mask[1].isFinite && !mask[4].isFinite)
        let copy = original.independentCopy()
        copy.commit(0);copy.commit(1)
        _ = copy.maskValues(count:5);copy.commit(2);copy.commit(3);copy.commit(4)
        #expect(copy.succeeded && copy.stats().forcedClose)
        #expect(original.stats().prefixTokens == 0 && !original.succeeded)
        #expect(try #require(original.maskValues(count:5))[0].isFinite)
    }

    @Test func diagnosticSeparatesNativeTransitionsAndResponseFailure() throws {
        let pending = try state()
        #expect(pending.diagnostic == "native_prefix_not_closed")
        let early = try state(); early.commit(4)
        #expect(early.diagnostic == "native_eos_before_close")
        #expect(early.independentCopy().diagnostic == early.diagnostic)
        let close = try state(); close.commit(2)
        #expect(close.diagnostic == "native_close_before_open")
        let missing = try state(); missing.commit(1); missing.commit(1); _ = missing.maskValues(count: 5)
        #expect(missing.diagnostic == "native_budget_before_open")
        let response = try state(); response.commit(0); response.commit(2); response.commit(1)
        #expect(response.diagnostic == "token_commit_error" && !response.succeeded)
        let success = try state(); success.commit(0); success.commit(2); success.commit(3); success.commit(4)
        #expect(success.diagnostic == "none" && success.succeeded)
    }

}
