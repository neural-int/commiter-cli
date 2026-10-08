"""Prospective symbol-history guardrail comparison, gold never schedules history."""
import json,pathlib,tempfile,subprocess,re,time
import inline_evaluate as base
from history_scan import scan
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[1]/'docs/benchmarks/issue-152'
def prepare(record):
 data,gold,_,_=base.payload(record,'/tmp/issue152-samefile-graph','/tmp/issue151-remeasure-symbols','repository')
 files={f['ID']:f for f in record['Files']}
 identities=sorted({(files[u['file']]['Path'],s) for u in data['change_units'] for s in u['symbols']})
 with tempfile.TemporaryDirectory(prefix='issue152-historyab-') as tmp:
  root=pathlib.Path(tmp)
  def git(*a):subprocess.run(['git',*a],cwd=root,capture_output=True,check=True)
  git('init','-q');git('config','user.name','Benchmark');git('config','user.email','benchmark@example.invalid')
  originals={f['Path']:f['Before'] for f in record['Files']}
  originals.update({f['Path']:f['Content'] for f in record['Repository']})
  for p,s in originals.items():
   q=root/p;q.parent.mkdir(parents=True,exist_ok=True);q.write_text(s)
  git('add','.');git('commit','-qm','initial')
  def changed(identity):
   p,name=identity;s=(root/p).read_text();defs=json.loads(subprocess.check_output(['/tmp/issue151-remeasure-symbols'],input=s.encode()));d=next(x for x in defs if x['name']==name);b=s.encode();part=b[d['start']:d['end']].decode();part=re.sub(r'\b\d+\b',lambda m:str(int(m[0])+1),part,count=1);(root/p).write_bytes(b[:d['start']]+part.encode()+b[d['end']:])
  groups=[identities] if record['HistoryRegime']=='batch' else [[x] for x in (identities[:1] if record['HistoryRegime']=='sparse' else identities)]
  for group in groups:
   for x in group:changed(x)
   git('add','.');git('commit','-qm','historical numeric change')
   for p in {p for p,s in group}:(root/p).write_text(originals[p])
   git('add','.');git('commit','-qm','historical numeric revert')
  start=time.perf_counter();history=scan(root,identities,'/tmp/issue151-remeasure-symbols');wall=time.perf_counter()-start
  repeated=scan(root,identities,'/tmp/issue151-remeasure-symbols');assert repeated==history
 return data,gold,history,wall
def main():
 records=json.loads((HERE/'history-independent.json').read_text());prepared=[]
 for r in records:
  d,g,h,w=prepare(r);prepared.append((r,d,g,h,w));print(json.dumps(dict(fixture=r['Name'],extraction_seconds=w,summary=h['summary'])),flush=True)
 (ROOT/'iteration-25-extraction.json').write_text(json.dumps([dict(fixture=r['Name'],history=h,extraction_seconds=w) for r,d,g,h,w in prepared],indent=2)+'\n')
 with (ROOT/'iteration-25-results.jsonl').open('w') as out:
  for r,d,g,h,w in prepared:
   for mode in ('A-only','history'):
    data=dict(d);data['symbol_history_evidence']=dict(observation=h['summary'] if mode=='history' else None,semantics='soft_history_not_intent; absence_unknown_no_default_merge')
    m,meta=base.api.invoke(data,'/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper','/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8')
    exact,fm,fs=base.api.quality(m,g) if m else (None,None,None)
    row=dict(fixture=r['Name'],source=mode,exact=exact,false_merge=fm,false_split=fs,complete=m is not None,membership=m,**meta);out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)
if __name__=='__main__':main()
