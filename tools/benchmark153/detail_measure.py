"""Local detail-stage diagnosis; gold remains outside model messages."""
import pathlib,sys,json,subprocess,time,base64
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark152'))
import inline_evaluate as base
sys.path.insert(0,str(HERE))
import refinement,payload_budget
HELPER='/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper'
MODEL='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'
cases=[('used-wire',b'package x\nfunc MaxUploadMiB() int { return 2 }\nfunc RetryAttempts() int { return 3 }\n',b'package x\nfunc MaxUploadMiB() int { return 4 }\nfunc RetryAttempts() int { return 5 }\n'),('fresh-boundary',b'package x\nfunc SessionMinutes() int { return 6 }\nfunc WorkerSlots() int { return 2 }\n',b'package x\nfunc SessionMinutes() int { return 8 }\nfunc WorkerSlots() int { return 3 }\n')]
with (HERE.parents[1]/'docs/benchmarks/issue-153/iteration-10-results.jsonl').open('w') as out:
 for name,old,new in cases:
  snap={'F1':(old,new)};h=refinement.contiguous.extract(old,new,'F1');assert len(h['operations'])==1
  parent=h['operations'][0];mapping={f'A{i+1:03}':a['id'] for i,a in enumerate(h['atoms'])}
  payload=dict(before=old.decode(),after=new.decode(),edits=[dict(id=k,old_span=a['old_span'],new_span=a['new_span'],before=base64.b64decode(a['before']).decode(),after=base64.b64decode(a['after']).decode()) for k,a in zip(mapping,h['atoms'])])
  schema=dict(type='object',properties=dict(groups=dict(type='array',items=dict(type='array',items=dict(type='string',enum=list(mapping)))),unresolved=dict(type='boolean')),required=['groups','unresolved'],additionalProperties=False)
  messages=[dict(role='system',content='Partition these edited atoms by commit purpose within this proposal. Include every edit ID exactly once. Independent purposes require separate groups; proximity alone is insufficient. If uncertain set unresolved true. Output JSON matching: '+json.dumps(schema)),dict(role='user',content=json.dumps(payload))]
  size=len(json.dumps(messages,ensure_ascii=False).encode());assert size<=payload_budget.MAX_MESSAGE_BYTES
  req=dict(schema=schema,messages=messages,context_tokens=16384,output_tokens=1536,model=base.api.MODEL+'@'+base.api.REVISION,model_path=MODEL,generation_profile='bounded-routed-grouping')
  row=dict(fixture=name,split='used_wire' if name.startswith('used') else 'independent_synthetic_detail_only',calls=1,complete=False,exact=None,false_merge=None,false_split=None,message_bytes=size);start=time.perf_counter()
  try:
   p=subprocess.run([HELPER],input=(json.dumps(req)+'\n').encode(),capture_output=True,timeout=120)
   if p.returncode:raise ValueError('helper_failure')
   r=json.loads(p.stdout,object_pairs_hook=base.api.strict);row.update(stop=r.get('stop_reason'),input_tokens=r.get('benchmark_input_tokens'),output_tokens=r.get('benchmark_output_tokens'))
   if not r.get('ok') or row['stop']!='completed':raise ValueError('backend_not_completed')
   answer=json.loads(r['generated_json'],object_pairs_hook=base.api.strict);row['answer']=answer
   if set(answer)!={'groups','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
   if answer['unresolved']:raise ValueError('unresolved')
   groups=[[mapping[k] for k in group] for group in answer['groups']]
   plan={parent['id']:dict(action='accept')} if len(groups)==1 and groups[0]==list(mapping.values()) else {parent['id']:dict(action='refine',groups=groups)}
   if len(groups)==1 and (len(groups[0])!=len(set(groups[0])) or set(groups[0])!=set(mapping.values())):raise ValueError('atom_coverage')
   result=refinement.validate(snap,plan);row.update(complete=True,exact=len(result)==2,false_merge=int(len(result)==1),false_split=0,reason='accepted')
  except subprocess.TimeoutExpired:row['reason']='timeout'
  except (ValueError,KeyError,TypeError) as e:row['reason']=str(e)
  row['wall_seconds']=time.perf_counter()-start;out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)
