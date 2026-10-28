"""Aggregate H182-005 without additional inference or changing raw evidence."""
import importlib.util,json,pathlib
source=pathlib.Path('$HOME/.codex/worktrees/issue-182-model-comparison/commiter-cli/tools/benchmark182/prompt_compare.py')
spec=importlib.util.spec_from_file_location('harness',source);h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)
p=pathlib.Path('<PRIVATE_TMP>/issue182-thinking-budget');baseline=pathlib.Path('<PRIVATE_TMP>/issue182-prompt-comparison')
contract=json.loads((p/'contract.json').read_text());before={r['cell']:r for r in json.loads((baseline/'final-summary.json').read_text())['cells']}
results=json.loads((p/'results.json').read_text());out=[];wire_count=0
for row in results:
 c=next(c for c in contract['cells'] if c['label']==row['cell']);d=p/c['label'];phases=[]
 for i in [1,2,3]:
  actual=d/f'{i:02d}-actual-request.json';request=d/f'{i:02d}-request.json';response=d/f'{i:02d}-response.json'
  if actual.exists():
   a=json.loads(request.read_text());b=json.loads(actual.read_text());b.pop('model_path',None)
   assert b==h.transform(a,c['arm']);h.invariant(a,b);wire_count+=1
  if response.exists():
   r=json.loads(response.read_text());native=r.get('native') or {};m=native.get('benchmark_metrics',{})
   thought=m.get('nativeThoughtTokens');assert thought is None or thought<=1024,(c['label'],thought)
   phases.append({'phase':i,'content':(r.get('response') or {}).get('Content'),'ok':native.get('ok'),'stop_reason':native.get('stop_reason'),'metrics':{k:m.get(k) for k in ['generatedTokens','nativeThoughtTokens','nativeThoughtForcedClose','runtimeGenerationSeconds','runtimePromptSeconds']}})
 old=before[c['label'].removesuffix('-thinking1024')]
 first=phases[0]['content'] if phases else None;groups=json.loads(first) if first else None
 count=sum(r['metrics']['generatedTokens'] or 0 for r in phases);seconds=sum(r['metrics']['runtimeGenerationSeconds'] or 0 for r in phases)
 out.append({'cell':c['label'],'error':row['result'].get('error'),'wall_seconds':row['result'].get('wall_seconds'),'actual_grouping_reference_match':len(set(groups.values()))==1 if groups else None,'output_tokens':count,'thought_tokens':sum(r['metrics']['nativeThoughtTokens'] or 0 for r in phases),'forced_closes':sum(r['metrics']['nativeThoughtForcedClose'] is True for r in phases),'decode_tokens_per_second':count/seconds if seconds else None,'phases':phases,'unlimited_reference':old})
samples=[json.loads(line) for line in (p/'monitor.jsonl').read_text().splitlines()]
summary={'id':contract['id'],'cells':out,'audited_requests':wire_count,'monitor_samples':len(samples),'invalid_monitor_samples':sum(not s['valid'] for s in samples),'completion':json.loads((p/'completion.json').read_text()) if (p/'completion.json').exists() else None,'limitations':contract['limitations']}
h.save(p/'summary.json',summary)
for r in out:print(r['cell'],r['error'],r['wall_seconds'],'G=',r['actual_grouping_reference_match'],'tokens=',r['output_tokens'],'thought=',r['thought_tokens'],'forced=',r['forced_closes'])
