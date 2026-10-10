"""Post-run independent recomputation and evidence linkage, no model calls."""
import collections
import json
import pathlib
import time
import diagnose as d


def audit():
    start=time.monotonic()
    fixed=d.loads((d.OUT/'preregistered.json').read_bytes())
    rows=[d.loads(l) for l in (d.OUT/'results.jsonl').read_text().splitlines()]
    summary=d.loads((d.OUT/'summary.json').read_bytes())
    keys=lambda r:(r['case'],r['reverse'],r['stage'])
    assert len(rows)==len(fixed['rows'])==80 and len({keys(r) for r in rows})==80
    assert {keys(r) for r in rows}=={keys(r) for r in fixed['rows']}
    specs={keys(r):r for r in fixed['rows']};cases={c['name']:c for c in fixed['cases']}
    manual=d.loads((d.OUT/'fact-review.json').read_bytes())
    reviewed={(r['case'],r['reverse']):r for r in manual['rows']}
    assert len(reviewed)==16
    historical=[d.loads(l) for l in (d.ROOT/'docs/benchmarks/issue-164/results.jsonl').read_text().splitlines()]
    comparisons=[];cache={};checks=[]
    fact_counts=collections.Counter();extra_counts=collections.Counter()
    for row in rows:
        spec=specs[keys(row)];c=cases[row['case']]
        assert row['prompt_sha256']==spec['prompt_sha256']==d.sha(d.encode(spec['messages']))
        assert row['calls']==1 and row['native_tokens_match'] and row['model_pin_matches']
        assert d.loads(row['resource']['stdout'])==row['outer_response']
        assert row['input_tokens']==spec['native_input_tokens']
        answer=row['answer']
        if row['stage'].startswith('C'):
            classified=d.classify(row['resource']['stdout'].encode(),'local',['EA','EB'])
            assert classified['status'] in ('completed','unresolved') and row['schema_valid']
            assert row['allowed']==d.allowed(c,answer['decision'])
            if answer['decision']!='defer':
                groups=[['EA','EB']] if answer['decision']=='merge' else d.split(c);key=(c['name'],d.encode(groups))
                if key not in cache:cache[key]=d.plan_audit(c,groups,fixed['validator_path'])
                assert row['groups']==groups
                fresh=cache[key]
                assert fresh['authoritative']['valid'] and fresh['reconstruction']['exact'] and fresh['all_ordered_pass']
                assert row['plan_audit']['authoritative']==fresh['authoritative']
        else:
            d.validate(answer,spec['schema']);assert row['schema_valid']
            cites=d.citation_audit(answer,c,row['stage']);assert cites==row['citation_audit']
            assert all(r['exists'] for r in cites)==row['citations_exist']
            if row['stage'].startswith('B'):
                assert (answer['relation']==c['expected_relation'])==row['relation_label_correct']
                assert all(answer[k]==v for k,v in c['expected_direction'].items())==row['direction_correct']
            else:
                rev=reviewed[(row['case'],row['reverse'])]
                assert rev['answer_sha256']==d.sha(d.encode(answer))
                assert {f['fact_id'] for f in rev['fact_reviews']}=={f['fact_id'] for f in c['audit_facts']}
                for f in rev['fact_reviews']:
                    assert f['verdict'] in ('present_correct','missing','wrong','partial')
                    assert all(1<=i<=len(answer['facts']) for i in f['claim_indices'])
                    fact_counts[f['verdict']]+=1
                for e in rev['extra_claims']:extra_counts[e['verdict']]+=1
        checks.append(dict(case=row['case'],reverse=row['reverse'],stage=row['stage'],raw_sha256=d.sha(row['resource']['stdout'].encode()),
                           source_sha256=c['input_sha256'],schema_valid=row['schema_valid'],citations_exist=row['citations_exist']))
    for c in fixed['cases']:
        pair=[]
        for reverse in (False,True):
            result={stage:next(r for r in rows if r['case']==c['name'] and r['reverse']==reverse and r['stage']==stage) for stage in ('A','B1','B2','C1','C2')}
            h={arm:next(r for r in historical if r['cohort']=='fresh' and r['case']==c['name'] and r['reverse']==reverse and r['arm']==arm) for arm in ('baseline','policy')}
            f=reviewed[(c['name'],reverse)]
            categories=[]
            if any(x['verdict'] in ('missing','partial') for x in f['fact_reviews']):categories.append('fact-missing')
            if any(x['verdict']=='wrong' for x in f['fact_reviews']) or any(x['verdict']=='wrong' for x in f['extra_claims']):categories.append('fact-wrong')
            if any(not result[s]['relation_label_correct'] or not result[s]['direction_correct'] for s in ('B1','B2')):categories.append('relation-wrong')
            if any(not result[s]['allowed'] for s in ('C1','C2')):categories.append('boundary-wrong')
            if c['expected_relation'] in ('multiple_defensible','insufficient') or any(not result[s]['citations_exist'] for s in ('A','B1','B2')):categories.append('unsupported-or-ambiguous')
            # No cross-task result identifies an internal causal stage or historical policy cause.
            categories.append('undetermined')
            pair.append(dict(reverse=reverse,fact_verdicts={x['fact_id']:x['verdict'] for x in f['fact_reviews']},
                             external_missing_recognized=f['external_missing_recognized'],
                             B={s:dict(relation=result[s]['answer']['relation'],provider=result[s]['answer']['provider'],consumer=result[s]['answer']['consumer'],
                                       label_correct=result[s]['relation_label_correct'],direction_correct=result[s]['direction_correct'],citations_exist=result[s]['citations_exist'],
                                       reason=result[s]['answer']['reason'],reason_source_supported=c['expected_relation']=='compensation',
                                       reason_support_audit='Explicit codec roundtrip supports compensation; generic assertion of joint contract in other cases is not a source-proven justification.') for s in ('B1','B2')},
                             C={s:dict(decision=result[s]['answer']['decision'],allowed=result[s]['allowed']) for s in ('C1','C2')},
                             historical164={arm:r['answer']['decision'] for arm,r in h.items()},
                             oracle_relation_changed=any(result['B1']['answer'][k]!=result['B2']['answer'][k] for k in ('relation','provider','consumer')),
                             oracle_boundary_changed=result['C1']['answer']['decision']!=result['C2']['answer']['decision'],
                             baseline_historical_reproduced=result['C1']['answer']==h['baseline']['answer'],
                             provisional_failure_categories=categories,
                             supported='Output omissions/false claims, relationship/direction errors and boundary disagreements are independently observable.',
                             counterevidence='Correct C on positive cases can coexist with imperfect A/B; oracle facts do not identify a sole missing-fact cause.',
                             undetermined='Internal thought, schema wording/class bias, format/length/salience effects, and historical policy-specific defer cause cannot be localized by this design.'))
        comparisons.append(dict(case=c['name'],expected_relation=c['expected_relation'],expected_direction=c['expected_direction'],gold_basis=c['gold_basis'],
                                source_sha256=c['input_sha256'],multiple_boundary=c['multiple_boundary'],external_information=c['missing_external'],observations=pair))
    allowed_counts={s:sum(r.get('allowed',False) for r in rows if r['stage']==s) for s in ('C1','C2')}
    for s,count in allowed_counts.items():assert count==summary['stages'][s]['allowed']
    assert sum(r['calls'] for r in rows)==summary['generation_calls']==80
    assert summary['post_identity_match'] and summary['all_native_tokens_match']
    total=summary['preparation_seconds']+summary['run_seconds']
    assert total<=fixed['whole_seconds']
    scope=d.command(['git','diff','--name-only',d.BASE]).stdout.decode().splitlines()
    assert all(p.startswith(('docs/benchmarks/issue-165/','tools/benchmark165/')) for p in scope)
    d.write('case-evidence.json',comparisons)
    d.write('result-audit.json',dict(issue=165,all_pass=True,scope=scope,checked_rows=checks,
                                    calls=80,retries=0,additional_generations=0,fact_counts=dict(fact_counts),extra_counts=dict(extra_counts),
                                    exact_quote_claims=sum(all(e['exists'] for e in d.citation_audit(r['answer'],cases[r['case']],'A')) for r in rows if r['stage']=='A'),
                                    fresh_plan_checks={c:dict(authoritative=v['authoritative'],reconstruction=v['reconstruction'],all_ordered_pass=v['all_ordered_pass']) for (c,_),v in cache.items()},
                                    total_experiment_seconds=total,post_audit_seconds=time.monotonic()-start,
                                    reference_audit=manual['reference_audit'],semantic_review_by=manual['reviewer'],
                                    fact_review_sha256=d.sha((d.OUT/'fact-review.json').read_bytes()),
                                    results_sha256=d.sha((d.OUT/'results.jsonl').read_bytes()),summary_sha256=d.sha((d.OUT/'summary.json').read_bytes()),
                                    production_changed=False,limits=fixed['causal_limits']))
    print(json.dumps(dict(all_pass=True,rows=len(rows),fact_counts=dict(fact_counts),post_audit_seconds=time.monotonic()-start)))


if __name__=='__main__':audit()
