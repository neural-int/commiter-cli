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
    var benchmarkMetrics: BenchmarkMetrics? = nil

    enum CodingKeys: String, CodingKey {
        case ok
        case stopReason = "stop_reason"
        case generatedJSON = "generated_json"
        case model
        case runtime
        case errorClass = "error_class"
        case generationProfile = "generation_profile"
        case benchmarkMetrics = "benchmark_metrics"
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

// Read-only model/tokenizer data. Constraints and matchers remain phase-local.
private struct GrammarPreparation: Sendable {
    let tokenizer: GrammarTokenizer
    let whitespaceTokenIDs: Set<Int>
    let runtimeStopIDs: Set<Int>
    let eosTokenID: Int
    let unknownTokenID: Int?

    init(context: ModelContext) throws {
        let vocab = TokenizerVocabExtractor.extractForGrammar(from: context.tokenizer)
        tokenizer = try GrammarTokenizer(vocab: vocab.vocab, vocabType: vocab.vocabType,
            eosTokenId: Int32(context.tokenizer.eosTokenId ?? 0))
        eosTokenID = context.tokenizer.eosTokenId ?? 0
        unknownTokenID = context.tokenizer.unknownTokenId
        var stops = context.configuration.eosTokenIds
        if let eos = context.tokenizer.eosTokenId { stops.insert(eos) }
        for token in context.configuration.extraEOSTokens {
            if let id = context.tokenizer.convertTokenToId(token) { stops.insert(id) }
        }
        runtimeStopIDs = stops
        whitespaceTokenIDs = WhitespaceTokenBias.compute(tokenizer: context.tokenizer).tokenIDs
    }
}

// Explicit research entry points only; normal one-shot protocol stays unchanged.
private final class BenchmarkState: @unchecked Sendable {
    var container: ModelContainer?
    var modelPath: String?
    var modelID: String?
    let retainModel: Bool
    let auditOnly: Bool
    var prefixTokens: [Int] = []
    var prefixCache: [KVCache]?
    var grammarPreparation: GrammarPreparation?
    var metrics = BenchmarkMetrics()
    init(retainModel: Bool, auditOnly: Bool) {
        self.retainModel = retainModel
        self.auditOnly = auditOnly
    }
}
private struct PrefixMemorySample: Encodable {
    let chunk: Int
    let processedTokens: Int
    let afterReclaim: Bool
    let activeBytes: Int
    let cacheBytes: Int
    let peakActiveBytes: Int
}

private struct KVArraySnapshot: Encodable {
    let shape: [Int]
    let dtype: String
    let bytes: Int
}

private struct KVLayerSnapshot: Encodable {
    let index: Int
    let kind: String
    let offset: Int
    let nativeType: String
    let bits: Int
    let bytes: Int
    let arrays: [KVArraySnapshot]
}

private func kvLayerSnapshots(_ cache: [KVCache]) -> [KVLayerSnapshot] {
    cache.enumerated().map { index, layer in
        let arrays = layer.innerState().map {
            KVArraySnapshot(shape: $0.shape, dtype: String(describing: $0.dtype), bytes: $0.nbytes)
        }
        let quantized = layer as? QuantizedKVCache
        return KVLayerSnapshot(index: index,
            kind: quantized != nil ? "affine" : (layer is RotatingKVCache ? "rotating" : "simple"),
            offset: layer.offset, nativeType: String(describing: type(of: layer)),
            bits: quantized?.bits ?? (arrays.map { item in item.bytes / max(1, item.shape.reduce(1, *)) * 8 }.max() ?? 16),
            bytes: arrays.reduce(0) { $0 + $1.bytes }, arrays: arrays)
    }
}

private func benchmarkKVCache(model: any LanguageModel, parameters: GenerateParameters, bits: Int?, configuration: ModelBenchmarkConfiguration? = nil) throws -> [KVCache] {
    let cache = try model.newCache(parameters: parameters)
    guard let bits else { return cache }
    // Quantize eligible global layers from the first prefill token. Native
    // sliding-window caches keep their existing storage and capacity.
    let converted: [KVCache] = cache.map { layer in
        if layer is KVCacheSimple {
            return QuantizedKVCache(groupSize: 64, bits: bits)
        }
        return layer
    }
    if let configuration {
        guard converted.count == configuration.attentionLayers + configuration.otherLayers,
              converted.filter({ $0 is QuantizedKVCache }).count == configuration.attentionLayers else { throw HelperError.malformedRequest }
        return converted
    }
    guard converted.count == 24,
          converted.filter({ $0 is QuantizedKVCache }).count == 4,
          converted.filter({ $0 is RotatingKVCache }).count == 20 else {
        throw HelperError.malformedRequest
    }
    return converted
}

private func cacheContainsPrefilledPrefix(_ cache: KVCache, tokenCount: Int) -> Bool {
    guard tokenCount > 0 else { return false }
    if let recurrentCache = cache as? MambaCache {
        let state = recurrentCache.state
        return state.count == 2 && state.allSatisfy { $0.size > 0 }
    }
    return cache.offset == tokenCount
}

// Research-only configuration; never read by normal CLI inference.
private struct ModelBenchmarkConfiguration: Decodable {
    let model: String
    let family: String
    let thinking: Bool
    let thinkingBudget: Int?
    let output_constraint: OutputConstraint?
    let temperature: Float
    let topP: Float
    let topK: Int
    let attentionLayers: Int
    let otherLayers: Int
    static func load() throws -> Self? {
        guard CommandLine.arguments.dropFirst().first.map({ $0.hasPrefix("--benchmark-") || $0 == "--three-phase" }) == true,
              let path = ProcessInfo.processInfo.environment["COMMITER_182_CASE_CONFIG"] else { return nil }
        let value = try JSONDecoder().decode(Self.self, from: Data(contentsOf: URL(fileURLWithPath: path)))
        guard ["gemma", "qwen", "ministral"].contains(value.family), value.temperature > 0,
              value.topP > 0, value.topP <= 1, value.topK > 0,
              value.thinkingBudget.map({ $0 > 0 && value.thinking }) ?? true else { throw HelperError.malformedRequest }
        return value
    }
}

// Research-only adapter: execute the official Python serializer locally, then
// feed its token IDs directly into the existing Swift/MLX generation runtime.
private func officialMistralInput(_ request: Request, messages: [(role: Chat.Message.Role, content: String)], reasoning: Bool, tokenizer: any MLXLMCommon.Tokenizer) throws -> LMInput? {
    let environment = ProcessInfo.processInfo.environment
    guard let python = environment["COMMITER_182_MISTRAL_PYTHON"] else { return nil }
    guard let bridge = environment["COMMITER_182_MISTRAL_BRIDGE"],
          let assets = environment["COMMITER_182_MISTRAL_ASSETS"] else { throw HelperError.malformedRequest }
    let process = Process()
    process.executableURL = URL(fileURLWithPath: python)
    process.arguments = [bridge, "--assets", assets] + (reasoning ? ["--reasoning"] : [])
    let input = Pipe(), output = Pipe()
    process.standardInput = input
    process.standardOutput = output
    process.standardError = FileHandle.standardError
    let payload = ["messages": messages.map { ["role": $0.role.rawValue, "content": $0.content] }]
    try process.run()
    try input.fileHandleForWriting.write(contentsOf: JSONSerialization.data(withJSONObject: payload))
    try input.fileHandleForWriting.close()
    let data = output.fileHandleForReading.readDataToEndOfFile()
    process.waitUntilExit()
    struct Serialized: Decodable { let tokens: [Int]; let eos: Int; let open: Int?; let close: Int? }
    guard process.terminationStatus == 0 else { throw HelperError.malformedRequest }
    let serialized = try JSONDecoder().decode(Serialized.self, from: data)
    guard serialized.eos == tokenizer.eosTokenId, serialized.tokens.first == 1,
          serialized.tokens.last == 4, !serialized.tokens.contains(serialized.eos),
          serialized.tokens.allSatisfy({ $0 >= 0 && $0 < 131072 }) else { throw HelperError.malformedRequest }
    if reasoning {
        guard serialized.open == tokenizer.convertTokenToId("[THINK]"),
              serialized.close == tokenizer.convertTokenToId("[/THINK]") else { throw HelperError.malformedRequest }
    }
    return LMInput(tokens: MLXArray(serialized.tokens))
}

private final class BenchmarkTokenProgress: @unchecked Sendable {
    let native: NativeThoughtBudgetState?
    let response: GrammarSamplingState
    let started = ProcessInfo.processInfo.systemUptime
    var count = 0
    let diagnostic = ProcessInfo.processInfo.environment["COMMITER_182_DIAGNOSTICS"] == "1"
    var diagnosticTotals: [String: Double] = [:]
    var diagnosticCounts: [String: Int] = [:]
    var firstReadSeconds = 0.0
    func phase() -> String { native?.isResponsePhase == false ? "thinking" : "response" }
    func record(_ name: String, seconds: Double) {
        diagnosticTotals[name, default: 0] += seconds
        diagnosticCounts[name, default: 0] += 1
    }
    func consume(_ token: MLXArray) {
        guard diagnostic else { sampled(token.item(Int.self)); return }
        let name = phase()
        let begin = ProcessInfo.processInfo.systemUptime
        let value = token.item(Int.self)
        let read = ProcessInfo.processInfo.systemUptime
        if count == 0 { firstReadSeconds = read - begin }
        else { record(name + "ReadSeconds", seconds: read - begin) }
        sampled(value)
        record(name + "UpdateAndProgressSeconds", seconds: ProcessInfo.processInfo.systemUptime - read)
    }
    init(native: NativeThoughtBudgetState?, response: GrammarSamplingState) {
        self.native = native; self.response = response
    }
    func sampled(_ token: Int) {
        if let native { native.commit(token) } else { response.commit(token) }
        count += 1
        if count == 1 || count % 64 == 0 {
            let elapsed = ProcessInfo.processInfo.systemUptime - started
            let entry: [String: Any] = ["event": "generation_progress", "tokens": count,
                "seconds_including_prefill": elapsed, "thinking_tokens": native?.stats().prefixTokens ?? 0]
            if let data = try? JSONSerialization.data(withJSONObject: entry, options: [.sortedKeys]) {
                FileHandle.standardError.write(data + Data([10]))
            }
        }
    }
}

private struct BenchmarkProgressProcessor: LogitProcessor {
    let progress: BenchmarkTokenProgress
    mutating func prompt(_ prompt: MLXArray) {}
    func process(logits: MLXArray) -> MLXArray {
        let begin = progress.diagnostic ? ProcessInfo.processInfo.systemUptime : 0
        let phase = progress.diagnostic ? progress.phase() : ""
        let mask: [Float]?
        if let native = progress.native { mask = native.maskValues(count: logits.dim(-1)) }
        else { mask = progress.response.enforcesGrammar ? progress.response.maskValues(count: logits.dim(-1)) : nil }
        let result = mask.map { logits + MLXArray($0) } ?? logits
        if progress.diagnostic {
            progress.record(phase + "MaskPreparationSeconds", seconds: ProcessInfo.processInfo.systemUptime - begin)
        }
        return result
    }
    mutating func didSample(token: MLXArray) { progress.consume(token) }
    func copy() -> Self {
        Self(progress: BenchmarkTokenProgress(native: progress.native?.independentCopy(), response: progress.response.independentCopy()))
    }
}

private struct BenchmarkMetrics: Encodable {
    var outputConstraint: String?
    var scopeConstraintRejectedTokens: Int?
    var prefillProtectionEnabled: Bool?
    var comparisonPath: String?
    var comparisonStopReason: String?
    var loadSeconds = 0.0
    var prepareSeconds = 0.0
    var grammarSetupSeconds = 0.0
    var grammarPreparationSeconds = 0.0
    var grammarPreparationReused = false
    var generationWallSeconds = 0.0
    var runtimePromptSeconds: Double?
    var runtimeGenerationSeconds: Double?
    var inputTokens = 0
    var generatedTokens: Int?
    var nativeThoughtTokens: Int?
    var nativeThoughtForcedClose: Bool?
    var prefixPrefillSeconds = 0.0
    var cacheReusedTokens = 0
    var modelReused = false
    var activeMemoryBytes = 0
    var peakMemoryBytes = 0
    var cacheMemoryBytes = 0
    var prefixChunkCount = 0
    var prefixReclaimCount = 0
    var prefixSynchronizeSeconds = 0.0
    var prefixClearCacheSeconds = 0.0
    var prefixMemorySamples: [PrefixMemorySample] = []
    var diagnosticCPUSeconds: [String: Double]?
    var diagnosticEventCounts: [String: Int]?
    var diagnosticFirstReadSeconds: Double?
    var benchmarkCacheMode: String?
    var requestedKVBits: Int? = nil
    var prefixKVLayers: [KVLayerSnapshot] = []
    var phaseEndKVLayers: [KVLayerSnapshot] = []
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
        if CommandLine.arguments.count == 2, CommandLine.arguments[1] == "--benchmark-output-contract" {
            do {
                guard let config = try ModelBenchmarkConfiguration.load(), let mode = config.output_constraint else { throw HelperError.malformedRequest }
                let request = try decodeRequest(try readBoundedRequest())
                let data = try JSONSerialization.data(withJSONObject: request.schema.foundationValue, options: [.sortedKeys])
                let schema = String(decoding: data, as: UTF8.self)
                let messages = try contractMessages(request, schema: schema, enabled: true)
                let audit: [String: Any] = ["output_constraint": mode.rawValue, "thinking": config.thinking,
                    "messages": messages.map { ["role": $0.role.rawValue, "content": $0.content] },
                    "schema": request.schema.foundationValue, "generation_calls": 0]
                try FileHandle.standardOutput.write(contentsOf: JSONSerialization.data(withJSONObject: audit, options: [.sortedKeys]))
                try FileHandle.standardOutput.write(contentsOf: Data([10]))
            } catch { exit(1) }
            return
        }
        if CommandLine.arguments.count == 2,
           ["--three-phase", "--benchmark-single", "--benchmark-cycle", "--benchmark-token-audit", "--benchmark-input-audit", "--benchmark-cache", "--benchmark-model-defaults"].contains(CommandLine.arguments[1]) {
            await benchmarkMain(mode: CommandLine.arguments[1])
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

    private static func contractMessages(_ request: Request, schema: String, enabled: Bool) throws -> [(role: Chat.Message.Role, content: String)] {
        var messages = request.messages.map { (role: $0.role, content: $0.content) }
        if enabled {
            var instruction = try StructuredOutputPrompt.instruction(schema: schema)
            if ProcessInfo.processInfo.environment["COMMITER_182_NEUTRAL_EXAMPLES"] == "1",
               let cut = instruction.range(of: "Structural example only;") {
                instruction = String(instruction[..<cut.lowerBound]) + "Choose values independently from the current Git evidence; enum order is not a preference."
            }
            if let index = messages.firstIndex(where: { $0.role == .system }) {
                messages[index].content += "\n\n" + instruction
            } else { messages.insert((role: .system, content: instruction), at: 0) }
        }
        return messages
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

    private static func handle(_ request: Request, benchmark: BenchmarkState? = nil) async throws -> Response {
        guard let modelPath = request.modelPath?.trimmingCharacters(in: .whitespacesAndNewlines),
              !modelPath.isEmpty,
              isModelDirectory(modelPath) else {
            throw HelperError.modelUnavailable
        }
        let modelID = request.model?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
            ? request.model!
            : URL(filePath: modelPath).lastPathComponent
        let research = try ModelBenchmarkConfiguration.load()
        if let research { guard research.model == modelID else { throw HelperError.malformedRequest } }
        if let profile = request.generationProfile {
            guard ["bounded-grouping", "bounded-category", "bounded-text"].contains(profile),
                  (research != nil || modelID == "mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2" ||
                   (benchmark != nil && CommandLine.arguments.contains("--benchmark-model-defaults"))) else {
                throw HelperError.malformedRequest
            }
            do {
                if research == nil { try BoundedRequestLimits.validate(profile: profile, context: request.contextTokens, output: request.outputTokens) } else { guard (request.outputTokens == 0 || request.outputTokens == 4096), request.contextTokens > 0, request.contextTokens <= 65536 else { throw HelperError.malformedRequest } }
            } catch { throw HelperError.malformedRequest }
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
            let loadStart = ProcessInfo.processInfo.systemUptime
            let container: ModelContainer
            if let retained = benchmark?.container {
                guard benchmark?.modelPath == modelPath, benchmark?.modelID == modelID else {
                    throw HelperError.malformedRequest
                }
                container = retained
                benchmark?.metrics.modelReused = true
            } else {
                container = try await loadModelContainer(from: modelDirectory, using: #huggingFaceTokenizerLoader())
                if benchmark?.retainModel == true {
                    benchmark?.container = container
                    benchmark?.modelPath = modelPath
                    benchmark?.modelID = modelID
                }
            }
            benchmark?.metrics.loadSeconds = ProcessInfo.processInfo.systemUptime - loadStart
            let outputBox = OutputBox()
            let generated = try await container.perform { context in
                try Task.checkCancellation()
                let chat = messages.map { Chat.Message(role: $0.role, content: $0.content) }
                if let profile = request.generationProfile {
                    return try await generateBounded(request, profile: profile, schema: schema, context: context, benchmark: benchmark)
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
                generationProfile: request.generationProfile,
                benchmarkMetrics: benchmark?.metrics
            )
        } catch {
            benchmark?.metrics.activeMemoryBytes = Memory.activeMemory
            benchmark?.metrics.peakMemoryBytes = Memory.peakMemory
            benchmark?.metrics.cacheMemoryBytes = Memory.cacheMemory
            let reason = classify(error)
            return Response(
                ok: false,
                stopReason: reason,
                generatedJSON: nil,
                model: modelID,
                runtime: "mlx",
                errorClass: classifyError(error),
                benchmarkMetrics: benchmark?.metrics
            )
        }
    }

    private static func generateBounded(_ request: Request, profile: String, schema: String, context: ModelContext, benchmark: BenchmarkState? = nil) async throws -> String {
        let research = try ModelBenchmarkConfiguration.load()
        let pipeline = research == nil ? "eager" : (ProcessInfo.processInfo.environment["COMMITER_182_PIPELINE"] ?? "eager")
        guard ["eager", "overlap"].contains(pipeline),
              pipeline != "overlap" || (benchmark != nil && ProcessInfo.processInfo.environment["COMMITER_182_PROGRESS"] == "1") else {
            throw HelperError.malformedRequest
        }
        let cacheMode = ProcessInfo.processInfo.environment["COMMITER_180_KV_BITS"] ?? "8"
        if research != nil {
            guard ["4", "8", "native"].contains(cacheMode) else { throw HelperError.malformedRequest }
            benchmark?.metrics.benchmarkCacheMode = cacheMode
        }
        let configuredMode = benchmark == nil ? nil : ProcessInfo.processInfo.environment["COMMITER_180_THINKING"]
        let modelDefaults = CommandLine.arguments.contains("--benchmark-model-defaults")
        let modelName = request.model?.split(separator: "@").first.map(String.init) ?? ""
        let qwenThinking = research.map { $0.family == "qwen" && $0.thinking } ?? (modelDefaults && modelName == "mlx-community/Qwen3.5-4B-4bit")
        let ministralThinking = research.map { $0.family == "ministral" && $0.thinking } ?? false
        let otherDefault = modelDefaults && !modelName.contains("gemma-4-E4B")
        let nativeLimit = research.map { $0.thinking ? Int.max : 0 } ?? (qwenThinking ? (profile == "bounded-grouping" ? 512 : 384) :
            (otherDefault ? 0 : (profile == "bounded-grouping" ? 512 : (configuredMode == "T" ? 384 : (configuredMode == "N" ? 0 : (profile == "bounded-category" ? 384 : 0)))))
        )
        let budget = request.outputTokens // Validated by handle before model loading.
        let outputConstraint = research?.output_constraint ?? .jsonSchema
        benchmark?.metrics.outputConstraint = research?.output_constraint?.rawValue
        var wireMessages = try contractMessages(request, schema: schema, enabled: research?.output_constraint != nil)
        // Research layout comparison: caller supplies the finalized wire; Schema still constrains sampling.
        if (benchmark?.auditOnly == true && ProcessInfo.processInfo.environment["COMMITER_184_AUDIT_WIRE"] == "1") || (research != nil && ProcessInfo.processInfo.environment["COMMITER_184_LAYOUT_WIRE"] == "1") {
            wireMessages = request.messages.map { (role: $0.role, content: $0.content) }
        }
        if research?.family == "ministral" {
            var merged: [(role: Chat.Message.Role, content: String)] = []
            for message in wireMessages {
                if let last = merged.last, last.role == message.role {
                    merged[merged.count - 1].content += "\n\n" + message.content
                } else { merged.append(message) }
            }
            if ministralThinking, ProcessInfo.processInfo.environment["COMMITER_182_MISTRAL_PYTHON"] == nil, merged.first?.role == .system {
                merged[0].content += "\nThink using [THINK]your thoughts[/THINK], then provide the requested JSON response."
            }
            wireMessages = merged
        }
        let chat = wireMessages.map { Chat.Message(role: $0.role, content: $0.content) }
        let prepareStart = ProcessInfo.processInfo.systemUptime
        // Candidate chat templates choose their own default thinking behavior.
        // Gemma preserves the established bounded-profile behavior.
        let templateContext: [String: any Sendable]? = research != nil ? ["enable_thinking": research!.thinking] : (otherDefault ? nil : ["enable_thinking": nativeLimit > 0])
        let officialInput = research?.family == "ministral"
            ? try officialMistralInput(request, messages: wireMessages, reasoning: ministralThinking, tokenizer: context.tokenizer) : nil
        var input: LMInput
        if let officialInput { input = officialInput }
        else { input = try await context.processor.prepare(input: UserInput(chat: chat, additionalContext: templateContext)) }
        benchmark?.metrics.prepareSeconds = ProcessInfo.processInfo.systemUptime - prepareStart
        benchmark?.metrics.inputTokens = input.text.tokens.size
        if benchmark?.auditOnly == true {
            var audit: [String: Any] = ["token_ids": input.text.tokens.asArray(Int.self), "enable_thinking": nativeLimit > 0]
            if let research {
                let cache = try benchmarkKVCache(model: context.model, parameters: GenerateParameters(), bits: cacheMode == "native" ? nil : Int(cacheMode), configuration: research)
                audit["planned_cache"] = try JSONSerialization.jsonObject(with: JSONEncoder().encode(kvLayerSnapshots(cache)))
            }
            let data = try JSONSerialization.data(withJSONObject: audit, options: [.sortedKeys])
            return String(decoding: data, as: UTF8.self)
        }
        // Research-only path comparison; ordinary generation is unchanged.
        let pathMode = ProcessInfo.processInfo.environment["COMMITER_182_PATH"] ?? "full"
        guard ["full", "bounded-cap", "standard-cap"].contains(pathMode),
              pathMode == "full" || (benchmark != nil && research != nil && pipeline == "eager" && research?.output_constraint == nil) else {
            throw HelperError.malformedRequest
        }
        let cappedComparison = pathMode != "full"
        if cappedComparison { benchmark?.metrics.comparisonPath = pathMode }
        var progressMetrics: BenchmarkTokenProgress?
        var responseState: GrammarSamplingState?
        let grammarStart = ProcessInfo.processInfo.systemUptime
        do { try TokenBudget.validate(context: request.contextTokens, prompt: input.text.tokens.size, output: budget) }
        catch { throw HelperError.inputTooLarge }
        var nativeState: NativeThoughtBudgetState?
        var components = GenerationComponents()
        if pathMode != "standard-cap" {
            let preparationStart = ProcessInfo.processInfo.systemUptime
            let preparation: GrammarPreparation
            if let retained = benchmark?.grammarPreparation {
                preparation = retained
                benchmark?.metrics.grammarPreparationReused = true
            } else {
                preparation = try GrammarPreparation(context: context)
                if benchmark?.retainModel == true {
                    benchmark?.grammarPreparation = preparation
                }
            }
            benchmark?.metrics.grammarPreparationSeconds = ProcessInfo.processInfo.systemUptime - preparationStart
            let tokenizer = preparation.tokenizer
            let factory: @Sendable () throws -> GrammarConstraint = { try GrammarConstraint(tokenizer: tokenizer, jsonSchema: schema, fastForward: false) }
            let response = outputConstraint == .none
                ? GrammarSamplingState(unconstrainedVocabSize: tokenizer.vocabSize, eos: preparation.eosTokenID, runtimeStopIDs: preparation.runtimeStopIDs)
                : GrammarSamplingState(constraint: try factory(), vocabSize: tokenizer.vocabSize,
                eos: preparation.eosTokenID, whitespace: preparation.whitespaceTokenIDs,
                runtimeStopIDs: preparation.runtimeStopIDs, unknownID: preparation.unknownTokenID,
                alignRuntimeStops: true, factory: factory)
            responseState = response
            if research != nil, profile == "bounded-text",
               ProcessInfo.processInfo.environment["COMMITER_184_SCOPE_CONSTRAINT"] == "1" {
                guard outputConstraint == .jsonSchema,
                      let path = ProcessInfo.processInfo.environment["COMMITER_184_SCOPE_POLICY"],
                      let words = try? JSONDecoder().decode([String].self, from: Data(contentsOf: URL(fileURLWithPath: path))) else { throw HelperError.malformedRequest }
                let forbidden = Set(words)
                let hostTokenizer = context.tokenizer
                // Only tokens containing a quote can close a JSON string.
                let quoteTokens = Set((0..<tokenizer.vocabSize).filter {
                    hostTokenizer.decode(tokenIds: [$0]).contains("\"")
                })
                response.setPrefixConstraint(candidate: { quoteTokens.contains($0) }) { ids in
                    return ScopeValueConstraint.allows(hostTokenizer.decode(tokenIds: ids), forbidden: forbidden)
                }
            }
            if nativeLimit > 0 {
                let open: Int
                let close: Int
                if qwenThinking || ministralThinking {
                    guard let opening = context.tokenizer.convertTokenToId(ministralThinking ? "[THINK]" : "<think>"),
                          let closing = context.tokenizer.convertTokenToId(ministralThinking ? "[/THINK]" : "</think>"), opening != closing,
                          context.tokenizer.decode(tokenIds: [opening]) == (ministralThinking ? "[THINK]" : "<think>"),
                          context.tokenizer.decode(tokenIds: [closing]) == (ministralThinking ? "[/THINK]" : "</think>") else { throw HelperError.grammarFailure }
                    open = opening; close = closing
                } else {
                    guard context.tokenizer.convertTokenToId("<|channel>") == 100,
                          context.tokenizer.convertTokenToId("<channel|>") == 101 else { throw HelperError.grammarFailure }
                    open = 100; close = 101
                }
                let state = NativeThoughtBudgetState(limit: research != nil ? research?.thinkingBudget : nativeLimit, open: open, close: close, eos: context.tokenizer.eosTokenId ?? 0, response: response, forceOpen: !qwenThinking && officialInput == nil, promptHasOpen: qwenThinking)
                nativeState = state
                components = GenerationComponents(logitProcessorFactory: { NativeThoughtBudgetProcessor(state: state) })
            } else {
                components = GenerationComponents(logitProcessorFactory: { GrammarSamplingProcessor(state: response) })
            }
            if officialInput != nil || (research != nil && ProcessInfo.processInfo.environment["COMMITER_182_PROGRESS"] == "1") {
                let progress = BenchmarkTokenProgress(native: nativeState, response: response)
                progressMetrics = progress
                if pipeline == "overlap" {
                    components = GenerationComponents(logitProcessorFactory: {
                        OverlappedTokenProcessor(BenchmarkProgressProcessor(progress: progress))
                    })
                } else {
                    components = GenerationComponents(logitProcessorFactory: { BenchmarkProgressProcessor(progress: progress) })
                }
            }
        }
        var parameters = GenerateParameters(maxTokens: cappedComparison ? 1024 : (research != nil && budget == 0 ? nil : budget), temperature: research?.temperature ?? 0, topP: research?.topP ?? 1, topK: research?.topK ?? 0, seed: 144)
        var kvBits: Int? = nil
        if benchmark != nil, (CommandLine.arguments.contains("--benchmark-cache") || research?.output_constraint != nil),
           let value = ProcessInfo.processInfo.environment["COMMITER_180_KV_BITS"] {
            if value == "native", research != nil {
                kvBits = nil
            } else {
                guard let bits = Int(value), [4, 8].contains(bits) else { throw HelperError.malformedRequest }
                kvBits = bits
                parameters.kvCache = KVCacheConfiguration(
                    strategy: .affine(try AffineKVCacheConfiguration(bits: bits, groupSize: 64)),
                    compatibility: .requireAtLeastOneLayer)
                benchmark?.metrics.requestedKVBits = bits
            }
        }
        let prefillPolicy = ProcessInfo.processInfo.environment["COMMITER_184_PREFILL_POLICY"] ?? "adaptive8k"
        guard ["adaptive8k", "legacy"].contains(prefillPolicy) else { throw HelperError.malformedRequest }
        let protectPrefill = request.contextTokens > 16384 && (prefillPolicy == "legacy" || input.text.tokens.size > 8192)
        benchmark?.metrics.prefillProtectionEnabled = protectPrefill
        if protectPrefill {
            // The dependency submits prefill chunks asynchronously. At long
            // contexts, wait for each chunk before submitting the next and
            // release unused buffers; keep token order and KV contents intact.
            parameters.prefill.progress = { _, _ in
                Stream().synchronize()
                Memory.clearCache()
            }
        }
        benchmark?.metrics.grammarSetupSeconds = ProcessInfo.processInfo.systemUptime - grammarStart
        let generationStart = ProcessInfo.processInfo.systemUptime
        var workingCache: [KVCache]? = nil
        if CommandLine.arguments.contains("--benchmark-cache"), profile != "bounded-grouping", let benchmark {
            let tokens = input.text.tokens.asArray(Int.self)
            guard let lengthText = ProcessInfo.processInfo.environment["COMMITER_180_PREFIX_TOKENS"], let length = Int(lengthText), length > 4, length < tokens.count else { throw HelperError.malformedRequest }
            if profile == "bounded-category" {
                guard research != nil || context.model is Gemma4Model || context.model is Gemma4TextModel else { throw HelperError.malformedRequest }
                let prefixStart = ProcessInfo.processInfo.systemUptime
                let cache = try benchmarkKVCache(model: context.model, parameters: parameters, bits: kvBits, configuration: research)
                for start in stride(from: 0, to: length, by: 512) {
                    let end = min(start + 512, length)
                    // Only materialize KV state, as in the runtime prefill path.
                    // Evaluating logits would also compute unused output projections.
                    _ = context.model(MLXArray(Array(tokens[start..<end])).expandedDimensions(axis: 0), cache: cache)
                    eval(cache)
                    benchmark.metrics.prefixChunkCount += 1
                    let chunk = benchmark.metrics.prefixChunkCount
                    benchmark.metrics.prefixMemorySamples.append(PrefixMemorySample(
                        chunk: chunk, processedTokens: end, afterReclaim: false,
                        activeBytes: Memory.activeMemory, cacheBytes: Memory.cacheMemory,
                        peakActiveBytes: Memory.peakMemory))
                    // Keep synchronous evaluation per 512 tokens; reclaim every
                    // four chunks and at the prefix boundary for this experiment.
                    if chunk % 4 == 0 || end == length {
                        let synchronizeStart = ProcessInfo.processInfo.systemUptime
                        Stream().synchronize()
                        benchmark.metrics.prefixSynchronizeSeconds += ProcessInfo.processInfo.systemUptime - synchronizeStart
                        let clearStart = ProcessInfo.processInfo.systemUptime
                        Memory.clearCache()
                        benchmark.metrics.prefixClearCacheSeconds += ProcessInfo.processInfo.systemUptime - clearStart
                        benchmark.metrics.prefixReclaimCount += 1
                        benchmark.metrics.prefixMemorySamples.append(PrefixMemorySample(
                            chunk: chunk, processedTokens: end, afterReclaim: true,
                            activeBytes: Memory.activeMemory, cacheBytes: Memory.cacheMemory,
                            peakActiveBytes: Memory.peakMemory))
                    }
                }
                guard cache.allSatisfy({ cacheContainsPrefilledPrefix($0, tokenCount: length) }) else {
                    throw HelperError.malformedRequest
                }
                benchmark.metrics.prefixPrefillSeconds = ProcessInfo.processInfo.systemUptime - prefixStart
                benchmark.metrics.prefixKVLayers = kvLayerSnapshots(cache)
                benchmark.prefixTokens = Array(tokens.prefix(length))
                benchmark.prefixCache = cache.map { $0.copy() }
                workingCache = cache
            } else {
                guard benchmark.prefixTokens == Array(tokens.prefix(length)), let saved = benchmark.prefixCache else { throw HelperError.malformedRequest }
                workingCache = saved.map { $0.copy() }
                benchmark.metrics.cacheReusedTokens = length
                benchmark.prefixCache = nil
                benchmark.prefixTokens = []
            }
            input = LMInput(tokens: MLXArray(Array(tokens.dropFirst(length))))
        }
        if (kvBits != nil || (research != nil && CommandLine.arguments.contains("--benchmark-cache"))), workingCache == nil {
            workingCache = try benchmarkKVCache(model: context.model, parameters: parameters, bits: kvBits, configuration: research)
        }
        if modelDefaults, workingCache == nil {
            workingCache = try context.model.newCache(parameters: parameters)
        }
        let stream = try generate(input: input, cache: workingCache,
            parameters: parameters,
            context: context, components: components)
        var output = ""
        var completed = false
        var lastPathProgress = generationStart
        for await event in stream {
            try Task.checkCancellation()
            switch event {
            case .chunk(let chunk):
                output += chunk
                if cappedComparison, ProcessInfo.processInfo.systemUptime - lastPathProgress >= 5 {
                    lastPathProgress = ProcessInfo.processInfo.systemUptime
                    FileHandle.standardError.write(Data("path_comparison_progress bytes=\(output.utf8.count)\n".utf8))
                }
            case .info(let info):
                benchmark?.metrics.runtimePromptSeconds = info.promptTime
                benchmark?.metrics.runtimeGenerationSeconds = info.generateTime
                benchmark?.metrics.generatedTokens = info.generationTokenCount
                if let progress = progressMetrics, progress.diagnostic {
                    benchmark?.metrics.diagnosticCPUSeconds = progress.diagnosticTotals
                    benchmark?.metrics.diagnosticEventCounts = progress.diagnosticCounts
                    benchmark?.metrics.diagnosticFirstReadSeconds = progress.firstReadSeconds
                }
                if let nativeState {
                    benchmark?.metrics.nativeThoughtTokens = nativeState.stats().prefixTokens
                    benchmark?.metrics.nativeThoughtForcedClose = nativeState.stats().forcedClose
                }
                if cappedComparison { benchmark?.metrics.comparisonStopReason = String(describing: info.stopReason) }
                switch info.stopReason {
                case .stop: completed = true
                case .length:
                    if cappedComparison { completed = true } else { throw HelperError.maxTokens }
                case .cancelled: throw HelperError.cancelled
                }
            default: break
            }
            guard output.utf8.count <= maxResponseBytes else { throw HelperError.responseTooLarge }
        }
        benchmark?.metrics.generationWallSeconds = ProcessInfo.processInfo.systemUptime - generationStart
        if research != nil, profile == "bounded-text",
           ProcessInfo.processInfo.environment["COMMITER_184_SCOPE_CONSTRAINT"] == "1" {
            benchmark?.metrics.scopeConstraintRejectedTokens = responseState?.prefixRejectedTokenCount
        }
        benchmark?.metrics.activeMemoryBytes = Memory.activeMemory
        benchmark?.metrics.peakMemoryBytes = Memory.peakMemory
        benchmark?.metrics.cacheMemoryBytes = Memory.cacheMemory
        if let workingCache {
            benchmark?.metrics.phaseEndKVLayers = kvLayerSnapshots(workingCache)
        }
        guard completed else { throw HelperError.cancelled }
        if cappedComparison { return output }
        if let nativeState, !nativeState.succeeded {
            FileHandle.standardError.write(Data("candidate_native_failure code=\(nativeState.diagnostic)\n".utf8))
            throw HelperError.grammarFailure
        }
        if nativeLimit == 0, let response = responseState, !response.succeeded {
            FileHandle.standardError.write(Data("candidate_text_failure code=\(response.diagnostic)\n".utf8))
            throw HelperError.grammarFailure
        }
        do {
            if qwenThinking || ministralThinking {
                let parts = output.components(separatedBy: ministralThinking ? "[/THINK]" : "</think>")
                guard parts.count == 2 else { throw CandidateOutputError.invalid }
                return try CandidateOutput.finalJSON(parts[1], native: false)
            }
            return try CandidateOutput.finalJSON(output, native: nativeLimit > 0)
        }
        catch {
            FileHandle.standardError.write(Data("candidate_final_json_invalid native=\(nativeLimit > 0)\n".utf8))
            throw HelperError.grammarFailure
        }
    }

    private static func benchmarkMain(mode: String) async {
        let audit = mode == "--benchmark-token-audit" || mode == "--benchmark-input-audit"
        let state = BenchmarkState(retainModel: mode != "--benchmark-single", auditOnly: audit)
        let reader = BenchmarkLineReader(handle: .standardInput, maximumBytes: maxRequestBytes)
        let profiles = mode == "--benchmark-token-audit" ? ["bounded-category", "bounded-text"] : ["bounded-grouping", "bounded-category", "bounded-text"]
        var count = 0
        do {
            while let data = try reader.next() {
                guard count < profiles.count else { throw HelperError.trailingInput }
                let request = try decodeRequest(data)
                if mode != "--benchmark-single" {
                    guard request.generationProfile == profiles[count] else { throw HelperError.malformedRequest }
                }
                state.metrics = BenchmarkMetrics()
                let response = try await handle(request, benchmark: state)
                try writeResponse(response)
                count += 1
                if !response.ok || mode == "--benchmark-single" || count == profiles.count { return }
            }
        } catch {
            try? writeResponse(Response(ok: false, stopReason: classify(error), generatedJSON: nil,
                model: nil, runtime: "mlx", errorClass: classifyError(error)))
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
