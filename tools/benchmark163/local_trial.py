"""Pre-register and run a bounded same-information local comparison."""
import argparse
import collections
import copy
import json
import pathlib
import re
import subprocess
import tempfile
import time

from harness import ROOT, MODEL_PINS, PROFILE, encode, sha, loads, messages, request, classify, authoritative, reconstruct, command, git_env
from resource import observed_call

OUT = ROOT / 'docs/benchmarks/issue-163'


def go(body):
    return 'package fixture\n\n' + body + '\n'


def file(name, before, after):
    return {'path': name, 'before': go(before), 'after': go(after)}


def dataset():
    specs = [
        ('path-escape-corresponding-assertion', 'join',
         file('escape.go', 'func Escape(s string) string { return s }', 'import "net/url"\nfunc Escape(s string) string { return url.PathEscape(s) }'),
         file('escape_test.go', 'import "testing"\nfunc TestSpace(t *testing.T) { if Escape("z z") != "z z" { t.Fatal("escape") } }', 'import "testing"\nfunc TestSpace(t *testing.T) { if Escape("z z") != "z%20z" { t.Fatal("escape") } }'), {}),
        ('inclusive-alert-corresponding-assertion', 'join',
         file('alert.go', 'func Alert(n int) bool { return n >= 100 }', 'func Alert(n int) bool { return n > 100 }'),
         file('alert_test.go', 'import "testing"\nfunc TestEdge(t *testing.T) { if !Alert(100) { t.Fatal("edge") } }', 'import "testing"\nfunc TestEdge(t *testing.T) { if Alert(100) { t.Fatal("edge") } }'), {}),
        ('shared-cleaner-independent-outputs', 'separate',
         file('job.go', 'func Job(s string) string { return Clean(s) }', 'func Job(s string) string { return "job:" + Clean(s) }'),
         file('quota.go', 'func Quota(s string) int { return len(Clean(s)) }', 'func Quota(s string) int { return len(Clean(s)) * 3 }'),
         {'clean.go': go('import "strings"\nfunc Clean(s string) string { return strings.TrimSpace(s) }'),
          'clean_test.go': go('import "testing"\nfunc TestClean(t *testing.T) { if Clean(" z ") != "z" { t.Fatal("clean") } }')}),
        ('weak-tests-independent-text-behaviors', 'separate',
         file('csv.go', 'func Csv(s string) string { return s }', 'import "strings"\nfunc Csv(s string) string { return strings.TrimSuffix(s, ",") }'),
         file('log.go', 'func LogLine(s string) string { return s }', 'import "strings"\nfunc LogLine(s string) string { return strings.ReplaceAll(s, "\\n", " ") }'),
         {'weak_test.go': go('import "testing"\nfunc TestNonempty(t *testing.T) { if Csv("a,") == "" || LogLine("a\\nb") == "" { t.Fatal("empty") } }')}),
        ('weak-tests-wire-compensation', 'join',
         file('writer.go', 'func WriteWire(s string) string { return s }', 'func WriteWire(s string) string { return "v2|" + s }'),
         file('reader.go', 'func ReadWire(s string) string { return s }', 'import "strings"\nfunc ReadWire(s string) string { return strings.TrimPrefix(s, "v2|") }'),
         {'invariant.go': go('// Roundtrip correctness is checked by this public invariant, but the weak test does not call it.\nfunc RoundtripStable(s string) bool { return ReadWire(WriteWire(s)) == s }'),
          'weak_test.go': go('import "testing"\nfunc TestNonempty(t *testing.T) { if ReadWire(WriteWire("abc")) == "" { t.Fatal("empty") } }')}),
        ('weak-tests-unit-scale-compensation', 'join',
         file('store.go', 'func Store(n int) int { return n * 2 }', 'func Store(n int) int { return n * 4 }'),
         file('restore.go', 'func Restore(n int) int { return n / 2 }', 'func Restore(n int) int { return n / 4 }'),
         {'invariant.go': go('// Stable is the source-visible roundtrip contract; the smoke test only observes positivity.\nfunc Stable(n int) bool { return Restore(Store(n)) == n }'),
          'weak_test.go': go('import "testing"\nfunc TestPositive(t *testing.T) { if Restore(Store(40)) <= 0 { t.Fatal("nonpositive") } }')}),
        ('new-cap-provider-ea', 'dependency',
         file('cap.go', 'func Identity(n int) int { return n }', 'func Identity(n int) int { return n }\nfunc Cap(n int) int { if n > 90 { return 90 }; return n }'),
         file('consume.go', 'func Consume(n int) int { return Identity(n) }', 'func Consume(n int) int { return Cap(n) }'), {}),
        ('new-wrap-provider-eb', 'dependency',
         file('consume.go', 'func Consume(s string) string { return Identity(s) }', 'func Consume(s string) string { return Wrap(s) }'),
         file('wrap.go', 'func Identity(s string) string { return s }', 'func Identity(s string) string { return s }\nfunc Wrap(s string) string { return "<" + s + ">" }'), {}),
        ('retry-policy-multiple-partitions', 'multiple',
         file('read.go', 'const ReadRetry = 5', 'const ReadRetry = 9'),
         file('write.go', 'const WriteRetry = 5', 'const WriteRetry = 9'), {}),
        ('external-limit-evidence-insufficient', 'insufficient',
         file('read.go', 'func ReadCapacity() int { return 8 }', 'func ReadCapacity() int { return 12 }'),
         file('write.go', 'func WriteCapacity() int { return 6 }', 'func WriteCapacity() int { return 10 }'),
         {'external.go': go('// The client-selected policy implementation and constraints are not supplied.\ntype ExternalLimit interface { Accepts(read, write int) bool }\nfunc ValidFor(policy ExternalLimit) bool { return policy.Accepts(ReadCapacity(), WriteCapacity()) }')}),
    ]
    rows = []
    for name, kind, a, b, support in specs:
        a = dict(a, id='EA'); b = dict(b, id='EB')
        support = dict(support, **{'go.mod': 'module example.test/issue163fixture\n\ngo 1.23.0\n'})
        rows.append({'name': name, 'kind': kind, 'files': [a, b], 'support': support,
                     'input_sha256': sha(encode({'files': [a, b], 'support': support})),
                     'gold_basis': {'join': 'direct changed implementation/assertion or source-visible roundtrip compensation',
                                    'separate': 'distinct observable behaviors; shared helper or weak smoke tests are not coupling',
                                    'dependency': 'new API provider may be committed alone; merge or provider-first split',
                                    'multiple': 'no requirement linking policy constants; merge or split are both allowed',
                                    'insufficient': 'external policy constraints needed to assess the changed pair are absent; defer required by this diagnostic'}[kind]})
    return rows


def edges(facts):
    result = []
    for provider in facts:
        added = set(provider['after']['definitions']) - set(provider['before']['definitions'])
        for user in facts:
            if provider['id'] == user['id'] or provider['after']['package'] != user['after']['package']:
                continue
            symbols = added & (set(user['after']['calls']) - set(user['before']['calls']))
            for symbol in symbols:
                result.append({'provider': provider['id'], 'consumer': user['id'], 'symbol': symbol})
    return sorted(result, key=lambda r: (r['provider'], r['consumer'], r['symbol']))


def split(case):
    return [[case['edges'][0]['provider']], [case['edges'][0]['consumer']]] if case['edges'] else [['EA'], ['EB']]


def allowed(case, decision):
    return (decision == 'merge' if case['kind'] == 'join' else
            decision == 'keep_separate' if case['kind'] == 'separate' else
            decision == 'defer' if case['kind'] == 'insufficient' else
            decision in ('merge', 'keep_separate'))


def state(case, selected):
    with tempfile.TemporaryDirectory(prefix='issue163-state-') as work:
        for name, content in case['support'].items(): pathlib.Path(work, name).write_text(content)
        for f in case['files']: pathlib.Path(work, f['path']).write_text(f['after' if f['id'] in selected else 'before'])
        proc = subprocess.run(['/usr/bin/sandbox-exec', '-p', '(version 1)(allow default)(deny network*)', 'go', 'test', './...'],
                              cwd=work, capture_output=True, timeout=30,
                              env=dict(git_env(), GOCACHE=str(pathlib.Path(tempfile.gettempdir()) / 'issue163-fixture-go-cache')))
        raw = proc.stdout + proc.stderr
        return {'pass': proc.returncode == 0, 'exit': proc.returncode, 'raw_sha256': sha(raw),
                'log': re.sub(r'/(?:private/)?(?:tmp|var/folders)/[^\s:]+', '<temporary>', raw.decode(errors='replace'))}


def plan_audit(case, groups, validator):
    final = {'review_valid': allowed(case, 'merge' if len(groups) == 1 else 'keep_separate'),
             'authoritative': authoritative(validator, ['EA', 'EB'], groups),
             'reconstruction': reconstruct(case['files'], groups, sha(encode({f['path']: f['after'] for f in case['files']}))),
             'ordered': [], 'independent_revert': []}
    selected = set()
    for phase, sequence in [('apply', groups), ('ordered_revert', list(reversed(groups)))]:
        for group in sequence:
            selected.update(group) if phase == 'apply' else selected.difference_update(group)
            final['ordered'].append({'phase': phase, 'group': group, 'test': state(case, selected)})
    for group in groups:
        final['independent_revert'].append({'group': group, 'test': state(case, {'EA', 'EB'} - set(group))})
    final['all_ordered_pass'] = all(r['test']['pass'] for r in final['ordered'])
    final['all_independent_revert_pass'] = all(r['test']['pass'] for r in final['independent_revert'])
    return final


def prepare(helper, facts_binary, validator):
    preparation_start = time.monotonic()
    cases = dataset(); rows = []; audits = []
    previous = subprocess.check_output(['git', 'show', 'b8a7fe14a798ff36680efffdbd879c9a02cb37f2:docs/benchmarks/issue-162/iteration-2-preregistered.json'])
    prior = loads(previous)
    prior_hashes = {sha(f[side].encode()) for c in prior['cases'] for f in c['files'] for side in ('before', 'after')}
    for case in cases:
        case['facts'] = loads(command([facts_binary], encode(case['files'])).stdout)
        case['edges'] = edges(case['facts'])
        case['used_before_this_experiment'] = False
        case['prior162_changed_source_matches'] = [f['id'] + ':' + side for f in case['files'] for side in ('before', 'after') if sha(f[side].encode()) in prior_hashes]
        probes = {name: state(case, selected) for name, selected in [('before', set()), ('only_a', {'EA'}), ('only_b', {'EB'}), ('after', {'EA', 'EB'})]}
        assert probes['before']['pass'] and probes['after']['pass'], (case['name'], probes)
        if case['kind'] == 'join' and 'corresponding' in case['name']:
            assert not probes['only_a']['pass'] and not probes['only_b']['pass']
        if 'weak-tests' in case['name']:
            assert all(p['pass'] for p in probes.values())
        if case['kind'] == 'dependency':
            assert len(case['edges']) == 1
            assert not probes['only_a' if case['edges'][0]['consumer'] == 'EA' else 'only_b']['pass']
        baselines = {'file-only': [['EA'], ['EB']], 'static-new-api-order': split(case)}
        audits.append({'case': case['name'], 'endpoint_probes': probes,
                       'baselines': {name: plan_audit(case, groups, validator) for name, groups in baselines.items()}})
        for reverse in (False, True):
            msg = messages('local', case['files'], case['support'], case['facts'], reverse)
            rows.append({'case': case['name'], 'reverse': reverse, 'messages': msg, 'prompt_sha256': sha(encode(msg))})
    assert all(not c['prior162_changed_source_matches'] for c in cases)
    fixed = {'issue': 163, 'cases': cases, 'rows': rows, 'models': MODEL_PINS,
             'helper_sha256': sha(pathlib.Path(helper).read_bytes()),
             'metal_library_sha256': sha(pathlib.Path(helper).parent.joinpath('mlx.metallib').read_bytes()),
             'facts_sha256': sha(pathlib.Path(facts_binary).read_bytes()), 'validator_sha256': sha(pathlib.Path(validator).read_bytes()),
             'source_sha256': {str(p.relative_to(ROOT)): sha(p.read_bytes()) for p in pathlib.Path(__file__).parent.rglob('*') if p.is_file()},
             'context_tokens': 16384, 'output_tokens': 1536, 'per_call_seconds': 30, 'whole_seconds': 1800,
             'audit_reserve_seconds': 400, 'max_model_calls': 40, 'retries': 0, 'profile': PROFILE,
             'input_tokens': 'must be reported by helper per native template; oversize is rejected, never truncated',
             'freshness': 'never inferred before this registration; self-authored diagnostic, not independent adoption holdout',
             'family_correlation': 'implementation/assertion and API cases share established task families with #162; compensation cases share a roundtrip family; source is not an independent author/repository corpus',
             'memory_gate_bytes': 5000000000, 'memory_metric': 'sampled process phys_footprint; observed lifetime peak retained separately',
             'parallel_load': 'current developer apps retained; per-call before/after pressure and swap; no artificial idle clearing',
             'ttft_definition': 'helper handle start after request validation to first sampled token via identical grammar processor; external process total includes startup/exit',
             'release_definition': 'one fresh helper process per call; exit and subsequent system snapshot; per-process allocation released by OS, system cache may remain',
             'gate': 'each model20/20 backend completed/schema/source-ref valid; all20 expected allowed decisions including insufficient defer; negative FM0/necessary FS0; both orders10/10 same decision; authoritative/byte/ordered states all pass for finalized plans; improves static baseline on corresponding assertions; footprint<=5e9, no swap growth or pressure warning; within whole budget',
             'preparation_audit_seconds': time.monotonic() - preparation_start, 'unknown_metrics': None, 'metadata_evaluated': False, 'production_evaluated': False}
    for name, data in [('iteration-2-preregistered.json', fixed), ('iteration-2-input-audit.json', audits)]:
        with (OUT / name).open('x') as out: json.dump(data, out, ensure_ascii=False, indent=2); out.write('\n')
    print(json.dumps({'cases': len(cases), 'rows_per_model': len(rows), 'max_calls': 40, 'model_calls': 0}), flush=True)


def swap_bytes(snapshot):
    text = snapshot.get('swap', {}).get('stdout', '')
    match = re.search(r'used = ([0-9.]+)([MG])', text)
    return float(match[1]) * (1024 ** (2 if match[2] == 'M' else 3)) if match else None


def run(helper, gemma, coder, facts_binary, validator):
    fixed = loads((OUT / 'iteration-2-preregistered.json').read_bytes())
    for path, digest in fixed['source_sha256'].items(): assert sha((ROOT / path).read_bytes()) == digest
    assert sha(pathlib.Path(helper).read_bytes()) == fixed['helper_sha256']
    assert sha(pathlib.Path(helper).parent.joinpath('mlx.metallib').read_bytes()) == fixed['metal_library_sha256']
    assert sha(pathlib.Path(facts_binary).read_bytes()) == fixed['facts_sha256']
    assert sha(pathlib.Path(validator).read_bytes()) == fixed['validator_sha256']
    start = time.monotonic(); deadline = start + fixed['whole_seconds'] - fixed.get('preparation_audit_seconds', 0); cases = {c['name']: c for c in fixed['cases']}
    results = []; cached_audits = {}; paths = {'gemma': gemma, 'coder': coder}
    with (OUT / 'iteration-2-results.jsonl').open('x') as out:
        # Alternate model order across input rows, to retain rather than hide order/pressure effects.
        for index, spec in enumerate(fixed['rows']):
            for model in (('gemma', 'coder') if index % 2 == 0 else ('coder', 'gemma')):
                case = cases[spec['case']]
                row = {'model': model, 'case': spec['case'], 'reverse': spec['reverse'], 'prompt_sha256': spec['prompt_sha256'],
                       'input_tokens': None, 'output_tokens': None, 'calls': 0, 'response_valid': False, 'allowed': False}
                remaining = deadline - time.monotonic() - fixed['audit_reserve_seconds']
                if remaining <= 0:
                    row.update(status='total_budget', resource=None)
                else:
                    req = request(model, paths[model], helper, fixed['helper_sha256'], 'local', case['files'], case['support'], case['facts'], spec['reverse'])
                    assert req['messages'] == spec['messages']
                    row['calls'] = 1
                    measured = observed_call([helper], encode(req) + b'\n', min(30, remaining))
                    row['resource'] = measured
                    response = classify(measured['stdout'].encode(), 'local', ['EA', 'EB'])
                    if measured['timed_out']: response.update(status='timeout', accepted=False)
                    elif measured['exit'] != 0: response.update(status='helper_exit', accepted=False)
                    row.update(status=response['status'], response=response,
                               response_valid=response['status'] in ('completed', 'unresolved'),
                               input_tokens=response['input_tokens'], output_tokens=response['output_tokens'])
                    if row['response_valid']:
                        answer = response['answer']; row['answer'] = answer; row['allowed'] = allowed(case, answer['decision'])
                        if answer['decision'] != 'defer':
                            groups = [['EA', 'EB']] if answer['decision'] == 'merge' else split(case)
                            key = (case['name'], encode(groups))
                            if key not in cached_audits: cached_audits[key] = plan_audit(case, groups, validator)
                            row.update(groups=groups, audit=cached_audits[key])
                    try:
                        metrics = loads(measured['stdout'].encode())
                        for name in ('ttft_seconds', 'load_seconds', 'prepared_seconds', 'first_chunk_seconds', 'prefill_seconds', 'decode_seconds', 'mlx_active_bytes', 'mlx_peak_bytes', 'mlx_cache_bytes'):
                            measured[name] = metrics.get('benchmark_' + name)
                    except ValueError: pass
                    peak = measured['sampled_peak_footprint_bytes']; before = swap_bytes(measured['system_before']); after = swap_bytes(measured['system_after'])
                    measured['swap_delta_bytes'] = after - before if before is not None and after is not None else None
                    measured['memory_within_limit'] = peak is not None and peak <= fixed['memory_gate_bytes']
                    levels = [p['level'] for p in measured.get('pressure_samples', [])]
                    for moment in ('system_before', 'system_after'):
                        raw_level = measured[moment].get('pressure_level', {}).get('stdout', '').strip()
                        levels.append(int(raw_level) if raw_level.isdigit() else None)
                    measured['pressure_levels'] = levels
                    measured['pressure_normal'] = bool(levels) and all(level == 1 for level in levels)
                    measured['resource_gate'] = bool(measured['pressure_normal'] and measured['memory_within_limit'] and measured['swap_delta_bytes'] is not None and measured['swap_delta_bytes'] <= 0 and measured['ttft_seconds'] is not None)
                row['elapsed_seconds'] = time.monotonic() - start
                results.append(row); out.write(json.dumps(row, ensure_ascii=False) + '\n'); out.flush()
                print(json.dumps({'model': model, 'case': row['case'], 'reverse': row['reverse'], 'status': row['status'], 'allowed': row['allowed'], 'elapsed': row['elapsed_seconds']}), flush=True)
    summary = {'issue': 163, 'elapsed_seconds': time.monotonic() - start + fixed.get('preparation_audit_seconds', 0), 'inference_and_post_audit_seconds': time.monotonic() - start, 'preparation_audit_seconds': fixed.get('preparation_audit_seconds', 0), 'max_calls': 40, 'model_calls': sum(r['calls'] for r in results), 'models': {}}
    for model in fixed['models']:
        rows = [r for r in results if r['model'] == model]
        pairs = [[r for r in rows if r['case'] == c['name']] for c in cases.values()]
        stable = sum(all(r['response_valid'] for r in pair) and pair[0]['answer']['decision'] == pair[1]['answer']['decision'] for pair in pairs)
        fm = sum(r.get('answer', {}).get('decision') == 'merge' and cases[r['case']]['kind'] == 'separate' for r in rows)
        fs = sum(r.get('answer', {}).get('decision') == 'keep_separate' and cases[r['case']]['kind'] == 'join' for r in rows)
        safe = all(r.get('audit', {}).get('authoritative', {}).get('valid') and r.get('audit', {}).get('reconstruction', {}).get('exact') and r.get('audit', {}).get('all_ordered_pass') for r in rows if r.get('answer', {}).get('decision') != 'defer')
        memory = all((r.get('resource') or {}).get('resource_gate', False) for r in rows)
        valid = sum(r['response_valid'] for r in rows); accepted = sum(r['allowed'] for r in rows)
        gate = valid == 20 and accepted == 20 and stable == 10 and fm == fs == 0 and safe and memory and summary['elapsed_seconds'] <= 1800
        summary['models'][model] = {'responses_valid': valid, 'allowed': accepted, 'total': len(rows), 'stable_cases': stable, 'cases': 10,
                                    'negative_false_merge': fm, 'required_false_split': fs,
                                    'negative_false_merge_cases': len({r['case'] for r in rows if r.get('answer', {}).get('decision') == 'merge' and cases[r['case']]['kind'] == 'separate'}),
                                    'required_false_split_cases': len({r['case'] for r in rows if r.get('answer', {}).get('decision') == 'keep_separate' and cases[r['case']]['kind'] == 'join'}),
                                    'input_tokens_total': sum(r['input_tokens'] for r in rows) if all(r['input_tokens'] is not None for r in rows) else None,
                                    'output_tokens_total': sum(r['output_tokens'] for r in rows) if all(r['output_tokens'] is not None for r in rows) else None, 'unresolved': sum(r['status'] == 'unresolved' for r in rows),
                                    'statuses': dict(collections.Counter(r['status'] for r in rows)), 'resource_gate': memory, 'plan_audits': safe,
                                    'sampled_peak_footprint_bytes': max((r['resource']['sampled_peak_footprint_bytes'] for r in rows if r.get('resource') and r['resource']['sampled_peak_footprint_bytes'] is not None), default=None),
                                    'local_gate': gate, 'global_status': 'eligible_for_separate_registration' if gate else 'gated_out',
                                    'metadata_and_production': 'not_evaluated'}
    with (OUT / 'iteration-2-summary.json').open('x') as out: json.dump(summary, out, indent=2); out.write('\n')
    print(json.dumps(summary), flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(); parser.add_argument('action', choices=('prepare', 'run'))
    for name in ('helper', 'facts', 'validator'): parser.add_argument('--' + name, required=True)
    parser.add_argument('--gemma'); parser.add_argument('--coder'); args = parser.parse_args()
    if args.action == 'prepare': prepare(args.helper, args.facts, args.validator)
    else: run(args.helper, args.gemma, args.coder, args.facts, args.validator)
