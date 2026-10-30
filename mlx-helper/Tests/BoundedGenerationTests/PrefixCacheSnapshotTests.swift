import Testing
import MLX
import MLXLMCommon
@testable import BoundedGeneration

struct PrefixCacheSnapshotTests {
    let verified: Set<String> = ["KVCacheSimple", "RotatingKVCache", "MambaCache"]

    private func append(_ values: [Float], to cache: KVCache) -> [Float] {
        let array = MLXArray(values).reshaped([1, 1, values.count, 1])
        let (keys, _) = cache.update(keys: array, values: array)
        eval(cache)
        return keys.asArray(Float.self)
    }

    @Test func attentionResumptionAndSiblingIsolation() throws {
        // Rotating cache crosses its capacity after resuming: ring metadata matters.
        for cache: KVCache in [KVCacheSimple(), RotatingKVCache(maxSize: 4)] {
            _ = append([1, 2, 3], to: cache)
            let snapshot = try #require(PrefixCacheSnapshot(tokens: [1, 2, 3], identity: "model/runtime/config", caches: [cache], verifiedTypes: verified))
            let fork = try #require(snapshot.fork(tokens: [1, 2, 3, 4, 5], identity: "model/runtime/config"))
            let sibling = try #require(snapshot.fork(tokens: [1, 2, 3, 9], identity: "model/runtime/config"))
            let resumed = append([4, 5], to: fork[0])
            let reference: KVCache = cache is RotatingKVCache ? RotatingKVCache(maxSize: 4) : KVCacheSimple()
            _ = append([1, 2, 3], to: reference)
            #expect(resumed == append([4, 5], to: reference))
            #expect(fork[0].offset == reference.offset)
            #expect(sibling[0].offset == 3)
            _ = append([9], to: sibling[0])
            let restored = try #require(snapshot.fork(tokens: [1, 2, 3, 8], identity: "model/runtime/config"))
            #expect(restored[0].offset == 3)
            #expect(restored[0].state[0].asArray(Float.self) == [1, 2, 3])
        }
    }

    @Test func recurrentStateAndPositionRemainIndependent() throws {
        let cache = MambaCache(leftPadding: [1])
        cache[0] = MLXArray([Float(2), 3]); cache[1] = MLXArray([Float(4), 5]); cache.offset = 3
        let snapshot = try #require(PrefixCacheSnapshot(tokens: [1, 2, 3], identity: "same", caches: [cache], verifiedTypes: verified))
        let fork = try #require(snapshot.fork(tokens: [1, 2, 3, 4], identity: "same"))
        let recurrent = try #require(fork[0] as? MambaCache)
        #expect(recurrent.metaState == cache.metaState)
        recurrent[0] = recurrent[0]! + 10
        recurrent[1] = recurrent[1]! * 2
        recurrent.offset += 1
        let again = try #require(snapshot.fork(tokens: [1, 2, 3, 4], identity: "same"))
        #expect(again[0].state[0].asArray(Float.self) == [2, 3])
        #expect(again[0].state[1].asArray(Float.self) == [4, 5])
        #expect(again[0].offset == 3)
        #expect(cache.state[0].asArray(Float.self) == [2, 3])
    }

    @Test func unsupportedAndUnverifiedTypesAndIdentityFallBack() throws {
        #expect(!PrefixCacheSnapshot.supports([ArraysCache(size: 2)], verifiedTypes: verified))
        #expect(!PrefixCacheSnapshot.supports([KVCacheSimple()], verifiedTypes: []))
        let cache = KVCacheSimple(); _ = append([1, 2, 3], to: cache)
        let snapshot = try #require(PrefixCacheSnapshot(tokens: [1, 2, 3], identity: "same", caches: [cache], verifiedTypes: verified))
        #expect(snapshot.fork(tokens: [1, 2, 8, 4], identity: "same") == nil)
        #expect(snapshot.fork(tokens: [1, 2, 3, 4], identity: "different") == nil)
        #expect(snapshot.fork(tokens: [1, 2, 3], identity: "same") == nil)
        #expect(PrefixCacheSnapshot(tokens: [1, 2], identity: "same", caches: [cache], verifiedTypes: verified) == nil)
    }
}
