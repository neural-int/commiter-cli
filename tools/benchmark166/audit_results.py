"""Recompute all fixed diagnostic metrics from immutable raw responses; no inference."""
import importlib.util
import pathlib
import statistics
from collections import Counter

module = importlib.util.spec_from_file_location('experiment166', pathlib.Path(__file__).with_name('diagnose.py'))
exp = importlib.util.module_from_spec(module)
module.loader.exec_module(exp)


def summarize(rows, cases, arm):
    valid = [r for r in rows if r['schema_valid']]
    resources = [r['resource'] for r in rows if r.get('resource')]
    result = dict(rows=len(rows), calls=sum(r['calls'] for r in rows),
        schema_valid=len(valid), statuses=dict(Counter(r['status'] for r in rows)),
        label_distribution=dict(Counter(r['answer']['relation'] for r in valid)),
        relation_correct=sum(r['answer']['relation'] == cases[r['case']]['expected_relation'] for r in valid),
        independent_correct=sum(r['answer']['relation'] == 'independent' and cases[r['case']]['expected_relation'] == 'independent' for r in valid),
        independent_rows=sum(cases[r['case']]['expected_relation'] == 'independent' for r in rows),
        citation_all_exist_rows=sum(r.get('citations_exist', False) for r in rows),
        exact_citation_claims=sum(a['exists'] for r in rows for a in r.get('citation_audit', [])),
        citation_claims=sum(len(r.get('citation_audit', [])) for r in rows),
        label_order_stable=sum(len(pair) == 2 and all(r['schema_valid'] for r in pair) and pair[0]['answer']['relation'] == pair[1]['answer']['relation']
            for c in cases for pair in [[r for r in rows if r['case'] == c]]),
        input_tokens=sum(r.get('input_tokens') or 0 for r in rows),
        output_tokens=sum(r.get('output_tokens') or 0 for r in rows),
        direction_gold_correct=sum(r.get('direction_correct', False) for r in rows) if arm != 'T2' else None,
        nondependency_direction_violations=sum(r['answer']['relation'] != 'dependency' and any(r['answer'][k] != 'none' for k in ('provider', 'consumer')) for r in valid) if arm != 'T2' else None,
        direction_fields_present_rows=sum(any(k in r['answer'] for k in ('provider', 'consumer')) for r in valid),
        direction_comparability='not scored; schema omission is not accuracy improvement' if arm == 'T2' else 'gold direction and output rule violations separately',
        pressure_warning_calls=sum(any(v != 1 for v in r.get('pressure_levels', [])) for r in resources),
        pressure_levels=dict(Counter(str(v) for r in resources for v in r.get('pressure_levels', []))),
        swap_increase_calls=sum((r.get('swap_delta_bytes') or 0) > 0 for r in resources))
    for name in ('ttft_seconds', 'total_seconds'):
        values = [r.get(name) for r in resources]
        result[name + '_median'] = statistics.median(values) if values and all(v is not None for v in values) else None
    for name in ('sampled_peak_footprint_bytes', 'observed_lifetime_peak_footprint_bytes', 'sampled_peak_rss_bytes', 'mlx_peak_bytes'):
        values = [r.get(name) for r in resources]
        result[name + '_max'] = max(values) if values and all(v is not None for v in values) else None
    return result


def audit():
    p = exp.load('preregistered.json')
    ref = exp.load('baseline-reference.json')
    raw = (exp.OUT / 'results.jsonl').read_bytes()
    rows = [exp.loads(line) for line in raw.splitlines()]
    meta = exp.load('run-metadata.json')
    cases = {c['name']: c for c in p['cases']}
    assert len(rows) == 32 and meta['generation_calls'] == sum(r['calls'] for r in rows) <= 32
    assert p['identity'] == exp.identity() and meta['post_identity_matches']
    for name, digest in p['artifacts_sha256'].items():
        assert exp.sha((exp.OUT / name).read_bytes()) == digest, name
    baseline_rows = ref['records']
    for r in baseline_rows:
        r['citation_audit'] = exp.baseline.citation_audit(r['answer'], cases[r['case']], 'B1')
        assert r['citations_exist'] == all(a['exists'] for a in r['citation_audit'])
    for spec, row in zip(p['rows'], rows):
        assert all(row[k] == spec[k] for k in ('case', 'stage', 'reverse', 'prompt_sha256', 'request_sha256'))
        assert exp.sha(spec['request_string'].encode()) == row['request_sha256']
        assert spec['request_string'] == (exp.encode(spec['request']) + b'\n').decode()
        c = cases[row['case']]
        if row['calls'] == 0:
            assert meta['stop_reason'] is not None
            continue
        outer = exp.loads(row['resource']['stdout'])
        assert outer == row.get('outer_response')
        if row['schema_valid']:
            assert outer['model'] == p['model'] and outer['benchmark_input_tokens'] == spec['native_input_tokens']
            assert outer['ok'] and outer['stop_reason'] == 'completed' and row['resource']['exit'] == 0 and not row['resource']['timed_out']
            answer = exp.loads(outer['generated_json'])
            assert answer == row['answer']
            exp.baseline.validate(answer, spec['schema'])
            citations = exp.baseline.citation_audit(answer, c, spec['stage'])
            assert citations == row['citation_audit']
            assert row['citations_exist'] == all(a['exists'] for a in citations)
            assert row['relation_label_correct'] == (answer['relation'] == c['expected_relation'])
            assert row['direction_correct'] == (all(answer[k] == v for k, v in c['expected_direction'].items()) if spec['stage'] == 'T1' else None)
        assert row['stop_after_call'] == exp.stop_reason(row['resource'], exp.swap_bytes(meta['system_before'])) if row['status'] == 'completed' else True
    all_rows = {'B1': baseline_rows, 'T1': [r for r in rows if r['stage'] == 'T1'], 'T2': [r for r in rows if r['stage'] == 'T2']}
    summary = {arm: summarize(rr, cases, arm) for arm, rr in all_rows.items()}
    review = exp.load('reason-review.json')
    assert len(review['rows']) == 48
    assert len({(r['case'], r['arm'], r['reverse']) for r in review['rows']}) == 48
    for arm, rr in all_rows.items():
        for row in rr:
            audited = next(r for r in review['rows'] if (r['case'], r['arm'], r['reverse']) == (row['case'], arm, row['reverse']))
            assert audited['answer_sha256'] == exp.sha(exp.encode(row['answer']))
            assert audited['reason'] == row['answer']['reason']
            assert audited['verdict'] in ('supported', 'partially_supported', 'unsupported', 'unavailable')
            assert audited['source_basis'] == cases[row['case']]['audit_facts']
        summary[arm]['reason_source_support'] = dict(Counter(r['verdict'] for r in review['rows'] if r['arm'] == arm))
    comparisons, case_evidence = {}, []
    for arm in ('T1', 'T2'):
        pairs = [(b, next(r for r in all_rows[arm] if (r['case'], r['reverse']) == (b['case'], b['reverse']))) for b in baseline_rows]
        complete = meta['stop_reason'] is None and all(r['schema_valid'] and r['status'] == 'completed' for _, r in pairs)
        label_changed = sum(b['answer']['relation'] != r['answer']['relation'] for b, r in pairs if r['schema_valid'])
        classification = ('schema-order effect supported' if arm == 'T1' else 'field-interference possible') if label_changed else 'no material change'
        comparisons[arm] = dict(classification=classification if complete else 'undetermined', complete_contract=complete,
            label_changed_rows=label_changed, reason_changed_rows=sum(b['answer']['reason'] != r['answer']['reason'] for b, r in pairs if r['schema_valid']),
            evidence_changed_rows=sum(b['answer']['evidence'] != r['answer']['evidence'] for b, r in pairs if r['schema_valid']),
            native_input_delta_rows=[dict(case=b['case'], reverse=b['reverse'], delta=r.get('input_tokens', 0) - b['input_tokens']) for b, r in pairs],
            limit='historical baseline and known exploratory corpus; dual schema/grammar and T2 prompt/length effects not isolated')
    for c in p['cases']:
        case_evidence.append(dict(case=c['name'], expected_relation=c['expected_relation'], expected_direction=c['expected_direction'],
            records={arm: [dict(reverse=r['reverse'], status=r['status'], answer=r['answer'], citation_audit=r.get('citation_audit'),
                input_tokens=r.get('input_tokens'), output_tokens=r.get('output_tokens'), direction_gold_correct=r.get('direction_correct'))
                for r in rr if r['case'] == c['name']] for arm, rr in all_rows.items()},
            semantic_truth='see separately audited reason-review.json; exact citations alone do not prove relation truth'))
    exp.write('summary.json', dict(issue=166, conditions=summary, comparisons=comparisons,
        run_metadata=meta, raw_sha256=exp.sha(raw), generated_retries=0, tokenizer_calls=33,
        initial_preparation_observation_gap=True, production_changed=False))
    exp.write('case-evidence.json', case_evidence)
    exp.write('result-audit.json', dict(pass_all=True, raw_sha256=exp.sha(raw), rows=32,
        independent_reparse=True, native_token_contract_checked=True, source_quote_recomputed=True,
        artifacts_hashes_checked=True, identity_matches=True, full_assignment_or_boundary_tests='not applicable; B relation diagnostic only',
        reason_truth='requires separate source audit', no_generation_calls=True))
    print(exp.encode(dict(conditions=summary, comparisons={k: {n: v for n, v in x.items() if n != 'native_input_delta_rows'} for k, x in comparisons.items()})).decode())


if __name__ == '__main__':
    audit()
