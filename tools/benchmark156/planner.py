"""Research-only fallback. Never imports gold or writes the source repository."""
import base64
import hashlib
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, str(pathlib.Path(__file__).parent / 'vendor151'))
from inline import extract
from evaluate import reconstruct

MAX_FILES = 16
MAX_TOTAL_UNITS = 4096

def encode(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False, separators=(',', ':')).encode()

def prepare(files):
    if not 1 <= len(files) <= MAX_FILES:
        raise ValueError('file_budget')
    ids, paths, units = set(), set(), []
    for f in files:
        path = pathlib.PurePosixPath(f['path'])
        if (f['id'] in ids or f['path'] in paths or path.is_absolute() or
                any(p in ('..', '.git') for p in path.parts) or str(path) != f['path']):
            raise ValueError('invalid_file_mapping')
        ids.add(f['id']); paths.add(f['path'])
        if f.get('mode', '100644') not in ('100644', '100755'):
            raise ValueError('unsupported_mode')
        before, after = base64.b64decode(f['before_b64'], validate=True), base64.b64decode(f['after_b64'], validate=True)
        atoms = extract(before, after, f['id'])
        if not atoms:
            raise ValueError('no_changed_units')
        coverage(before, after, atoms)
        units.extend(atoms)
    if len(units) > MAX_TOTAL_UNITS:
        raise ValueError('total_unit_budget')
    return units

def coverage(before, after, atoms):
    old, new = [], []
    for a in atoms:
        for name, data, accum in [('old_span', before, old), ('new_span', after, new)]:
            start, end = a[name]
            if not 0 <= start <= end <= len(data):
                raise ValueError('span_range')
            accum.extend(range(start, end))
        if before[slice(*a['old_span'])] != base64.b64decode(a['before']):
            raise ValueError('before_source_mapping')
        if after[slice(*a['new_span'])] != base64.b64decode(a['after']):
            raise ValueError('after_source_mapping')
    if len(old) != len(set(old)) or len(new) != len(set(new)):
        raise ValueError('duplicate_byte_ownership')
    if reconstruct(before, atoms, [a['id'] for a in atoms]) != after:
        raise ValueError('byte_reconstruction')
    if reconstruct(before, atoms, []) != before:
        raise ValueError('empty_reconstruction')

def fallback(files, units):
    commits = []
    for f in sorted(files, key=lambda f: f['id']):
        commits.append(dict(file_ids=[f['id']], unit_ids=[u['id'] for u in units if u['file'] == f['id']],
            type='chore', scope='changes', breaking=False,
            summary='目的未確定の変更をファイル単位で保存',
            purpose_status='unknown', mixed_intent_risk='unresolved',
            compose_candidate=False, fallback=True))
    validate(units, commits)
    return commits

def validate(units, commits):
    expected = [u['id'] for u in units]
    assigned = [uid for c in commits for uid in c['unit_ids']]
    if len(expected) != len(set(expected)) or sorted(expected) != sorted(assigned) or len(assigned) != len(set(assigned)):
        raise ValueError('invalid_assignment')
    by_id = {u['id']:u for u in units}
    for c in commits:
        if not c['unit_ids'] or set(c['file_ids']) != {by_id[uid]['file'] for uid in c['unit_ids']}:
            raise ValueError('invalid_commit_mapping')
    return True

def run_git(root, env, *args, data=None):
    p = subprocess.run(['git', *args], cwd=root, env=env, input=data, capture_output=True)
    if p.returncode:
        raise ValueError('git_' + args[0] + ':' + p.stderr.decode(errors='replace'))
    return p.stdout

def replay(files, units, commits):
    validate(units, commits)
    started = time.perf_counter()
    with tempfile.TemporaryDirectory(prefix='benchmark156-') as tmp:
        root = pathlib.Path(tmp)
        env = dict(os.environ, GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull)
        run_git(root, env, 'init', '-q')
        env['GIT_INDEX_FILE'] = str(root / 'temporary-index')
        run_git(root, env, 'read-tree', '--empty')
        by_file = {f['id']:[u for u in units if u['file']==f['id']] for f in files}
        def install(f, data, exists):
            if exists:
                blob = run_git(root, env, 'hash-object', '-w', '--stdin', data=data).decode().strip()
                run_git(root, env, 'update-index', '--add', '--cacheinfo', f.get('mode', '100644'), blob, f['path'])
            else:
                run_git(root, env, 'update-index', '--force-remove', '--', f['path'])
        for f in files:
            install(f, base64.b64decode(f['before_b64']), f.get('before_exists', True))
        before_tree = run_git(root, env, 'write-tree').decode().strip()
        for f in files:
            install(f, base64.b64decode(f['after_b64']), f.get('after_exists', True))
        expected_tree = run_git(root, env, 'write-tree').decode().strip()
        trees, stages = [], 0
        for order in (commits, list(reversed(commits))):
            run_git(root, env, 'read-tree', before_tree)
            selected = set()
            for c in order:
                selected.update(c['unit_ids'])
                for f in files:
                    atoms = by_file[f['id']]
                    own = [u['id'] for u in atoms if u['id'] in selected]
                    target = reconstruct(base64.b64decode(f['before_b64']), atoms, own)
                    all_selected = len(own) == len(atoms)
                    exists = f.get('after_exists', True) if all_selected else f.get('before_exists', True) or bool(own)
                    # Build an independent target index; apply its delta to the replay index.
                    target_env = dict(env, GIT_INDEX_FILE=str(root / 'target-index'))
                    run_git(root, target_env, 'read-tree', run_git(root, env, 'write-tree').decode().strip())
                    if exists:
                        blob = run_git(root, target_env, 'hash-object', '-w', '--stdin', data=target).decode().strip()
                        run_git(root, target_env, 'update-index', '--add', '--cacheinfo', f.get('mode', '100644'), blob, f['path'])
                    else:
                        run_git(root, target_env, 'update-index', '--force-remove', '--', f['path'])
                    current = run_git(root, env, 'write-tree').decode().strip()
                    next_tree = run_git(root, target_env, 'write-tree').decode().strip()
                    patch = run_git(root, env, 'diff-tree', '-p', '--binary', current, next_tree, '--', f['path'])
                    if patch:
                        run_git(root, env, 'apply', '--cached', '--binary', data=patch)
                    if run_git(root, env, 'write-tree').decode().strip() != next_tree:
                        raise ValueError('intermediate_index_mismatch')
                    stages += 1
            final = run_git(root, env, 'write-tree').decode().strip()
            if final != expected_tree:
                raise ValueError('final_tree_mismatch')
            trees.append(final)
        return dict(valid_staging=True, final_tree_equality=True, forward_reverse=True,
            before_tree=before_tree, after_tree=expected_tree, file_stage_steps=stages,
            seconds=time.perf_counter()-started)
