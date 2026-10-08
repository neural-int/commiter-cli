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
f=json.loads((HERE/'behavior-fixture.json').read_text());old,new=f['before'].encode(),f['after'].encode();snap={'F1':(old,new)};h=refinement.contiguous.extract(old,new,'F1');parent=h['operations'][0]
row=dict(fixture=f['name'],split='independent_synthetic',complete=False,exact=None,false_merge=None,false_split=None,stages=[])
try:
 messages,schema,mapping,_=payload_budget.build(snap);answer,meta=invoke(messages,schema);row['stages'].append(dict(stage='initial',answer=answer,**meta))
 if answer is None:raise ValueError('backend_not_completed')
 decisions=payload_budget.validate_decisions(answer,mapping)
 if decisions[parent['id']]=='accept':plan={parent['id']:dict(action='accept')}
 else:
  aliases={f'A{i+1:03}':a['id'] for i,a in enumerate(h['atoms'])}
  payload=dict(before=old.decode(),after=new.decode(),edits=[dict(id=k,old_span=a['old_span'],new_span=a['new_span'],before=base64.b64decode(a['before']).decode(),after=base64.b64decode(a['after']).decode()) for k,a in zip(aliases,h['atoms'])])
  schema=dict(type='object',properties=dict(groups=dict(type='array',items=dict(type='array',items=dict(type='string',enum=list(aliases)))),unresolved=dict(type='boolean')),required=['groups','unresolved'],additionalProperties=False)
  messages=[dict(role='system',content='Partition these edited atoms by commit purpose within this proposal. Include every edit ID exactly once. Independent purposes require separate groups; proximity alone is insufficient. If uncertain set unresolved true. Output JSON matching: '+json.dumps(schema)),dict(role='user',content=json.dumps(payload))]
  answer,meta=invoke(messages,schema);row['stages'].append(dict(stage='detail',answer=answer,**meta))
  if answer is None:raise ValueError('backend_not_completed')
  if set(answer)!={'groups','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
  if answer['unresolved']:raise ValueError('unresolved')
  groups=[[aliases[k] for k in group] for group in answer['groups']];flat=[a for g in groups for a in g]
  if len(flat)!=len(set(flat)) or set(flat)!=set(aliases.values()):raise ValueError('atom_coverage')
  plan={parent['id']:dict(action='accept')} if len(groups)==1 else {parent['id']:dict(action='refine',groups=groups)}
 result=refinement.validate(snap,plan)
 actual={a:i for i,g in enumerate(result) for a in g['atoms']}
 targets=[s for s in f['expected_states'] if s['a']!=s['b']];gold={}
 for i,state in enumerate(targets):
  witnesses=remeasure.matching_subsets(old,h['atoms'],state['content'].encode());assert len(witnesses)==1
  gold.update({a:i for a in witnesses[0]})
 assert set(actual)==set(gold)
 fm=sum(actual[a]==actual[b] and gold[a]!=gold[b] for a,b in itertools.combinations(gold,2));fs=sum(actual[a]!=actual[b] and gold[a]==gold[b] for a,b in itertools.combinations(gold,2))
 row.update(complete=True,exact=fm==fs==0,false_merge=fm,false_split=fs,units=len(result),groups=result,reason='accepted')
except subprocess.TimeoutExpired:row['reason']='timeout'
except (ValueError,KeyError,TypeError) as e:row['reason']=str(e)
row['calls']=len(row['stages'])
(HERE.parents[1]/'docs/benchmarks/issue-153/iteration-14-results.json').write_text(json.dumps(row,indent=2)+'\n');print(json.dumps(row))
