import MLX
import MLXLMCommon

/// Research adapter for TokenIterator's one-token lookahead. State is committed
/// before the next sampling mask, while that step's forward logits are scheduled.
/// A final unreturned lookahead token is intentionally never committed.
public struct OverlappedTokenProcessor: LogitProcessor {
    private final class Storage {
        var processor: any LogitProcessor
        var pending: MLXArray?
        init(_ processor: any LogitProcessor, pending: MLXArray? = nil) {
            self.processor = processor
            self.pending = pending
        }
    }
    private let storage: Storage
    public init(_ processor: any LogitProcessor) { storage = Storage(processor) }
    private init(storage: Storage) { self.storage = storage }
    public mutating func prompt(_ prompt: MLXArray) { storage.processor.prompt(prompt) }
    public func process(logits: MLXArray) -> MLXArray {
        if let previous = storage.pending {
            // Schedule the unconstrained forward before any CPU read in didSample.
            // No next token is sampled until the previous state has been committed.
            asyncEval(logits)
            storage.pending = nil
            storage.processor.didSample(token: previous)
        }
        return storage.processor.process(logits: logits)
    }
    public mutating func didSample(token: MLXArray) {
        precondition(storage.pending == nil, "Previous token must be committed before sampling")
        storage.pending = token
    }
    public func copy() -> Self {
        // MLXArray is retained as an immutable value; matcher/counter state is copied.
        Self(storage: Storage(storage.processor.copy(), pending: storage.pending))
    }
}
