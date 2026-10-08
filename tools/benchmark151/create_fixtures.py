"""Author requirements and expected snapshots without importing the extractor."""
import hashlib
import itertools
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent / 'fixtures'


def create(name, split, files, requirements, edits):
    root = ROOT / split / name
    intents = list(requirements)
    gold = {'name': name, 'split': split, 'requirements': requirements,
            'edits': edits, 'expected_states': []}
    for area in ('before', 'after'):
        for path, source in files.items():
            target = root / area / path
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(source.encode())
    for count in range(len(intents) + 1):
        for chosen in itertools.combinations(intents, count):
            state = '+'.join(chosen) or 'none'
            output = dict(files)
            for intent in chosen:
                for path, old, new in edits[intent]:
                    if output[path].count(old) != 1:
                        raise ValueError((name, intent, path, 'ambiguous authored edit'))
                    output[path] = output[path].replace(old, new, 1)
            for path, source in output.items():
                target = root / 'expected' / state / path
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(source.encode())
                if len(chosen) == len(intents):
                    (root / 'after' / path).write_bytes(source.encode())
            gold['expected_states'].append({'selected_intents': list(chosen), 'directory': state})
    (root / 'gold.json').write_text(json.dumps(gold, ensure_ascii=False, indent=2)+'\n')


def main():
    for split in ('development', 'independent'):
        independent = split == 'independent'
        a, b = ('timeout', 'upload_limit') if independent else ('retry_limit', 'page_size')
        old_a, new_a, old_b, new_b = ('10', '30', '1024', '4096') if independent else ('3', '5', '20', '50')
        req = {'R1': f'{a}を{old_a}から{new_a}へ変更する', 'R2': f'{b}を{old_b}から{new_b}へ変更する'}
        edits = {'R1': [('settings.py', f'return {old_a}\n', f'return {new_a}\n')],
                 'R2': [('settings.py', f'return {old_b}\n', f'return {new_b}\n')]}
        for name, gap in (('separated', '\n'), ('adjacent', '')):
            source = f'def {a}():\n    return {old_a}\n{gap}def {b}():\n    return {old_b}\n'
            if name == 'adjacent':
                source = f'{a} = {old_a}\n{b} = {old_b}\n'
                case_edits = {'R1': [('settings.py', f'{a} = {old_a}', f'{a} = {new_a}')],
                              'R2': [('settings.py', f'{b} = {old_b}', f'{b} = {new_b}')]}
            else:
                case_edits = edits
            create(name, split, {'settings.py': source}, req, case_edits)
        files = {'src/settings.py': f'def {a}():\n    return {old_a}\n\ndef {b}():\n    return {old_b}\n',
                 'tests/test_first.py': f'from src.settings import {a}\nassert {a}() == {old_a}\n',
                 'tests/test_second.py': f'from src.settings import {b}\nassert {b}() == {old_b}\n'}
        create('source-test', split, files, req,
               {'R1': [('src/settings.py', f'return {old_a}\n', f'return {new_a}\n'), ('tests/test_first.py', f'== {old_a}', f'== {new_a}')],
                'R2': [('src/settings.py', f'return {old_b}\n', f'return {new_b}\n'), ('tests/test_second.py', f'== {old_b}', f'== {new_b}')]})
        path = 'defaults.py'
        create('same-line', split, {path: f'defaults = dict({a}={old_a}, {b}={old_b})\n'}, req,
               {'R1': [(path, f'{a}={old_a}', f'{a}={new_a}')], 'R2': [(path, f'{b}={old_b}', f'{b}={new_b}')]})
        version, next_version = ('7', '8') if independent else ('1', '2')
        create('coupled-protocol', split,
               {'protocol.py': f'WIRE_VERSION = {version}\nENCODER_VERSION = {version}\n',
                'tests/protocol.txt': f'wire-version={version}\nencoder-version={version}\n'},
               {'R1': 'wireとencoderのprotocol version、および対応するtest期待値を同時に更新する'},
               {'R1': [('protocol.py', f'WIRE_VERSION = {version}', f'WIRE_VERSION = {next_version}'),
                       ('protocol.py', f'ENCODER_VERSION = {version}', f'ENCODER_VERSION = {next_version}'),
                       ('tests/protocol.txt', f'wire-version={version}', f'wire-version={next_version}'),
                       ('tests/protocol.txt', f'encoder-version={version}', f'encoder-version={next_version}')]})
        key = 'retired_upload' if independent else 'retired_page'
        addition = 'enable_upload=true\n' if independent else 'enable_pagination=true\n'
        create('insert-delete', split, {'settings.conf': f'header=true\n{key}=true\nseparator=true\nfooter=true\n'},
               {'R1': f'旧設定{key}を削除する', 'R2': '独立した新機能の設定を追加する'},
               {'R1': [('settings.conf', f'{key}=true\n', '')],
                'R2': [('settings.conf', 'footer=true\n', addition+'footer=true\n')]})
    manifest = {}
    for path in sorted(ROOT.rglob('*')):
        if path.is_file() and path.name != 'manifest.json':
            manifest[str(path.relative_to(ROOT))] = hashlib.sha256(path.read_bytes()).hexdigest()
    (ROOT/'manifest.json').write_text(json.dumps(manifest, indent=2)+'\n')
    print(json.dumps({'fixtures': len(list(ROOT.glob('*/*/gold.json'))), 'sha256_manifest': hashlib.sha256((ROOT/'manifest.json').read_bytes()).hexdigest()}))


if __name__ == '__main__':
    main()
