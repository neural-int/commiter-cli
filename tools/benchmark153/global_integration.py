"""Replay validated refinement, then one fixed global-score call."""
import pathlib,sys,json,subprocess,time,itertools
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark152'))
import inline_evaluate as base
sys.path.insert(0,str(HERE));import refinement,payload_budget
from partition import partition
from evaluate import reconstruct
f=json.loads((HERE/'behavior-fixture.json').read_text());old,new=f['before'].encode(),f['after'].encode();prior=json.loads((HERE.parents[1]/'docs/benchmarks/issue-153/iteration-14-results.json').read_text())
plan={prior['groups'][0]['parent']:dict(action='refine',groups=[g['atoms'] for g in prior['groups']])};groups=refinement.validate({'F1':(old,new)},plan);atoms=refinement.contiguous.extract(old,new,'F1')['atoms']
data=dict(change_units=[dict(id=f'U{i+1:03}',file=g['file'],before=old.decode(),after=reconstruct(old,atoms,g['atoms']).decode()) for i,g in enumerate(groups)],selected_file_context=[dict(id='F1',before=old.decode(),after=new.decode())])
ids=sorted(u['id'] for u in data['change_units']);pairs=[a+'__'+b for a,b in itertools.combinations(ids,2)]
schema=dict(type='object',properties=dict(scores=dict(type='object',properties={p:dict(type='integer',enum=[-2,-1,0,1,2]) for p in pairs},required=pairs,additionalProperties=False),unresolved=dict(type='boolean')),required=['scores','unresolved'],additionalProperties=False)
system='Evaluate whether each pair of changed units serves the same commit purpose. Return +2 strong same intent, +1 weak same, -1 weak independent, -2 strong independent, 0 unknown. Do not produce a partition. Syntax proximity alone does not establish shared purpose. If evidence cannot support evaluation set unresolved true. Output JSON matching this schema: '+json.dumps(schema,sort_keys=True)
messages=[dict(role='system',content=system),dict(role='user',content=json.dumps(data,sort_keys=True))];size=len(json.dumps(messages,ensure_ascii=False).encode());assert size<=8192
req=dict(schema=schema,messages=messages,context_tokens=16384,output_tokens=1536,model=base.api.MODEL+'@'+base.api.REVISION,model_path='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8',generation_profile='bounded-routed-grammar')
row=dict(fixture=f['name'],split='used_integration',calls=1,cached_refinement_calls=2,complete=False,exact=None,false_merge=None,false_split=None,message_bytes=size);t=time.perf_counter()
try:
 p=subprocess.run(['/tmp/issue153-bounded-grammar-helper/.build/release/commiter-mlx-helper'],input=(json.dumps(req)+'\n').encode(),capture_output=True,timeout=120)
 if p.returncode:raise ValueError('helper_failure')
 r=json.loads(p.stdout,object_pairs_hook=base.api.strict);row.update(stop=r.get('stop_reason'),input_tokens=r.get('benchmark_input_tokens'),output_tokens=r.get('benchmark_output_tokens'))
 if not r.get('ok') or row['stop']!='completed':raise ValueError('backend_not_completed')
 answer=json.loads(r['generated_json'],object_pairs_hook=base.api.strict);row['answer']=answer
 if set(answer)!={'scores','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
 if answer['unresolved']:raise ValueError('unresolved')
 membership,meta=partition(ids,answer['scores']);row.update(membership=membership,solver=meta,complete=True,exact=membership[ids[0]]!=membership[ids[1]],false_merge=int(membership[ids[0]]==membership[ids[1]]),false_split=0,reason='accepted')
except subprocess.TimeoutExpired:row['reason']='timeout'
except (ValueError,TypeError,KeyError) as e:row['reason']=str(e)
row['wall_seconds']=time.perf_counter()-t
(HERE.parents[1]/'docs/benchmarks/issue-153/iteration-15-results.json').write_text(json.dumps(row,indent=2)+'\n');print(json.dumps(row))
