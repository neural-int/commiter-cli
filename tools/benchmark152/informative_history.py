"""Used-fixture attribution with valid prior source/test commits."""
import json,pathlib,tempfile,subprocess,re,time,os
import inline_evaluate as base
from history_scan import scan
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[1]/'docs/benchmarks/issue-152'
SYMBOLS='/tmp/issue151-remeasure-symbols'
def prepare(r):
 d,g,_,e=base.payload(r,'/tmp/issue152-samefile-graph',SYMBOLS,'repository');files={f['ID']:f for f in r['Files']};units={u['id']:u for u in d['change_units']}
 identities=sorted({(files[u['file']]['Path'],s) for u in units.values() for s in u['symbols']})
 pairs=[]
 for edge in e['unit_relations']:
  pair=sorted({(units[x]['file'],s) for x in [edge['from_unit'],edge['to_unit']] for s in units[x]['symbols']})
  if pair not in pairs:pairs.append(pair)
 groups=[sorted({x for p in pairs for x in p})] if r['HistoryRegime']=='batch' else pairs
 with tempfile.TemporaryDirectory(prefix='issue152-inf-history-') as tmp:
  root=pathlib.Path(tmp)
  def git(*a):subprocess.run(['git',*a],cwd=root,capture_output=True,check=True)
  def test():
   v=subprocess.run(['go','test','./...'],cwd=root,env=dict(os.environ,GOCACHE='/tmp/issue151-remeasure-gocache'),capture_output=True,text=True);assert v.returncode==0,v.stdout+v.stderr
  git('init','-q');git('config','user.name','Benchmark');git('config','user.email','benchmark@example.invalid');(root/'go.mod').write_text('module fixture\n\ngo 1.22\n')
  for f in r['Files']:
   old=re.sub(r'(\* |!= )(\d+)',lambda m:m[1]+str(int(m[2])-1),f['Before']);p=root/f['Path'];p.parent.mkdir(parents=True,exist_ok=True);p.write_text(old)
  for f in r['Repository']:
   p=root/f['Path'];p.parent.mkdir(parents=True,exist_ok=True);p.write_text(f['Content'])
  test();git('add','.');git('commit','-qm','prior valid snapshot');checks=1
  for group in groups:
   for fid,name in group:
    f=files[fid];p=root/f['Path'];current=p.read_bytes();target=f['Before'].encode()
    def find(b):return next(s for s in json.loads(subprocess.check_output([SYMBOLS],input=b)) if s['name']==name)
    a,b=find(current),find(target);p.write_bytes(current[:a['start']]+target[b['start']:b['end']]+current[a['end']:])
   test();checks+=1;git('add','.');git('commit','-qm','valid historical feature update')
  assert all((root/f['Path']).read_text()==f['Before'] for f in r['Files'])
  start=time.perf_counter();h=scan(root,identities,SYMBOLS);wall=time.perf_counter()-start;assert h==scan(root,identities,SYMBOLS)
 return d,g,h,wall,checks
def main():
 prepared=[]
 for r in json.loads((HERE/'informative-history.json').read_text()):
  d,g,h,w,c=prepare(r);prepared.append((r,d,g,h,w,c));print(json.dumps(dict(fixture=r['Name'],historical_test_pass=c,extraction_seconds=w)),flush=True)
 (ROOT/'iteration-26-extraction.json').write_text(json.dumps([dict(fixture=r['Name'],history=h,extraction_seconds=w,historical_test_pass=c) for r,d,g,h,w,c in prepared],indent=2)+'\n')
 with (ROOT/'iteration-26-results.jsonl').open('w') as out:
  for r,d,g,h,w,c in prepared:
   for mode in ['A-only','history']:
    data=dict(d);data['symbol_history_evidence']=dict(observation=h['summary'] if mode=='history' else None,semantics='soft_history_not_intent; absence_unknown_no_default_merge')
    m,meta=base.api.invoke(data,'/tmp/issue149-qwen8-helper/.build/release/commiter-mlx-helper','/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8');exact,fm,fs=base.api.quality(m,g) if m else (None,None,None)
    row=dict(fixture=r['Name'],source=mode,exact=exact,false_merge=fm,false_split=fs,membership=m,complete=m is not None,**meta);out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)
if __name__=='__main__':main()
