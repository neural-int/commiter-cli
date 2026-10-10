"""Issue166 fixed single-factor diagnostic, reusing the Issue165 B1 contract."""
import argparse
import copy
import difflib
import json
import os
import pathlib
import statistics
import subprocess
import sys
import time
from collections import Counter

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'tools/benchmark165'))
import diagnose as baseline
from harness import encode, loads, sha
from local_trial import swap_bytes
from resource import observed_call, system

OUT = ROOT / 'docs/benchmarks/issue-166'
BRANCH = 'codex/issue-166-schema-diagnostic'
BASE = '4b9ac67e1cf1317986b659935c3970192e83a41f'
FIELDS = ('helper', 'coder', 'facts', 'validator', 'helper_source')
T2_PROMPT = baseline.B_PROMPT.replace(' (provider/consumer direction required)', '').replace(
    'For nondependency use provider=consumer=none. ', '')


def write(name, value):
    with (OUT / name).open('x') as f:
        json.dump(value, f, ensure_ascii=False, indent=2)
        f.write('\n')


def load(name):
    return loads((OUT / name).read_bytes())


def old():
    return loads((ROOT / 'docs/benchmarks/issue-165/preregistered.json').read_bytes())


def args():
    p = old()
    return argparse.Namespace(**{k: p[k + ('_path' if k != 'coder' else '')]
        if k != 'coder' else p['model_path'] for k in FIELDS})


def identity():
    found = baseline.identity(args())
    assert found == old()['identity'], 'baseline model/helper/runtime/source contract mismatch'
    return dict(baseline=found, diagnostic_sha256=sha(pathlib.Path(__file__).read_bytes()))


def treatment(spec, arm):
    result = copy.deepcopy(spec)
    result['stage'] = arm
    if arm == 'T1':
        result['schema']['properties']['relation']['enum'].reverse()
        prompt = baseline.B_PROMPT
    else:
        assert arm == 'T2'
        for k in ('provider', 'consumer'):
            del result['schema']['properties'][k]
            result['schema']['required'].remove(k)
        prompt = T2_PROMPT
    result['messages'][0]['content'] = prompt + '\nSchema: ' + encode(result['schema']).decode()
    result['prompt_sha256'] = sha(encode(result['messages']))
    result.pop('native_input_tokens')
    return result


def diff_audit(original, changed):
    arm = changed['stage']
    expected = treatment(original, arm)
    assert changed == expected
    assert changed['messages'][1:] == original['messages'][1:]
    original_prompt, original_schema = original['messages'][0]['content'].split('\nSchema: ', 1)
    changed_prompt, changed_schema = changed['messages'][0]['content'].split('\nSchema: ', 1)
    assert original_prompt == baseline.B_PROMPT
    assert loads(original_schema) == original['schema']
    assert loads(changed_schema) == changed['schema']
    restored = copy.deepcopy(changed['schema'])
    if arm == 'T1':
        restored['properties']['relation']['enum'].reverse()
        assert restored == original['schema'] and changed_prompt == original_prompt
        paths = ['schema.properties.relation.enum', 'messages[0].content embedded Schema relation.enum']
    else:
        assert restored['properties']['relation'] == original['schema']['properties']['relation']
        for k in ('provider', 'consumer'):
            restored['properties'][k] = original['schema']['properties'][k]
        restored['required'] = original['schema']['required']
        assert restored == original['schema'] and changed_prompt == T2_PROMPT
        paths = ['schema.properties.provider/consumer', 'schema.required provider/consumer',
                 'messages[0].content embedded Schema equivalents',
                 'messages[0].content removal of two direction instructions']
    return dict(case=original['case'], reverse=original['reverse'], arm=arm,
                changed_paths=paths, user_byte_identical=True,
                prose_label_definitions_unchanged=(arm == 'T1'),
                schema_diff=''.join(difflib.unified_diff(
                    json.dumps(original['schema'], indent=2).splitlines(True),
                    json.dumps(changed['schema'], indent=2).splitlines(True), fromfile='B1', tofile=arm)),
                system_prose_diff=''.join(difflib.unified_diff(original_prompt.splitlines(True),
                    changed_prompt.splitlines(True), fromfile='B1', tofile=arm)))


def make_request(spec, c):
    return baseline.make_request(args(), old(), spec, c)


def resource_fields(res):
    outer = loads(res['stdout'])
    for name in ('ttft_seconds', 'load_seconds', 'prepared_seconds', 'first_chunk_seconds',
                 'prefill_seconds', 'decode_seconds', 'mlx_active_bytes', 'mlx_peak_bytes', 'mlx_cache_bytes'):
        res[name] = outer.get('benchmark_' + name)
    before, after = (swap_bytes(res[k]) for k in ('system_before', 'system_after'))
    res['swap_delta_bytes'] = after - before if before is not None and after is not None else None
    res['pressure_levels'] = [p['level'] for p in res['pressure_samples']]
    for k in ('system_before', 'system_after'):
        p = res[k]['pressure_level']
        res['pressure_levels'].append(int(p['stdout'].strip()) if p.get('exit') == 0 and p.get('stdout', '').strip().isdigit() else None)
    return outer


def stop_reason(res, initial_swap):
    if res['sampled_peak_footprint_bytes'] is None or res['observed_lifetime_peak_footprint_bytes'] is None:
        return 'resource_unmeasurable'
    if res['swap_delta_bytes'] is None or not res['pressure_levels'] or any(p is None for p in res['pressure_levels']):
        return 'resource_unmeasurable'
    if max(res['sampled_peak_footprint_bytes'], res['observed_lifetime_peak_footprint_bytes']) > 5_000_000_000:
        return 'footprint_limit'
    if max(res['pressure_levels']) >= 4:
        return 'critical_pressure'
    current = swap_bytes(res['system_after'])
    if res['swap_delta_bytes'] > 256 * 1024**2 or current - initial_swap > 512 * 1024**2:
        return 'swap_limit'
    if res['timed_out'] or res['exit'] != 0:
        return 'timeout_or_helper_exit'
    return None


def prepare():
    start = time.monotonic()
    OUT.mkdir(parents=True, exist_ok=True)
    ident = identity()
    p = old()
    cases = {c['name']: c for c in p['cases']}
    originals = [s for s in p['rows'] if s['stage'] == 'B1']
    records = [loads(l) for l in (ROOT / 'docs/benchmarks/issue-165/results.jsonl').read_bytes().splitlines()]
    records = [r for r in records if r['stage'] == 'B1']
    assert len(originals) == len(records) == 16
    assert len({(r['case'], r['reverse']) for r in records}) == 16
    for s in originals:
        r = next(r for r in records if (r['case'], r['reverse']) == (s['case'], s['reverse']))
        assert r['calls'] == 1 and r['status'] == 'completed' and r['schema_valid']
        assert r['prompt_sha256'] == s['prompt_sha256'] == sha(encode(s['messages']))
        baseline.validate(r['answer'], s['schema'])
        assert r['answer']['relation'] == 'compensation'
    reference = dict(original_specs=originals, records=records,
        hashes={name: sha((ROOT / 'docs/benchmarks/issue-165' / name).read_bytes())
                for name in ('preregistered.json', 'results.jsonl', 'source-audit.json')},
        baseline_regenerated=False, baseline_is_historical=True)
    assert load('baseline-reference.json') == reference
    rows, diffs, tokens = [], [], []
    initial_system = system()
    initial_swap = swap_bytes(initial_system)
    assert initial_swap is not None
    recovered = load('preparation-failure-diagnostic.json')
    assert recovered['stop_reason'] is None and recovered['generation_calls'] == 0
    token_stream = (OUT / 'tokenizer-observations.jsonl').open('x')
    for i, s in enumerate(originals):
        for arm in (('T1', 'T2') if i % 2 == 0 else ('T2', 'T1')):
            spec = treatment(s, arm)
            diffs.append(diff_audit(s, spec))
            req = make_request(spec, cases[s['case']])
            spec['request'] = req
            spec['request_string'] = (encode(req) + b'\n').decode()
            spec['request_sha256'] = sha(spec['request_string'].encode())
            pre = dict(req, preflight_only=True)
            res = recovered['resource'] if not rows else observed_call(baseline.SANDBOX + [args().helper], encode(pre) + b'\n', 30)
            r = resource_fields(res)
            stop = stop_reason(res, initial_swap)
            token_stream.write(json.dumps(dict(case=s['case'], reverse=s['reverse'], arm=arm, resource=res, stop_reason=stop), ensure_ascii=False) + '\n')
            token_stream.flush()
            assert stop is None, stop
            n = r.get('benchmark_input_tokens')
            assert r.get('ok') is True and r.get('stop_reason') == 'completed' and r.get('generated_json') == '{}'
            assert r.get('model') == p['model'] and r.get('benchmark_output_tokens') is None
            assert type(n) is int and n <= 4096 and n + 1536 <= 16384
            spec['native_input_tokens'] = n
            rows.append(spec)
            tokens.append(dict(case=s['case'], reverse=s['reverse'], arm=arm, resource=res, input_tokens=n))
            print(json.dumps(dict(case=s['case'], arm=arm, reverse=s['reverse'], tokens=n, generation_calls=0)), flush=True)
    token_stream.close()
    write('diff-audit.json', diffs)
    write('native-token-audit.json', tokens)
    write('preregistered.json', dict(issue=166, base_revision=BASE, identity=ident,
        cases=p['cases'], rows=rows, model=p['model'], profile=p['profile'],
        decoder='same helper neutral grammar; schema is BOTH system text and grammar argument',
        temperature=0, seed=144, top_p=1, top_k=0, native_thought_tokens=0,
        context_tokens=16384, output_tokens=1536, max_input_tokens=4096,
        max_generation_calls=32, max_tokenizer_calls=33, tokenizer_calls=33, retries=0,
        preparation_repair='Initial tokenizer resource assertion stopped before raw preservation; cause undetermined. One diagnostic tokenizer-only repeat saved full data and passed; reused as first row, then 31 calls. No generated outcome or treatment/gold/limit change. Initial data gap retained, not reconstructed.',
        whole_seconds=1800, per_call_seconds=30, audit_reserve_seconds=200,
        preparation_seconds=time.monotonic() - start,
        initial_system=initial_system, artifacts_sha256={name: sha((OUT / name).read_bytes()) for name in
            ('baseline-reference.json', 'diff-audit.json', 'native-token-audit.json', 'tokenizer-observations.jsonl', 'preparation-failure-diagnostic.json', 'contract.md', 'goal-source.json', 'issue-source.json')},
        stop_policy='Stop further calls on unmeasurable footprint/pressure/swap, digest/input-token/model/backend contract mismatch, total/time budget, footprint>5e9, critical pressure>=4, per-call swap increase>256MiB or cumulative increase>512MiB. Warning level2 and smaller swap increases retained, not resource qualification. Limits checked between bounded 30sec calls; sampling may miss exit peak.',
        comparison='historical B1; no concurrent regenerated baseline, no holdout or adoption qualification',
        classification='T1 valid label changes => schema-order effect supported, else no material change. T2 valid label changes => field-interference possible, else no material change. Protocol failure, stop or mismatch => undetermined. Reason/citation changes reported separately. No accuracy rescue inferred from distribution or field omission.',
        reason_rubric='Review ALL B1/T1/T2 rows against unchanged source; supported / partially_supported / unsupported / unavailable. Exact quote existence separate from relation truth; no string-overlap scoring. Agent audit, not independent human gold.',
        production_changed=False))


def run():
    start = time.monotonic()
    p = load('preregistered.json')
    assert identity() == p['identity']
    head = baseline.command(['git', 'rev-parse', 'HEAD']).stdout.decode().strip()
    remote = baseline.command(['git', 'ls-remote', 'origin', 'refs/heads/' + BRANCH]).stdout.decode().split()[0]
    assert remote == head and not baseline.command(['git', 'status', '--porcelain']).stdout
    registration = load('remote-registration.json')
    baseline.command(['git', 'merge-base', '--is-ancestor', registration['commit'], head])
    assert registration['manifest_byte_matches'] and registration['manifest_sha256'] == sha((OUT / 'preregistered.json').read_bytes())
    for name, digest in p['artifacts_sha256'].items():
        assert sha((OUT / name).read_bytes()) == digest
    cases = {c['name']: c for c in p['cases']}
    before = system()
    initial_swap = swap_bytes(before)
    results, stopped = [], None if initial_swap is not None else 'resource_unmeasurable'
    with (OUT / 'results.jsonl').open('x') as out:
        for spec in p['rows']:
            row = {k: spec[k] for k in ('case', 'stage', 'reverse', 'prompt_sha256', 'request_sha256')}
            row.update(calls=0, status='not_run', schema_valid=False, answer=None)
            remaining = p['whole_seconds'] - p['preparation_seconds'] - (time.monotonic() - start) - p['audit_reserve_seconds']
            if remaining <= 0: stopped = stopped or 'total_budget'
            if stopped is None:
                c = cases[spec['case']]
                assert spec['request'] == make_request(spec, c)
                assert sha(spec['request_string'].encode()) == spec['request_sha256']
                row['calls'] = 1
                res = observed_call(baseline.SANDBOX + [args().helper], spec['request_string'].encode(), min(30, remaining))
                row['resource'] = res
                try:
                    r = resource_fields(res)
                    row['outer_response'] = r
                    stopped = stop_reason(res, initial_swap)
                    row.update(input_tokens=r.get('benchmark_input_tokens'), output_tokens=r.get('benchmark_output_tokens'))
                    assert r.get('model') == p['model'], 'model_mismatch'
                    assert row['input_tokens'] == spec['native_input_tokens'], 'native_token_mismatch'
                    assert r.get('ok') is True and r.get('stop_reason') == 'completed', 'backend_stop'
                    assert row['output_tokens'] <= p['output_tokens'], 'output_budget'
                    assert res['exit'] == 0 and not res['timed_out'], 'helper_exit_or_timeout'
                    answer = loads(r['generated_json'])
                    row['answer'] = answer
                    baseline.validate(answer, spec['schema'])
                    audit = baseline.citation_audit(answer, c, spec['stage'])
                    row.update(schema_valid=True, status='completed', citation_audit=audit,
                        citations_exist=all(a['exists'] for a in audit),
                        relation_label_correct=answer['relation'] == c['expected_relation'],
                        direction_correct=all(answer[k] == v for k, v in c['expected_direction'].items()) if spec['stage'] == 'T1' else None)
                except (ValueError, KeyError, TypeError, AssertionError) as e:
                    row['status'] = str(e) or type(e).__name__
                    stopped = stopped or 'protocol_or_contract_failure'
                row['stop_after_call'] = stopped
            else:
                row['not_run_reason'] = stopped
            results.append(row)
            out.write(json.dumps(row, ensure_ascii=False) + '\n')
            out.flush()
            print(json.dumps({k: row.get(k) for k in ('case', 'stage', 'reverse', 'status', 'schema_valid', 'relation_label_correct', 'stop_after_call')}), flush=True)
    write('run-metadata.json', dict(registration_commit=head, generation_calls=sum(r['calls'] for r in results),
        run_seconds=time.monotonic() - start, retries=0, stop_reason=stopped,
        system_before=before, system_after=system(), post_identity_matches=identity() == p['identity']))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=('prepare', 'run'))
    action = parser.parse_args().action
    prepare() if action == 'prepare' else run()
