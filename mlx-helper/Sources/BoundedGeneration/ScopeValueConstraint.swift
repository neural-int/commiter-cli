import Foundation

/// Additional runtime language constraint, intersected with the JSON matcher.
/// Only completed values for the JSON key `scope` are inspected.
public enum ScopeValueConstraint {
    public static func allows(_ prefix: String, forbidden: Set<String>) -> Bool {
        let chars = Array(prefix)
        var i = 0
        var key: String?
        while i < chars.count {
            guard chars[i] == "\"" else { i += 1; continue }
            let start = i
            i += 1
            var escaped = false
            while i < chars.count {
                if escaped { escaped = false }
                else if chars[i] == "\\" { escaped = true }
                else if chars[i] == "\"" { break }
                i += 1
            }
            if i == chars.count { return true }
            let literal = String(chars[start...i])
            guard let data = literal.data(using: .utf8),
                  let text = try? JSONSerialization.jsonObject(with: data, options: .fragmentsAllowed) as? String else { return false }
            i += 1
            var next = i
            while next < chars.count && chars[next].isWhitespace { next += 1 }
            // A key whose colon is not generated yet remains undecided.
            if next == chars.count {
                return key != "scope" || !forbidden.contains(normalized(text))
            }
            if chars[next] == ":" { key = text }
            else {
                if key == "scope" && forbidden.contains(normalized(text)) { return false }
                key = nil
            }
        }
        return true
    }

    public static func normalized(_ text: String) -> String {
        let whitespace = CharacterSet(charactersIn: "\u{0009}\u{000A}\u{000B}\u{000C}\u{000D}\u{0020}\u{0085}\u{00A0}\u{1680}\u{2000}\u{2001}\u{2002}\u{2003}\u{2004}\u{2005}\u{2006}\u{2007}\u{2008}\u{2009}\u{200A}\u{2028}\u{2029}\u{202F}\u{205F}\u{3000}")
        // Go strings.ToLower uses the simple mapping for U+0130.
        return text.trimmingCharacters(in: whitespace).replacingOccurrences(of: "\u{0130}", with: "i").lowercased()
    }
}
