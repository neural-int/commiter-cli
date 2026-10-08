"""Fixed model comparison; observations contain no gold or author requirements."""
import importlib.util,json,pathlib,sys,subprocess,time,hashlib,argparse
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE))
spec=importlib.util.spec_from_file_location('api152',HERE/'evaluate.py');api=importlib.util.module_from_spec(spec);spec.loader.exec_module(api)
from inline import extract


def main():
 parser=argparse.ArgumentParser();parser.add_argument('--modes',nargs='+',choices=['A-only','impact','context-only','metadata-only'],default=['A-only','impact']);parser.add_argument('--output',default='iteration-8-results.jsonl');args=parser.parse_args()
 root=HERE.parents[1]/'docs/benchmarks/issue-152'
 records=json.loads((HERE/'impact-ablation.json').read_text())
 probes=json.loads((root/'iteration-7-results.json').read_text())+json.loads((root/'iteration-8-probes.json').read_text())
 helper='/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper'
 model='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'
 with (root/args.output).open('w') as output:
  for r in records:
   units=[];gold={};aliases={}
   for f in r['Files']:
    symbols=json.loads(subprocess.run(['/tmp/issue151-remeasure-symbols'],input=f['After'].encode(),capture_output=True,check=True).stdout)
    for u in extract(f['Before'].encode(),f['After'].encode(),f['ID'],symbols):
     uid=f'U{len(units)+1:03}';aliases[u['id']]=uid
     units.append(dict(id=uid,file=f['ID'],old_span=u['old_span'],new_span=u['new_span'],symbols=u['symbols'],before=u['before'],after=u['after']))
     groups=set()
     for g,edits in enumerate(r['IntentEdits']):
      for fid,old,new in edits:
       if fid==f['ID']:
        start=f['Before'].index(old);end=start+len(old)
        if start<u['old_span'][1] and u['old_span'][0]<end:groups.add(g)
     assert len(groups)==1;gold[uid]=groups.pop()
   assert 1<=len(units)<=64
   observations=[dict(units=[aliases[x] for x in p['selected_unit_ids']],returncode=p['returncode'],failed_tests=p['failed_tests'],assertion_locations=p['assertion_locations'],unknown=p['unknown']) for p in probes if p['fixture']==r['Name']]
   assert observations
   for mode in args.modes:
    data=dict(change_units=units,selected_file_context=[dict(id=f['ID'],path=f['Path'],before=f['Before'],after=f['After']) for f in r['Files']])
    if mode in ('impact','context-only','metadata-only'):data['test_impact_evidence']=dict(observations=observations if mode=='impact' else [],unchanged_sources=r['Repository'] if mode!='metadata-only' else [],semantics='soft_behavioral_consistency_not_shared_intent; absence_unknown_no_default_merge')
    assert 'IntentEdits' not in json.dumps(data) and 'Requirements' not in json.dumps(data)
    m,meta=api.invoke(data,helper,model);exact,fm,fs=api.quality(m,gold) if m else (None,None,None)
    row=dict(fixture=r['Name'],split=r['Split'],source=mode,units=len(units),exact=exact,false_merge=fm,false_split=fs,complete=m is not None,unresolved=m is None,calls=1,membership=m,context_bytes=len(json.dumps(data).encode()),**meta)
    output.write(json.dumps(row)+'\n');output.flush();print(json.dumps(row),flush=True)


if __name__=='__main__':main()
