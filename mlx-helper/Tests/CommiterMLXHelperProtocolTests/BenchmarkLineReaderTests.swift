import Foundation
import Testing
@testable import CommiterMLXHelperProtocol

struct BenchmarkLineReaderTests {
    @Test func retainsFollowingLineAndHandlesEOF() throws {
        let pipe = Pipe()
        pipe.fileHandleForWriting.write(Data("{}\n{\"next\":true}\n".utf8))
        try pipe.fileHandleForWriting.close()
        let reader = BenchmarkLineReader(handle: pipe.fileHandleForReading, maximumBytes: 32)
        #expect(try reader.next() == Data("{}".utf8))
        #expect(try reader.next() == Data("{\"next\":true}".utf8))
        #expect(try reader.next() == nil)
    }
    @Test func rejectsOversizedAndUnterminatedLines() throws {
        for (data, error) in [("12345\n", IPCFramingError.inputTooLarge), ("{}", .unterminatedLine)] {
            let pipe = Pipe()
            pipe.fileHandleForWriting.write(Data(data.utf8))
            try pipe.fileHandleForWriting.close()
            let reader = BenchmarkLineReader(handle: pipe.fileHandleForReading, maximumBytes: 5)
            #expect(throws: error) { _ = try reader.next() }
        }
    }
}
