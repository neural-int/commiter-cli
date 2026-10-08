"""A-only and repository share the complete evidence-container contract."""
import json,pathlib,hashlib,subprocess
import inline_evaluate as base

HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[1]/'docs/benchmarks/issue-152'


def payload(r,mode):
 data,gold,wall,_=base.payload(r,'/tmp/issue152-inline-graph','/tmp/issue151-remeasure-symbols','A-only')
 sources=[dict(ID=f['ID'],Path=f['Path'],Content=f['After']) for f in r['Files']]
 unchanged=r.get('Repository',[]) if mode=='repository' else []
 graph=json.loads(subprocess.run(['/tmp/issue152-inline-graph'],input=json.dumps(sources+unchanged).encode(),capture_output=True,check=True).stdout)
 data['test_impact_evidence']=dict(observations=[],unchanged_sources=[],semantics='soft_behavioral_consistency_not_shared_intent; absence_unknown_no_default_merge')
 data['repository_evidence']=dict(unit_relations=base.relations(data['change_units'],graph),syntax_graph=graph,absence='unknown_no_default_merge',unchanged_sources=unchanged,max_hops=2)
 assert len(json.dumps(data).encode())<=base.api.MAX_CONTEXT_BYTES
 return data,gold


def main():
 records=json.loads((HERE/'normalized-fixtures.json').read_text())
 legacy=json.loads(pathlib.Path('/tmp/issue151-remeasure-fixtures.json').read_text())
 names=['same-directory-independent-5','cross-directory-single-intent-9','multiple-intents-boundary-12','shared-callee-independent-6']
 records += [dict(r,Split='regression') for r in legacy if r['Name'] in names]
 helper='/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper'
 model='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'
 with (ROOT/'iteration-11-results.jsonl').open('w') as out:
  for r in records:
   previous={}
   for mode in ('A-only','repository'):
    try:data,gold=payload(r,mode)
    except ValueError as e:
     assert str(e)=='unit_budget'
     row=dict(fixture=r['Name'],split=r['Split'],source=mode,complete=False,exact=None,false_merge=None,false_split=None,calls=0,stop='unit_budget',validation_reason='preflight_rejected')
    else:
     encoded=json.dumps(data,sort_keys=True);digest=hashlib.sha256(encoded.encode()).hexdigest()
     if digest in previous:
      row=dict(previous[digest],source=mode,calls=0,reused_identical_request=True,wall_seconds=0,input_tokens=0,output_tokens=0)
     else:
      membership,meta=base.api.invoke(data,helper,model)
      exact,fm,fs=base.api.quality(membership,gold) if membership else (None,None,None)
      row=dict(fixture=r['Name'],split=r['Split'],source=mode,complete=membership is not None,exact=exact,false_merge=fm,false_split=fs,calls=1,membership=membership,request_sha256=digest,context_bytes=len(encoded),**meta)
      previous[digest]=row
    out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)


if __name__=='__main__':main()
