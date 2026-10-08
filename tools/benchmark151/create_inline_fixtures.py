"""New prospective requirements, independent of either extractor."""
import hashlib
import json
import pathlib
import create_fixtures as author


def main():
    author.ROOT = pathlib.Path(__file__).resolve().parent / 'inline_fixtures'
    cases = [
        ('unicode', 'config.py', '設定 = dict(名前="猫", 色="赤")\n',
         ('名前を猫から犬へ変更する', '"猫"', '"犬"'), ('色を赤から青へ変更する', '"赤"', '"青"')),
        ('nested', 'options.py', 'options = dict(network=dict(timeout=7), storage=dict(limit=128))\n',
         ('network timeoutを変更する', 'timeout=7', 'timeout=19'), ('storage limitを変更する', 'limit=128', 'limit=256')),
        ('inline-add-delete', 'flags.py', 'flags = dict(active=True, retired=False, limit=8)\n',
         ('retired設定を削除する', 'retired=False, ', ''), ('独立したaudit設定を追加する', 'limit=8', 'limit=8, audit=True')),
        ('crlf', 'config.ini', 'retries=6; window=15\r\n',
         ('retry上限を変更する', 'retries=6', 'retries=9'), ('windowを変更する', 'window=15', 'window=45')),
        ('no-newline', 'config.json', '{"width":640,"height":480}',
         ('widthを変更する', '640', '800'), ('heightを変更する', '480', '600')),
        ('contiguous-edits', 'labels.txt', 'ab\n',
         ('一文字目の独立labelを変更する', 'a', 'c'), ('二文字目の独立labelを変更する', 'b', 'd')),
    ]
    for name, path, source, first, second in cases:
        author.create(name, 'independent', {path: source}, {'R1': first[0], 'R2': second[0]},
                      {'R1': [(path, first[1], first[2])], 'R2': [(path, second[1], second[2])]})
    manifest = {str(p.relative_to(author.ROOT)): hashlib.sha256(p.read_bytes()).hexdigest()
                for p in sorted(author.ROOT.rglob('*')) if p.is_file() and p.name != 'manifest.json'}
    (author.ROOT/'manifest.json').write_text(json.dumps(manifest, indent=2)+'\n')
    print(hashlib.sha256((author.ROOT/'manifest.json').read_bytes()).hexdigest())


if __name__ == '__main__':
    main()
