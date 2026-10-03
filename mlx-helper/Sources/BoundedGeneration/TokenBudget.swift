public enum TokenBudgetError: Error, Equatable {
    case invalidCount
    case exceedsContext
}

public enum TokenBudget {
    public static func validate(context: Int, prompt: Int, output: Int) throws {
        guard context >= 0, prompt >= 0, output >= 0 else {
            throw TokenBudgetError.invalidCount
        }
        guard context > 0 else { return }
        guard prompt <= context, output <= context - prompt else {
            throw TokenBudgetError.exceedsContext
        }
    }
}
