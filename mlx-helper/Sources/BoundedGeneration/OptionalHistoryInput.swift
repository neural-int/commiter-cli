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
    public static let maximumCandidatePreparations = 3
    public static let maximumPreparationCalls = 2 + maximumCandidatePreparations

    /// All-fit stays one preparation. On overflow, secure the mandatory input
    /// first, then use observed token/byte density only to shortlist records.
    /// Exact validation alone admits an input; keep that same prepared object.
    public static func prepare<Input>(recordByteCounts: [Int], context: Int, output: Int,
        prepare: ([Int]) async throws -> (Input, Int)
    ) async throws -> PreparedHistoryInput<Input> {
        guard !recordByteCounts.isEmpty, context > 0, output >= 0, output <= context else {
            throw TokenBudgetError.invalidCount
        }
        var totalBytes = 0
        for bytes in recordByteCounts {
            let sum = totalBytes.addingReportingOverflow(bytes)
            guard bytes > 0, !sum.overflow else { throw TokenBudgetError.invalidCount }
            totalBytes = sum.partialValue
        }
        let limit = context - output
        let all = Array(recordByteCounts.indices)
        let full = try await prepare(all)
        guard full.1 >= 0 else { throw TokenBudgetError.invalidCount }
        if full.1 <= limit {
            return PreparedHistoryInput(input: full.0, tokenCount: full.1, indices: all, preparationCalls: 1)
        }
        let base = try await prepare([])
        try TokenBudget.validate(context: context, prompt: base.1, output: output)
        var calls = 2
        let remaining = limit - base.1
        // Full-width arithmetic avoids overflow when scaling the byte heuristic.
        // Since full exceeds the limit, the quotient is strictly below bytes.
        func estimatedByteBudget(bytes: Int, observedTokens: Int) -> Int {
            let product = UInt(bytes).multipliedFullWidth(by: UInt(remaining))
            return Int(UInt(observedTokens).dividingFullWidth(product).quotient)
        }
        var byteBudget = estimatedByteBudget(bytes: totalBytes, observedTokens: full.1 - base.1)
        for _ in 0..<maximumCandidatePreparations {
            var indices: [Int] = []
            var selectedBytes = 0
            // Preserve the host's priority order. Skip whole records that do not
            // fit the estimate; do not cut Git diffs or truncate history text.
            for index in all where recordByteCounts[index] <= byteBudget - selectedBytes {
                indices.append(index)
                selectedBytes += recordByteCounts[index]
            }
            if indices.isEmpty { break }
            let candidate = try await prepare(indices)
            calls += 1
            guard candidate.1 >= 0 else { throw TokenBudgetError.invalidCount }
            if candidate.1 <= limit {
                return PreparedHistoryInput(input: candidate.0, tokenCount: candidate.1, indices: indices, preparationCalls: calls)
            }
            // Adapt to the last exact measurement, but halve retained bytes at
            // least. The next shortlist can admit smaller, lower-priority records.
            byteBudget = min(selectedBytes / 2,
                estimatedByteBudget(bytes: selectedBytes, observedTokens: candidate.1 - base.1))
        }
        // Already validated: fallback needs no further prepare/tokenize call.
        return PreparedHistoryInput(input: base.0, tokenCount: base.1, indices: [], preparationCalls: calls)
    }
}
