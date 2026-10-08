"""Fixed two-stage local refinement evaluation, never final global grouping."""
import pathlib,sys,json,base64,subprocess,time,itertools
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark152'))
import inline_evaluate as base
sys.path.insert(0,str(HERE))
import refinement,payload_budget
import remeasure
HELPER='/tmp/issue153-bounded-grammar-helper/.build/release/commiter-mlx-helper'
MODEL='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'
def invoke(messages,schema):
 size=len(json.dumps(messages,ensure_ascii=False).encode())
 if size>8192:raise ValueError('message_byte_budget')
 req=dict(schema=schema,messages=messages,context_tokens=16384,output_tokens=1536,model=base.api.MODEL+'@'+base.api.REVISION,model_path=MODEL,generation_profile='bounded-routed-grammar')
 t=time.perf_counter();p=subprocess.run([HELPER],input=(json.dumps(req)+'\n').encode(),capture_output=True,timeout=120)
 if p.returncode:raise ValueError('helper_failure')
 r=json.loads(p.stdout,object_pairs_hook=base.api.strict)
 meta=dict(stop=r.get('stop_reason'),input_tokens=r.get('benchmark_input_tokens'),output_tokens=r.get('benchmark_output_tokens'),wall_seconds=time.perf_counter()-t,message_bytes=size)
 if not r.get('ok') or meta['stop']!='completed':return None,meta
 return json.loads(r['generated_json'],object_pairs_hook=base.api.strict),meta

from partition import partition
f=json.loads((HERE/'multifile-fixture.json').read_text());files=f['files'][:4];snap={x['id']:(x['before'].encode(),x['after'].encode()) for x in files}
row=dict(fixture=f['name'],split='used_projection_comparison',complete=False,exact=None,false_merge=None,false_split=None,stages=[])
output=HERE.parents[1]/'docs/benchmarks/issue-153/iteration-19-results.json'
def save():output.write_text(json.dumps(row,indent=2)+'\n')
def call(stage,messages,schema):
 answer,meta=invoke(messages,schema);row['stages'].append(dict(stage=stage,answer=answer,**meta));save();print(json.dumps(row['stages'][-1]),flush=True)
 if answer is None:raise ValueError('backend_not_completed')
 return answer
try:
 messages,schema,mapping,_=payload_budget.build(snap);decisions=payload_budget.validate_decisions(call('initial',messages,schema),mapping)
 hs={fid:refinement.contiguous.extract(*pair,fid) for fid,pair in snap.items()};parents={p['id']:p for h in hs.values() for p in h['operations']};plan={}
 for pid,action in decisions.items():
  if action=='accept':plan[pid]=dict(action='accept');continue
  parent=parents[pid];atoms=[a for a in hs[parent['file']]['atoms'] if a['id'] in parent['children']];aliases={f'A{i+1:03}':a['id'] for i,a in enumerate(atoms)};old,new=snap[parent['file']]
  payload=dict(before=old.decode(),after=new.decode(),edits=[dict(id=k,old_span=a['old_span'],new_span=a['new_span'],before=base64.b64decode(a['before']).decode(),after=base64.b64decode(a['after']).decode()) for k,a in zip(aliases,atoms)])
  schema=dict(type='object',properties=dict(groups=dict(type='array',items=dict(type='array',items=dict(type='string',enum=list(aliases)))),unresolved=dict(type='boolean')),required=['groups','unresolved'],additionalProperties=False)
  messages=[dict(role='system',content='Partition these edited atoms by commit purpose within this proposal. Include every edit ID exactly once. Independent purposes require separate groups; proximity alone is insufficient. If uncertain set unresolved true. Output JSON matching: '+json.dumps(schema)),dict(role='user',content=json.dumps(payload))]
  answer=call('detail:'+parent['file'],messages,schema)
  if set(answer)!={'groups','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
  if answer['unresolved']:raise ValueError('unresolved')
  groups=[[aliases[k] for k in g] for g in answer['groups']];flat=[a for g in groups for a in g]
  if len(flat)!=len(set(flat)) or set(flat)!=set(aliases.values()):raise ValueError('atom_coverage')
  plan[pid]=dict(action='accept') if len(groups)==1 else dict(action='refine',groups=groups)
 units=refinement.validate(snap,plan);row['refined_units']=units;save()
 data=dict(change_units=[dict(id=f'U{i+1:03}',file=u['file'],before=snap[u['file']][0].decode(),after=refinement.reconstruct(snap[u['file']][0],hs[u['file']]['atoms'],u['atoms']).decode()) for i,u in enumerate(units)],selected_file_context=[dict(id=x['id'],path=x['path'],before=x['before'],after=x['after']) for x in files])
 ids=[u['id'] for u in data['change_units']];pairs=[a+'__'+b for a,b in itertools.combinations(sorted(ids),2)]
 schema=dict(type='object',properties=dict(scores=dict(type='object',properties={p:dict(type='integer',enum=[-2,-1,0,1,2]) for p in pairs},required=pairs,additionalProperties=False),unresolved=dict(type='boolean')),required=['scores','unresolved'],additionalProperties=False)
 system='Evaluate whether each pair of changed units serves the same commit purpose. Return +2 strong same intent, +1 weak same, -1 weak independent, -2 strong independent, 0 unknown. Do not produce a partition. Syntax proximity alone does not establish shared purpose. If evidence cannot support evaluation set unresolved true. Output JSON matching this schema: '+json.dumps(schema,sort_keys=True)
 answer=call('score',[dict(role='system',content=system),dict(role='user',content=json.dumps(data,sort_keys=True))],schema)
 if set(answer)!={'scores','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
 if answer['unresolved']:raise ValueError('unresolved')
 membership,solver=partition(ids,answer['scores']);actual={a:membership[f'U{i+1:03}'] for i,u in enumerate(units) for a in u['atoms']};gold={a['id']:x['eval_intent'] for x in files for a in hs[x['id']]['atoms']};assert set(actual)==set(gold)
 fm=sum(actual[a]==actual[b] and gold[a]!=gold[b] for a,b in itertools.combinations(gold,2));fs=sum(actual[a]!=actual[b] and gold[a]==gold[b] for a,b in itertools.combinations(gold,2))
 row.update(complete=True,exact=fm==fs==0,false_merge=fm,false_split=fs,membership=membership,solver=solver,reason='accepted')
 file_members={u['file']:membership[f'U{i+1:03}'] for i,u in enumerate(units)}
 if len(units)==len(files) and len(file_members)==len(files):
  file_gold={x['id']:x['eval_intent'] for x in files}
  file_fm=sum(file_members[a]==file_members[b] and file_gold[a]!=file_gold[b] for a,b in itertools.combinations(file_gold,2))
  file_fs=sum(file_members[a]!=file_members[b] and file_gold[a]==file_gold[b] for a,b in itertools.combinations(file_gold,2))
  row.update(file_membership=file_members,file_false_merge=file_fm,file_false_split=file_fs,file_exact=file_fm==file_fs==0)

except subprocess.TimeoutExpired:row['reason']='timeout'
except (ValueError,KeyError,TypeError) as e:row['reason']=str(e)
row['calls']=len(row['stages']);save();print(json.dumps(row),flush=True)
