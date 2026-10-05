"""Copy the pinned helper to an experiment directory and add numeric telemetry.

Production source is unchanged. Experimental global profiles are separate.
The experiment protocol is intentionally consumed only by benchmark149.
"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil


def replace_once(text, old, new):
    if text.count(old) != 1:
        raise ValueError('instrumentation anchor changed; inspect before measuring')
    return text.replace(old, new, 1)


def prepare(source, destination):
    if destination.exists():
        raise ValueError('use a new experiment directory')
    destination.mkdir(parents=True)
    for name in ('Package.swift', 'Package.resolved'):
        shutil.copy2(source / name, destination / name)
    for name in ('Sources', 'Tests'):
        shutil.copytree(source / name, destination / name)
    path = destination / 'Sources/commiter-mlx-helper/CommiterMLXHelper.swift'
    original = path.read_text()
    text = replace_once(original, 'private struct Response: Encodable {',
        'private final class BenchmarkCounts: @unchecked Sendable {\n'
        '    var input: Int? = nil\n    var output: Int? = nil\n}\n\n'
        'private struct Response: Encodable {')
    text = replace_once(text, '    var generationProfile: String? = nil',
        '    var generationProfile: String? = nil\n'
        '    var benchmarkInputTokens: Int? = nil\n'
        '    var benchmarkOutputTokens: Int? = nil')
    text = replace_once(text, '        case generationProfile = "generation_profile"\n    }\n}\n\nprivate enum HelperError',
        '        case generationProfile = "generation_profile"\n'
        '        case benchmarkInputTokens = "benchmark_input_tokens"\n'
        '        case benchmarkOutputTokens = "benchmark_output_tokens"\n    }\n}\n\nprivate enum HelperError')
    text = replace_once(text, '        let messages = request.messages',
        '        let benchmarkCounts = BenchmarkCounts()\n        let messages = request.messages')
    text = replace_once(text, 'generateBounded(request, profile: profile, schema: schema, context: context)',
        'generateBounded(request, profile: profile, schema: schema, context: context, counts: benchmarkCounts)')
    text = replace_once(text, '                generationProfile: request.generationProfile\n',
        '                generationProfile: request.generationProfile,\n'
        '                benchmarkInputTokens: benchmarkCounts.input,\n'
        '                benchmarkOutputTokens: benchmarkCounts.output\n')
    text = replace_once(text, '                model: modelID,\n                runtime: "mlx",\n                errorClass: classifyError(error)\n',
        '                model: modelID,\n                runtime: "mlx",\n                errorClass: classifyError(error),\n'
        '                benchmarkInputTokens: benchmarkCounts.input,\n'
        '                benchmarkOutputTokens: benchmarkCounts.output\n')
    text = replace_once(text, 'context: ModelContext) async throws -> String {',
        'context: ModelContext, counts: BenchmarkCounts) async throws -> String {')
    text = replace_once(text, '        do { try TokenBudget.validate(context: request.contextTokens, prompt: input.text.tokens.size, output: budget) }',
        '        counts.input = input.text.tokens.size\n'
        '        do { try TokenBudget.validate(context: request.contextTokens, prompt: input.text.tokens.size, output: budget) }')
    text = replace_once(text, '            case .info(let info):\n                switch info.stopReason {',
        '            case .info(let info):\n                counts.output = info.generationTokenCount\n                switch info.stopReason {')
    # A dedicated larger global decision contract and native-free routing.
    text = replace_once(text, '["bounded-grouping", "bounded-category", "bounded-text"].contains(profile),', '["bounded-grouping", "bounded-category", "bounded-text", "bounded-global-contract", "bounded-routed-grouping"].contains(profile),')
    text = replace_once(text, 'modelID == "mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2" else {', '(modelID == "mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2" || (profile == "bounded-routed-grouping" && ['
        '"mlx-community/Qwen3.5-4B-MLX-4bit@32f3e8ecf65426fc3306969496342d504bfa13f3",'
        '"mlx-community/Phi-4-mini-instruct-4bit@ac1c269cb4222a4e136a3d09edad301056c1f36a",'
        '"ibm-granite/granite-4.2-3b-q4-mlx@0c6f39b1827afd5eb2c1c3b13751929857434953",'
        '"mlx-community/NVIDIA-Nemotron-3-Nano-4B-4bit@c4d79ba1901d99806ef757642a552acebb851a35",'
        '"mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee",'
        '"mlx-community/Ministral-3-3B-Reasoning-2512-4bit@2cd2087aad40c28747f8ede17851de6035f12b16"'
        '].contains(modelID))) else {')
    text = replace_once(text, 'let nativeLimit = profile == "bounded-grouping" ? 512 : (profile == "bounded-category" ? 384 : 0)', 'let nativeLimit = profile == "bounded-global-contract" ? 1024 : (profile == "bounded-grouping" ? 512 : (profile == "bounded-category" ? 384 : 0))')
    text = replace_once(text, 'let budget = profile == "bounded-category" ? 512 : 768', 'let budget = (profile == "bounded-global-contract" || profile == "bounded-routed-grouping") ? 1536 : (profile == "bounded-category" ? 512 : 768)')
    path.write_text(text)
    manifest = {
        'original_helper_source_sha256': hashlib.sha256(original.encode()).hexdigest(),
        'instrumented_helper_source_sha256': hashlib.sha256(text.encode()).hexdigest(),
        'global_profiles': {'bounded-global-contract': {'native': 1024, 'output': 1536}, 'bounded-routed-grouping': {'native': 0, 'output': 1536}},
        'measurement': 'input.text.tokens.size and info.generationTokenCount; output includes native thought and channel tokens',
    }
    (destination / 'benchmark-instrumentation.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print(json.dumps(manifest))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('source', type=Path)
    parser.add_argument('destination', type=Path)
    args = parser.parse_args()
    prepare(args.source, args.destination)
