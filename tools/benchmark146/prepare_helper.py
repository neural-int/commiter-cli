"""Copy the pinned helper to an experiment directory and add numeric telemetry.

Production source, generation parameters, prompts, and sampling are unchanged.
The experiment protocol is intentionally consumed only by benchmark146.
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
    path.write_text(text)
    manifest = {
        'original_helper_source_sha256': hashlib.sha256(original.encode()).hexdigest(),
        'instrumented_helper_source_sha256': hashlib.sha256(text.encode()).hexdigest(),
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
