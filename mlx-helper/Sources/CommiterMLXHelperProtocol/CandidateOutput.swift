import Foundation

public enum CandidateOutputError: Error { case invalid }

public enum CandidateOutput {
    // Strip only the registered Gemma final-channel delimiter and one JSON
    // fence. Never repair JSON or retain/return native thought text.
    public static func finalJSON(_ text: String, native: Bool) throws -> String {
        var value = text.trimmingCharacters(in: .whitespacesAndNewlines)
        if native {
            // The bounded processor enables JSON grammar immediately after
            // token 101. There is no free-form "final" label after this token.
            let pieces = value.components(separatedBy: "<channel|>")
            guard value.hasPrefix("<|channel>thought"), pieces.count == 2 else {
                throw CandidateOutputError.invalid
            }
            value = pieces[1].trimmingCharacters(in: .whitespacesAndNewlines)
        }
        if value.hasPrefix("```json\n"), value.hasSuffix("\n```") {
            value = String(value.dropFirst(8).dropLast(4)).trimmingCharacters(in: .whitespacesAndNewlines)
        }
        guard let data = value.data(using: .utf8),
              (try? JSONSerialization.jsonObject(with: data)) is [String: Any] else {
            throw CandidateOutputError.invalid
        }
        return value
    }
}
