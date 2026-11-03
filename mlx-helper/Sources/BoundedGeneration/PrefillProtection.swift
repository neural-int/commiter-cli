/// Use the full template-applied prompt, including any reused prefix. A 64K
/// context ceiling by itself does not enable per-chunk synchronization/reclaim.
public enum PrefillProtection {
    public static let inputTokenBoundary = 12_000

    public static func enabled(inputTokens: Int) -> Bool {
        inputTokens > inputTokenBoundary
    }
}
