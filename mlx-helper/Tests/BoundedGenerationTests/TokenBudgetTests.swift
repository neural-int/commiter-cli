import Testing
@testable import BoundedGeneration

@Suite("Full prompt and output reservation")
struct TokenBudgetTests {
    @Test func exactBoundaryFits() throws {
        try TokenBudget.validate(context: 8192, prompt: 6144, output: 2048)
    }
    @Test func oneTokenOverFails() {
        #expect(throws: TokenBudgetError.exceedsContext) {
            try TokenBudget.validate(context: 8192, prompt: 6145, output: 2048)
        }
    }
    @Test func promptExceedsContextFails() {
        #expect(throws: TokenBudgetError.exceedsContext) {
            try TokenBudget.validate(context: 8192, prompt: 8193, output: 0)
        }
    }
    @Test func measuredA136RequiresLargerContext() throws {
        #expect(throws: TokenBudgetError.exceedsContext) {
            try TokenBudget.validate(context: 8192, prompt: 7933, output: 2048)
        }
        try TokenBudget.validate(context: 16384, prompt: 7933, output: 2048)
    }
    @Test func unspecifiedContextRetainsLegacyBehavior() throws {
        try TokenBudget.validate(context: 0, prompt: Int.max, output: Int.max)
    }
    @Test func negativeCountsFail() {
        for values in [(-1, 0, 0), (0, -1, 0), (0, 0, -1)] {
            #expect(throws: TokenBudgetError.invalidCount) {
                try TokenBudget.validate(context: values.0, prompt: values.1, output: values.2)
            }
        }
    }
    @Test func maximumCountsAvoidOverflow() throws {
        try TokenBudget.validate(context: Int.max, prompt: Int.max - 1, output: 1)
        #expect(throws: TokenBudgetError.exceedsContext) {
            try TokenBudget.validate(context: Int.max, prompt: Int.max, output: 1)
        }
    }
}
