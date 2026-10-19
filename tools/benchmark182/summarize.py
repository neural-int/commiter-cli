"""Aggregate the frozen study without inference or rewriting raw evidence."""
import argparse,datetime,json,pathlib,re,statistics

def values(rows,key):
 return [m[key] for r in rows for m in r.get('metrics',[]) if m.get(key) is not None]
def stats(v):return {'n':len(v),'median':statistics.median(v) if v else None,'min':min(v) if v else None,'max':max(v) if v else None}
def summarize(out):
 rows=json.loads((out/'results.json').read_text());contract=json.loads((out/'execution-contract.json').read_text());result=[]
 monitor=[json.loads(line) for line in (out/'monitor.jsonl').read_text().splitlines()]
 reference=json.loads((out/'fixture-reference.json').read_text())
 def grouping(row):
  response=out/row['label']/'01-response.json'
  if not response.exists():return None
  native=json.loads(response.read_text())['native']
  if not native.get('ok'):return None
  membership=json.loads(native['generated_json']);groups={}
  for path,label in membership.items():groups.setdefault(label,[]).append(path.removeprefix('path:'))
  expected=[frozenset(g['files']) for g in reference['groups']]
  actual=[frozenset(files) for files in groups.values()]
  return {'exact_groups':sum(files in expected for files in actual),'reference_groups':len(expected),'predicted_groups':len(actual)}
 def resources(row):
  folder=out/row['label'];lo=(folder/'input.json').stat().st_mtime;hi=(folder/'result.json').stat().st_mtime
  selected=[r for r in monitor if lo<=datetime.datetime.fromisoformat(r['utc']).timestamp()<=hi]
  gpu=[d['Device Utilization %'] for r in selected for d in r.get('gpu',[]) if 'Device Utilization %' in d]
  rss=[d['rss_kib'] for r in selected for d in r.get('helper_processes',[])]
  memory=[float(v.group(1)) for r in selected if (v:=re.search(r'free percentage: ([0-9.]+)%',r.get('pressure',{}).get('stdout','')))]
  return {'samples':len(selected),'system_gpu_percent':stats(gpu),'helper_peak_sampled_rss_kib':max(rss,default=None),'system_free_percentage':stats(memory)}
 for model in contract['models']:
  runs=[r for r in rows if r['model']==model['repo'] and not r['warmup']]
  accepted=[r for r in runs if not r['error']]
  phase={}
  for i,name in enumerate(['grouping','category','text'],1):
   replies=[]
   for r in runs:
    path=out/r['label']/f'{i:02d}-response.json'
    if path.exists():replies.append(json.loads(path.read_text())['native'])
   phase[name]={'attempted':sum((out/r['label']/f'{i:02d}-request.json').exists() for r in runs),'responses':len(replies),'completed':sum(r.get('ok',False) and r.get('stop_reason')=='completed' for r in replies)}
  exact=sum(r['score']['exact_groups'] for r in runs);den=sum(r['score']['reference_groups'] for r in runs)
  entry={'model':model['repo'],'revision':model['revision'],'measured_cycles':len(runs),'accepted_cycles':len(accepted),'acceptance_rate':len(accepted)/len(runs) if runs else None,'exact_group_recall_end_to_end':exact/den if den else None,'matched_groups':exact,'reference_groups':den,'category_correct_on_exact_groups':sum(r['score']['category_correct_on_exact_groups'] for r in runs),'category_denominator':sum(r['score']['category_denominator'] for r in runs),'fallback_count':0,'all_attempt_latency_seconds':stats([r['wall_seconds'] for r in runs]),'accepted_latency_seconds':stats([r['wall_seconds'] for r in accepted]),'phase':phase,'runtime_prompt_seconds':stats(values(runs,'runtimePromptSeconds')),'runtime_decode_seconds':stats(values(runs,'runtimeGenerationSeconds')),'input_tokens':stats(values(runs,'inputTokens')),'output_tokens':stats(values(runs,'generatedTokens')),'mlx_peak_active_bytes':max(values(runs,'peakMemoryBytes'),default=None),'native_thought_forced_closes':sum(values(runs,'nativeThoughtForcedClose')),'native_thought_tokens':stats(values(runs,'nativeThoughtTokens')),'kv_array_bytes':stats([sum(layer['bytes'] for layer in m.get('phaseEndKVLayers',[])) for r in runs for m in r.get('metrics',[]) if m.get('phaseEndKVLayers')]),'errors':[r['error'] for r in runs if r['error']]}
  observed=[v for r in runs if (v:=grouping(r)) is not None]
  entry['raw_grouping']={'completed':len(observed),'exact_group_recall_when_completed':sum(r['exact_groups'] for r in observed)/sum(r['reference_groups'] for r in observed) if observed else None,'predicted_group_counts':[r['predicted_groups'] for r in observed]}
  entry['resources']=[resources(r) for r in runs]
  result.append(entry)
 completion=json.loads((out/'completion.json').read_text()) if (out/'completion.json').exists() else None
 return {'harness_id':contract['harness_id'],'models':result,'completion':completion,'limitations':['one authored 16-file fixture, two measured repeats, agent reference without independent approval','120s timeout is shared across all three phases, not per-phase','runtimePromptSeconds includes first-token work and is not pure prefill','Swift Debug build, uncontrolled background workload','MLX active, cache-array bytes and OS RSS are different scopes and must not be summed','monitor collection duration is measured; causal latency overhead is not separately measured']}
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('study',type=pathlib.Path);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args();a.output.write_text(json.dumps(summarize(a.study),ensure_ascii=False,indent=2)+'\n')
