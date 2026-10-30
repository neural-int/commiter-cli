import Foundation

public enum OutputConstraint: String, Codable, Sendable {
    case jsonSchema = "json_schema"
    case none
}

// Derive documentation from the exact request schema, including dynamic IDs.
// Reject unsupported example construction instead of silently showing an invalid example.
public enum StructuredOutputPrompt {
    public enum PromptError: Error { case unsupportedSchema }
    public static func instruction(schema: String) throws -> String {
        guard let data = schema.data(using: .utf8),
              let root = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw PromptError.unsupportedSchema
        }
        var fields: [String] = []
        try describe(root, path: "$", requirement: "required", fields: &fields)
        let example = try sample(root)
        let exampleData = try JSONSerialization.data(withJSONObject: example, options: [.sortedKeys, .fragmentsAllowed])
        return """
        Output contract for the FINAL ANSWER only:
        Return exactly one JSON value matching the schema below. No Markdown fences, prose, or extra fields.
        When thinking is enabled, think freely in the model's native thinking section; apply this contract only after that section ends. When thinking is disabled, begin the JSON answer directly.
        Field names, types, and required/optional fields:
        \(fields.joined(separator: "\n"))
        Exact JSON Schema (including enum values, lengths, patterns, and other format constraints):
        \(schema)
        Structural example only; choose actual values from the current Git evidence and task rules, never copy this example as a factual answer:
        \(String(decoding: exampleData, as: UTF8.self))
        """
    }
    private static func describe(_ schema: [String: Any], path: String, requirement: String, fields: inout [String]) throws {
        let type = schema["type"] as? String ?? "schema-defined"
        fields.append("\(path): type=\(type); \(requirement)")
        if let properties = schema["properties"] as? [String: [String: Any]] {
            let required = Set(schema["required"] as? [String] ?? [])
            for name in properties.keys.sorted() {
                try describe(properties[name]!, path: path + "." + name, requirement: required.contains(name) ? "required" : "optional", fields: &fields)
            }
        }
        if let items = schema["items"] as? [String: Any] {
            try describe(items, path: path + "[]", requirement: "array item", fields: &fields)
        }
    }
    private static func sample(_ schema: [String: Any]) throws -> Any {
        if let constant = schema["const"] { return constant }
        if let values = schema["enum"] as? [Any], let first = values.first { return first }
        // These branches are not part of the current phase schemas. Fail closed on expansion.
        guard schema["$ref"] == nil, schema["oneOf"] == nil, schema["anyOf"] == nil,
              schema["allOf"] == nil, schema["not"] == nil else { throw PromptError.unsupportedSchema }
        switch schema["type"] as? String {
        case "object":
            let properties = schema["properties"] as? [String: [String: Any]] ?? [:]
            var result: [String: Any] = [:]
            for name in schema["required"] as? [String] ?? [] {
                guard let property = properties[name] else { throw PromptError.unsupportedSchema }
                result[name] = try sample(property)
            }
            return result
        case "array":
            let count = schema["minItems"] as? Int ?? 0
            guard count >= 0, count <= 1024, count <= (schema["maxItems"] as? Int ?? 1024),
                  let items = schema["items"] as? [String: Any] else { throw PromptError.unsupportedSchema }
            if schema["uniqueItems"] as? Bool == true, count > 1 {
                guard let values = items["enum"] as? [Any], values.count >= count else { throw PromptError.unsupportedSchema }
                return Array(values.prefix(count))
            }
            return try (0..<count).map { _ in try sample(items) }
        case "string":
            let count = max(1, schema["minLength"] as? Int ?? 0)
            guard count <= 1024, count <= (schema["maxLength"] as? Int ?? 1024) else { throw PromptError.unsupportedSchema }
            let value = String(repeating: "例", count: count)
            if let pattern = schema["pattern"] as? String {
                let regex = try NSRegularExpression(pattern: pattern)
                guard regex.firstMatch(in: value, range: NSRange(value.startIndex..., in: value)) != nil else { throw PromptError.unsupportedSchema }
            }
            return value
        case "boolean": return false
        case "integer", "number":
            let value = schema["minimum"] as? Double ?? 0
            guard value <= (schema["maximum"] as? Double ?? value), schema["multipleOf"] == nil,
                  schema["exclusiveMinimum"] == nil, schema["exclusiveMaximum"] == nil else { throw PromptError.unsupportedSchema }
            return value
        case "null": return NSNull()
        default: throw PromptError.unsupportedSchema
        }
    }
}
