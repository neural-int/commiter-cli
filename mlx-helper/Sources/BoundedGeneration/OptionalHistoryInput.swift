/// Host-filtered, already rendered shared-message fragments. No JSON parsing or
/// reserialization is needed in the helper, and file diffs occur only once on IPC.
public struct OptionalHistoryInput: Decodable, Sendable {
    public let before: String
    public let after: String
    public let records: [String]
    public let systemWithoutHistory: String

    enum CodingKeys: String, CodingKey {
        case before, after, records
        case systemWithoutHistory = "system_without_history"
    }

    public func sharedContent(indices: [Int]) -> String {
        var members: [String] = []
        if !before.isEmpty { members.append(before) }
        if !indices.isEmpty {
            members.append("\"codex_context\":[" + indices.map { records[$0] }.joined(separator: ",") + "]")
        }
        if !after.isEmpty { members.append(after) }
        return "{" + members.joined(separator: ",") + "}"
    }
}

public struct PreparedHistoryInput<Input> {
    public let input: Input
    public let tokenCount: Int
    public let indices: [Int]
    public let preparationCalls: Int
}

public enum HistoryInputPreparation {
    /// Prepare all safe records once in the common case. Only overflow invokes
    /// ordered whole-record admission. Keep the accepted prepared object; never
    /// tokenize it again before generation. Preparation errors are not omissions.
    public static func prepare<Input>(recordCount: Int, context: Int, output: Int,
        prepare: ([Int]) async throws -> (Input, Int)
    ) async throws -> PreparedHistoryInput<Input> {
        guard recordCount > 0, context > 0, output >= 0, output <= context else {
            throw TokenBudgetError.invalidCount
        }
        let limit = context - output
        let all = Array(0..<recordCount)
        let full = try await prepare(all)
        guard full.1 >= 0 else { throw TokenBudgetError.invalidCount }
        if full.1 <= limit {
            return PreparedHistoryInput(input: full.0, tokenCount: full.1, indices: all, preparationCalls: 1)
        }
        let base = try await prepare([])
        try TokenBudget.validate(context: context, prompt: base.1, output: output)
        var accepted = base
        var indices: [Int] = []
        var calls = 2
        for index in all {
            let trial = indices + [index]
            let candidate: (Input, Int)
            if trial == all { candidate = full }
            else { candidate = try await prepare(trial); calls += 1 }
            guard candidate.1 >= 0 else { throw TokenBudgetError.invalidCount }
            if candidate.1 <= limit { accepted = candidate; indices = trial }
        }
        return PreparedHistoryInput(input: accepted.0, tokenCount: accepted.1, indices: indices, preparationCalls: calls)
    }
}
