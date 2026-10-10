"""Add numeric measurement to a copied pinned helper; never alter generation."""
import importlib.util
from pathlib import Path
import hashlib
import json
import sys


def prepare(source, destination):
    existing = Path(__file__).resolve().parents[1] / 'benchmark146/prepare_helper.py'
    spec = importlib.util.spec_from_file_location('benchmark146_instrumentation', existing)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    module.prepare(source, destination)
    path = destination / 'Sources/commiter-mlx-helper/CommiterMLXHelper.swift'
    text = path.read_text()
    replace = module.replace_once
    text = replace(text, '    var input: Int? = nil',
        '    let started = DispatchTime.now().uptimeNanoseconds\n'
        '    var loadSeconds: Double? = nil\n'
        '    var ttftSeconds: Double? = nil\n'
        '    var peakBytes: Int? = nil\n'
        '    var input: Int? = nil')
    text = replace(text, '    var benchmarkInputTokens: Int? = nil',
        '    var benchmarkLoadSeconds: Double? = nil\n'
        '    var benchmarkTTFTSeconds: Double? = nil\n'
        '    var benchmarkPeakBytes: Int? = nil\n'
        '    var benchmarkInputTokens: Int? = nil')
    text = replace(text, '        case benchmarkInputTokens = "benchmark_input_tokens"',
        '        case benchmarkLoadSeconds = "benchmark_load_seconds"\n'
        '        case benchmarkTTFTSeconds = "benchmark_ttft_seconds"\n'
        '        case benchmarkPeakBytes = "benchmark_peak_bytes"\n'
        '        case benchmarkInputTokens = "benchmark_input_tokens"')
    text = replace(text, '            let container = try await loadModelContainer(',
        '            let benchmarkLoadStart = DispatchTime.now().uptimeNanoseconds\n'
        '            let container = try await loadModelContainer(')
    text = replace(text, '            let outputBox = OutputBox()',
        '            benchmarkCounts.loadSeconds = Double(DispatchTime.now().uptimeNanoseconds - benchmarkLoadStart) / 1e9\n'
        '            let outputBox = OutputBox()')
    text = replace(text, '        let stream = try generate(input: input,',
        '        let benchmarkBeforeStream = Double(DispatchTime.now().uptimeNanoseconds - counts.started) / 1e9\n'
        '        let stream = try generate(input: input,')
    text = replace(text, '                counts.output = info.generationTokenCount',
        '                counts.output = info.generationTokenCount\n'
        '                counts.ttftSeconds = benchmarkBeforeStream + info.promptTime\n'
        '                counts.peakBytes = MLX.Memory.peakMemory')
    # Both success and failure carry numeric availability; failures may be null.
    text = text.replace('                benchmarkInputTokens: benchmarkCounts.input,',
        '                benchmarkLoadSeconds: benchmarkCounts.loadSeconds,\n'
        '                benchmarkTTFTSeconds: benchmarkCounts.ttftSeconds,\n'
        '                benchmarkPeakBytes: benchmarkCounts.peakBytes,\n'
        '                benchmarkInputTokens: benchmarkCounts.input,')
    path.write_text(text)
    manifest = json.loads((destination / 'benchmark-instrumentation.json').read_text())
    manifest['instrumented_helper_source_sha256'] = hashlib.sha256(text.encode()).hexdigest()
    manifest['ttft_definition'] = 'handle setup through stream setup + runtime promptTime (prefill through first generated token); excludes process startup'
    (destination / 'benchmark-instrumentation.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print(json.dumps(manifest))


if __name__ == '__main__':
    prepare(Path(sys.argv[1]), Path(sys.argv[2]))
