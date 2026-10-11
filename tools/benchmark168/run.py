"""Run preregistered local-only discovery calls once; preserve failure receipts."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import stat
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]
DATA = ROOT / 'docs/benchmarks/issue-168'

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def repo_digest(root):
    entries = {}
    for path in sorted(root.rglob('*')):
        if path.is_symlink():
            raise ValueError('unexpected fixture symlink')
        if path.is_file():
            entries[str(path.relative_to(root))] = f'{stat.S_IMODE(path.stat().st_mode):o}:{sha(path)}'
    encoded = json.dumps(entries, sort_keys=True, separators=(',', ':')).encode()
    return hashlib.sha256(encoded).hexdigest()

def resources():
    result = {'pressure_level': None, 'swap_used_mib': None}
    for key in ['kern.memorystatus_vm_pressure_level', 'vm.swapusage']:
        p = subprocess.run(['/usr/sbin/sysctl', '-n', key], capture_output=True)
        if p.returncode:
            continue
        if key.endswith('level'):
            result['pressure_level'] = int(p.stdout)
        else:
            m = re.search(rb'used = ([0-9.]+)M', p.stdout)
            if m:
                result['swap_used_mib'] = float(m[1])
    return result

def decode_unique(pairs):
    obj = {}
    for key, value in pairs:
        if key in obj:
            raise ValueError('duplicate JSON key')
        obj[key] = value
    return obj

def validate_discovery(text):
    obj = json.loads(text, object_pairs_hook=decode_unique)
    if not isinstance(obj, dict) or set(obj) != {'purposes'}:
        raise ValueError('invalid discovery object')
    purposes = obj['purposes']
    if not isinstance(purposes, list) or not 1 <= len(purposes) <= 8:
        raise ValueError('invalid purpose count')
    for i, purpose in enumerate(purposes, 1):
        if not isinstance(purpose, dict) or set(purpose) != {'purpose_id', 'description'}:
            raise ValueError('invalid purpose fields')
        if purpose['purpose_id'] != f'P{i:03d}':
            raise ValueError('nonunique/nonsequential purpose ID')
        desc = purpose['description']
        if not isinstance(desc, str) or not desc.strip() or not 1 <= len(desc) <= 240 or '\n' in desc or '\r' in desc:
            raise ValueError('invalid purpose description')
    return obj

def check_pins(pr, local):
    for path, value in pr['sha256'].items():
        p = Path(local[path.split(':',1)[1]]) if path.startswith('runtime:') else ROOT / path
        if sha(p) != value:
            raise ValueError('preregistered file drift: ' + path)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--cohort', choices=['known', 'holdout'], required=True)
    parser.add_argument('--runtime', type=Path, required=True)
    args = parser.parse_args()
    local = json.loads(args.runtime.read_text())
    pr = json.loads((DATA/'preregistration.json').read_text())
    check_pins(pr, local)
    if args.cohort == 'holdout' and not (DATA/'known-evaluation.json').is_file():
        raise ValueError('record known evaluation first; no prompt tuning permitted')
    manifest = json.loads((DATA/'frozen-inputs/manifest.json').read_text())
    results = DATA / (args.cohort+'-measurements.jsonl')
    # Never replay cells after timeout/failure or on process restart.
    seen = set()
    if results.exists():
        seen = {(r['fixture'], r['order']) for r in map(json.loads, results.read_text().splitlines())}
    for row in manifest:
        if row['cohort'] != args.cohort or 'admission_failure' in row or (row['fixture'],row['order']) in seen:
            continue
        check_pins(pr, local)
        request_path = DATA/'frozen-inputs'/f"{row['fixture']}-{row['order']}-discovery.json"
        root = Path(local['fixture_roots'][row['fixture']])
        if repo_digest(root) != row['repository_sha256']:
            raise ValueError('fixture changed before inference')
        before = resources()
        if before['pressure_level'] is None or before['swap_used_mib'] is None or before['pressure_level'] >= 4:
            raise ValueError('pre-call resource observer unavailable/critical')
        started = time.monotonic()
        proc = subprocess.Popen(['/usr/bin/time', '-l', local['helper']], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
        # communicate bounds each process group; no arbitrary subprocess survives a timeout.
        try:
            request = json.loads(request_path.read_text())
            request['model_path'] = local['model_path']
            out, err = proc.communicate(json.dumps(request).encode()+b'\n', timeout=pr['call_timeout_seconds'])
            stop = 'process_failure' if proc.returncode else None
        except subprocess.TimeoutExpired:
            os.killpg(proc.pid, signal.SIGKILL)
            out, err = proc.communicate()
            stop = 'timeout'
        elapsed = time.monotonic()-started
        after = resources()
        rec = {**row, 'request_sha256': sha(request_path), 'calls': 1, 'wall_seconds': elapsed, 'resource_before': before, 'resource_after': after,
               'git_unchanged': repo_digest(root)==row['repository_sha256'], 'response': None, 'schema_valid': False, 'status': stop,
               'helper_peak_rss_bytes': None, 'input_tokens': None, 'output_tokens': None, 'load_seconds': None, 'ttft_seconds': None, 'mlx_peak_bytes': None}
        rss = re.search(rb'(?m)^\s*(\d+)\s+maximum resident set size\s*$',err)
        if rss:
            rec['helper_peak_rss_bytes'] = int(rss[1])
        if stop is None and len(out) <= 4*1024*1024:
            try:
                response = json.loads(out,object_pairs_hook=decode_unique)
                rec['status'] = response['stop_reason']
                for name, source in [('input_tokens','input_tokens'),('output_tokens','output_tokens'),('load_seconds','load_seconds'),('ttft_seconds','ttft_seconds'),('mlx_peak_bytes','peak_bytes')]:
                    rec[name] = response.get('benchmark_'+source)
                if response.get('generation_profile') != 'bounded-grouping' or response.get('model') != pr['model']+'@'+pr['revision']:
                    rec['status'] = 'protocol_mismatch'
                elif response.get('ok') and rec['status'] == 'completed':
                    rec['response'] = validate_discovery(response['generated_json'])
                    rec['schema_valid'] = True
            except (ValueError, KeyError, TypeError):
                rec['status'] = 'invalid_schema_or_protocol'
        with results.open('a') as dest:
            dest.write(json.dumps(rec,ensure_ascii=False)+'\n')
            dest.flush()
            os.fsync(dest.fileno())
        print(json.dumps({k:rec[k] for k in ['fixture','order','status','schema_valid','wall_seconds','input_tokens','output_tokens','git_unchanged']}),flush=True)
        if not rec['git_unchanged']:
            raise ValueError('fixture changed; fail closed')
        if after['pressure_level'] is None or after['swap_used_mib'] is None or after['pressure_level'] >= 4 or after['swap_used_mib']-before['swap_used_mib']>1024:
            raise ValueError('hard resource stop after saved receipt')
    check_pins(pr, local)

if __name__ == '__main__':
    main()
