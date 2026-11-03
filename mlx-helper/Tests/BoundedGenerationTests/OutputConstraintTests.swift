import Foundation
import Testing
import MLXGuidedGeneration
@testable import BoundedGeneration

@Suite("Independent output constraint")
struct OutputConstraintTests {
    @Test func unconstrainedTerminalPrefetchDoesNotInvalidateStop() {
        let state = GrammarSamplingState(unconstrainedVocabSize: 5, eos: 4, runtimeStopIDs: [4])
        state.commit(1); state.commit(4)
        #expect(state.succeeded)
        state.commit(1) // The iterator may already have sampled its lookahead.
        #expect(state.succeeded)
        #expect(state.diagnostic == "none")
        #expect(state.independentCopy().succeeded)
    }
    @Test func modesRejectUnknownValues() throws {
        #expect(try JSONDecoder().decode(OutputConstraint.self, from: Data("\"json_schema\"".utf8)) == .jsonSchema)
        #expect(try JSONDecoder().decode(OutputConstraint.self, from: Data("\"none\"".utf8)) == .none)
        #expect(throws: (any Error).self) { try JSONDecoder().decode(OutputConstraint.self, from: Data("\"json\"".utf8)) }
    }
    @Test func freeResponsePreservesThinkingBoundaryAndBudget() throws {
        let response = GrammarSamplingState(unconstrainedVocabSize: 5, eos: 4, runtimeStopIDs: [4])
        let state = NativeThoughtBudgetState(limit: 2, open: 0, close: 2, eos: 4, response: response, promptHasOpen: true)
        #expect(state.maskValues(count: 5) == nil)
        state.commit(1); state.commit(1)
        #expect(try #require(state.maskValues(count: 5))[2].isFinite)
        state.commit(2)
        #expect(state.maskValues(count: 5) == nil)
        state.commit(1) // Non-JSON text is permitted after thinking in `none`.
        #expect(!state.succeeded)
        let copy = state.independentCopy()
        copy.commit(4)
        #expect(copy.succeeded && copy.stats().forcedClose)
        #expect(!state.succeeded)
        state.commit(4)
        #expect(state.succeeded)
    }
    @Test func promptOpenedThinkingIsFreeUntilNaturalClose() {
        let response = GrammarSamplingState(unconstrainedVocabSize: 5, eos: 4, runtimeStopIDs: [4])
        let state = NativeThoughtBudgetState(limit: nil, open: 0, close: 2, eos: 4, response: response, promptHasOpen: true)
        for _ in 0..<1100 { #expect(state.maskValues(count: 5) == nil); state.commit(1) }
        state.commit(2); state.commit(1); state.commit(4)
        #expect(state.succeeded && !state.stats().forcedClose)
    }
    @Test func schemaControlsRequiredOptionalAndDynamicExample() throws {
        let schema = #"{"type":"object","properties":{"F9":{"type":"string","enum":["G1","G2"]},"optional":{"type":"boolean"}},"required":["F9"],"additionalProperties":false}"#
        let prompt = try StructuredOutputPrompt.instruction(schema: schema)
        #expect(prompt.contains("$.F9: type=string; required"))
        #expect(prompt.contains("$.optional: type=boolean; optional"))
        #expect(prompt.contains(schema))
        #expect(prompt.contains(#"{"F9":"G1"}"#))
        #expect(prompt.contains("FINAL ANSWER only"))
        #expect(prompt.contains("never copy"))
    }
    @Test func unsupportedExampleFailsBeforeInference() {
        #expect(throws: (any Error).self) { try StructuredOutputPrompt.instruction(schema: #"{"$ref":"unknown"}"#) }
    }
}
