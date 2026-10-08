"""Fixed behavior contracts; validation/gold never enter model payload."""
import json,pathlib,subprocess,tempfile,os,sys
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark151'))
import contiguous,remeasure,evaluate
OLD_A='func ClampNegative(n int) int { return n }\n'
NEW_A='func ClampNegative(n int) int { if n < 0 { return 0 }; return n }\n'
OLD_B='func Characters(s string) int { return len([]byte(s)) }\n'
NEW_B='func Characters(s string) int { return len([]rune(s)) }\n'
def state(a,b):return ('package behavior\n'+(NEW_A if a else OLD_A)+(NEW_B if b else OLD_B)).encode()
def main():
 old,new=state(False,False),state(True,True)
 fixture=dict(name='negative-clamp-and-unicode-count',before=old.decode(),after=new.decode(),requirements=['negative clamping preserves positive values','Unicode rune count replaces byte count'],expected_states=[dict(a=a,b=b,content=state(a,b).decode()) for a,b in [(False,False),(True,False),(False,True),(True,True)]])
 path=HERE/'behavior-fixture.json';path.write_text(json.dumps(fixture,indent=2)+'\n')
 h=contiguous.extract(old,new,'F1');rows=[]
 with tempfile.TemporaryDirectory(prefix='issue153-behavior-') as d:
  root=pathlib.Path(d);(root/'go.mod').write_text('module behavior\ngo 1.24\n')
  for a,b in [(False,False),(True,False),(False,True),(True,True)]:
   target=state(a,b);witnesses=remeasure.matching_subsets(old,h['atoms'],target);assert len(witnesses)==1
   (root/'behavior.go').write_bytes(target)
   neg=0 if a else -2;chars=1 if b else 3
   (root/'behavior_test.go').write_text(f'package behavior\nimport "testing"\nfunc TestContracts(t *testing.T) {{ if ClampNegative(-2)!={neg} || ClampNegative(7)!=7 || Characters("あ")!={chars} || Characters("abc")!=3 {{t.Fatal("behavior contract")}} }}\n')
   p=subprocess.run(['go','test','./...','-count=1'],cwd=root,env={**os.environ,'GOCACHE':'/tmp/issue151-remeasure-gocache'},capture_output=True,text=True,timeout=120)
   rows.append(dict(a=a,b=b,test_pass=p.returncode==0,atom_count=len(witnesses[0]),output=p.stdout+p.stderr));assert p.returncode==0
 remeasure.stage_states({'behavior.go':old},[{'behavior.go':state(a,b)} for a,b in [(False,False),(True,False),(False,True),(True,True)]])
 result=dict(atoms=len(h['atoms']),proposals=len(h['operations']),states=rows,staging=True,calls=0,gold_runtime=False)
 (HERE.parents[1]/'docs/benchmarks/issue-153/iteration-13-preflight.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
if __name__=='__main__':main()
