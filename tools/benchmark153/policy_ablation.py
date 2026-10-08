"""One preregistered scorer-policy contrast; no production or upstream changes."""
import argparse, hashlib, itertools, json, os, pathlib, subprocess, sys, tempfile, time
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark152'))
import inline_evaluate as base
sys.path.insert(0,str(HERE))
import refinement, remeasure
from partition import partition
ROOT=HERE.parents[1]/'docs/benchmarks/issue-153'
HELPER=pathlib.Path('/tmp/issue153-bounded-grammar-helper/.build/release/commiter-mlx-helper')
HELPER_SHA='3b50561e987fb166a794bbe7d6a6dd8aca00514a3619a391a523f001f8c64f80'
MODEL='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'
ORIGINAL='Evaluate whether each pair of changed units serves the same commit purpose. Return +2 strong same intent, +1 weak same, -1 weak independent, -2 strong independent, 0 unknown. Do not produce a partition. Syntax proximity alone does not establish shared purpose. If evidence cannot support evaluation set unresolved true.'
POLICY=' An implementation behavior change and a changed test assertion or expectation directly verifying that same behavior serve the same commit purpose. A test may contain assertions for multiple independent purposes; evaluate the changed assertion, not the test file as a whole. Test logging, diagnostics or refactoring unrelated to the changed behavior remain independent. A matching filename, shared call or shared failing test alone does not prove shared purpose. Do not invent a correspondence when the supplied before/after code is insufficient.'
def digest(value):
 data=value if isinstance(value,bytes) else json.dumps(value,sort_keys=True).encode()
 return hashlib.sha256(data).hexdigest()
def file(fid,path,before,after,intent):
 return dict(id=fid,path=path,before=before,after=after,eval_intent=intent)
def fixtures():
 used=json.loads((HERE/'multifile-fixture.json').read_text())
 used['split']='used_failure_diagnostic'
 fresh1=dict(name='fee-rounding-and-wrap',split='model_unused_synthetic',requirements=['Fee rounding is changed to ceiling independently of text wrapping.','Each changed expectation verifies its corresponding changed behavior.'],files=[
 file('F001','fee.go','package policy\nfunc Fee(n int) int { return n/100 }\n','package policy\nfunc Fee(n int) int { return (n+99)/100 }\n',0),
 file('F002','fee_test.go','package policy\nimport "testing"\nfunc TestFee(t *testing.T) { if Fee(101)!=1 {t.Fatal("fee")} }\n','package policy\nimport "testing"\nfunc TestFee(t *testing.T) { if Fee(101)!=2 {t.Fatal("fee")} }\n',0),
 file('F003','wrap.go','package policy\nfunc Wrap(s string) string { return "("+s+")" }\n','package policy\nfunc Wrap(s string) string { return "{"+s+"}" }\n',1),
 file('F004','wrap_test.go','package policy\nimport "testing"\nfunc TestWrap(t *testing.T) { if Wrap("a")!="(a)" {t.Fatal("wrap")} }\n','package policy\nimport "testing"\nfunc TestWrap(t *testing.T) { if Wrap("a")!="{a}" {t.Fatal("wrap")} }\n',1)])
 fresh2=dict(name='same-call-independent-diagnostic',split='model_unused_synthetic',requirements=['Negative-count handling changes independently of test failure wording.','The existing positive-input assertion and its expected value are unchanged.'],files=[
 file('F001','count.go','package policy\nfunc Count(n int) int { return n }\n','package policy\nfunc Count(n int) int { if n<0 {return 0}; return n }\n',0),
 file('F002','count_test.go','package policy\nimport "testing"\nfunc TestCount(t *testing.T) { if Count(3)!=3 {t.Fatal("count")} }\n','package policy\nimport "testing"\nfunc TestCount(t *testing.T) { if Count(3)!=3 {t.Fatal("unexpected positive count")} }\n',1)])
 before='package policy\nimport "testing"\nfunc TestBoth(t *testing.T) {\n if Left(2)!=3 {t.Fatal("left")}\n // Keep both checks in one existing test.\n t.Log("checking right")\n if Right(2)!=4 {t.Fatal("right")}\n}\n'
 after=before.replace('Left(2)!=3','Left(2)!=4').replace('Right(2)!=4','Right(2)!=6')
 shared=file('F003','both_test.go',before,after,None)
 shared['eval_line_intents']={'4':0,'7':1}
 fresh3=dict(name='shared-test-independent-assertions',split='model_unused_synthetic',requirements=['Left increment and Right multiplier are independent behavior changes.','The two assertions share a test function, but each verifies only one changed behavior.'],files=[
 file('F001','left.go','package policy\nfunc Left(n int) int {return n+1}\n','package policy\nfunc Left(n int) int {return n+2}\n',0),
 file('F002','right.go','package policy\nfunc Right(n int) int {return n*2}\n','package policy\nfunc Right(n int) int {return n*3}\n',1),shared])
 return [used,fresh1,fresh2,fresh3]
def prepare(record):
 files=record['files'];snap={f['id']:(f['before'].encode(),f['after'].encode()) for f in files}
 hs={fid:refinement.contiguous.extract(*pair,fid) for fid,pair in snap.items()}
 # Fixed mechanical upstream: accept every contiguous proposal in both arms.
 plan={p['id']:dict(action='accept') for h in hs.values() for p in h['operations']}
 units=refinement.validate(snap,plan);atom_gold={}
 for f in files:
  for a in hs[f['id']]['atoms']:
   line=f['before'].encode()[:a['old_span'][0]].count(b'\n')+1
   atom_gold[a['id']]=f['eval_line_intents'][str(line)] if 'eval_line_intents' in f else f['eval_intent']
 gold={}
 for i,u in enumerate(units):
  purposes={atom_gold[a] for a in u['atoms']}
  if len(purposes)!=1:raise ValueError('unrepresentable_gold_parent')
  gold[f'U{i+1:03}']=purposes.pop()
 data=dict(change_units=[dict(id=f'U{i+1:03}',file=u['file'],before=snap[u['file']][0].decode(),after=refinement.reconstruct(snap[u['file']][0],hs[u['file']]['atoms'],u['atoms']).decode()) for i,u in enumerate(units)],selected_file_context=[{k:f[k] for k in ('id','path','before','after')} for f in files])
 if any(k.startswith('eval') or k=='requirements' for k in data):raise ValueError('gold_leak')
 return data,gold,units,hs,atom_gold

def preflight(records):
 rows=[]
 for record in records:
  data,gold,units,hs,ag=prepare(record);states=[];intents=sorted(set(ag.values()))
  with tempfile.TemporaryDirectory(prefix='issue153-policy-') as tmp:
   root=pathlib.Path(tmp);(root/'go.mod').write_text('module policy\ngo 1.24\n')
   for mask in range(1<<len(intents)):
    enabled={v for i,v in enumerate(intents) if mask&(1<<i)};state={}
    for f in record['files']:
     atoms=hs[f['id']]['atoms'];ids=[a['id'] for a in atoms if ag[a['id']] in enabled]
     content=refinement.reconstruct(f['before'].encode(),atoms,ids)
     state[f['path']]=content;dest=root/f['path'];dest.parent.mkdir(parents=True,exist_ok=True);dest.write_bytes(content)
    proc=subprocess.run(['go','test','./...','-count=1'],cwd=root,env={**os.environ,'GOCACHE':'/tmp/issue151-remeasure-gocache'},capture_output=True,timeout=120)
    if proc.returncode:raise ValueError('fixture_test_failure:'+record['name']+':'+str(mask))
    states.append(state)
  remeasure.stage_states({f['path']:f['before'].encode() for f in record['files']},states)
  rows.append(dict(fixture=record['name'],units=len(units),gold=gold,unit_order=[dict(id=f'U{i+1:03}',file=u['file'],parent=u['parent']) for i,u in enumerate(units)],payload_sha256=digest(data),states_passed=len(states),staging=True))
 return rows

def messages(data,condition):
 ids=sorted(u['id'] for u in data['change_units']);pairs=[a+'__'+b for a,b in itertools.combinations(ids,2)]
 schema=dict(type='object',properties=dict(scores=dict(type='object',properties={p:dict(type='integer',enum=[-2,-1,0,1,2]) for p in pairs},required=pairs,additionalProperties=False),unresolved=dict(type='boolean')),required=['scores','unresolved'],additionalProperties=False)
 system=ORIGINAL+(POLICY if condition=='policy' else '')+' Output JSON matching this schema: '+json.dumps(schema,sort_keys=True)
 return [dict(role='system',content=system),dict(role='user',content=json.dumps(data,sort_keys=True))],schema

def run(records):
 assert digest(HELPER.read_bytes())==HELPER_SHA
 out=ROOT/'iteration-21-results.jsonl'
 if out.exists():raise ValueError('results_already_exist')
 deadline=time.monotonic()+960
 with out.open('x') as stream:
  for index,record in enumerate(records):
   data,gold,units,hs,ag=prepare(record)
   conditions=['original','policy'] if index%2==0 else ['policy','original']
   for condition in conditions:
    msg,schema=messages(data,condition);size=len(json.dumps(msg,ensure_ascii=False).encode())
    if size>8192:raise ValueError('message_byte_budget')
    row=dict(fixture=record['name'],split=record['split'],condition=condition,calls=1,complete=False,exact=None,false_merge=None,false_split=None,payload_sha256=digest(data),schema_sha256=digest(schema),prompt_sha256=digest(msg),message_bytes=size)
    req=dict(schema=schema,messages=msg,context_tokens=16384,output_tokens=1536,model=base.api.MODEL+'@'+base.api.REVISION,model_path=MODEL,generation_profile='bounded-routed-grammar')
    start=time.perf_counter()
    try:
     remaining=deadline-time.monotonic()
     if remaining<=0:raise ValueError('whole_budget')
     proc=subprocess.run([str(HELPER)],input=(json.dumps(req)+'\n').encode(),capture_output=True,timeout=min(120,remaining))
     if proc.returncode:raise ValueError('helper_failure')
     result=json.loads(proc.stdout,object_pairs_hook=base.api.strict)
     row.update(stop=result.get('stop_reason'),input_tokens=result.get('benchmark_input_tokens'),output_tokens=result.get('benchmark_output_tokens'))
     if not result.get('ok') or row['stop']!='completed':raise ValueError('backend_not_completed')
     answer=json.loads(result['generated_json'],object_pairs_hook=base.api.strict)
     if set(answer)!={'scores','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
     row['scores']=answer['scores']
     if answer['unresolved']:raise ValueError('unresolved')
     membership,solver=partition(sorted(gold),answer['scores'])
     pairs=list(itertools.combinations(gold,2));fm=sum(membership[a]==membership[b] and gold[a]!=gold[b] for a,b in pairs);fs=sum(membership[a]!=membership[b] and gold[a]==gold[b] for a,b in pairs)
     row.update(complete=True,exact=fm==fs==0,false_merge=fm,false_split=fs,membership=membership,solver=solver,reason='accepted')
    except subprocess.TimeoutExpired:row.update(reason='timeout',stop='timeout')
    except (ValueError,KeyError,TypeError) as exc:row['reason']=str(exc)
    row['wall_seconds']=time.perf_counter()-start;stream.write(json.dumps(row)+'\n');stream.flush();print(json.dumps(row),flush=True)

def main():
 parser=argparse.ArgumentParser();parser.add_argument('mode',choices=['preflight','run']);args=parser.parse_args()
 path=HERE/'policy-fixtures.json'
 if args.mode=='preflight':
  records=fixtures();path.write_text(json.dumps(records,indent=2)+'\n');rows=preflight(records)
  sizes=[dict(fixture=r['name'],condition=c,message_bytes=len(json.dumps(messages(prepare(r)[0],c)[0],ensure_ascii=False).encode())) for r in records for c in ('original','policy')]
  if any(s['message_bytes']>8192 for s in sizes):raise ValueError('message_byte_budget')
  result=dict(calls=0,fixture_sha256=digest(path.read_bytes()),helper_sha256=digest(HELPER.read_bytes()),model=base.api.MODEL+'@'+base.api.REVISION,rows=rows,message_sizes=sizes)
  (ROOT/'iteration-21-preflight.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
 else:
  fixed=json.loads((ROOT/'iteration-21-preflight.json').read_text());assert digest(path.read_bytes())==fixed['fixture_sha256'];run(json.loads(path.read_text()))
if __name__=='__main__':main()
