import Testing
import Foundation
@testable import BoundedGeneration

@Suite("History selection in the generation request")
struct HistoryInputPreparationTests {
    final class Prepared { let label: String; init(_ label: String) { self.label = label } }

    @Test func overflowPreparationCountDoesNotGrowWithHistoryCount() async throws {
        for recordCount in [20, 100, 2_000] {
            var calls = 0
            var baseCalls = 0
            let base = Prepared("mandatory")
            let result = try await HistoryInputPreparation.prepare(recordByteCounts: Array(repeating: 1, count: recordCount), context: 100, output: 4) { indices in
                calls += 1
                if indices.isEmpty { baseCalls += 1 }
                return (indices.isEmpty ? base : Prepared("over"), indices.isEmpty ? 20 : 200)
            }
            #expect(calls == 5 && result.preparationCalls == 5)
            #expect(calls == HistoryInputPreparation.maximumPreparationCalls)
            #expect(baseCalls == 1 && result.indices.isEmpty && result.input === base)
        }
    }

    @Test func fittingHistoryPreparesOnceAndKeepsThePreparedObject() async throws {
        var calls = 0
        let object = Prepared("all")
        let result = try await HistoryInputPreparation.prepare(recordByteCounts: [6, 2, 2], context: 65_536, output: 4_096) { indices in
            calls += 1
            #expect(indices == [0, 1, 2])
            return (object, 61_440)
        }
        #expect(calls == 1 && result.preparationCalls == 1)
        #expect(result.input === object)
        #expect(result.indices == [0, 1, 2])
    }

    @Test func overflowShortlistsInPriorityOrderAndReusesAcceptedInput() async throws {
        var visited: [[Int]] = []
        var objects: [String: Prepared] = [:]
        let result = try await HistoryInputPreparation.prepare(recordByteCounts: [6, 2, 2], context: 14, output: 4) { indices in
            visited.append(indices)
            let label = indices.map(String.init).joined(separator: ",")
            let object = Prepared(label); objects[label] = object
            let counts = ["0,1,2": 13, "": 2, "0": 11, "1": 8]
            return (object, counts[label]!)
        }
        #expect(visited == [[0, 1, 2], [], [0], [1]])
        #expect(result.indices == [1] && result.tokenCount == 8)
        #expect(result.input === objects["1"] && result.preparationCalls == 4)
    }

    @Test func mandatoryInputAboveSixtyThousandStillHasUsableHistorySpace() async throws {
        var visited: [[Int]] = []
        let result = try await HistoryInputPreparation.prepare(recordByteCounts: [400, 600], context: 65_536, output: 4_096) { indices in
            visited.append(indices)
            let count = indices.isEmpty ? 61_000 : (indices == [0] ? 61_440 : 62_000)
            return (Prepared("\(indices)"), count)
        }
        #expect(visited == [[0, 1], [], [0]])
        #expect(result.indices == [0] && result.tokenCount == 61_440)
        #expect(result.preparationCalls == 3)
    }

    @Test func mandatoryInputAtExactLimitUsesCachedBaseWithoutRetry() async throws {
        var visited: [[Int]] = []
        let base = Prepared("mandatory")
        let result = try await HistoryInputPreparation.prepare(recordByteCounts: [100], context: 65_536, output: 4_096) { indices in
            visited.append(indices)
            return (indices.isEmpty ? base : Prepared("full"), indices.isEmpty ? 61_440 : 61_441)
        }
        #expect(visited == [[0], []])
        #expect(result.input === base && result.indices.isEmpty && result.preparationCalls == 2)
    }

    @Test func estimatedEmptyShortlistDoesNotPretendHistoryWasExactlyTested() async throws {
        var calls = 0
        let base = Prepared("mandatory")
        let result = try await HistoryInputPreparation.prepare(recordByteCounts: [1], context: 14, output: 4) { indices in
            calls += 1
            return (indices.isEmpty ? base : Prepared("full"), indices.isEmpty ? 9 : 20)
        }
        #expect(calls == 2 && result.input === base && result.indices.isEmpty)
    }

    @Test func candidateTokenizerFailureStopsEvenWhenBaseFits() async {
        enum Failure: Error, Equatable { case tokenizer }
        var visited: [[Int]] = []
        await #expect(throws: Failure.tokenizer) {
            _ = try await HistoryInputPreparation.prepare(recordByteCounts: [8, 2], context: 100, output: 4) { indices -> (Prepared, Int) in
                visited.append(indices)
                if indices == [1] { throw Failure.tokenizer }
                return (Prepared("\(indices)"), indices.isEmpty ? 20 : 200)
            }
        }
        #expect(visited == [[0, 1], [], [1]])
    }

    @Test func invalidSizeHintsDoNotTriggerAnyPreparation() async {
        for sizes in [[], [0], [-1], [Int.max, 1]] {
            var calls = 0
            await #expect(throws: TokenBudgetError.invalidCount) {
                _ = try await HistoryInputPreparation.prepare(recordByteCounts: sizes, context: 100, output: 4) { _ in
                    calls += 1
                    return (Prepared("unused"), 1)
                }
            }
            #expect(calls == 0)
        }
    }

    @Test func oversizedMandatoryInputAndTokenizerFailureStop() async {
        await #expect(throws: TokenBudgetError.exceedsContext) {
            _ = try await HistoryInputPreparation.prepare(recordByteCounts: [1], context: 14, output: 4) { indices in
                (indices, indices.isEmpty ? 11 : 12)
            }
        }
        enum Failure: Error, Equatable { case tokenizer }
        var calls = 0
        await #expect(throws: Failure.tokenizer) {
            _ = try await HistoryInputPreparation.prepare(recordByteCounts: [1], context: 14, output: 4) { _ -> (String, Int) in
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
