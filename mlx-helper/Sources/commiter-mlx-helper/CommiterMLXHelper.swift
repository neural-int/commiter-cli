import Foundation
import HuggingFace
import MLX
import MLXGuidedGeneration
import MLXHuggingFace
import MLXLLM
import MLXLMCommon
import Tokenizers
import CommiterMLXHelperProtocol
import BoundedGeneration

private let maxRequestBytes = 1_048_576
private let maxResponseBytes = 4_194_304

private enum StopReason: String, Encodable {
    case completed
    case maxTokens = "max_tokens"
    case timeout
    case cancelled
    case grammarFailure = "grammar_failure"
    case internalError = "internal_error"
}

private struct Message: Decodable, Sendable {
    let role: Chat.Message.Role
    let content: String

    init(from decoder: Swift.Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let rawRole = try container.decode(String.self, forKey: .role)
        guard let role = Chat.Message.Role(rawValue: rawRole) else {
            throw HelperError.malformedRequest
        }
        self.role = role
        content = try container.decode(String.self, forKey: .content)
    }

    private enum CodingKeys: String, CodingKey {
        case role
        case content
    }
}

private enum JSONValue: Decodable {
    case object([String: JSONValue])
    case array([JSONValue])
    case string(String)
    case number(Double)
    case boolean(Bool)
    case null

    init(from decoder: Swift.Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
        } else if let value = try? container.decode([String: JSONValue].self) {
            self = .object(value)
        } else if let value = try? container.decode([JSONValue].self) {
            self = .array(value)
        } else if let value = try? container.decode(String.self) {
            self = .string(value)
        } else if let value = try? container.decode(Double.self) {
            self = .number(value)
        } else if let value = try? container.decode(Bool.self) {
            self = .boolean(value)
        } else {
            throw HelperError.malformedRequest
        }
    }

    var foundationValue: Any {
        switch self {
        case .object(let value):
            return value.mapValues(\.foundationValue)
        case .array(let value):
            return value.map(\.foundationValue)
        case .string(let value):
            return value
        case .number(let value):
            return value
        case .boolean(let value):
            return value
        case .null:
            return NSNull()
        }
    }
}

private struct Request: Decodable {
    let schema: JSONValue
    let messages: [Message]
    let contextTokens: Int
    let outputTokens: Int
    let model: String?
    let modelPath: String?
    let generationProfile: String?

    enum CodingKeys: String, CodingKey {
        case schema
        case messages
        case contextTokens = "context_tokens"
        case outputTokens = "output_tokens"
        case model
        case modelPath = "model_path"
        case generationProfile = "generation_profile"
    }

    init(from decoder: Swift.Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        schema = try container.decode(JSONValue.self, forKey: .schema)
        messages = try container.decode([Message].self, forKey: .messages)
        contextTokens = try container.decodeIfPresent(Int.self, forKey: .contextTokens) ?? 0
        outputTokens = try container.decodeIfPresent(Int.self, forKey: .outputTokens) ?? 0
        model = try container.decodeIfPresent(String.self, forKey: .model)
        modelPath = try container.decodeIfPresent(String.self, forKey: .modelPath)
        generationProfile = try container.decodeIfPresent(String.self, forKey: .generationProfile)
        guard !messages.isEmpty, contextTokens >= 0, outputTokens >= 0 else {
            throw HelperError.malformedRequest
        }
    }
}

private struct Response: Encodable {
    let ok: Bool
    let stopReason: StopReason
    let generatedJSON: String?
    let model: String?
    let runtime: String?
    let errorClass: String?
    var generationProfile: String? = nil

    enum CodingKeys: String, CodingKey {
        case ok
        case stopReason = "stop_reason"
        case generatedJSON = "generated_json"
        case model
        case runtime
        case errorClass = "error_class"
        case generationProfile = "generation_profile"
    }
}

private enum HelperError: Error {
    case inputTooLarge
    case missingInput
    case trailingInput
    case malformedRequest
    case responseTooLarge
    case modelUnavailable
    case grammarFailure
    case maxTokens
    case cancelled
}

private final class OutputBox: @unchecked Sendable {
    var value = ""
}

@main
struct CommiterMLXHelper {
    static func main() async {
        if CommandLine.arguments == [CommandLine.arguments[0], "--smoke-metal"] {
            let ready = Device.withDefaultDevice(.gpu) {
                let result = add(MLXArray([1.0 as Float]), MLXArray([2.0 as Float]))
                eval(result)
                return result.item(Float.self) == 3
            }
            guard ready else { exit(1) }
            FileHandle.standardOutput.write(Data("metal_ok\n".utf8))
            return
        }
        do {
            let request = try decodeRequest(try readBoundedRequest())
            let response = try await handle(request)
            try writeResponse(response)
        } catch {
            do {
                try writeResponse(Response(
                    ok: false,
                    stopReason: classify(error),
                    generatedJSON: nil,
                    model: nil,
                    runtime: "mlx",
                    errorClass: classifyError(error)
                ))
            } catch {
                FileHandle.standardError.write(Data("response_write_failed\n".utf8))
                exit(1)
            }
        }
    }

    private static func readBoundedRequest() throws -> Data {
        do {
            return try IPCFraming.readLine(
                from: FileHandle.standardInput,
                maximumBytes: maxRequestBytes)
        } catch IPCFramingError.inputTooLarge {
            throw HelperError.inputTooLarge
        } catch IPCFramingError.missingInput {
            throw HelperError.missingInput
        } catch IPCFramingError.unterminatedLine {
            throw HelperError.malformedRequest
        } catch IPCFramingError.trailingInput {
            throw HelperError.trailingInput
        }
    }

    private static func decodeRequest(_ data: Data) throws -> Request {
        guard !data.isEmpty else {
            throw HelperError.malformedRequest
        }
        do {
            return try JSONDecoder().decode(Request.self, from: data)
        } catch {
            throw HelperError.malformedRequest
        }
    }

    private static func handle(_ request: Request) async throws -> Response {
        guard let modelPath = request.modelPath?.trimmingCharacters(in: .whitespacesAndNewlines),
              !modelPath.isEmpty,
              isModelDirectory(modelPath) else {
            throw HelperError.modelUnavailable
        }
        let modelID = request.model?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
            ? request.model!
            : URL(filePath: modelPath).lastPathComponent
        if let profile = request.generationProfile {
            guard ["bounded-grouping", "bounded-category", "bounded-text"].contains(profile),
                  modelID == "mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2" else {
                throw HelperError.malformedRequest
            }
        }
        let schemaData = try JSONSerialization.data(withJSONObject: request.schema.foundationValue, options: [.sortedKeys])
        guard let schema = String(data: schemaData, encoding: .utf8) else {
            throw HelperError.malformedRequest
        }
        let messages = request.messages
        let maxTokens = min(max(request.outputTokens == 0 ? 1024 : request.outputTokens, 1), 8192)

        do {
            try Task.checkCancellation()
            let modelDirectory = URL(fileURLWithPath: modelPath, isDirectory: true)
            let container = try await loadModelContainer(
                from: modelDirectory,
                using: #huggingFaceTokenizerLoader())
            let outputBox = OutputBox()
            let generated = try await container.perform { context in
                try Task.checkCancellation()
                let chat = messages.map { Chat.Message(role: $0.role, content: $0.content) }
                if let profile = request.generationProfile {
                    return try await generateBounded(request, profile: profile, schema: schema, context: context)
                }
                let grammarVocab = TokenizerVocabExtractor.extractForGrammar(from: context.tokenizer)
                let grammarTokenizer = try GrammarTokenizer(
                    vocab: grammarVocab.vocab,
                    vocabType: grammarVocab.vocabType,
                    eosTokenId: Int32(context.tokenizer.eosTokenId ?? 0)
                )
                let constraint = try GrammarConstraint(
                    tokenizer: grammarTokenizer,
                    jsonSchema: schema,
                    fastForward: true,
                    hostTokenizer: context.tokenizer
                )
                let input = try await context.processor.prepare(input: UserInput(chat: chat))
                do {
                    _ = try GuidedGenerationLoop.run(
                        input: input,
                        context: context,
                        constraint: constraint,
                        maxTokens: maxTokens,
                        vocabSize: grammarTokenizer.vocabSize,
                        hardReserve: 0,
                        closingBias: nil
                    ) { delta in
                        if Task.isCancelled {
                            return false
                        }
                        outputBox.value += delta
                        return outputBox.value.utf8.count <= maxResponseBytes
                    }
                } catch GuidedGenerationError.incompleteOutput {
                    throw HelperError.maxTokens
                } catch GuidedGenerationError.prematureEOS {
                    throw HelperError.grammarFailure
                }
                if Task.isCancelled {
                    throw HelperError.cancelled
                }
                if outputBox.value.utf8.count > maxResponseBytes {
                    throw HelperError.responseTooLarge
                }
                return outputBox.value
            }
            return Response(
                ok: true,
                stopReason: .completed,
                generatedJSON: generated,
                model: modelID,
                runtime: "mlx",
                errorClass: nil,
                generationProfile: request.generationProfile
            )
        } catch {
            let reason = classify(error)
            return Response(
                ok: false,
                stopReason: reason,
                generatedJSON: nil,
                model: modelID,
                runtime: "mlx",
                errorClass: classifyError(error)
            )
        }
    }

    private static func generateBounded(_ request: Request, profile: String, schema: String, context: ModelContext) async throws -> String {
        let nativeLimit = profile == "bounded-grouping" ? 512 : (profile == "bounded-category" ? 384 : 0)
        let budget = profile == "bounded-category" ? 512 : 768
        guard request.outputTokens == budget, request.contextTokens > 0, request.contextTokens <= 16384 else {
            throw HelperError.malformedRequest
        }
        let chat = request.messages.map { Chat.Message(role: $0.role, content: $0.content) }
        let input = try await context.processor.prepare(input: UserInput(chat: chat, additionalContext: ["enable_thinking": nativeLimit > 0]))
        do { try TokenBudget.validate(context: request.contextTokens, prompt: input.text.tokens.size, output: budget) }
        catch { throw HelperError.inputTooLarge }
        var components = GenerationComponents()
        var nativeState: NativeThoughtBudgetState?
        if nativeLimit > 0 {
            guard context.tokenizer.convertTokenToId("<|channel>") == 100,
                  context.tokenizer.convertTokenToId("<channel|>") == 101 else { throw HelperError.grammarFailure }
            let vocab = TokenizerVocabExtractor.extractForGrammar(from: context.tokenizer)
            let tokenizer = try GrammarTokenizer(vocab: vocab.vocab, vocabType: vocab.vocabType, eosTokenId: Int32(context.tokenizer.eosTokenId ?? 0))
            let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
            var stops = context.configuration.eosTokenIds
            if let eos = context.tokenizer.eosTokenId { stops.insert(eos) }
            for token in context.configuration.extraEOSTokens {
                if let id = context.tokenizer.convertTokenToId(token) { stops.insert(id) }
            }
            let response = GrammarSamplingState(constraint: try factory(), vocabSize: tokenizer.vocabSize,
                eos: context.tokenizer.eosTokenId ?? 0, whitespace: WhitespaceTokenBias.compute(tokenizer: context.tokenizer).tokenIDs,
                runtimeStopIDs: stops, unknownID: context.tokenizer.unknownTokenId, alignRuntimeStops: true, factory: factory)
            let state = NativeThoughtBudgetState(limit: nativeLimit, open: 100, close: 101, eos: context.tokenizer.eosTokenId ?? 0, response: response, forceOpen: true)
            nativeState = state
            components = GenerationComponents(logitProcessorFactory: { NativeThoughtBudgetProcessor(state: state) })
        }
        let stream = try generate(input: input,
            parameters: GenerateParameters(maxTokens: budget, temperature: 0, topP: 1, topK: 0, seed: 144),
            context: context, components: components)
        var output = ""
        var completed = false
        for await event in stream {
            try Task.checkCancellation()
            switch event {
            case .chunk(let chunk): output += chunk
            case .info(let info):
                switch info.stopReason {
                case .stop: completed = true
                case .length: throw HelperError.maxTokens
                case .cancelled: throw HelperError.cancelled
                }
            default: break
            }
            guard output.utf8.count <= maxResponseBytes else { throw HelperError.responseTooLarge }
        }
        guard completed else { throw HelperError.cancelled }
        if let nativeState, !nativeState.succeeded {
            FileHandle.standardError.write(Data("candidate_native_failure code=\(nativeState.diagnostic)\n".utf8))
            throw HelperError.grammarFailure
        }
        do { return try CandidateOutput.finalJSON(output, native: nativeLimit > 0) }
        catch {
            FileHandle.standardError.write(Data("candidate_final_json_invalid native=\(nativeLimit > 0)\n".utf8))
            throw HelperError.grammarFailure
        }
    }

    private static func isModelDirectory(_ path: String) -> Bool {
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory) else {
            return false
        }
        return isDirectory.boolValue
    }

    private static func writeResponse(_ response: Response) throws {
        let data = try JSONEncoder().encode(response)
        guard data.count + 1 <= maxResponseBytes else {
            throw HelperError.responseTooLarge
        }
        var line = data
        line.append(0x0A)
        try FileHandle.standardOutput.write(contentsOf: line)
    }

    private static func classify(_ error: Error) -> StopReason {
        if Task.isCancelled {
            return .cancelled
        }
        if let helperError = error as? HelperError {
            switch helperError {
            case .maxTokens:
                return .maxTokens
            case .grammarFailure:
                return .grammarFailure
            case .cancelled:
                return .cancelled
            default:
                return .internalError
            }
        }
        let description = String(describing: error).lowercased()
        if description.contains("incomplete") || description.contains("max token") {
            return .maxTokens
        }
        if description.contains("grammar") || description.contains("schema") {
            return .grammarFailure
        }
        return .internalError
    }

    private static func classifyError(_ error: Error) -> String {
        switch error {
        case HelperError.inputTooLarge:
            return "input_too_large"
        case HelperError.missingInput:
            return "missing_input"
        case HelperError.trailingInput:
            return "trailing_input"
        case HelperError.malformedRequest:
            return "malformed_request"
        case HelperError.responseTooLarge:
            return "response_too_large"
        case HelperError.modelUnavailable:
            return "model_unavailable"
        case HelperError.grammarFailure:
            return "grammar_failure"
        case HelperError.maxTokens:
            return "max_tokens"
        case HelperError.cancelled:
            return "cancelled"
        default:
            return "internal_error"
        }
    }
}
