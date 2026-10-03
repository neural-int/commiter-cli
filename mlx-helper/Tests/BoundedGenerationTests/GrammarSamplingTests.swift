import Testing
import MLXGuidedGeneration
@testable import BoundedGeneration

@Suite("Sampling grammar state")
struct GrammarSamplingTests {
    func state() throws -> GrammarSamplingState {
        let tokenizer = try GrammarTokenizer(vocab: ["\"", "a", "b", "<eos>", " "], vocabType: .raw, eosTokenId: 3)
        let constraint = try GrammarConstraint(tokenizer: tokenizer, jsonSchema: "{\"type\":\"string\",\"minLength\":1,\"maxLength\":1}", fastForward: false)
        return GrammarSamplingState(constraint: constraint, vocabSize: 5, eos: 3, factory: { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: "{\"type\":\"string\",\"minLength\":1,\"maxLength\":1}", fastForward: false) })
    }
    @Test func enforcesLengthAndStop() throws {
        let s = try state()
        #expect(!s.maskValues(count: 6)[3].isFinite)
        #expect(!s.maskValues(count: 6)[5].isFinite)
        s.commit(0); s.commit(1)
        #expect(!s.maskValues(count: 6)[2].isFinite)
        s.commit(0)
        #expect(!s.succeeded)
        #expect(s.maskValues(count: 6)[3].isFinite)
        s.commit(3)
        #expect(s.succeeded)
        // Public TokenIterator prefetches after sampling the terminal token.
        #expect(s.maskValues(count: 6)[3].isFinite)
        #expect(!s.maskValues(count: 6)[1].isFinite)
        s.commit(3)
        #expect(s.succeeded)
    }
    @Test func rejectsEarlyStop() throws {
        let s = try state(); s.commit(3)
        #expect(!s.succeeded)
        #expect(s.maskValues(count: 5)[3].isFinite)
    }
    @Test func copyHasIndependentMatcher() throws {
        let original = try state(); original.commit(0)
        let copy = original.independentCopy()
        copy.commit(1); copy.commit(0); copy.commit(3)
        #expect(copy.succeeded)
        #expect(!original.succeeded)
        #expect(original.maskValues(count: 5)[1].isFinite)
        #expect(!original.maskValues(count: 5)[3].isFinite)
    }
    @Test func bothBooleanValuesAreAllowed() throws {
        let tokenizer = try GrammarTokenizer(vocab: ["{\"analysis\":\"x\",\"approve\":", "true", "false", "}", "<eos>"], vocabType: .raw, eosTokenId: 4)
        let schema = "{\"type\":\"object\",\"properties\":{\"analysis\":{\"type\":\"string\",\"minLength\":1,\"maxLength\":640},\"approve\":{\"type\":\"boolean\"}},\"required\":[\"analysis\",\"approve\"],\"additionalProperties\":false}"
        let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
        for booleanToken in [1, 2] {
            let state = GrammarSamplingState(constraint: try factory(), vocabSize: 5, eos: 4, factory: factory)
            state.commit(0)
            let mask = state.maskValues(count: 5)
            #expect(mask[1].isFinite && mask[2].isFinite)
            state.commit(booleanToken); state.commit(3); state.commit(4)
            #expect(state.succeeded)
        }
    }

    @Test func topLevelBooleanIsStrictAndTerminal() throws {
        let tokenizer = try GrammarTokenizer(vocab: ["true", "false", "<eos>", "0", "{}"], vocabType: .raw, eosTokenId: 2)
        let schema = "{\"type\":\"boolean\"}"
        let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
        for token in [0, 1] {
            let state = GrammarSamplingState(constraint: try factory(), vocabSize: 5, eos: 2, factory: factory)
            let mask = state.maskValues(count: 5)
            #expect(mask[0].isFinite && mask[1].isFinite)
            #expect(!mask[2].isFinite && !mask[3].isFinite && !mask[4].isFinite)
            state.commit(token)
            #expect(state.maskValues(count: 5)[2].isFinite)
            state.commit(2); state.commit(2)
            #expect(state.succeeded)
        }
    }

    @Test func sharedTokenizerKeepsSchemaMatchersIndependent() throws {
        let tokenizer = try GrammarTokenizer(vocab: ["\"", "a", "b", "<eos>"], vocabType: .raw, eosTokenId: 3)
        let schemaA = "{\"type\":\"string\",\"enum\":[\"a\"]}"
        let schemaB = "{\"type\":\"string\",\"enum\":[\"b\"]}"
        let factoryA: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schemaA, fastForward: false) }
        let factoryB: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schemaB, fastForward: false) }
        let a = GrammarSamplingState(constraint: try factoryA(), vocabSize: 4, eos: 3, factory: factoryA)
        let b = GrammarSamplingState(constraint: try factoryB(), vocabSize: 4, eos: 3, factory: factoryB)
        a.commit(0); a.commit(1); a.commit(0); a.commit(3)
        #expect(a.succeeded && !b.succeeded)
        #expect(b.maskValues(count: 4)[0].isFinite && !b.maskValues(count: 4)[3].isFinite)
        b.commit(0)
        #expect(b.maskValues(count: 4)[2].isFinite && !b.maskValues(count: 4)[1].isFinite)
        b.commit(2); b.commit(0); b.commit(3)
        #expect(b.succeeded)
    }

    @Test func diagnosticSeparatesBoundsCommitAndPrefetchFailures() throws {
        let bounds = try state(); bounds.commit(6)
        #expect(bounds.diagnostic == "token_bounds")
        #expect(bounds.independentCopy().diagnostic == "token_bounds")
        #expect(bounds.maskValues(count: 5)[3].isFinite && !bounds.succeeded)
        let illegal = try state(); illegal.commit(3)
        #expect(illegal.diagnostic == "token_commit_error")
        let prefetch = try state(); prefetch.commit(0); prefetch.commit(1); prefetch.commit(0); prefetch.commit(3)
        #expect(prefetch.diagnostic == "none" && prefetch.succeeded)
        prefetch.commit(1)
        #expect(prefetch.diagnostic == "unexpected_prefetch_token" && !prefetch.succeeded)
    }
    @Test func diagnosticMarksUnterminatedAndUnavailableStates() throws {
        let pending = try state(); pending.commit(0)
        #expect(pending.diagnostic == "terminal_no_accepted_stop")
        let unavailable = GrammarSamplingState(constraint: nil, vocabSize: 5, eos: 3, factory: { throw DiagnosticTestError.unavailable })
        #expect(unavailable.diagnostic == "constraint_unavailable")
        _ = unavailable.maskValues(count: 5); unavailable.commit(3)
        #expect(unavailable.diagnostic == "constraint_unavailable" && !unavailable.succeeded)
    }
    private enum DiagnosticTestError: Error { case unavailable }

    @Test func runtimeStopCountsDistinguishAllowedWhitespaceFromEOS() throws {
        let tokenizer = try GrammarTokenizer(vocab: ["{", "\"a\":", "true", "}", "<eos>", "\n", "<unk>"], vocabType: .raw, eosTokenId: 4)
        let schema = "{\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"boolean\"}},\"required\":[\"a\"],\"additionalProperties\":false}"
        let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
        let s = GrammarSamplingState(constraint: try factory(), vocabSize: 7, eos: 4, runtimeStopIDs: [4,5], unknownID: 6, factory: factory)
        s.commit(0)
        #expect(s.maskValues(count: 7)[5].isFinite)
        s.commit(5)
        #expect(!s.succeeded && s.diagnostic == "terminal_no_accepted_stop")
        #expect(s.runtimeStopCounts() == [0,1,0,0,1,0])
        #expect(s.independentCopy().runtimeStopCounts() == s.runtimeStopCounts())
        s.commit(1); s.commit(2); s.commit(3); s.commit(4); s.commit(4)
        #expect(s.succeeded && s.runtimeStopCounts() == [2,1,0,1,1,0])
        let unknown = GrammarSamplingState(constraint: try factory(), vocabSize: 7, eos: 4, runtimeStopIDs: [4,5], unknownID: 6, factory: factory)
        unknown.commit(6)
        #expect(unknown.runtimeStopCounts() == [0,0,1,0,0,1] && !unknown.succeeded)
    }

    @Test func alignedRuntimeStopsPreserveJSONAndTerminalEOS() throws {
        let tokenizer = try GrammarTokenizer(vocab: ["{", "\"a\":", "true", "false", "}", "<eos>", "\n", " "], vocabType: .raw, eosTokenId: 5)
        let schema = "{\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"boolean\"}},\"required\":[\"a\"],\"additionalProperties\":false}"
        let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
        for value in [2,3] {
            let s = GrammarSamplingState(constraint: try factory(), vocabSize: 8, eos: 5, runtimeStopIDs: [5,6], unknownID: 7, alignRuntimeStops: true, factory: factory)
            s.commit(0)
            let mask = s.maskValues(count: 8)
            #expect(!mask[6].isFinite && !mask[7].isFinite && !mask[5].isFinite && mask[1].isFinite)
            let clone = s.independentCopy()
            #expect(!clone.maskValues(count: 8)[6].isFinite)
            s.commit(1)
            #expect(s.maskValues(count: 8)[2].isFinite && s.maskValues(count: 8)[3].isFinite)
            s.commit(value); s.commit(4)
            #expect(s.maskValues(count: 8)[5].isFinite)
            s.commit(5); s.commit(5)
            #expect(s.succeeded && s.diagnostic == "none")
            #expect(s.runtimeStopCounts()[1] == 0 && s.runtimeStopCounts()[2] == 0)
        }
    }

}
