import XCTest
@testable import BoundedGeneration

final class PrefillProtectionTests: XCTestCase {
    func testOutputReservationRemainsInside64K() throws {
        try TokenBudget.validate(context: 65_536, prompt: 61_440, output: 4_096)
        XCTAssertThrowsError(try TokenBudget.validate(context: 65_536, prompt: 61_441, output: 4_096))
    }
    func testStrictTwelveThousandTokenBoundary() {
        for count in [0, 5_123, 11_999, 12_000] {
            XCTAssertFalse(PrefillProtection.enabled(inputTokens: count))
        }
        for count in [12_001, 12_284, 61_440] {
            XCTAssertTrue(PrefillProtection.enabled(inputTokens: count))
        }
    }
}
