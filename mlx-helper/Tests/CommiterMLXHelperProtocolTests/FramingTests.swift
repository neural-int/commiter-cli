import Foundation
import Testing

@testable import CommiterMLXHelperProtocol

@Suite("IPC framing")
struct FramingTests {
    @Test("accepts one bounded line followed by EOF")
    func acceptsOneLine() throws {
        let input = try temporaryInput(Data("{\"ok\":true}\n".utf8))
        defer { try? FileManager.default.removeItem(at: input.url) }

        let result = try IPCFraming.readLine(from: input.handle, maximumBytes: 128)

        #expect(result == Data("{\"ok\":true}".utf8))
    }

    @Test("rejects trailing bytes in the same chunk")
    func rejectsTrailingBytesInSameChunk() throws {
        let input = try temporaryInput(Data("{}\n{}".utf8))
        defer { try? FileManager.default.removeItem(at: input.url) }

        #expect(throws: IPCFramingError.trailingInput) {
            _ = try IPCFraming.readLine(from: input.handle, maximumBytes: 128)
        }
    }

    @Test("rejects trailing bytes in a later chunk")
    func rejectsTrailingBytesInLaterChunk() throws {
        let pipe = Pipe()
        pipe.fileHandleForWriting.write(Data("{}\n".utf8))
        pipe.fileHandleForWriting.write(Data("{}".utf8))
        try pipe.fileHandleForWriting.close()

        #expect(throws: IPCFramingError.trailingInput) {
            _ = try IPCFraming.readLine(
                from: pipe.fileHandleForReading,
                maximumBytes: 128)
        }
    }

    @Test("rejects a line that exceeds the bound")
    func rejectsOversizedLine() throws {
        let input = try temporaryInput(Data("12345\n".utf8))
        defer { try? FileManager.default.removeItem(at: input.url) }

        #expect(throws: IPCFramingError.inputTooLarge) {
            _ = try IPCFraming.readLine(from: input.handle, maximumBytes: 4)
        }
    }

    @Test("rejects a non-empty line without a delimiter")
    func rejectsUnterminatedLine() throws {
        let input = try temporaryInput(Data("{}".utf8))
        defer { try? FileManager.default.removeItem(at: input.url) }

        #expect(throws: IPCFramingError.unterminatedLine) {
            _ = try IPCFraming.readLine(from: input.handle, maximumBytes: 128)
        }
    }

    private func temporaryInput(_ data: Data) throws -> (url: URL, handle: FileHandle) {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("commiter-mlx-framing-" + UUID().uuidString)
        try data.write(to: url, options: .atomic)
        return (url, try FileHandle(forReadingFrom: url))
    }
}
