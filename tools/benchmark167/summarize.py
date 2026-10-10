"""Summarize recorded attempts without discarding failures or fabricating metrics."""
import argparse
from collections import defaultdict, Counter
import hashlib
import json
import math
from pathlib import Path


def distribution(values):
    if not values:
        return None
    ordered = sorted(values)
    return {
        'n': len(ordered),
        'p50': ordered[math.ceil(len(ordered) * .5) - 1],
        'p95': ordered[math.ceil(len(ordered) * .95) - 1],
        'values': values,
        'method': 'nearest-rank; n=3 p95 is maximum, not a production tail estimate',
    }


def summarize(rows, fixtures):
    groups = defaultdict(list)
    for row in rows:
        groups[(row['workload'], row['route'])].append(row)
    expected = {(f['name'], route): 3 for f in fixtures
                for route in (['three-phase', 'file-first'] if len(f['files']) <= 4 else ['file-first'])}
    complete_matrix = set(groups) == set(expected) and all(
        len(groups[key]) == count and sorted(r['repeat'] for r in groups[key]) == [0, 1, 2]
        for key, count in expected.items())
    summaries = []
    for (name, route), samples in groups.items():
        successful = [r for r in samples if r['status'] == 'completed']
        stage_times = defaultdict(list)
        for r in samples:
            for profile in set(c['profile'] for c in r['calls']):
                stage_times[profile].append(sum(c['wall_seconds'] for c in r['calls'] if c['profile'] == profile))
        token_totals = {}
        for key in ('input_tokens', 'output_tokens'):
            token_totals[key] = distribution([sum(c[key] for c in r['calls']) for r in samples
                                             if all(c[key] is not None for c in r['calls'])])
        calls = [c for r in samples for c in r['calls']]
        summaries.append({
            'workload': name, 'route': route, 'files': samples[0]['files'],
            'attempts': len(samples), 'successes': len(successful),
            'statuses': dict(Counter(r['status'] for r in samples)),
            'attempt_wall_seconds': distribution([r['wall_seconds'] for r in samples]),
            'successful_wall_seconds': distribution([r['wall_seconds'] for r in successful]),
            'stage_wall_seconds': {p: distribution(v) for p, v in stage_times.items()},
            'replay_seconds': distribution([r['replay_seconds'] for r in samples]),
            'call_counts': [len(r['calls']) for r in samples], 'tokens': token_totals,
            'runtime_ttft_seconds': distribution([c['runtime_ttft_seconds'] for c in calls if c['runtime_ttft_seconds'] is not None]),
            'load_seconds': distribution([c['load_seconds'] for c in calls if c['load_seconds'] is not None]),
            'helper_peak_rss_bytes': max((c['helper_peak_rss_bytes'] for c in calls if c['helper_peak_rss_bytes'] is not None), default=None),
            'mlx_peak_bytes': max((c['mlx_peak_bytes'] for c in calls if c['mlx_peak_bytes'] is not None), default=None),
            'swap_increase_mib': [r['resource_after']['swap_used_mib'] - r['resource_before']['swap_used_mib']
                                  if r['resource_after']['swap_used_mib'] is not None and r['resource_before']['swap_used_mib'] is not None else None
                                  for r in samples],
            'pressure_levels': [[r['resource_before']['pressure_level'], r['resource_after']['pressure_level']] for r in samples],
            'diff_bytes': samples[0]['diff_bytes'], 'diff_lines': samples[0]['diff_lines'],
            'validated_plan_groups': [r['groups'] if r['status'] == 'completed' else None for r in samples],
            'semantic_quality_of_validated_plans': [r['quality'] for r in samples],
            'semantic_unavailable_reason': 'no complete validated plan' if not successful else None,
            'git_unchanged': all(r['git_unchanged'] for r in samples),
        })
    paired = []
    for f in fixtures:
        if len(f['files']) > 4:
            continue
        a = sorted(groups.get((f['name'], 'three-phase'), []), key=lambda r: r['repeat'])
        b = sorted(groups.get((f['name'], 'file-first'), []), key=lambda r: r['repeat'])
        paired.append({'workload': f['name'], 'files': len(f['files']),
                       'same_snapshot_and_prepared': len(a) == len(b) == 3 and all(
                           x['snapshot_sha256'] == y['snapshot_sha256'] and x['prepared_sha256'] == y['prepared_sha256']
                           for x, y in zip(a, b)),
                       'file_first_minus_default_attempt_seconds': distribution([y['wall_seconds'] - x['wall_seconds'] for x, y in zip(a, b)]),
                       'successes': {'three-phase': sum(r['status'] == 'completed' for r in a), 'file-first': sum(r['status'] == 'completed' for r in b)},
                       'interpretation': 'attempt/failure latency; not usable-plan latency when either route fails'})
    def contrast(left, right):
        a = sorted(groups.get(left, []), key=lambda r: r['repeat'])
        b = sorted(groups.get(right, []), key=lambda r: r['repeat'])
        return {'right_minus_left_attempt_seconds': distribution([y['wall_seconds'] - x['wall_seconds'] for x, y in zip(a, b)]),
                'complete_success_pairs': sum(x['status'] == y['status'] == 'completed' for x, y in zip(a, b))}
    swap_violations = [r for r in rows if r['resource_after']['swap_used_mib'] is not None and r['resource_before']['swap_used_mib'] is not None
                       and r['resource_after']['swap_used_mib'] - r['resource_before']['swap_used_mib'] > 256]
    successes = sum(r['status'] == 'completed' for r in rows)

    def bounded(row):
        calls = row['calls']
        if not calls or len(calls) > (8 if row['route'] == 'file-first' else 3) or row['wall_seconds'] > 120:
            return False
        before, after = row['resource_before'], row['resource_after']
        if before['pressure_level'] != 1 or after['pressure_level'] != 1 or before['swap_used_mib'] is None or after['swap_used_mib'] is None:
            return False
        if after['swap_used_mib'] - before['swap_used_mib'] > 256:
            return False
        for call in calls:
            if call['mlx_peak_bytes'] is None or call['helper_peak_rss_bytes'] is None or max(call['mlx_peak_bytes'], call['helper_peak_rss_bytes']) > 8 * 1024 ** 3:
                return False
            reserved = 512 if call['profile'] == 'bounded-category' else 768
            if call['input_tokens'] is None or call['output_tokens'] is None or call['input_tokens'] + reserved > 16384 or call['output_tokens'] > reserved:
                return False
        return True

    bounds_count = sum(bounded(r) for r in rows)
    return {
        'issue': 167, 'phase': 2, 'matrix_complete': complete_matrix,
        'attempts': len(rows), 'successes': successes, 'workloads': summaries, 'paired': paired,
        'four_to_five': {
            'same_base_before_after_bytes': fixtures[3]['files'] == fixtures[4]['files'][:4],
            'same_file_first_route_increment': contrast(('boundary-base', 'file-first'), ('boundary-plus-consumer', 'file-first')),
            'route_and_input_change': contrast(('boundary-base', 'three-phase'), ('boundary-plus-consumer', 'file-first')),
            'interpretation': 'includes added dependent test and packet composition; route contrast also changes planner; not file-count-only causality',
        },
        'git_unchanged_count': sum(r['git_unchanged'] for r in rows),
        'runtime_bounds_satisfied_count': bounds_count,
        'metadata_gate_pass': complete_matrix and successes == len(rows) and bounds_count == len(rows) and all(r['git_unchanged'] for r in rows),
        'swap_threshold_violation_count': len(swap_violations),
        'phase_3': 'gated_out' if successes != len(rows) or swap_violations else 'requires_full_gate_audit',
        'production_go': False,
        'limits': ['authored synthetic fixtures; no independent Phase 3 holdout',
                   'failed attempts retained; no semantic scores invented for absent final plans',
                   'model reload on each call; true storage-cold and resident-model warm not controlled',
                   'shared host; pressure/swap before-after changes are not solely attributable to the measured process',
                   'initial matrix lacks detailed failure codes; separate bounded numeric diagnostics required',
                   'n=3 does not support a stable production p95 claim'],
    }


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('measurements', type=Path)
    parser.add_argument('fixtures', type=Path)
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    raw = args.measurements.read_bytes()
    rows = [json.loads(line) for line in raw.splitlines()]
    result = summarize(rows, json.loads(args.fixtures.read_text()))
    result['raw_measurements_sha256'] = hashlib.sha256(raw).hexdigest()
    args.output.write_text(json.dumps(result, indent=2) + '\n')
