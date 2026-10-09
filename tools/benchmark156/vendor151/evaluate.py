"""Bounded deterministic line-edit representation; synthetic inputs only."""
import argparse
import base64
import difflib
import hashlib
import json
import pathlib
import subprocess
import tempfile
import time

MAX_BYTES = 1024 * 1024
MAX_LINES = 20000
MAX_UNITS = 256


def extract(before, after, file_id, symbols=()):
    if max(len(before), len(after)) > MAX_BYTES:
        raise ValueError('byte_budget')
    a, b = before.splitlines(keepends=True), after.splitlines(keepends=True)
    if max(len(a), len(b)) > MAX_LINES:
        raise ValueError('line_budget')
    old_offsets, new_offsets = [0], [0]
    for line in a:
        old_offsets.append(old_offsets[-1] + len(line))
    for line in b:
        new_offsets.append(new_offsets[-1] + len(line))
    # No semantic labels or path/name rules participate in extraction.
    out = []
    blocks = []
    for tag, i, j, k, l in difflib.SequenceMatcher(None, a, b, autojunk=False).get_opcodes():
        if tag == 'equal':
            continue
        # Preserve every selectable line edit; correspondence is positional,
        # never a claim of shared intent. Extra additions/deletions remain units.
        for offset in range(max(j-i, l-k)):
            oi, ni = min(i+offset, j), min(k+offset, l)
            blocks.append((tag, oi, min(oi+1, j), ni, min(ni+1, l)))
    for tag, i, j, k, l in blocks:
        if tag == 'equal':
            continue
        start, end = old_offsets[i], old_offsets[j]
        ns, ne = new_offsets[k], new_offsets[l]
        payload = b''.join(b[k:l])
        digest = hashlib.sha256(before + b'\0' + after).hexdigest()
        uid = hashlib.sha256(f'{file_id}:{digest}:{start}:{end}:{ns}:{ne}'.encode()).hexdigest()[:24]
        out.append(dict(id=uid, file=file_id, source_digest=digest, before_sha256=hashlib.sha256(before).hexdigest(),
                        old_span=[start, end], new_span=[ns, ne], kind=tag,
                        old_lines=[i, j], new_lines=[k, l],
                        before=base64.b64encode(before[start:end]).decode(),
                        after=base64.b64encode(payload).decode(),
                        symbols=[s['name'] for s in symbols if s['start'] < ne and ns < s['end']]))
    if len(out) > MAX_UNITS:
        raise ValueError('unit_budget')
    return out


def reconstruct(before, units, selected):
    known = {u['id'] for u in units}
    if len(known) != len(units) or len(set(selected)) != len(selected) or not set(selected) <= known:
        raise ValueError('invalid_assignment')
    parts, cursor = [], 0
    for u in units:
        if u['before_sha256'] != hashlib.sha256(before).hexdigest():
            raise ValueError('snapshot_mismatch')
        start, end = u['old_span']
        if start < cursor or not cursor <= start <= end <= len(before):
            raise ValueError('overlap_or_range')
        if before[start:end] != base64.b64decode(u['before']):
            raise ValueError('source_mismatch')
        parts.append(before[cursor:start])
        parts.append(base64.b64decode(u['after']) if u['id'] in selected else before[start:end])
        cursor = end
    parts.append(before[cursor:])
    return b''.join(parts)


def git(cwd, *args, data=None, allowed=(0,)):
    p = subprocess.run(['git', *args], cwd=cwd, input=data, capture_output=True)
    if p.returncode not in allowed:
        raise RuntimeError('git_' + args[0] + ': ' + p.stderr.decode(errors='replace'))
    return p.stdout


def stage_verify(before, after, units):
    # Mutations are confined to a disposable synthetic repository.
    with tempfile.TemporaryDirectory(prefix='benchmark151-') as tmp:
        root = pathlib.Path(tmp)
        git(root, 'init', '-q')
        (root / 'f').write_bytes(before)
        git(root, 'add', 'f')
        for order in (units, list(reversed(units))):
            git(root, 'hash-object', '-w', '--stdin', data=before)
            blob = hashlib.sha1(b'blob ' + str(len(before)).encode() + b'\0' + before).hexdigest()
            git(root, 'update-index', '--cacheinfo', '100644', blob, 'f')
            selected, current = [], before
            for u in order:
                selected.append(u['id'])
                target = reconstruct(before, units, selected)
                (root / 'f').write_bytes(target)
                patch = git(root, 'diff', '--binary', '--', 'f')
                git(root, 'apply', '--cached', '--binary', data=patch)
                if git(root, 'show', ':f') != target:
                    raise AssertionError('index_mismatch')
                current = target
            if current != after:
                raise AssertionError('final_mismatch')
    return True


def main():
    p = argparse.ArgumentParser()
    p.add_argument('--fixtures', required=True)
    p.add_argument('--symbols', required=True)
    p.add_argument('--output', required=True)
    args = p.parse_args()
    records = json.loads(pathlib.Path(args.fixtures).read_text())
    rows = []
    for record in records:
        start = time.perf_counter()
        unit_count = size = changed_bytes = 0
        extraction_wall = 0.0
        counts = []
        for f in record['Files']:
            before, after = f['Before'].encode(), f['After'].encode()
            symbols = json.loads(subprocess.run([args.symbols], input=after, capture_output=True, check=True).stdout)
            extraction_start = time.perf_counter()
            units = extract(before, after, f['ID'], symbols)
            extraction_wall += time.perf_counter() - extraction_start
            assert units == extract(before, after, f['ID'], symbols)
            assert reconstruct(before, units, [u['id'] for u in units]) == after
            assert reconstruct(before, units, []) == before
            assert stage_verify(before, after, units)
            counts.append(len(units))
            unit_count += len(units)
            size += len(json.dumps(units, sort_keys=True).encode())
            changed_bytes += sum(u['old_span'][1]-u['old_span'][0] + u['new_span'][1]-u['new_span'][0] for u in units)
        rows.append(dict(fixture=record['Name'], files=len(record['Files']), units=unit_count,
                         units_per_file=counts, complete=True, deterministic=True,
                         stage_forward_reverse=True, gold_representable=True,
                         file_gold_representable=True, changed_bytes=changed_bytes,
                         representation_bytes=size, extraction_seconds=extraction_wall, wall_seconds=time.perf_counter()-start,
                         calls=0, input_tokens=0, output_tokens=0,
                         exact=None, false_merge=None, false_split=None,
                         unresolved=None, semantic_prediction=False))
    synthetic = [
        ('one-file-two-intents', b'package x\nfunc A() int { return 1 }\n\nfunc B() int { return 2 }\n', b'package x\nfunc A() int { return 3 }\n\nfunc B() int { return 4 }\n', 2),
        ('adjacent-edit', b'a\nb\n', b'c\nd\n', 2),
        ('same-line', b'a b\n', b'c d\n', 1),
        ('insert', b'a\nb\n', b'a\nz\nb\n', 1),
        ('delete', b'a\nb\n', b'a\n', 1),
        ('empty-to-content', b'', b'a\n', 1),
        ('content-to-empty', b'a\n', b'', 1),
        ('no-final-newline', b'a', b'b', 1),
        ('non-utf8', b'\xff\n', b'\xfe\n', 1),
        ('crlf', b'a\r\nx\r\nb\r\n', b'c\r\nx\r\nd\r\n', 2),
    ]
    for name, before, after, expected in synthetic:
        start = time.perf_counter()
        units = extract(before, after, 'F001')
        assert len(units) == expected
        assert stage_verify(before, after, units)
        rows.append(dict(fixture=name, files=1, units=len(units), complete=True,
                         stage_forward_reverse=True, gold_representable=name != 'same-line',
                         file_gold_representable=name not in ('one-file-two-intents','adjacent-edit','same-line','crlf'),
                         wall_seconds=time.perf_counter()-start, calls=0,
                         semantic_prediction=False))
    pathlib.Path(args.output).write_text(json.dumps(rows, indent=2)+'\n')
    print(json.dumps(dict(cases=len(rows), regression=len(records), staging_pass=sum(r['stage_forward_reverse'] for r in rows),
                          files=sum(r['files'] for r in rows), units=sum(r['units'] for r in rows))))


if __name__ == '__main__':
    main()
