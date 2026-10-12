"""Aggregate offline semantic judgments; failed/missing cells stay in denominators."""
import hashlib,json,sys
from collections import Counter
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2];DATA=ROOT/'docs/benchmarks/issue-170'
sys.path.insert(0,str(ROOT/'tools/benchmark169'))
from summarize import costs
def read(p):return json.loads((DATA/p).read_text())
def lines(p):return list(map(json.loads,(DATA/p).read_text().splitlines())) if (DATA/p).exists() else []
def metrics(cohort):
 rows=lines(cohort+'-discovery-measurements.jsonl');ev=read(cohort+'-evaluation.json')['cells'];manifest=read('frozen-inputs/manifest.json');fixtures=json.loads((ROOT/'tools/benchmark170/fixtures.json').read_text());gold={f['name']:len(f['gold_purposes']) for f in fixtures}
 keyed={(e['fixture'],e['order'],e['arm']):e for e in ev};assert len(keyed)==len(ev)
 result={}
 for arm in 'ABC':
  planned=[r for r in manifest if r['arm']==arm and r['cohort']==cohort and r['phase']=='discovery' and (cohort!='known' or r['order']=='normal')];rs=[r for r in rows if r['arm']==arm];out={ (r['fixture'],r['order'],r['arm']):r for r in rs };assert len(out)==len(rs)
  m=costs(rs);m.update(planned_cells=len(planned),admitted_cells=sum(r['admitted'] for r in planned),expected_gold=sum(gold[r['fixture']] for r in planned),covered_gold=0,target_covered=0,exact=0,over_merge=0,over_split=0,grounded=0,hallucinations=0,unknown_purposes=0,generated_purposes=0);orders={}
  for row in planned:
   key=(row['fixture'],row['order'],arm);r=out.get(key);e=keyed.get(key)
   if not r or not r['schema_valid']:
    orders[(row['fixture'],row['order'])]=('unresolved',r['status'] if r else 'missing');continue
   assert e is not None and e['request_sha256']==r['request_sha256'] and len(e['purposes'])==len(r['response']['purposes'])
   semantic=set();targets=set();counts=Counter();scounts=Counter();signatures=[]
   for p,generated in zip(e['purposes'],r['response']['purposes']):
    assert p['description']==generated['description'] and p['purpose_id']==generated['purpose_id']
    ss=p['semantic_gold'];ts=p['target_gold'];assert set(ss)<=set(ts) and all(type(x)is int and 1<=x<=gold[row['fixture']] for x in ts)
    assert type(p['grounded'])is bool and type(p['hallucination'])is bool and p['reason'].strip()
    semantic.update(ss);targets.update(ts);counts.update(ts);scounts.update(ss);m['over_merge']+=max(0,len(ts)-1);m['grounded']+=p['grounded'];m['hallucinations']+=p['hallucination'];m['unknown_purposes']+=not ts;m['generated_purposes']+=1
    signatures.append((tuple(ss),tuple(ts),p['grounded'],p['hallucination']))
   extra=sum(max(0,n-1) for n in counts.values());m['over_split']+=extra;m['covered_gold']+=len(semantic);m['target_covered']+=len(targets)
   exact=len(semantic)==gold[row['fixture']] and len(e['purposes'])==gold[row['fixture']] and all(len(p['semantic_gold'])==1 for p in e['purposes']) and all(n==1 for n in scounts.values()) and extra==0 and all(p['grounded'] and not p['hallucination'] for p in e['purposes'])
   m['exact']+=exact;orders[(row['fixture'],row['order'])]=tuple(sorted(signatures))
  m.update(coverage=m['covered_gold']/m['expected_gold'],exact_rate=m['exact']/m['planned_cells'],grounding_rate=m['grounded']/m['generated_purposes'] if m['generated_purposes'] else 0,order_disagreement=sum(orders.get((f['name'],'normal'))!=orders.get((f['name'],'reverse')) for f in fixtures if f['cohort']==cohort) if cohort=='holdout' else None)
  m['absolute_quality_pass']=m['admitted_cells']==m['planned_cells']==len(rs)==m['schema_valid'] and m['coverage']>=.9 and m['exact_rate']>=.8 and m['over_merge']==m['over_split']==m['hallucinations']==m['unknown_purposes']==0 and m['grounding_rate']>=.95 and m['order_disagreement']==0
  m['absolute_pass']=m['absolute_quality_pass'] and m['resources_pass'] and m['preflight_token_match'];result[arm]=m
 return result
def main():
 known=metrics('known');holdout=metrics('holdout');a=holdout['A'];candidates=[]
 for arm in 'BC':
  m=holdout[arm];m['improvement_over_A']= (m['coverage']-a['coverage']>=.25-1e-9 or m['exact_rate']-a['exact_rate']>=.25-1e-9) and m['coverage']>=a['coverage'] and m['exact_rate']>=a['exact_rate'] and all(m[k]<=a[k] for k in ['over_merge','over_split','hallucinations'])
  m['relative_cost_pass']=bool(m['wall_p95_seconds'] is not None and a['wall_p95_seconds'] is not None and m['wall_p95_seconds']<=a['wall_p95_seconds']*1.25)
  if m['absolute_pass'] and m['improvement_over_A'] and m['relative_cost_pass']:candidates.append(arm)
 rows=lines('known-discovery-measurements.jsonl')+lines('holdout-discovery-measurements.jsonl');journal=lines('attempts.jsonl');key=lambda r:(r['fixture'],r['order'],r['arm'],r['phase'])
 no_lost=len(journal)==len(rows)==36 and {key(r) for r in journal}=={key(r) for r in rows}
 gate=bool(candidates) and no_lost and all(costs([r])['resources_pass'] for r in rows)
 summary=dict(gate1='GO' if gate else 'NO-GO',candidates=candidates if gate else [],known=known,arms=holdout,generation_calls=len(journal),lost_attempts=not no_lost,phase2='required' if gate else 'not_evaluated_gate1_no_go',production_go=False,evaluation='offline same-agent source/gold binding; four synthetic holdouts, two paired orders; not independent human rating')
 (DATA/'phase1-summary.json').write_text(json.dumps(summary,indent=2)+'\n');print(json.dumps(summary,indent=2))
if __name__=='__main__':main()
