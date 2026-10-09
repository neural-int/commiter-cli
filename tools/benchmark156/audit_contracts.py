"""Scale-specific structural/metadata audit only; oracle is not semantic ability."""
import base64,copy,hashlib,json,pathlib,subprocess,sys
from coarse import prepare
from planner import encode,fallback,replay,validate
from evaluate_c2 import authoritative
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def rows():
 for n in (6,16):
  files=[]
  for i in range(n):
   before=b'first=1\nsecond=2\n' if i==0 else b'value=1\n'
   after=b'first=3\nsecond=4\n' if i==0 else b'value=2\n'
   files.append(dict(id=f'F{i+1:03}',path=f'data/f{i}.txt',before_b64=base64.b64encode(before).decode(),after_b64=base64.b64encode(after).decode()))
  yield dict(files=files,n=n,input_sha256=hashlib.sha256(encode(files)).hexdigest())
def preflight():
 fixed=dict(contract='oracle structural partial assignment at6/16 only; no model calls or semantic exact metrics; same-file raw partial planning.Validate rejection and safe whole-file alternative',rows=list(rows()),source_sha256={name:hashlib.sha256((ROOT/'tools/benchmark156'/name).read_bytes()).hexdigest() for name in ('audit_contracts.py','planner.py','coarse.py')})
 with (OUT/'iteration-8-contract-preregistered.json').open('x') as f:json.dump(fixed,f,ensure_ascii=False,indent=2);f.write('\n')
def run():
 fixed=json.loads((OUT/'iteration-8-contract-preregistered.json').read_text());output=[]
 for name,d in fixed['source_sha256'].items():
  if hashlib.sha256((ROOT/'tools/benchmark156'/name).read_bytes()).hexdigest()!=d:raise ValueError('source_digest')
 for spec in fixed['rows']:
  files=spec['files'];units=prepare(files);base=fallback(files,units);first=[u for u in units if u['file']=='F001'];raw=[copy.deepcopy(base[0])]
  other=copy.deepcopy(raw[0]);other['unit_ids']=[first[1]['id']]
  raw[0]['unit_ids']=[first[0]['id']]+[u['id'] for u in units if u['file']!='F001'];raw[0]['file_ids']=[f['id'] for f in files];raw.append(other)
  validate(units,raw)
  output.append(dict(files=spec['n'],input_sha256=spec['input_sha256'],oracle_only=True,semantic_prediction=False,calls=0,partial_validation=authoritative(files,raw),partial_staging=replay(files,units,raw),fallback_validation=authoritative(files,base)))
 base=fallback(fixed['rows'][0]['files'],prepare(fixed['rows'][0]['files']))[0]
 requests=[]
 for type_,body in [('compose',False),('chore',True),('compose',True)]:
  commit={k:base[k] for k in ('type','scope','breaking','summary','file_ids')};commit['type']=type_
  if body:commit['body']='- 独立目的A\n- 独立目的B'
  requests.append(dict(candidate=dict(schema_version=1,commits=[commit]),file_ids=['F001']))
 compatibility=json.loads(subprocess.run(['/private/tmp/benchmark156-validate'],input=encode(requests),capture_output=True,check=True).stdout)
 with (OUT/'iteration-8-contract-results.json').open('x') as f:json.dump(dict(rows=output,compatibility_requests=requests,compatibility=compatibility),f,ensure_ascii=False,indent=2);f.write('\n')
 print(json.dumps(dict(partial=[r['partial_validation'] for r in output],fallback=[r['fallback_validation'] for r in output],compose_body=compatibility),ensure_ascii=False))
if __name__=='__main__':preflight() if sys.argv[1]=='preflight' else run()
