import collections,json,pathlib,statistics
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def aggregate(rows):
 primary=[r for r in rows if r['primary']];walls=sorted(r['wall_seconds'] for r in rows)
 grains={}
 for g in sorted({r['metric_granularity'] for r in primary}):
  relevant=[r for r in primary if r['metric_granularity']==g];complete=[r for r in relevant if r['complete']]
  grains[g]=dict(cases=len(relevant),complete=len(complete),null_pair_cases=sum(r['false_merge'] is None or r['false_split'] is None for r in relevant),fm=sum(r['false_merge'] or 0 for r in complete) if complete else None,fs=sum(r['false_split'] or 0 for r in complete) if complete else None,fm_case=sum(bool(r['false_merge']) for r in complete),fs_case=sum(bool(r['false_split']) for r in complete))
 return dict(observations=len(rows),primary=len(primary),excluded_gold=len(rows)-len(primary),exact=sum(r['exact'] is True for r in primary),complete=sum(r['complete'] for r in rows),grains=grains,wall_total=sum(walls),wall_mean=statistics.mean(walls),wall_p95_lower=walls[int((len(walls)-1)*.95)],calls=sum(r.get('calls',0) for r in rows),input_tokens_lower_bound=sum(r.get('input_tokens_lower_bound',r.get('input_tokens',0)) or 0 for r in rows),output_tokens_lower_bound=sum(r.get('output_tokens_lower_bound',r.get('output_tokens',0)) or 0 for r in rows),token_incomplete_cases=sum(r.get('token_telemetry_complete') is False for r in rows),call_telemetry_missing=sum('calls' not in r for r in rows),stops=dict(collections.Counter(r['reason'] for r in rows if not r['complete'])),strata={s:dict(primary=sum(r['stratum']==s for r in primary),exact=sum(r['stratum']==s and r['exact'] is True for r in primary),complete=sum(r['stratum']==s and r['complete'] for r in rows),observations=sum(r['stratum']==s for r in rows)) for s in ('1-4','5-8','9-16')})
def summary():
 rows=[json.loads(s) for s in (OUT/'iteration-7-results.jsonl').read_text().splitlines()]
 if len(rows)!=23:raise ValueError('incomplete_results')
 result={a:aggregate([r for r in rows if r['architecture']==a]) for a in ('A2','baseline','h23')}
 result['canonical_failure_classes']=[]
 for r in rows:
  if r['complete']:continue
  calls=r.get('observed',{}).get('calls',[]);stops=[c.get('stop') for c in calls]
  if 'context_overflow' in r['reason']:kind='observed_host_context_budget'
  elif any('timeout' in str(s) for s in stops) or 'TimeoutExpired' in r['reason']:kind='observed_resource_timeout'
  elif any(c.get('input_tokens',0) and c['input_tokens']+1536>16384 for c in calls):kind='inferred_context_budget'
  elif stops and all(s=='completed' for s in stops):kind='final_invalid_or_unlogged_error_after_completed_backend'
  else:kind='recorded_noncompleted_or_unlogged_error'
  result['canonical_failure_classes'].append(dict(architecture=r['architecture'],name=r['name'],reverse=r.get('reverse'),reason=r['reason'],call_stops=stops,classification=kind))
 result['contract']=dict(excluded_input_rate=3/15,target=.8,population_inference=False,components_correlated=True,stop_pairs='null excluded from pair sums but stop retained as exact failure',cost='audit wall includes temp-index; tokens lower bounds; peak RSS cumulative max')
 result['comparison_4file']=dict(cases=['mock-sentinel','mock-plus-assert-negative'],orders=2,a_exact=0,baseline_exact=sum(r['exact'] for r in rows if r['architecture']=='baseline'),baseline_observations=4)
 result['decision']='A2 semantic candidate NO-GO; target overall and 9-16 not reached; production unchanged. A2 structural fallback limited GO only.'
 with (OUT/'iteration-7-summary-clarified.json').open('x') as f:json.dump(result,f,ensure_ascii=False,indent=2);f.write('\n')
 print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':summary()
