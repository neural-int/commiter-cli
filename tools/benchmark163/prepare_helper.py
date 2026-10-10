"""Materialize a pinned research helper plus an explicit experiment-only patch."""
import base64
import hashlib
import json
import pathlib
import subprocess

ARCHIVE_COMMIT = '82a179a11e0c75562fadd595f208584f7a43cdb0'
ARCHIVE_PATH = 'docs/benchmarks/issue-156/measured-helper-source-archive.json'
CODER_PIN = 'mlx-community/Qwen2.5-Coder-3B-Instruct-4bit@3dd939c621c08e5753d5b89f35a2642cd83b98ca'


def prepare(destination):
    root = pathlib.Path(__file__).resolve().parents[2]
    archive = json.loads(subprocess.check_output(['git', 'show', ARCHIVE_COMMIT + ':' + ARCHIVE_PATH], cwd=root))
    dest = pathlib.Path(destination); dest.mkdir(parents=True, exist_ok=True)
    provenance = {'archive_commit': ARCHIVE_COMMIT, 'archive_path': ARCHIVE_PATH, 'source': {},
                  'production_modified': False, 'new_dependencies': False}
    for entry in archive['files']:
        raw = base64.b64decode(entry['bytes_b64']) if 'bytes_b64' in entry else subprocess.check_output([
            'git', 'show', archive['base_revision'] + ':' + archive['base_source_prefix'] + '/' + entry['path']], cwd=root)
        assert hashlib.sha256(raw).hexdigest() == entry['sha256']
        target = dest / entry['path']; target.parent.mkdir(parents=True, exist_ok=True)
        if not target.exists() or target.read_bytes() != raw: target.write_bytes(raw)
        provenance['source'][entry['path']] = {'before_sha256': entry['sha256']}
    test_paths = subprocess.check_output(['git', 'ls-tree', '-r', '--name-only', archive['base_revision'], 'mlx-helper/Tests'], cwd=root, text=True).splitlines()
    for test_path in test_paths:
        raw = subprocess.check_output(['git', 'show', archive['base_revision'] + ':' + test_path], cwd=root)
        relative = test_path.removeprefix('mlx-helper/')
        target = dest / relative; target.parent.mkdir(parents=True, exist_ok=True)
        if not target.exists() or target.read_bytes() != raw: target.write_bytes(raw)
        provenance['source'][relative] = {'before_sha256': hashlib.sha256(raw).hexdigest()}
    target = dest / 'Sources/commiter-mlx-helper/CommiterMLXHelper.swift'
    source = target.read_text()
    edits = [
        ('["mlx-community/Qwen3.5-4B-MLX-4bit@', '["' + CODER_PIN + '","mlx-community/Qwen3.5-4B-MLX-4bit@'),
        ('    var output: Int? = nil\n', '''    var output: Int? = nil
    let started = Date()
    var loadedSeconds: Double? = nil
    var preparedSeconds: Double? = nil
    var firstChunkSeconds: Double? = nil
    var firstTokenSeconds: Double? = nil
    var prefillSeconds: Double? = nil
    var decodeSeconds: Double? = nil
'''),
        ('    var benchmarkOutputTokens: Int? = nil\n', '''    var benchmarkOutputTokens: Int? = nil
    var benchmarkLoadSeconds: Double? = nil
    var benchmarkPreparedSeconds: Double? = nil
    var benchmarkFirstChunkSeconds: Double? = nil
    var benchmarkTTFTSeconds: Double? = nil
    var benchmarkPrefillSeconds: Double? = nil
    var benchmarkDecodeSeconds: Double? = nil
    var benchmarkMLXActiveBytes: Int? = nil
    var benchmarkMLXPeakBytes: Int? = nil
    var benchmarkMLXCacheBytes: Int? = nil
'''),
        ('        case benchmarkOutputTokens = "benchmark_output_tokens"\n', '''        case benchmarkOutputTokens = "benchmark_output_tokens"
        case benchmarkLoadSeconds = "benchmark_load_seconds"
        case benchmarkPreparedSeconds = "benchmark_prepared_seconds"
        case benchmarkFirstChunkSeconds = "benchmark_first_chunk_seconds"
        case benchmarkTTFTSeconds = "benchmark_ttft_seconds"
        case benchmarkPrefillSeconds = "benchmark_prefill_seconds"
        case benchmarkDecodeSeconds = "benchmark_decode_seconds"
        case benchmarkMLXActiveBytes = "benchmark_mlx_active_bytes"
        case benchmarkMLXPeakBytes = "benchmark_mlx_peak_bytes"
        case benchmarkMLXCacheBytes = "benchmark_mlx_cache_bytes"
'''),
        ('            let outputBox = OutputBox()\n', '            benchmarkCounts.loadedSeconds = Date().timeIntervalSince(benchmarkCounts.started)\n            let outputBox = OutputBox()\n'),
        ('                benchmarkOutputTokens: benchmarkCounts.output\n', '''                benchmarkOutputTokens: benchmarkCounts.output,
                benchmarkLoadSeconds: benchmarkCounts.loadedSeconds,
                benchmarkPreparedSeconds: benchmarkCounts.preparedSeconds,
                benchmarkFirstChunkSeconds: benchmarkCounts.firstChunkSeconds,
                benchmarkTTFTSeconds: benchmarkCounts.firstTokenSeconds,
                benchmarkPrefillSeconds: benchmarkCounts.prefillSeconds,
                benchmarkDecodeSeconds: benchmarkCounts.decodeSeconds,
                benchmarkMLXActiveBytes: Memory.activeMemory,
                benchmarkMLXPeakBytes: Memory.peakMemory,
                benchmarkMLXCacheBytes: Memory.cacheMemory
'''),
        ('        counts.input = input.text.tokens.size\n', '        counts.preparedSeconds = Date().timeIntervalSince(counts.started)\n        counts.input = input.text.tokens.size\n'),
        ('            case .chunk(let chunk): output += chunk\n', '''            case .chunk(let chunk):
                if counts.firstChunkSeconds == nil { counts.firstChunkSeconds = Date().timeIntervalSince(counts.started) }
                output += chunk
'''),
    ]
    edits += [
        ('    let generationProfile: String?\n', '    let generationProfile: String?\n    let preflightOnly: Bool\n'),
        ('        case modelPath = "model_path"\n        case generationProfile = "generation_profile"\n', '        case modelPath = "model_path"\n        case generationProfile = "generation_profile"\n        case preflightOnly = "preflight_only"\n'),
        ('        generationProfile = try container.decodeIfPresent(String.self, forKey: .generationProfile)\n', '        generationProfile = try container.decodeIfPresent(String.self, forKey: .generationProfile)\n        preflightOnly = try container.decodeIfPresent(Bool.self, forKey: .preflightOnly) ?? false\n'),
        ('        catch { throw HelperError.inputTooLarge }\n', '        catch { throw HelperError.inputTooLarge }\n        if request.preflightOnly { return "{}" }\n'),
        ('components = GenerationComponents(logitProcessorFactory: { GrammarSamplingProcessor(state: state) })', 'components = GenerationComponents(logitProcessorFactory: { MeasuredGrammarProcessor(state: state, counts: counts) })'),
        ('                counts.output = info.generationTokenCount', '                counts.prefillSeconds = info.promptTime\n                counts.decodeSeconds = info.generateTime\n                counts.output = info.generationTokenCount'),
        ('private struct Response: Encodable {', '''// Experiment-only observation wrapper. Mask/commit/copy semantics match the pinned processor.
private struct MeasuredGrammarProcessor: LogitProcessor {
    let state: GrammarSamplingState
    let counts: BenchmarkCounts
    mutating func prompt(_ prompt: MLXArray) {}
    func process(logits: MLXArray) -> MLXArray {
        logits + MLXArray(state.maskValues(count: logits.dim(-1)))
    }
    mutating func didSample(token: MLXArray) {
        let value = token.item(Int.self)
        if counts.firstTokenSeconds == nil { counts.firstTokenSeconds = Date().timeIntervalSince(counts.started) }
        state.commit(value)
    }
    func copy() -> Self { Self(state: state.independentCopy(), counts: counts) }
}

private struct Response: Encodable {'''),
    ]
    provenance['changes'] = []
    for before, after in edits:
        count = source.count(before)
        assert count in (1, 2), (before, count)
        source = source.replace(before, after)
        provenance['changes'].append({'before': before, 'after': after, 'occurrences': count})
    target.write_text(source)
    for path in provenance['source']:
        provenance['source'][path]['after_sha256'] = hashlib.sha256((dest / path).read_bytes()).hexdigest()
    return provenance


if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(); parser.add_argument('--destination', required=True); parser.add_argument('--provenance', required=True)
    args = parser.parse_args(); result = prepare(args.destination)
    with open(args.provenance, 'x') as out: json.dump(result, out, indent=2); out.write('\n')
    print(json.dumps({'files': len(result['source']), 'new_dependencies': False}))
