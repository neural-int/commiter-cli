"""Fresh six-file behavior fixture; requirements used only by evaluation."""
import pathlib,json,sys,tempfile,subprocess,os,hashlib
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE));import refinement,payload_budget,remeasure
specs=[('division', 'package behavior\nfunc SafeRatio(n,d int) int { return n/d }\n','package behavior\nfunc SafeRatio(n,d int) int { if d==0 { return 0 }; return n/d }\n','package behavior\nimport "testing"\nfunc TestSafeRatio(t *testing.T) { if SafeRatio(8,2)!=4 {t.Fatal("ratio")} }\n','package behavior\nimport "testing"\nfunc TestSafeRatio(t *testing.T) { if SafeRatio(8,2)!=4 || SafeRatio(8,0)!=0 {t.Fatal("ratio")} }\n'),('label','package behavior\nimport "strings"\nfunc NormalLabel(s string) string { return strings.ToLower(s) }\n','package behavior\nimport "strings"\nfunc NormalLabel(s string) string { return strings.ToLower(strings.TrimSpace(s)) }\n','package behavior\nimport "testing"\nfunc TestNormalLabel(t *testing.T) { if NormalLabel(" A ")!=" a " {t.Fatal("label")} }\n','package behavior\nimport "testing"\nfunc TestNormalLabel(t *testing.T) { if NormalLabel(" A ")!="a" {t.Fatal("label")} }\n'),('upper','package behavior\nfunc BoundLevel(n int) int { if n<0 {return 0}; return n }\n','package behavior\nfunc BoundLevel(n int) int { if n<0 {return 0}; if n>100 {return 100}; return n }\n','package behavior\nimport "testing"\nfunc TestBoundLevel(t *testing.T) { if BoundLevel(-1)!=0 || BoundLevel(101)!=101 {t.Fatal("level")} }\n','package behavior\nimport "testing"\nfunc TestBoundLevel(t *testing.T) { if BoundLevel(-1)!=0 || BoundLevel(101)!=100 {t.Fatal("level")} }\n')]
def main():
 files=[]
 for i,(name,b,a,tb,ta) in enumerate(specs):
  for suffix,before,after in [('.go',b,a),('_test.go',tb,ta)]:files.append(dict(id=f'F{len(files)+1:03}',path=name+suffix,before=before,after=after,eval_intent=i))
 fixture=dict(name='fresh-six-file-behaviors',split='independent_synthetic',files=files,baseline_projection=[f['id'] for f in files[:4]],baseline_projection_scope='4file/2intent; full6 outside current production limit')
 (HERE/'multifile-fixture.json').write_text(json.dumps(fixture,indent=2)+'\n')
 snap={f['id']:(f['before'].encode(),f['after'].encode()) for f in files};messages,schema,mapping,size=payload_budget.build(snap)
 counts=[dict(id=f['id'],atoms=len(refinement.contiguous.extract(*snap[f['id']],f['id'])['atoms'])) for f in files];rows=[];states=[]
 with tempfile.TemporaryDirectory(prefix='issue153-six-') as d:
  root=pathlib.Path(d);(root/'go.mod').write_text('module behavior\ngo 1.24\n')
  for mask in range(8):
   state={}
   for f in files:
    content=f['after' if mask&(1<<f['eval_intent']) else 'before'].encode();(root/f['path']).write_bytes(content);state[f['path']]=content
   proc=subprocess.run(['go','test','./...','-count=1'],cwd=root,env={**os.environ,'GOCACHE':'/tmp/issue151-remeasure-gocache'},capture_output=True,text=True,timeout=120);rows.append(dict(mask=mask,test_pass=proc.returncode==0,output=proc.stdout+proc.stderr));assert proc.returncode==0;states.append(state)
 remeasure.stage_states({f['path']:f['before'].encode() for f in files},states)
 result=dict(files=len(files),intents=3,atom_counts=counts,initial_proposals=len(mapping),initial_message_bytes=size,states=rows,staging=True,calls=0,fixture_sha256=hashlib.sha256((HERE/'multifile-fixture.json').read_bytes()).hexdigest())
 (HERE.parents[1]/'docs/benchmarks/issue-153/iteration-16-preflight.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
if __name__=='__main__':main()
