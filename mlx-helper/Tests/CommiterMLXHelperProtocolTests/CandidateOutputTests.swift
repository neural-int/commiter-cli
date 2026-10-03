import Testing
@testable import CommiterMLXHelperProtocol

@Suite("Candidate final-channel boundary")
struct CandidateOutputTests {
    @Test func discardsNativeContent() throws {
        #expect(try CandidateOutput.finalJSON("<|channel>thought private text<channel|>{\"G001\":\"fix\"}", native: true) == "{\"G001\":\"fix\"}")
    }
    @Test func rejectsMissingOrRepeatedFinalChannel() {
        for text in ["{\"x\":1}", "<|channel>thought<channel|>{}<channel|>{}", "<|channel>thought<channel|>final{}", "untrusted prefix<channel|>{}"] {
            #expect(throws: CandidateOutputError.self) { try CandidateOutput.finalJSON(text, native: true) }
        }
    }
    @Test func permitsSingleJSONFenceButNoProseOrPartialOutput() throws {
        #expect(try CandidateOutput.finalJSON("```json\n{}\n```", native: false) == "{}")
        for text in ["prose {}", "{", "{}{}", "[]", "null"] {
            #expect(throws: CandidateOutputError.self) { try CandidateOutput.finalJSON(text, native: false) }
        }
    }
}
