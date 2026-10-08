"""Real-history preflight, no model calls or commit-boundary-derived gold."""
import io,json,os,pathlib,subprocess,sys,tarfile,tempfile
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark151'))
import evaluate as representation
import inline
COMMIT='fd0428947c752766d936275f2d42caa10732993d'
PATHS=['internal/relation/extract.go','internal/relation/extract_test.go']
def git(*args):return subprocess.check_output(['git',*args])
rows=[]
with tempfile.TemporaryDirectory(prefix='issue153-real-') as d:
 root=pathlib.Path(d)
 with tarfile.open(fileobj=io.BytesIO(git('archive',COMMIT))) as archive:
  archive.extractall(root,filter='data')
 blobs={p:(git('show',COMMIT+'^:'+p),git('show',COMMIT+':'+p)) for p in PATHS}
 for i,p in enumerate(PATHS):
  before,after=blobs[p]
  try:
   units=inline.extract(before,after,f'F{i+1:03}')
   assert representation.reconstruct(before,units,{u['id'] for u in units})==after
   rows.append(dict(path=p,units=len(units),reconstruction=True,staging=representation.stage_verify(before,after,units)))
  except ValueError as e:rows.append(dict(path=p,rejection=str(e),units=None))
 states=[]
 for source,test in [(0,0),(0,1),(1,0),(1,1)]:
  for p,state in zip(PATHS,[source,test]):(root/p).write_bytes(blobs[p][state])
  proc=subprocess.run(['go','test','./internal/relation','-run','^TestDirectImportUsesObservedSyntaxAndUniqueChangedTarget$','-count=1'],cwd=root,env={**os.environ,'GOCACHE':'/tmp/issue151-remeasure-gocache'},capture_output=True,text=True,timeout=120)
  states.append(dict(source_after=bool(source),test_after=bool(test),passed=proc.returncode==0,output=proc.stdout+proc.stderr))
result=dict(commit=COMMIT,parent=git('rev-parse',COMMIT+'^').decode().strip(),files=rows,states=states,model_calls=0,semantic_gold=None)
out=HERE.parents[1]/'docs/benchmarks/issue-153/iteration-5-preflight.json';out.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
