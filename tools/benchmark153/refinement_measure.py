"""Fixed local refinement diagnosis; no final commit grouping claims."""
import json,pathlib,sys,subprocess,time
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark152'))
import inline_evaluate as base
sys.path.insert(0,str(HERE))
import refinement
HELPER='/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper'
MODEL='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'
CASES=[('fresh-independent',{'F1':(b'package x\nfunc MaxUploadMiB() int { return 2 }\nfunc RetryAttempts() int { return 3 }\n',b'package x\nfunc MaxUploadMiB() int { return 4 }\nfunc RetryAttempts() int { return 5 }\n')},2),('fresh-single',{'F1':(b'package x\nfunc RetryAttempts() int { return 3 }\n',b'package x\nfunc RetryAttempts() int { return 5 }\n')},1)]
real={}
for i,p in enumerate(['internal/relation/extract.go','internal/relation/extract_test.go']):real[f'F{i+1}']=(subprocess.check_output(['git','show','fd042894^:'+p]),subprocess.check_output(['git','show','fd042894:'+p]))
CASES.append(('real-fd042894',real,None))
with (HERE.parents[1]/'docs/benchmarks/issue-153/iteration-7-results.jsonl').open('w') as out:
 for name,snap,expected in CASES:
  parents={};payload={'files':[]}
  for fid,(old,new) in snap.items():
   h=refinement.contiguous.extract(old,new,fid);parents.update({p['id']:p for p in h['operations']})
   payload['files'].append(dict(id=fid,before=old.decode(),after=new.decode(),hierarchy=h))
  decision=dict(type='object',properties=dict(action=dict(type='string',enum=['accept','refine','unresolved']),groups=dict(type='array',items=dict(type='array',items=dict(type='string')))),required=['action'],additionalProperties=False)
  schema=dict(type='object',properties={pid:decision for pid in parents},required=list(parents),additionalProperties=False)
  system='For each edit proposal decide whether its changed atoms serve one commit purpose (accept), multiple independent purposes (refine with groups containing every child atom ID exactly once), or cannot be determined (unresolved). Do not group across proposals. For accept or unresolved omit groups. Proximity alone is not shared purpose. Output JSON matching: '+json.dumps(schema)
  req=dict(schema=schema,messages=[dict(role='system',content=system),dict(role='user',content=json.dumps(payload))],context_tokens=16384,output_tokens=1536,model=base.api.MODEL+'@'+base.api.REVISION,model_path=MODEL,generation_profile='bounded-routed-grouping')
  start=time.perf_counter();row=dict(fixture=name,expected_local_units=expected,calls=1,complete=False,local_exact=None,false_merge=None,false_split=None)
  try:
   p=subprocess.run([HELPER],input=(json.dumps(req)+'\n').encode(),capture_output=True,timeout=120)
   if p.returncode:raise ValueError('helper_failure')
   r=json.loads(p.stdout,object_pairs_hook=base.api.strict);row.update(stop=r.get('stop_reason'),input_tokens=r.get('benchmark_input_tokens'),output_tokens=r.get('benchmark_output_tokens'))
   if not r.get('ok') or row['stop']!='completed':raise ValueError('backend_not_completed')
   plan=json.loads(r['generated_json'],object_pairs_hook=base.api.strict);row['plan']=plan
   result=refinement.validate(snap,plan);row.update(complete=True,units=len(result),reason='accepted',local_exact=(len(result)==expected) if expected is not None else None)
   # Authored two-atom local gold only; real-history gold is unassigned.
   if expected is not None:row.update(false_merge=int(expected==2 and len(result)==1),false_split=int(expected==1 and len(result)>1))
  except subprocess.TimeoutExpired:row.update(reason='timeout')
  except (ValueError,TypeError,KeyError) as e:row['reason']=str(e)
  row['wall_seconds']=time.perf_counter()-start;out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)
