import Foundation
import Testing
@testable import BoundedGeneration

@Suite("Host-compiled attributed history display")
struct HistoryDisplayTests {
    @Test func optionalElementsNeverLeaveDanglingCommasOrCommands() throws {
        let source = #"{"parts":[{"literal":"["},{"parts":[{"literal":"H1","require":[0]},{"literal":"H2","require":[1]},{"literal":"command1","require":[0,2]},{"literal":"command2","require":[0,3]}],"separator":","},{"literal":"]"}]}"#
        let template = try JSONDecoder().decode(DisplayTemplate.self, from: Data(source.utf8))
        #expect(template.render(selected: [0,1,2,3]) == "[H1,H2,command1,command2]")
        #expect(template.render(selected: [1,2,3]) == "[H2]")
        #expect(template.render(selected: []) == "[]")
        #expect(template.render(selected: [0,3]) == "[H1,command2]")
    }

    @Test func omittedCandidateUsesHostSuppliedUncertaintyAndPreservesLiteralText() throws {
        let source = #"{"parts":[{"literal":"raw\n\\n\"quoted\""},{"literal":"matched","require":[0],"if_missing":[{"literal":"ambiguous:candidate_context_omitted"}]}],"separator":"|"}"#
        let template = try JSONDecoder().decode(DisplayTemplate.self, from: Data(source.utf8))
        #expect(template.render(selected: [0]) == "raw\n\\n\"quoted\"|matched")
        #expect(template.render(selected: []) == "raw\n\\n\"quoted\"|ambiguous:candidate_context_omitted")
    }
}
