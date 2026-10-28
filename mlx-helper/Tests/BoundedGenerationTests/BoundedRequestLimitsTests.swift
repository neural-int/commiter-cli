import Testing
@testable import BoundedGeneration

@Suite("Three-phase host/helper context contract")
struct BoundedRequestLimitsTests {
    @Test func hostTiersAcceptedForEveryProfile() throws {
        for profile in ["bounded-grouping", "bounded-category", "bounded-text"] {
            let output = profile == "bounded-category" ? 512 : 768
            for context in [16384, 32768, 65536] {
                try BoundedRequestLimits.validate(profile: profile, context: context, output: output)
                try TokenBudget.validate(context: context, prompt: context - output, output: output)
                #expect(throws: TokenBudgetError.exceedsContext) {
                    try TokenBudget.validate(context: context, prompt: context - output + 1, output: output)
                }
            }
        }
    }

    @Test func invalidContextAndOutputRemainRejected() {
        for context in [-1, 0, 65537, Int.max] {
            #expect(throws: BoundedRequestError.invalidContext) {
                try BoundedRequestLimits.validate(profile: "bounded-text", context: context, output: 768)
            }
        }
        for profile in ["bounded-grouping", "bounded-category", "bounded-text"] {
            let wrongOutput = profile == "bounded-category" ? 768 : 512
            #expect(throws: BoundedRequestError.invalidOutputBudget) {
                try BoundedRequestLimits.validate(profile: profile, context: 65536, output: wrongOutput)
            }
        }
        #expect(throws: BoundedRequestError.invalidProfile) {
            try BoundedRequestLimits.validate(profile: "unknown", context: 65536, output: 768)
        }
    }
}
