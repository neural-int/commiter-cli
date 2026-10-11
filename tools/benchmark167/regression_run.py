"""Run the fixed four cells sequentially; no retries, raw outputs, or downloads."""
import argparse
import json
import os
import re
from pathlib import Path
import subprocess


def resources():
    result = {}
    for name in ('kern.memorystatus_vm_pressure_level', 'vm.swapusage'):
        proc = subprocess.run(['sysctl', '-n', name], capture_output=True, check=False)
        if proc.returncode:
            continue
        if name.endswith('level'):
            result['pressure_level'] = int(proc.stdout)
        else:
            match = re.search(rb'used = ([0-9.]+)M', proc.stdout)
            if match:
                result['swap_used_mib'] = float(match[1])
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=('static', 'real'), required=True)
    parser.add_argument('--output-dir', type=Path, required=True)
    parser.add_argument('--benchmark', required=True)
    parser.add_argument('--cache', required=True)
    parser.add_argument('--product-helper', required=True)
    parser.add_argument('--measured-helper', required=True)
    args = parser.parse_args()
    args.output_dir.mkdir(mode=0o700, parents=True, exist_ok=False)
    repo = Path(__file__).resolve().parents[2]
    proxy = str(Path(__file__).with_name('regression_proxy.py'))
    base = os.environ.copy()
    base.update(GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL='/dev/null', GIT_OPTIONAL_LOCKS='0',
                COMMITER_167_PROXY_MODE=args.mode, COMMITER_CANDIDATE_SMOKE_CACHE=args.cache,
                COMMITER_167_PRODUCT_HELPER=args.product_helper)
    for cell, fixture, route in [('A', 'smoke-143', 'product'), ('B', 'smoke-143', 'benchmark'),
                                 ('C', 'one-independent', 'product'), ('D', 'one-independent', 'benchmark')]:
        env = base.copy()
        trace = args.output_dir / (cell + '-calls.jsonl')
        env['COMMITER_167_PROXY_TRACE'] = str(trace)
        receipt = args.output_dir / (cell + '-receipt.json')
        if route == 'product':
            env.update(COMMITER_167_REGRESSION_FIXTURE=fixture, COMMITER_CANDIDATE_SMOKE_HELPER=proxy,
                       COMMITER_167_REGRESSION_OUT=str(receipt))
            command = ['go', 'test', './internal/cli', '-run', '^TestIssue167RegressionProductPath$', '-count=1']
        else:
            if args.mode == 'static':
                env['COMMITER_167_PROXY_INSTRUMENTED'] = '1'
            helper = proxy if args.mode == 'static' else args.measured_helper
            command = [args.benchmark, '-regression-control', '-diagnostic', '-fixture', fixture,
                       '-helper', helper, '-cache', args.cache]
        # Go planning retains its shared 120-second deadline. This timeout
        # additionally bounds host compilation/fixture setup and observation.
        resource_before = resources()
        result = subprocess.run(command, cwd=repo, env=env, capture_output=True, timeout=180)
        resource_after = resources()
        if result.returncode:
            raise RuntimeError('cell host execution failed: ' + cell)
        if route == 'benchmark':
            record = json.loads(result.stdout)
            receipt.write_text(json.dumps(record, indent=2) + '\n')
            calls = record['calls']
            trace.write_text(''.join(json.dumps(c) + '\n' for c in calls))
        else:
            record = json.loads(receipt.read_text())
            record.update(resource_before=resource_before, resource_after=resource_after)
            receipt.write_text(json.dumps(record, indent=2) + '\n')
            calls = [json.loads(line) for line in trace.read_text().splitlines()]
        if not record['git_unchanged'] or not 1 <= len(calls) <= 3:
            raise RuntimeError('cell safety/call bound failed: ' + cell)
        print(json.dumps({'cell': cell, 'status': record['status'], 'calls': len(calls),
                          'model_inference': args.mode == 'real'}), flush=True)


if __name__ == '__main__':
    main()
