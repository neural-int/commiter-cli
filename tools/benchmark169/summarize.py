"""Deterministic frozen Gate calculations; semantic binding remains offline-reviewed."""
import hashlib,json,math,statistics
from collections import Counter
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2];DATA=ROOT/'docs/benchmarks/issue-169'
def read(p):return json.loads((DATA/p).read_text())
def lines(p):return list(map(json.loads,(DATA/p).read_text().splitlines())) if (DATA/p).exists() else []
def percentile(v,p):return sorted(v)[max(0,math.ceil(p*len(v))-1)] if v else None
def costs(rows):
    complete=all(r[k] is not None for r in rows for k in ['input_tokens','output_tokens','helper_peak_rss_bytes']) and all(None not in r[k].values() for r in rows for k in ['resource_before','resource_after'])
    walls=[r['wall_seconds'] for r in rows]
    swaps=[r['resource_after']['swap_used_mib']-r['resource_before']['swap_used_mib'] for r in rows if r['resource_after']['swap_used_mib'] is not None and r['resource_before']['swap_used_mib'] is not None]
    pressure=[r[k]['pressure_level'] for r in rows for k in ['resource_before','resource_after'] if r[k]['pressure_level'] is not None]
    result={'attempts':len(rows),'schema_valid':sum(r['schema_valid'] for r in rows),'input_tokens':sum(r['input_tokens'] or 0 for r in rows),'output_tokens':sum(r['output_tokens'] or 0 for r in rows),'wall_total_seconds':sum(walls),'wall_p50_seconds':statistics.median(walls) if walls else None,'wall_p95_seconds':percentile(walls,.95),'rss_max_bytes':max((r['helper_peak_rss_bytes'] or 0 for r in rows),default=0),'swap_delta_max_mib':max(swaps,default=None),'pressure_max':max(pressure,default=None),'observer_complete':complete,'git_unchanged':all(r['git_unchanged'] for r in rows),'stop_reasons':dict(Counter(r['status'] for r in rows))}
    result['resources_pass']=bool(rows) and complete and result['wall_p95_seconds']<=120 and result['rss_max_bytes']<=6*1024**3 and result['swap_delta_max_mib']<=256 and result['pressure_max']<4 and result['git_unchanged']
    result['preflight_token_match']=all(r['input_tokens']==r['token_count_before_inference'] for r in rows)
    return result

def discovery():
    pr=read('preregistration.json');manifest=read('frozen-inputs/manifest.json');rows=lines('holdout-discovery-measurements.jsonl');known=lines('known-discovery-measurements.jsonl');ev=read('holdout-evaluation.json')['cells'];fx=json.loads((ROOT/'tools/benchmark169/fixtures.json').read_text());gold={f['name']:len(f['gold_purposes']) for f in fx}
    all_by_key={(r['fixture'],r['order'],r['arm']):r for r in rows};judgments={(e['fixture'],e['order'],e['arm']):e for e in ev};metrics={}
    for arm in ['A','B','C']:
        planned=[r for r in manifest if r['cohort']=='holdout' and r['phase']=='discovery' and r['arm']==arm];rs=[r for r in rows if r['arm']==arm];es=[e for e in ev if e['arm']==arm]
        assert len({(e['fixture'],e['order']) for e in es})==len(es)
        expected=sum(gold[r['fixture']] for r in planned);covered=exact=overmerge=oversplit=grounded=hallucinated=total=target=0
        order_maps={}
        for row in planned:
            key=(row['fixture'],row['order'],arm);e=judgments.get(key);r=all_by_key.get(key)
            if not e or not r or not r['schema_valid']:continue
            assert e['request_sha256']==r['request_sha256'];assert len(e['purposes'])==len(r['response']['purposes'])
            semantic=set();targets=set();targetcounts=Counter();scounts=Counter()
            for p,out in zip(e['purposes'],r['response']['purposes']):
                assert p['description']==out['description'] and p['purpose_id']==out['purpose_id']
                ss=p['semantic_gold'];ts=p['target_gold'];assert set(ss)<=set(ts) and all(1<=x<=gold[row['fixture']] for x in ts)
                semantic.update(ss);targets.update(ts);targetcounts.update(ts);scounts.update(ss);overmerge+=max(0,len(ts)-1);grounded+=p['grounded'];hallucinated+=p['hallucination'];total+=1
            extra=sum(max(0,n-1) for n in targetcounts.values());oversplit+=extra;covered+=len(semantic);target+=len(targets)
            cell_exact=len(semantic)==gold[row['fixture']] and len(e['purposes'])==gold[row['fixture']] and all(len(p['semantic_gold'])==1 for p in e['purposes']) and all(n==1 for n in scounts.values()) and extra==0 and all(p['grounded'] and not p['hallucination'] for p in e['purposes'])
            exact+=cell_exact
            order_maps[(row['fixture'],row['order'])]=sorted(tuple(sorted(p['semantic_gold'])) for p in e['purposes'])
        m=costs(rs);m.update(planned_cells=len(planned),admitted_cells=sum(r['admitted'] for r in planned),covered_gold=covered,expected_gold=expected,coverage=covered/expected,exact=exact,exact_rate=exact/len(planned),target_only_coverage=target,over_merge=overmerge,over_split=oversplit,grounded=grounded,generated_purposes=total,grounding_rate=grounded/total if total else 0,hallucinations=hallucinated,order_disagreement=sum(order_maps.get((f['name'],'normal'))!=order_maps.get((f['name'],'reverse')) for f in fx if f['cohort']=='holdout'))
        m['absolute_quality_pass']=m['admitted_cells']==m['planned_cells']==len(rs)==m['schema_valid'] and m['coverage']>=.9 and m['exact_rate']>=.8 and m['over_merge']==0 and m['over_split']==0 and m['grounding_rate']>=.95 and m['hallucinations']==0
        m['absolute_pass']=m['absolute_quality_pass'] and m['resources_pass'] and m['preflight_token_match']
        metrics[arm]=m
    a,b,c=[metrics[x] for x in ['A','B','C']]
    def improves(x,y):return (x['coverage']-y['coverage']>=.25-1e-9 or x['exact_rate']-y['exact_rate']>=.25-1e-9) and x['coverage']>=y['coverage'] and x['exact_rate']>=y['exact_rate'] and x['hallucinations']<=y['hallucinations'] and x['over_split']<=y['over_split']
    no_lost_attempts=len(lines('attempts.jsonl'))==len(known)+len(rows)
    gate=(b['absolute_pass'] and improves(b,a) or c['absolute_pass'] and improves(c,a)) and all(r['git_unchanged'] for r in rows+known) and all(costs([r])['resources_pass'] for r in rows+known) and no_lost_attempts
    cost_ok=c['input_tokens']<=b['input_tokens']*1.25 and c['wall_p95_seconds']<=b['wall_p95_seconds']*1.25
    c_increment=c['covered_gold']>=b['covered_gold']+1 and c['exact']>=b['exact']+1 and c['over_merge']<=b['over_merge'] and c['over_split']<=b['over_split'] and c['hallucinations']<=b['hallucinations'] and c['grounding_rate']>=b['grounding_rate'] and cost_ok
    decision='C_CANDIDATE' if c['absolute_pass'] and c_increment else ('B_PREFERRED' if b['absolute_pass'] else ('DEFER' if c['absolute_pass'] else 'B_C_NO_GO'))
    result={'phase':1,'gate1':'GO' if gate else 'NO-GO','arms':metrics,'known_costs':{x:costs([r for r in known if r['arm']==x]) for x in ['A','B','C']},'C_minus_B':{'covered_gold':c['covered_gold']-b['covered_gold'],'exact_cells':c['exact']-b['exact'],'coverage_pp':(c['coverage']-b['coverage'])*100,'exact_pp':(c['exact_rate']-b['exact_rate'])*100,'input_token_ratio':c['input_tokens']/b['input_tokens'],'wall_p95_ratio':c['wall_p95_seconds']/b['wall_p95_seconds'],'cost_pass':cost_ok,'beneficial_increment_pass':c_increment},'phase1_absolute_and_increment_decision':decision,'generation_calls':len(known)+len(rows),'max_generation_calls':pr['max_generation_calls'],'lost_attempts':not no_lost_attempts,'phase2':'gated_out' if not gate else 'required','production_go':False,'evaluation':'Offline agent binding against preregistered gold/input, not independent human rating; four synthetic holdout fixtures, two orders are paired observations'}
    (DATA/'phase1-summary.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result,indent=2))
if __name__=='__main__':discovery()
