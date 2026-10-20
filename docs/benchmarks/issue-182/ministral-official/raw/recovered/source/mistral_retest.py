"""H182-003: four Ministral cells, official serialization, no retry/repair.

Reuses the frozen H182-002 Go binary/fixtures. Swift uses mistral-common 1.8.6
for input, audited HF byte decoding for output, and the existing JSON grammar.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import time

from mistral_audit import audit
from monitor_v2 import Monitor, sample
from suite_v2 import ROOT, CONFIG, sha, save

MODELS = {
    'instruct': 'd3a59d531fc3a5f486072eaf1bc199943abe0df9f290b18ca3d29ff71e899230',
    'reasoning': '5254a062953a4616c8b2b5ee1d02d5a4e940da55646111d86ee5e1109ad1edb7',
}


def run(args):
    args.out.mkdir(parents=True, exist_ok=True)
    assert not (args.out / 'execution-contract.json').exists(), 'never rerun a frozen suite'
    previous = json.loads((args.previous / 'preparation.json').read_text())
    if args.repos is not None:
        previous['repos'] = str(args.repos)
    files = [Path(__file__), Path(__file__).with_name('mistral_tokenize.py'),
             Path(__file__).with_name('mistral_audit.py'), Path(__file__).with_name('monitor_v2.py'),
             ROOT / 'mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift',
             args.runtime / 'commiter-mlx-helper', args.runtime / 'mlx.metallib',
             args.previous / 'evaluation.test', args.previous / 'evaluation.go',
             args.previous / 'three_phase.go', args.runtime / 'assets/manifest.json']
    cells = []
    for kind in args.kinds:
        model_hash = MODELS[kind]
        assets = args.runtime / 'assets' / kind
        model = Path('$HOME/Library/Caches/commiter/mlx-models') / model_hash
        config = CONFIG / f'ministral-{kind}.json'
        files += [config, model / 'model.safetensors', model / 'tokenizer.json', model / 'tokenizer_config.json',
                  model / 'config.json', assets / 'tekken.json', assets / 'SYSTEM_PROMPT.txt']
        for case in ['C01', 'C02']:
            row = audit(assets, model, args.previous / ('capture-' + case), kind == 'reasoning')
            save(args.out / f'{kind}-{case}-token-audit.json', row)
            cells.append({'kind': kind, 'case': case, 'model_path': str(model),
                          'config': str(config), 'prefix_tokens': row['prefix_tokens']})
    packages = subprocess.check_output(['uv', 'pip', 'freeze', '--python', str(args.runtime / 'venv/bin/python')], text=True)
    (args.out / 'requirements.lock.txt').write_text(packages)
    contract = {'harness_id': 'H182-003-ministral-official', 'baseline': 'H182-002',
                'cells': cells, 'maximum_calls': 12, 'repeats': 1, 'retry': 0, 'repair': 0,
                'output_budget': None, 'thinking_budget': None, 'cycle_timeout': None,
                'changes': ['official mistral-common serialization', 'official Reasoning system prompt with final JSON override',
                            'natural THINK opening rather than forced opening', 'numeric progress logging'],
                'unchanged': ['weight revisions', 'sampling', 'seed144', 'KV8/group64', 'C01/C02',
                              'original task prompts', 'schema/validator', 'fixed partition injection downstream'],
                'safety_stop': 'monitor failure or RSS >=12GiB; helper no progress for 300s; process group killed, recorded as interruption',
                'hashes': {str(path): sha(path) for path in files},
                'official_assets': json.loads((args.runtime / 'assets/manifest.json').read_text())}
    preflight = [sample() for _ in range(2)]
    save(args.out / 'monitor-preflight.json', preflight)
    assert all('gpu' in row and all(row[key].get('exit') == 0 for key in ['cpu', 'vm', 'pressure', 'swap', 'thermal', 'power']) for row in preflight)
    save(args.out / 'execution-contract.json', contract)
    monitor = Monitor(args.out / 'monitor.jsonl')
    monitor.start()
    started = time.monotonic()
    results = []
    try:
        for cell in cells:
            assert not monitor.failed, 'monitor gate failed'
            assert all(sha(path) == expected for path, expected in contract['hashes'].items()), 'frozen runtime drift'
            label = f"ministral-{cell['kind']}-{cell['case']}"
            target = args.out / label
            assert not target.exists()
            config = json.loads(Path(cell['config']).read_text())
            env = {key: value for key, value in os.environ.items() if not key.startswith(('COMMITER_180_', 'COMMITER_182_'))}
            env.update(COMMITER_182_CASE_CONFIG=cell['config'], COMMITER_182_MODEL=config['model'],
                       COMMITER_182_MODEL_PATH=cell['model_path'], COMMITER_180_MODE='run',
                       COMMITER_180_CONDITION='D', COMMITER_180_THINKING='T', COMMITER_180_CASE=cell['case'],
                       COMMITER_180_OUT=str(target), COMMITER_180_REPOS=previous['repos'],
                       COMMITER_180_FIXTURE_DIR=previous['fixtures'], COMMITER_180_HELPER=str(args.runtime / 'commiter-mlx-helper'),
                       COMMITER_180_KV_BITS='8', COMMITER_180_PREFIX_TOKENS=str(cell['prefix_tokens']),
                       COMMITER_182_MISTRAL_PYTHON=str(args.runtime / 'venv/bin/python'),
                       COMMITER_182_MISTRAL_BRIDGE=str(Path(__file__).with_name('mistral_tokenize.py').resolve()),
                       COMMITER_182_MISTRAL_ASSETS=str(args.runtime / 'assets' / cell['kind']))
            print('START', label, flush=True)
            with (args.out / (label + '.log')).open('x') as log:
                process = subprocess.Popen([str(args.previous / 'evaluation.test'), '-test.run=^TestIssue180Evaluation$', '-test.timeout=0'],
                                           cwd=ROOT / 'internal/cli', env=env, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
                interrupted = None
                last_progress = time.monotonic()
                last_size = 0
                while process.poll() is None:
                    progress_size = sum(path.stat().st_size for path in target.glob('*helper-stderr.log')) if target.exists() else 0
                    if progress_size != last_size:
                        last_progress = time.monotonic()
                        last_size = progress_size
                    if monitor.failed or time.monotonic() - last_progress > 300:
                        import signal
                        interrupted = 'monitor_failure' if monitor.failed else 'no_progress_300s'
                        # The Go client starts the helper in a separate group.
                        # Stop that exact child group before stopping the runner.
                        inventory = subprocess.check_output(['ps', '-axo', 'pid=,ppid=,comm='], text=True)
                        for line in inventory.splitlines():
                            values = line.split(None, 2)
                            if len(values) == 3 and int(values[1]) == process.pid and values[2] == str(args.runtime / 'commiter-mlx-helper'):
                                os.killpg(int(values[0]), signal.SIGTERM)
                        if process.poll() is None:
                            os.killpg(process.pid, signal.SIGTERM)
                        process.wait(timeout=10)
                        break
                    time.sleep(1)
                monitor.active = None
            result_path = target / 'result.json'
            result = json.loads(result_path.read_text()) if result_path.exists() else {'error': 'runner incomplete'}
            results.append({'cell': label, 'exit': process.returncode, 'interrupted': interrupted, 'result': result})
            save(args.out / 'results.json', results)
            print('END', label, 'error', result.get('error'), 'seconds', result.get('wall_seconds'), flush=True)
            if interrupted:
                break
    finally:
        monitor.stop()
        save(args.out / 'completion.json', {'cells': len(results), 'elapsed_seconds': time.monotonic() - started,
                                          'monitor_failed': monitor.failed,
                                          'drift': [path for path, expected in contract['hashes'].items() if sha(path) != expected]})


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    for name in ['runtime', 'previous', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--repos', type=Path)
    parser.add_argument('--kinds', nargs='+', choices=list(MODELS), default=list(MODELS))
    run(parser.parse_args())
