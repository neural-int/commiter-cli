public enum BoundedRequestError: Error, Equatable {
    case invalidProfile
    case invalidContext
    case invalidOutputBudget
}

// The host selects a context tier; the helper enforces the same 64k ceiling
// and the existing per-profile output reservation before loading the model.
public enum BoundedRequestLimits {
    public static let maximumContext = 65536

    public static func outputBudget(profile: String) throws -> Int {
        switch profile {
        case "bounded-grouping", "bounded-text": return 768
        case "bounded-category": return 512
        default: throw BoundedRequestError.invalidProfile
        }
    }

    public static func validate(profile: String, context: Int, output: Int) throws {
        let budget = try outputBudget(profile: profile)
        guard context > 0, context <= maximumContext else {
            throw BoundedRequestError.invalidContext
        }
        guard output == budget else {
            throw BoundedRequestError.invalidOutputBudget
        }
    }
}
