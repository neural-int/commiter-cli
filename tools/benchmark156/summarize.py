"""Aggregate immutable observations, retaining unknown/failed denominators."""
import collections,json,math,pathlib,statistics
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'

def p95(xs):return sorted(xs)[max(0,math.ceil(.95*len(xs))-1)] if xs else None
if __name__=='__main__':
 rows=[json.loads(s) for s in (OUT/'iteration-4-results.jsonl').read_text().splitlines()]
 groups=collections.defaultdict(list)
 for r in rows:groups[(r['architecture'],r['stratum'])].append(r)
 summary=[]
 for (architecture,stratum),rs in groups.items():
  delivered=[r for r in rs if r['complete']]
  lat=[r['wall_seconds'] for r in rs]
  summary.append(dict(architecture=architecture,stratum=stratum,observations=len(rs),exact=sum(bool(r['exact']) for r in rs),exact_rate=sum(bool(r['exact']) for r in rs)/len(rs),coverage=len(delivered)/len(rs),valid_staging=sum(r['valid_staging'] for r in rs),fm_known=sum(r['false_merge'] for r in rs if r['false_merge'] is not None),fs_known=sum(r['false_split'] for r in rs if r['false_split'] is not None),pair_metrics_unknown=sum(r['false_merge'] is None or r['false_split'] is None for r in rs),calls=sum(r.get('calls',0) for r in rs),input_tokens_observed=sum(r.get('input_tokens',0) for r in rs),output_tokens_observed=sum(r.get('output_tokens',0) for r in rs),token_telemetry_incomplete=sum(r.get('token_telemetry_complete',True) is False for r in rs),mean_wall_seconds=statistics.mean(lat),p95_wall_seconds=p95(lat),commits_delivered=sum(r.get('commits',0) for r in delivered)))
 full=[]
 for architecture in ('A','baseline','h23'):
  rs=[r for r in rows if r['architecture']==architecture]
  full.append(dict(architecture=architecture,cases=len(rs),exact=sum(r['exact'] for r in rs),exact_rate=sum(r['exact'] for r in rs)/len(rs),complete=sum(r['complete'] for r in rs),wall_seconds=sum(r['wall_seconds'] for r in rs),calls=sum(r.get('calls',0) for r in rs)))
 canonical_stops=[]
 for r in rows:
  if r['complete']:continue
  obs=r.get('observed',{});calls=obs.get('calls',[])
  if r.get('reason')=='unit_budget':kind='structural_resource_unit_budget'
  elif any((c.get('input_tokens') or 0)+1536>16384 for c in calls if c.get('phase')=='global-purpose-assignment'):kind='context_resource_limit_inferred_from_recorded_tokens'
  elif 'timeout' in r.get('reason','') or any(c.get('stop')=='timeout' for c in calls):kind='resource_timeout'
  else:kind='invalid_model_metadata_or_semantic_output'
  canonical_stops.append(dict(name=r['name'],architecture=r['architecture'],reverse=r.get('reverse'),canonical_kind=kind,raw_reason=r.get('reason')))
 result=dict(summary=summary,totals=full,canonical_stops=canonical_stops,target='NOT_REACHED / current candidate NO-GO',production='NO-GO; no implementation issue or rollout is authorized by these observations',final_verification_complete=False,
   limitations=['D primary has only positive, mostly single-purpose, public-history projections; negative/mixed/shared-test/weak-edge independent coverage remains incomplete.', 'C oracle stage is not independent semantic evaluation.', '4/8/16 projections share one commit; no population-performance claim.', 'No retry, case exclusion, threshold relaxation or gold rewrite. Rejected 16-file remains in primary denominator.', 'All token sums are observed lower bounds when telemetry incomplete; null is not zero success.'])
 with (OUT/'iteration-4-summary.json').open('x') as f:json.dump(result,f,ensure_ascii=False,indent=2);f.write('\n')
 print(json.dumps(full,ensure_ascii=False))
