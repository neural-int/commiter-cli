import Foundation
import Testing
import MLXGuidedGeneration
@testable import BoundedGeneration

struct ScopeValueConstraintTests {
    let words: Set<String> = ["feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"]
    @Test func checksOnlyCompletedScopeValues() {
        for word in words {
            for value in [word, word.uppercased(), " \u{3000}" + word + "\t"] {
                let literal = String(data: try! JSONEncoder().encode(value), encoding: .utf8)!
                #expect(!ScopeValueConstraint.allows("{\"scope\":" + literal + "}", forbidden: words))
            }
        }
        for prefix in [#"{"scope":"doc"#, #"{"scope":"docs"#, #"{"type":"docs","scope":"readme"}"#, #"{"scope":"docs-api"}"#, #"{"scope":"文書"}"#, #"{"summary":"\"scope\":\"docs\"","scope":"mail"}"#] {
            #expect(ScopeValueConstraint.allows(prefix, forbidden: words))
        }
        #expect(!ScopeValueConstraint.allows(#"{"scope":"\u0064ocs"}"#, forbidden: words))
        #expect(!ScopeValueConstraint.allows(#"{"scope":"FİX"}"#, forbidden: words))
        #expect(!ScopeValueConstraint.allows("{\"scope\":\"docs\"", forbidden: words))
    }
    @Test func masksForbiddenClosingTokenAndPreservesOtherFields() throws {
        let vocab = ["{\"scope\":\"", "docs", "readme", "\"}", "<eos>"]
        let tokenizer = try GrammarTokenizer(vocab: vocab, vocabType: .raw, eosTokenId: 4)
        let schema = #"{"type":"object","properties":{"scope":{"type":"string","minLength":1,"maxLength":16}},"required":["scope"],"additionalProperties":false}"#
        let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
        let state = GrammarSamplingState(constraint: try factory(), vocabSize: 5, eos: 4, factory: factory)
        let blocked = words
        state.setPrefixConstraint(candidate: { $0 == 3 }) { ids in ScopeValueConstraint.allows(ids.filter { $0 != 4 }.map { vocab[$0] }.joined(), forbidden: blocked) }
        state.commit(0); state.commit(1)
        #expect(!state.maskValues(count: 5)[3].isFinite)
        let clone = state.independentCopy()
        #expect(!clone.maskValues(count: 5)[3].isFinite)
    }
}
