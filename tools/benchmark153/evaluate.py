"""A-only direct assignment versus relation scoring + exact host partition."""
import json,pathlib,sys,subprocess,time,itertools
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark152'))
import inline_evaluate as base
from partition import partition
ROOT=HERE.parents[1]/'docs/benchmarks/issue-153'
HELPER='/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper'
MODEL='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'
def score(data):
 ids=sorted(u['id'] for u in data['change_units']);pairs=[a+'__'+b for a,b in itertools.combinations(ids,2)]
 schema=dict(type='object',properties=dict(scores=dict(type='object',properties={p:dict(type='integer',enum=[-2,-1,0,1,2]) for p in pairs},required=pairs,additionalProperties=False),unresolved=dict(type='boolean')),required=['scores','unresolved'],additionalProperties=False)
 system='Evaluate whether each pair of changed units serves the same commit purpose. Return +2 strong same intent, +1 weak same, -1 weak independent, -2 strong independent, 0 unknown. Do not produce a partition. Syntax proximity alone does not establish shared purpose. If evidence cannot support evaluation set unresolved true. Output JSON matching this schema: '+json.dumps(schema,sort_keys=True)
 req=dict(schema=schema,messages=[dict(role='system',content=system),dict(role='user',content=json.dumps(data,sort_keys=True))],context_tokens=16384,output_tokens=1536,model=base.api.MODEL+'@'+base.api.REVISION,model_path=MODEL,generation_profile='bounded-routed-grouping')
 start=time.perf_counter();meta=dict(input_tokens=None,output_tokens=None,stop='helper_failure')
 try:
  p=subprocess.run([HELPER],input=(json.dumps(req)+'\n').encode(),capture_output=True,timeout=120)
  if p.returncode:raise ValueError('helper_failure')
  r=json.loads(p.stdout,object_pairs_hook=base.api.strict);meta.update(stop=r.get('stop_reason','unknown'),input_tokens=r.get('benchmark_input_tokens'),output_tokens=r.get('benchmark_output_tokens'))
  if not r.get('ok') or meta['stop']!='completed':raise ValueError('backend_not_completed')
  answer=json.loads(r.get('generated_json',''),object_pairs_hook=base.api.strict)
  if set(answer)!={'scores','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
  if answer['unresolved']:raise ValueError('unresolved')
  if not isinstance(answer['scores'],dict):raise ValueError('invalid_scores')
  meta['scores']=answer['scores']
  m,solver=partition(ids,answer['scores']);meta.update(validation_reason='accepted',solver=solver)
 except subprocess.TimeoutExpired:m=None;meta.update(stop='timeout',validation_reason='timeout')
 except (ValueError,TypeError,KeyError) as e:m=None;meta['validation_reason']=str(e)
 meta['wall_seconds']=time.perf_counter()-start
 return m,meta

def main():
 records=json.loads((HERE.parent/'benchmark152'/'normalized-fixtures.json').read_text())[:2]
 with (ROOT/'iteration-1-results.jsonl').open('w') as out:
  for r in records:
   data,gold,_,_=base.payload(r,'/tmp/issue152-samefile-graph','/tmp/issue151-remeasure-symbols','A-only')
   if not 2<=len(gold)<=8:raise ValueError('unit_budget')
   for mode in ['direct','score_partition']:
    m,meta=base.api.invoke(data,HELPER,MODEL) if mode=='direct' else score(data)
    exact,fm,fs=base.api.quality(m,gold) if m else (None,None,None)
    row=dict(fixture=r['Name'],split='used_qualification',mode=mode,complete=m is not None,exact=exact,false_merge=fm,false_split=fs,membership=m,calls=1,**meta);out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)
if __name__=='__main__':main()
