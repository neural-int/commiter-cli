import MLX
import MLXLMCommon

/// A checkpoint before any phase-specific tokens. Unknown concrete types fail closed.
/// Model/runtime validation is a separate activation gate in the caller.
public final class PrefixCacheSnapshot {
    public let tokens: [Int]
    public let identity: String
    private let layers: [KVCache]

    public static func supportedTypeName(_ cache: KVCache) -> String? {
        let concrete = ObjectIdentifier(type(of: cache))
        if concrete == ObjectIdentifier(KVCacheSimple.self) { return "KVCacheSimple" }
        if concrete == ObjectIdentifier(RotatingKVCache.self) { return "RotatingKVCache" }
        if concrete == ObjectIdentifier(MambaCache.self) { return "MambaCache" }
        return nil
    }

    public static func supports(_ caches: [KVCache], verifiedTypes: Set<String>) -> Bool {
        !caches.isEmpty && caches.allSatisfy {
            guard let name = supportedTypeName($0) else { return false }
            return verifiedTypes.contains(name)
        }
    }

    public init?(tokens: [Int], identity: String, caches: [KVCache], verifiedTypes: Set<String>) {
        guard !tokens.isEmpty, Self.supports(caches, verifiedTypes: verifiedTypes),
              caches.allSatisfy({ cache in
                  if cache is MambaCache {
                      return cache.state.count == 2 && cache.state.allSatisfy { $0.size > 0 }
                  }
                  return cache.offset == tokens.count
              }), let copies = Self.copyLayers(caches) else { return nil }
        self.tokens = tokens
        self.identity = identity
        self.layers = copies
    }

    public func fork(tokens input: [Int], identity: String) -> [KVCache]? {
        guard self.identity == identity, input.count > tokens.count,
              Array(input.prefix(tokens.count)) == tokens else { return nil }
        return Self.copyLayers(layers)
    }

    private static func copyLayers(_ caches: [KVCache]) -> [KVCache]? {
        var result: [KVCache] = []
        for original in caches {
            let copied = original.copy()
            guard (original as AnyObject) !== (copied as AnyObject),
                  type(of: original) == type(of: copied),
                  original.offset == copied.offset, original.metaState == copied.metaState,
                  original.state.count == copied.state.count,
                  zip(original.state, copied.state).allSatisfy({ $0.shape == $1.shape && $0.dtype == $1.dtype }) else { return nil }
            result.append(copied)
        }
        eval(result)
        return result
    }
}
