"""Aggregate frozen gold bindings from offline evaluation; never infer semantics."""
import hashlib
import json
import math
from pathlib import Path
import statistics
from collections import Counter

ROOT=Path(__file__).resolve().parents[2]
DATA=ROOT/'docs/benchmarks/issue-168'

def read(name):
    return json.loads((DATA/name).read_text())

def lines(name):
    return list(map(json.loads,(DATA/name).read_text().splitlines()))

def main():
    fixtures=json.loads((ROOT/'tools/benchmark168/fixtures.json').read_text())
    gold={f['name']:f['gold_purposes'] for f in fixtures}
    records=lines('known-v2-measurements.jsonl')+lines('holdout-v2-measurements.jsonl')
    evaluations=read('known-evaluation.json')['cells']+read('holdout-evaluation.json')['cells']
    by={(e['fixture'],e['order']):e for e in evaluations}
    assert len(by)==len(evaluations)==len(records)
    for record in records:
        e=by[record['fixture'],record['order']]
        purposes=(record['response'] or {}).get('purposes',[])
        assert len(e['purposes'])==len(purposes)
        for p,rp in zip(e['purposes'],purposes):
            assert p['purpose_id']==rp['purpose_id'] and p['description']==rp['description']
            assert set(p['semantic_gold']).issubset(set(range(1,len(gold[e['fixture']])+1)))
            assert set(p['target_gold']).issubset(set(range(1,len(gold[e['fixture']])+1)))
            assert isinstance(p['grounded'],bool) and p['reason']
        semantic=Counter(g for p in e['purposes'] for g in p['semantic_gold'])
        targets=Counter(g for p in e['purposes'] for g in p['target_gold'])
        e['gold_count']=len(gold[e['fixture']])
        e['covered']=len(semantic)
        e['over_merge_pairs']=sum(len(p['target_gold'])*(len(p['target_gold'])-1)//2 for p in e['purposes'])
        e['over_split_extra']=sum(max(0,v-1) for v in targets.values())
        e['grounded']=sum(p['grounded'] for p in e['purposes'])
        e['hallucinations']=sum(p['hallucination'] for p in e['purposes'])
        e['exact']=record['schema_valid'] and len(semantic)==len(gold[e['fixture']]) and e['over_merge_pairs']==0 and e['over_split_extra']==0 and e['hallucinations']==0 and e['grounded']==len(e['purposes'])
    summary={}
    for cohort in ['known','holdout','independent_holdout','known_reused']:
        rows=[r for r in records if (r['cohort']==cohort if cohort in ['known','holdout'] else r['fixture'] in ({'H1','H2','H3','H5'} if cohort=='independent_holdout' else {'H4'}))]
        es=[by[r['fixture'],r['order']] for r in rows]
        expected=sum(len(gold[e['fixture']]) for e in es)
        if cohort in ['holdout','independent_holdout']:
            expected_planned=2*sum(len(f['gold_purposes']) for f in fixtures if f['cohort']=='holdout' and (cohort=='holdout' or f['name']!='H4'))
            planned_cells=12 if cohort=='holdout' else 10
        else:
            expected_planned=expected;planned_cells=2
        walls=sorted(r['wall_seconds'] for r in rows)
        total_purposes=sum(len(e['purposes']) for e in es)
        summary[cohort]={'attempts':len(rows),'planned_cells':planned_cells,'schema_valid':sum(r['schema_valid'] for r in rows),'purpose_coverage_reached':sum(e['covered'] for e in es)/expected,'purpose_coverage_planned':sum(e['covered'] for e in es)/expected_planned,'covered_gold':sum(e['covered'] for e in es),'reached_gold':expected,'planned_gold':expected_planned,'exact':sum(e['exact'] for e in es),'exact_rate_planned':sum(e['exact'] for e in es)/planned_cells,'over_merge_pairs':sum(e['over_merge_pairs'] for e in es),'over_split_extra':sum(e['over_split_extra'] for e in es),'grounded_descriptions':sum(e['grounded'] for e in es),'generated_purposes':total_purposes,'grounding_rate':sum(e['grounded'] for e in es)/total_purposes if total_purposes else 0,'hallucinations':sum(e['hallucinations'] for e in es),'wall_p50_seconds':statistics.median(walls),'wall_p95_seconds':walls[math.ceil(.95*len(walls))-1],'input_tokens':sum(r['input_tokens'] or 0 for r in rows),'output_tokens':sum(r['output_tokens'] or 0 for r in rows),'helper_peak_rss_max_bytes':max(r['helper_peak_rss_bytes'] or 0 for r in rows),'swap_delta_max_mib':max(r['resource_after']['swap_used_mib']-r['resource_before']['swap_used_mib'] for r in rows),'swap_delta_over_256_count':sum(r['resource_after']['swap_used_mib']-r['resource_before']['swap_used_mib']>256 for r in rows),'git_unchanged':sum(r['git_unchanged'] for r in rows),'stop_reasons':dict(Counter(r['status'] for r in rows))}
    holdout=summary['holdout'];pr=read('preregistration.json');g=pr['gate1'];initial=lines('known-measurements.jsonl')
    allrows=initial+records
    manifest=read('frozen-inputs/manifest.json')
    criteria={'all_planned_inputs_admitted':not any('admission_failure' in r for r in manifest),'v2_schema_valid_rate_1':all(r['schema_valid'] for r in records),'coverage_at_least_90_percent':holdout['purpose_coverage_planned']>=g['holdout_purpose_coverage_min'],'exact_at_least_80_percent':holdout['exact_rate_planned']>=g['holdout_exact_min'],'over_merge_zero':holdout['over_merge_pairs']==0,'over_split_zero':holdout['over_split_extra']==0,'grounded_at_least_95_percent':holdout['grounding_rate']>=g['holdout_grounded_descriptions_min'],'hallucinations_zero':holdout['hallucinations']==0,'git_unchanged_all':all(r['git_unchanged'] for r in allrows),'wall_within_120':all(r['wall_seconds']<=120 for r in allrows),'rss_within_6gib':all(r['helper_peak_rss_bytes'] is not None and r['helper_peak_rss_bytes']<=g['rss_max_bytes'] for r in allrows),'swap_delta_within_256mib':all(r['resource_after']['swap_used_mib']-r['resource_before']['swap_used_mib']<=g['swap_delta_max_mib'] for r in allrows),'pressure_below_4':all(r['resource_before']['pressure_level']<4 and r['resource_after']['pressure_level']<4 for r in allrows),'process_calls_at_most_16':len(allrows)<=pr['max_calls']}
    summary.update(process_calls=len(allrows),invalid_request_calls=len(initial),generation_calls=len(records),baseline_generation_calls=0,additional_llm_verification_calls=0,criteria=criteria,decision='GO' if all(criteria.values()) else 'NO_GO',phase2='not-evaluated: Gate 1 unmet',phase3='not-evaluated: Gate 1 unmet',production_go=False,cells=evaluations)
    assert len(allrows)==16
    assert summary['decision']=='NO_GO'
    # Verify the frozen file evidence and projection equality independent of runtime paths.
    for row in manifest:
        if 'admission_failure' in row:continue
        requests=[read(f"frozen-inputs/{row['fixture']}-{row['order']}-{arm}.json") for arm in ['baseline','discovery']]
        files=[json.loads(r['messages'][1]['content'])['files'] for r in requests]
        assert files[0]==files[1]
        for file in files[0]:
            assert not set(file)&{'old_mode','new_mode','head_identity','worktree_identity','worktree_kind','change_hash','size','staged','unstaged'}
            for evidence in file.get('evidence',[]):assert not set(evidence)&{'start_line','end_line','start_byte','end_byte'}
    summary['same_evidence_ordered_pairs']=14
    summary['cohort_correction']='H4 matches prior #167 fixture exactly; independent holdout subset excludes H4; original planned-cohort metrics retained'
    summary['independent_holdout_gate_unmet']=True
    (DATA/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({k:v for k,v in summary.items() if k!='cells'},ensure_ascii=False,indent=2))

if __name__=='__main__':main()
