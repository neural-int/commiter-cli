import Foundation
import Testing
import Tokenizers
import MLXHuggingFace
import MLXLMCommon
import MLXGuidedGeneration
@testable import BoundedGeneration

@Suite("Score wire tokenizer and grammar audit")
struct ScoreWireGrammarTests {
    struct Message: Decodable { let role: String; let content: String }
    struct Vector: Decodable { let legal: Bool; let score: Int; let output: String }
    struct Case: Decodable {
        let name: String; let schema: String; let original: [Message]; let policy: [Message]
        let expected_original_tokens: Int; let expected_policy_tokens: Int
        let criterion: String; let vectors: [Vector]
    }
    @Test func replayActualScorerSchemasAndPrompts() async throws {
        let input = URL(fileURLWithPath: "/tmp/issue153-score-wire-input.json")
        let cases = try JSONDecoder().decode([Case].self, from: Data(contentsOf: input))
        let folder = URL(fileURLWithPath: "/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8")
        let upstream = try await Tokenizers.AutoTokenizer.from(modelFolder: folder)
        let tokenizer = #adaptHuggingFaceTokenizer(upstream)
        let vocabulary = TokenizerVocabExtractor.extractForGrammar(from: tokenizer)
        let eos = try #require(tokenizer.eosTokenId)
        let grammarTokenizer = try GrammarTokenizer(vocab: vocabulary.vocab, vocabType: vocabulary.vocabType, eosTokenId: Int32(eos))
        let config = try JSONSerialization.jsonObject(with: Data(contentsOf: folder.appendingPathComponent("config.json"))) as! [String: Any]
        let stopIDs = (config["eos_token_id"] as? [Int]) ?? [config["eos_token_id"] as? Int ?? eos]
        var accepted = 0; var rejected = 0; var promptPasses = 0
        for item in cases {
            for (messages, expected) in [(item.original,item.expected_original_tokens),(item.policy,item.expected_policy_tokens)] {
                let chat: [[String: any Sendable]] = messages.map { ["role": $0.role, "content": $0.content] }
                let tokens = try tokenizer.applyChatTemplate(messages: chat, additionalContext: ["enable_thinking": false])
                let decoded = tokenizer.decode(tokenIds: tokens)
                #expect(tokens.count == expected)
                #expect(decoded.contains(messages[0].content))
                #expect(decoded.contains(messages[1].content))
                if messages[0].content.contains(item.criterion) { #expect(decoded.contains(item.criterion)) }
                promptPasses += 1
            }
            let factory: @Sendable () throws -> GrammarConstraint = {
                try GrammarConstraint(tokenizer: grammarTokenizer, jsonSchema: item.schema, fastForward: false)
            }
            for vector in item.vectors {
                let state = GrammarSamplingState(constraint: try factory(), vocabSize: grammarTokenizer.vocabSize, eos: eos,
                    whitespace: WhitespaceTokenBias.compute(tokenizer: tokenizer).tokenIDs,
                    runtimeStopIDs: Set(stopIDs).union([eos]), unknownID: tokenizer.unknownTokenId,
                    alignRuntimeStops: true, factory: factory)
                let tokens = tokenizer.encode(text: vector.output, addSpecialTokens: false)
                var allowed = true
                for token in tokens + [eos] {
                    if !state.maskValues(count: grammarTokenizer.vocabSize)[token].isFinite { allowed = false; break }
                    state.commit(token)
                }
                if vector.legal {
                    #expect(allowed && state.succeeded && state.diagnostic == "none")
                    accepted += allowed && state.succeeded ? 1 : 0
                } else {
                    #expect(!allowed && !state.succeeded)
                    rejected += !allowed ? 1 : 0
                }
            }
        }
        let report: [String: Any] = ["cases":cases.count,"prompt_checks":promptPasses,"legal_replays_accepted":accepted,"illegal_replays_rejected":rejected,"model_calls":0,"weights_loaded":false,"vocab_size":grammarTokenizer.vocabSize,"eos":eos]
        let data = try JSONSerialization.data(withJSONObject: report, options: [.prettyPrinted,.sortedKeys])
        try data.write(to: URL(fileURLWithPath: "/tmp/issue153-score-wire-result.json"))
    }
}
