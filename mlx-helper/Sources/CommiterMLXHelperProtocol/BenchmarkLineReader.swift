import Foundation

// Bounded multi-request reader used only by the opt-in, single-cycle benchmark.
// Unlike production readLine, it preserves subsequent lines and need not await EOF.
public final class BenchmarkLineReader {
    private let handle: FileHandle
    private let maximumBytes: Int
    private var buffer = Data()

    public init(handle: FileHandle, maximumBytes: Int) {
        self.handle = handle
        self.maximumBytes = maximumBytes
    }

    public func next() throws -> Data? {
        guard maximumBytes > 0 else { throw IPCFramingError.inputTooLarge }
        while true {
            if let newline = buffer.firstIndex(of: 0x0A) {
                let length = buffer.distance(from: buffer.startIndex, to: newline)
                guard length + 1 <= maximumBytes else { throw IPCFramingError.inputTooLarge }
                let line = Data(buffer.prefix(length))
                buffer.removeFirst(length + 1)
                return line
            }
            guard buffer.count < maximumBytes else { throw IPCFramingError.inputTooLarge }
            let chunk = handle.availableData
            if chunk.isEmpty {
                if buffer.isEmpty { return nil }
                throw IPCFramingError.unterminatedLine
            }
            // availableData is bounded by the OS pipe/read buffer; each complete
            // line is checked before decoding, including lines read together.
            buffer.append(chunk)
        }
    }
}
