"""Fixed common-container expression observation comparison."""
import json,pathlib,subprocess
import inline_evaluate as base
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[1]/'docs/benchmarks/issue-152'
def payload(record,mode):
 data,gold,_,_=base.payload(record,'/tmp/issue152-samefile-graph','/tmp/issue151-remeasure-symbols','repository')
 observations=[]
 if mode=='expression':
  files={f['ID']:f for f in record['Files']}
  for u in data['change_units']:
   scopes=json.loads(subprocess.check_output(['/tmp/issue152-flow-expressions'],input=files[u['file']]['After'].encode()))
   a,z=u['new_span'];matches=sorted([s for s in scopes if s['start']<=a and z<=s['end']],key=lambda s:s['end']-s['start'])
   observations.append(dict(unit=u['id'],observation=matches[0] if matches else None,status=matches[0]['status'] if matches else 'unknown'))
 data['expression_evidence']=dict(observations=observations,semantics='soft_syntactic_dependency_not_shared_intent; unknown_or_empty_calls_not_unrelated')
 return data,gold
def main():
 records=json.loads((HERE/'expression-fresh.json').read_text())+[dict(r,Split='regression') for r in json.loads((HERE/'statement-guardrails.json').read_text())]
 with (ROOT/'iteration-20-results.jsonl').open('w') as out:
  for r in records:
   for mode in ('A-only','expression'):
    d,g=payload(r,mode);m,meta=base.api.invoke(d,'/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper','/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8')
    exact,fm,fs=base.api.quality(m,g) if m else (None,None,None)
    row=dict(fixture=r['Name'],split=r['Split'],source=mode,exact=exact,false_merge=fm,false_split=fs,complete=m is not None,membership=m,calls=1,**meta)
    out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)
if __name__=='__main__':main()
