import Testing
import Foundation
@testable import BoundedGeneration

@Suite("History selection in the generation request")
struct HistoryInputPreparationTests {
    final class Prepared { let label: String; init(_ label: String) { self.label = label } }

    @Test func fittingHistoryPreparesOnceAndKeepsThePreparedObject() async throws {
        var calls = 0
        let object = Prepared("all")
        let result = try await HistoryInputPreparation.prepare(recordCount: 3, context: 65_536, output: 4_096) { indices in
            calls += 1
            #expect(indices == [0, 1, 2])
            return (object, 61_440)
        }
        #expect(calls == 1 && result.preparationCalls == 1)
        #expect(result.input === object)
        #expect(result.indices == [0, 1, 2])
    }

    @Test func overflowSelectsWholeRecordsInOrderAndReusesLastAcceptedInput() async throws {
        var visited: [[Int]] = []
        var objects: [String: Prepared] = [:]
        let result = try await HistoryInputPreparation.prepare(recordCount: 3, context: 14, output: 4) { indices in
            visited.append(indices)
            let label = indices.map(String.init).joined(separator: ",")
            let object = Prepared(label); objects[label] = object
            let counts = ["0,1,2": 13, "": 2, "0": 11, "1": 8, "1,2": 9]
            return (object, counts[label]!)
        }
        #expect(visited == [[0, 1, 2], [], [0], [1], [1, 2]])
        #expect(result.indices == [1, 2] && result.tokenCount == 9)
        #expect(result.input === objects["1,2"] && result.preparationCalls == 5)
    }

    @Test func oversizedMandatoryInputAndTokenizerFailureStop() async {
        await #expect(throws: TokenBudgetError.exceedsContext) {
            _ = try await HistoryInputPreparation.prepare(recordCount: 1, context: 14, output: 4) { indices in
                (indices, indices.isEmpty ? 11 : 12)
            }
        }
        enum Failure: Error, Equatable { case tokenizer }
        var calls = 0
        await #expect(throws: Failure.tokenizer) {
            _ = try await HistoryInputPreparation.prepare(recordCount: 1, context: 14, output: 4) { _ -> (String, Int) in
                calls += 1; throw Failure.tokenizer
            }
        }
        #expect(calls == 1)
    }

    @Test func sharedFragmentsKeepOrderNewlinesAndEmptySelection() throws {
        let data = #"{"before":"\"agent_history_contexts\":[]","after":"\"files\":[{\"raw_diff\":\"+line\n-next\"}]","records":["{\"final_response\":\"完了\n次行\",\"paths\":[\"a.go\"]}"],"system_without_history":"rules"}"#.data(using: .utf8)!
        let history = try JSONDecoder().decode(OptionalHistoryInput.self, from: data)
        #expect(history.sharedContent(indices: [0]) == "{\"agent_history_contexts\":[],\"codex_context\":[{\"final_response\":\"完了\n次行\",\"paths\":[\"a.go\"]}],\"files\":[{\"raw_diff\":\"+line\n-next\"}]}")
        #expect(history.sharedContent(indices: []) == "{\"agent_history_contexts\":[],\"files\":[{\"raw_diff\":\"+line\n-next\"}]}")
    }
}
