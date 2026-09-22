import Foundation

public enum IPCFramingError: Error, Equatable {
    case inputTooLarge
    case missingInput
    case unterminatedLine
    case trailingInput
}

public enum IPCFraming {
    public static func readLine(from handle: FileHandle, maximumBytes: Int) throws -> Data {
        guard maximumBytes > 0 else {
            throw IPCFramingError.inputTooLarge
        }

        var data = Data()
        while true {
            guard let chunk = try handle.read(upToCount: 4096), !chunk.isEmpty else {
                if data.isEmpty {
                    throw IPCFramingError.missingInput
                }
                throw IPCFramingError.unterminatedLine
            }

            if let newline = chunk.firstIndex(of: 0x0A) {
                let line = chunk[..<newline]
                guard data.count + line.count + 1 <= maximumBytes else {
                    throw IPCFramingError.inputTooLarge
                }
                data.append(line)

                if newline + 1 < chunk.endIndex {
                    throw IPCFramingError.trailingInput
                }

                // A line can end exactly at a read boundary. Confirm EOF so
                // trailing bytes in a later chunk cannot be accepted.
                if let trailing = try handle.read(upToCount: 1), !trailing.isEmpty {
                    throw IPCFramingError.trailingInput
                }
                return data
            }

            guard data.count + chunk.count < maximumBytes else {
                throw IPCFramingError.inputTooLarge
            }
            data.append(chunk)
        }
    }
}
