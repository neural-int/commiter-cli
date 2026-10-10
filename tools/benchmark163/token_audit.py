"""Audit native prompt lengths before generation; no grouping/model inference."""
import argparse
import json
import pathlib
import subprocess
import time
from harness import ROOT, encode, sha, loads, request

OUT = ROOT / 'docs/benchmarks/issue-163'


def audit(helper, gemma, coder):
    fixed = loads((OUT / 'iteration-2-preregistered.json').read_bytes())
    digest = sha(pathlib.Path(helper).read_bytes())
    paths = {'gemma': gemma, 'coder': coder}; cases = {c['name']: c for c in fixed['cases']}
    results = []; start = time.monotonic()
    for spec in fixed['rows']:
        case = cases[spec['case']]
        for model in ('gemma', 'coder'):
            req = request(model, paths[model], helper, digest, 'local', case['files'], case['support'], case['facts'], spec['reverse'])
            assert req['messages'] == spec['messages']
            req['preflight_only'] = True
            p = subprocess.run([helper], input=encode(req) + b'\n', capture_output=True, timeout=30,
                               env=dict(__import__('os').environ, HF_HUB_OFFLINE='1', TRANSFORMERS_OFFLINE='1'))
            row = {'model': model, 'case': spec['case'], 'reverse': spec['reverse'], 'exit': p.returncode,
                   'prompt_sha256': spec['prompt_sha256'], 'stdout_sha256': sha(p.stdout), 'stderr_sha256': sha(p.stderr),
                   'raw_response': p.stdout.decode(errors='replace'), 'input_tokens': None, 'generation_calls': 0}
            try:
                response = loads(p.stdout); row['input_tokens'] = response.get('benchmark_input_tokens')
                row['stop_reason'] = response.get('stop_reason')
                row['pass'] = bool(p.returncode == 0 and response.get('ok') is True and response.get('stop_reason') == 'completed'
                                   and response.get('generated_json') == '{}' and type(row['input_tokens']) is int
                                   and row['input_tokens'] + 1536 <= 16384 and response.get('benchmark_output_tokens') is None)
            except ValueError:
                row['pass'] = False
            results.append(row); print(json.dumps({k: row[k] for k in ('model', 'case', 'reverse', 'input_tokens', 'pass')}), flush=True)
    record = {'issue': 163, 'helper_sha256': digest, 'generation_calls': 0, 'tokenizer_only_helper_calls': len(results),
              'elapsed_seconds': time.monotonic() - start, 'all_pass': all(row['pass'] for row in results), 'rows': results,
              'meaning': 'load/tokenize and full input+output reservation only; no prefill/decode/grammar or semantic qualification'}
    with (OUT / 'iteration-2-token-audit.json').open('x') as out: json.dump(record, out, ensure_ascii=False, indent=2); out.write('\n')
    print(json.dumps({'all_pass': record['all_pass'], 'generation_calls': 0, 'helper_calls': len(results)}), flush=True)
    return record['all_pass']


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    for name in ('helper', 'gemma', 'coder'): parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    raise SystemExit(0 if audit(args.helper, args.gemma, args.coder) else 1)
