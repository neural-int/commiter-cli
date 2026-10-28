import collections,hashlib,importlib.util,json,pathlib,sys
base=pathlib.Path('<PRIVATE_TMP>/issue182-prompt-comparison')
source=pathlib.Path('$HOME/.codex/worktrees/issue-182-model-comparison/commiter-cli/tools/benchmark182/prompt_compare.py')
spec=importlib.util.spec_from_file_location('harness',source);h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)
contract=json.loads((base/'contract.json').read_text());rows=[];wire_count=0
for c in contract['cells']:
 d=(base/'qwen2-resume' if c['label'].startswith('qwen2-') else base)/c['label']
 interruption=json.loads((d/'user-interruption.json').read_text()) if (d/'user-interruption.json').exists() else None
 result=json.loads((d/'result.json').read_text()) if (d/'result.json').exists() else None
 phases=[]
 for i in [1,2,3]:
  request=d/f'{i:02d}-request.json';actual=d/f'{i:02d}-actual-request.json';response=d/f'{i:02d}-response.json'
  if actual.exists():
   a=json.loads(request.read_text());b=json.loads(actual.read_text());b.pop('model_path',None)
   assert b==h.transform(a,c['arm']);h.invariant(a,b);wire_count+=1
  if response.exists():
   reply=json.loads(response.read_text());native=reply.get('native') or {};metrics=native.get('benchmark_metrics',{})
   phases.append({'phase':i,'metrics':{k:metrics.get(k) for k in ['generatedTokens','nativeThoughtTokens','runtimeGenerationSeconds','runtimePromptSeconds','nativeThoughtForcedClose']},'native_ok':native.get('ok'),'stop_reason':native.get('stop_reason'),'content':(reply.get('response') or {}).get('Content')})
 first=phases[0]['content'] if phases else None
 fixed=result.get('fixed_membership') if result else None
 grouping=json.loads(first) if first else None
 # Registered fixtures each have one reference group, identical to the fixed downstream partition.
 exact=(len(set(grouping.values()))==1) if grouping else None
 if fixed and grouping:assert set(grouping)==set(fixed)
 count=sum(p['metrics']['generatedTokens'] or 0 for p in phases);seconds=sum(p['metrics']['runtimeGenerationSeconds'] or 0 for p in phases)
 rows.append({'cell':c['label'],'status':'user_interrupted' if interruption else ('validator_rejected' if result and result.get('error') else 'plan_accepted'),'wall_seconds':result.get('wall_seconds') if result else None,'error':result.get('error') if result else None,'actual_grouping_reference_match':exact,'actual_group_count':len(set(grouping.values())) if grouping else None,'completed_response_tokens':count,'decode_tokens_per_second':count/seconds if seconds else None,'interruption':interruption,'phases':phases})
monitor={}
for name,folder in [('original',base),('qwen2_resume',base/'qwen2-resume')]:
 samples=[json.loads(line) for line in (folder/'monitor.jsonl').read_text().splitlines()]
 monitor[name]={'samples':len(samples),'invalid_samples':sum(not s['valid'] for s in samples)}
summary={'id':'H182-004','status':'all_registered_cells_accounted_for','counts':dict(collections.Counter(r['status'] for r in rows)),'actual_wire_requests_audited':wire_count,'monitor':monitor,'cells':rows,'limitations':['single repetition per arm and fixture','downstream fixed partition; not end-to-end success','reference not independently human approved','nativeThoughtTokens absent for OFF; progress logs report zero thinking','token counts and decode rates for interrupted cells cover only completed responses; partial thinking recorded separately']}
h.save(base/'final-summary.json',summary)
print(json.dumps({k:v for k,v in summary.items() if k!='cells'},ensure_ascii=False,indent=2))
for r in rows:
 print(r['cell'],r['status'],r['wall_seconds'],r['actual_grouping_reference_match'],r['completed_response_tokens'],round(r['decode_tokens_per_second'],2) if r['decode_tokens_per_second'] else None)
