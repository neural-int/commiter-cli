"""Gold feasibility oracle; labels never participate in production extraction."""
import argparse
import hashlib
import itertools
import json
import pathlib
import tempfile
import time
from evaluate import extract, reconstruct, git


def subsets(items):
    for n in range(len(items)+1):
        yield from itertools.combinations(items, n)


def snapshot(root, paths):
    return {p: (root/p).read_bytes() for p in paths}


def stage_states(before, states):
    with tempfile.TemporaryDirectory(prefix='benchmark151-oracle-') as tmp:
        root = pathlib.Path(tmp)
        for p, data in before.items():
            (root/p).parent.mkdir(parents=True, exist_ok=True)
            (root/p).write_bytes(data)
        git(root, 'init', '-q')
        git(root, 'add', '.')
        for state in states:
            for p, data in state.items():
                (root/p).write_bytes(data)
            patch = git(root, 'diff', '--binary')
            if patch:
                git(root, 'apply', '--cached', '--binary', data=patch)
            for p, expected in state.items():
                if git(root, 'show', ':'+p) != expected:
                    raise AssertionError('index_mismatch')


def coverage(before, after, units):
    # Verify raw source locations, unchanged gaps and payloads independently.
    import base64
    old_cursor = new_cursor = 0
    for u in units:
        os, oe = u['old_span']
        ns, ne = u['new_span']
        assert old_cursor <= os <= oe <= len(before)
        assert new_cursor <= ns <= ne <= len(after)
        assert before[old_cursor:os] == after[new_cursor:ns]
        assert base64.b64decode(u['before']) == before[os:oe]
        assert base64.b64decode(u['after']) == after[ns:ne]
        old_cursor, new_cursor = oe, ne
    assert before[old_cursor:] == after[new_cursor:]


def matching_subsets(before, units, target):
    """Prefix DP, at most two witnesses; gold stays exclusively in evaluator."""
    import base64
    from functools import lru_cache
    visited = 0

    @lru_cache(None)
    def visit(index, position):
        nonlocal visited
        visited += 1
        if visited > 4096:
            raise ValueError('oracle_state_budget')
        cursor = units[index-1]['old_span'][1] if index else 0
        if index == len(units):
            return [()] if target[position:] == before[cursor:] else []
        unit = units[index]
        start, end = unit['old_span']
        gap = before[cursor:start]
        if not target.startswith(gap, position):
            return []
        position += len(gap)
        matches = []
        for chosen, payload in ((False, before[start:end]), (True, base64.b64decode(unit['after']))):
            if target.startswith(payload, position):
                for rest in visit(index+1, position+len(payload)):
                    matches.append(((unit['id'],) if chosen else ()) + rest)
                    if len(matches) == 2:
                        return matches
        return matches

    return visit(0, 0)


def measure(root, extractor=extract, oracle='exhaustive'):
    gold = json.loads((root/'gold.json').read_text())
    intents = list(gold['requirements'])
    paths = sorted(str(p.relative_to(root/'before')) for p in (root/'before').rglob('*') if p.is_file())
    before, after = snapshot(root/'before', paths), snapshot(root/'after', paths)
    expected = {tuple(s['selected_intents']): snapshot(root/'expected'/s['directory'], paths) for s in gold['expected_states']}
    assert len(intents) <= 3 and len(expected) == 2**len(intents)
    assert expected[()] == before and expected[tuple(intents)] == after
    start = time.perf_counter()
    units = {p: extractor(before[p], after[p], p) for p in paths}
    latency = time.perf_counter()-start
    deterministic = units == {p: extractor(before[p], after[p], p) for p in paths}
    assert deterministic
    for p in paths:
        coverage(before[p], after[p], units[p])
        assert reconstruct(before[p], units[p], []) == before[p]
        assert reconstruct(before[p], units[p], [u['id'] for u in units[p]]) == after[p]
        assert len(units[p]) <= (256 if oracle == 'bounded-dp' else 16)
    assignments = {i: {} for i in intents}
    missing = []
    for i in intents:
        for p in paths:
            if oracle == 'bounded-dp':
                matches = matching_subsets(before[p], units[p], expected[(i,)][p])
                assert all(reconstruct(before[p], units[p], choice) == expected[(i,)][p] for choice in matches)
            else:
                matches = [choice for choice in subsets([u['id'] for u in units[p]])
                           if reconstruct(before[p], units[p], choice) == expected[(i,)][p]]
            if len(matches) != 1:
                missing.append({'intent': i, 'path': p, 'matching_subsets': len(matches)})
            else:
                assignments[i][p] = list(matches[0])
    representable = not missing
    if representable:
        for p in paths:
            ids = [uid for i in intents for uid in assignments[i][p]]
            assert len(ids) == len(set(ids))
            assert set(ids) == {u['id'] for u in units[p]}
        for selected, state in expected.items():
            actual = {p: reconstruct(before[p], units[p], [u for i in selected for u in assignments[i][p]]) for p in paths}
            assert actual == state
        stage_states(before, list(expected.values()))
        for order in itertools.permutations(intents):
            stages = []
            for n in range(1, len(order)+1):
                selected = tuple(i for i in intents if i in order[:n])
                stages.append(expected[selected])
            stage_states(before, stages)
    # An independent expected-snapshot patch confirms representational failures
    # concern the extractor, not inability of Git to hold the desired blobs.
    stage_states(before, list(expected.values()))
    file_representable = True
    for p in paths:
        if sum(expected[(i,)][p] != before[p] for i in intents) > 1:
            file_representable = False
    return dict(fixture=f"{gold['split']}/{gold['name']}", files=len(paths),
                units=sum(map(len, units.values())), units_per_file={p: len(v) for p,v in units.items()},
                file_gold_representable=file_representable, gold_representable=representable,
                missing_boundaries=missing, assignments=assignments if representable else None,
                complete=True, source_coverage=True, deterministic=deterministic,
                oracle_snapshots_stageable=True, unit_subset_staging=representable,
                unit_order_staging=representable, intent_subsets=len(expected),
                representation_bytes=len(json.dumps(units, sort_keys=True).encode()),
                extraction_seconds=latency, calls=0, input_tokens=0, output_tokens=0,
                semantic_prediction=False, exact=None, false_merge=None, false_split=None, unresolved=None)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--output', required=True)
    ap.add_argument('--fixtures-root', type=pathlib.Path)
    ap.add_argument('--representation', choices=['line','inline'], default='line')
    ap.add_argument('--oracle', choices=['exhaustive','bounded-dp'], default='exhaustive')
    args = ap.parse_args()
    root = args.fixtures_root or pathlib.Path(__file__).resolve().parent/'fixtures'
    extractor = extract
    if args.representation == 'inline':
        from inline import extract as extractor
    manifest = json.loads((root/'manifest.json').read_text())
    actual_paths = {str(p.relative_to(root)) for p in root.rglob('*') if p.is_file() and p.name != 'manifest.json'}
    assert actual_paths == set(manifest)
    for path, digest in manifest.items():
        assert hashlib.sha256((root/path).read_bytes()).hexdigest() == digest
    rows = [measure(p.parent, extractor, args.oracle) for p in sorted(root.glob('*/*/gold.json'))]
    result = {'representation': args.representation, 'oracle': args.oracle, 'manifest_sha256': hashlib.sha256((root/'manifest.json').read_bytes()).hexdigest(),
              'rows': rows, 'cases': len(rows),
              'representable': sum(r['gold_representable'] for r in rows),
              'file_representable': sum(r['file_gold_representable'] for r in rows),
              'bounded_representation_gate': 'GO' if all(r['gold_representable'] for r in rows) else 'NO-GO',
              'parent_known149_gate': 'NOT_PROVEN'}
    pathlib.Path(args.output).write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k != 'rows'}))


if __name__ == '__main__':
    main()
