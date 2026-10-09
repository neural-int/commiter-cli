import json,pathlib,statistics
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def summary():
 fixed=json.loads((OUT/'iteration-6-preregistered.json').read_text());rows=[json.loads(v) for v in (OUT/'iteration-6-results.jsonl').read_text().splitlines()]
 if len(rows)!=20:raise ValueError('incomplete_observations')
 output={}
 for arch in ('A2+C','B'):
  allrows=[r for r in rows if r['architecture']==arch];primary=[r for r in allrows if r['primary']]
  raw=[r for r in primary if 'raw_quality' in r];calls=[c for r in allrows for c in r['calls']];walls=sorted(r['wall_seconds'] for r in allrows)
  output[arch]=dict(observations=len(allrows),primary=len(primary),excluded_gold=len(allrows)-len(primary),final_exact=sum(r['exact'] for r in primary),baseline_exact=sum(r['baseline']['exact'] for r in primary),complete=sum(r['complete'] for r in allrows),fm=sum(r['false_merge'] for r in primary),fs=sum(r['false_split'] for r in primary),baseline_fm=sum(r['baseline']['false_merge'] for r in primary),baseline_fs=sum(r['baseline']['false_split'] for r in primary),raw_completed=len(raw),raw_exact=sum(r['raw_quality']['exact'] for r in raw),raw_fm=sum(r['raw_quality']['false_merge'] for r in raw),raw_fs=sum(r['raw_quality']['false_split'] for r in raw),raw_authoritative_rejections=sum(not r['raw_validation']['valid'] for r in raw),backend_calls=sum(r['backend_calls'] for r in allrows),recorded_call_responses=len(calls),call_response_missing=sum(r['backend_calls'] for r in allrows)-len(calls),input_tokens_lower_bound=sum(c['response'].get('benchmark_input_tokens') or 0 for c in calls),output_tokens_lower_bound=sum(c['response'].get('benchmark_output_tokens') or 0 for c in calls),wall_total=sum(walls),wall_mean=statistics.mean(walls),wall_p95=walls[int((len(walls)-1)*.95)],refined_observations=sum(bool(r['refinement']) for r in allrows),model_failure_fallback=sum(r['reason']=='model_failure_fallback' for r in allrows),partial_contract_fallback=sum(r['reason']=='partial_file_contract_fallback' for r in allrows))
 mixed=[]
 for spec in fixed['rows']:
  c=spec['case']
  if not c['primary']:continue
  by_file={f['id']:{c['gold'][u['id']] for u in __import__('planner').prepare(c['files']) if u['file']==f['id']} for f in c['files']}
  r=next(r for r in rows if r['architecture']=='A2+C' and r['name']==c['name'] and r['reverse']==spec['reverse'])
  if 'coarse_answer' in r:
   for fid,actual in by_file.items():mixed.append(dict(name=c['name'],reverse=r['reverse'],file=fid,gold_mixed=len(actual)>1,status=r['coarse_answer']['status'][fid]))
 output['mixed_classification']=dict(observed_files=len(mixed),gold_mixed=sum(r['gold_mixed'] for r in mixed),mixed_true_positive=sum(r['gold_mixed'] and r['status']=='mixed' for r in mixed),mixed_false_negative=sum(r['gold_mixed'] and r['status']!='mixed' for r in mixed),false_mixed=sum(not r['gold_mixed'] and r['status']=='mixed' for r in mixed),unknown=sum(r['status']=='unknown' for r in mixed),rows=mixed)
 output['judgment']='C NO-GO: final raw split contract incompatible and no final mixed-file improvement; any control improvement is reported separately; B fixed remains NO-GO. Do not use gold to fix assignments.'
 with (OUT/'iteration-6-summary.json').open('x') as f:json.dump(output,f,ensure_ascii=False,indent=2);f.write('\n')
 print(json.dumps({k:v for k,v in output.items() if k!='mixed_classification'},ensure_ascii=False))
if __name__=='__main__':summary()
