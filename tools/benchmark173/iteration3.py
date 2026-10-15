"""Once-only short Category prompt evaluation. Historical comparison, no adoption."""
import argparse, hashlib, json, os, subprocess, time
from pathlib import Path
import run as prior
SRC=Path(__file__).resolve().parents[2]; BASE=SRC/'docs/benchmarks/issue-173'; DATA=BASE/'iteration-3'
SEMANTIC='''Classify each group by its overall change purpose.
Use only supplied evidence. Do not infer intent.
Supporting changes inherit the group's purpose.

Types:
- feat: Clearly introduces new behavior, not a correction.
- fix: Corrects a demonstrated violation of an existing contract.
- refactor: Restructures without behavior changes.
- perf: Improves performance with evidence.
- docs: Documentation only.
- style: Formatting only, not UI changes.
- test: Tests only.
- build: Build or dependency changes.
- ci: CI/CD changes.
- chore: Non-behavioral maintenance.
- revert: Explicitly undoes a previous change.
- update: Behavior changed; feat vs fix cannot be distinguished.
- change: No specific category can be reliably identified.

Rules:
- Behavior changes alone do not prove feat or fix.
- If only feat vs fix is ambiguous, use update.
- Otherwise use change only if no specific type is supported.
- Missing tests do not determine the type.
- Never change group membership.
'''
def h(b):return hashlib.sha256(b).hexdigest()
def save(p,x):p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(x,indent=2,ensure_ascii=False)+'\n')
def main():
 ap=argparse.ArgumentParser();ap.add_argument('--root',type=Path,required=True);a=ap.parse_args();root=a.root
 runtime=json.loads((root/'runtime.json').read_text());old=json.loads((BASE/'preregistration.json').read_text());gold=json.loads((BASE/'gold.json').read_text())
 assert h(Path(runtime['helper']).read_bytes())==old['helper_sha256']
 cases=[*'ABCDE',*[f'F{i:02d}' for i in range(1,21)]]
 freeze=DATA/'preregistration.json'
 if not freeze.exists():
  assert not (DATA/'attempts.jsonl').exists()
  hashes={}
  for case in cases:
   source=json.loads((BASE/f'inputs/{case}-candidate.json').read_text());req=json.loads(json.dumps(source))
   body,schema=source['messages'][0]['content'].split(' The required JSON Schema is: ',1)
   suffix=body[body.index('Choose an applicable observed evidence reference'):]
   req['messages'][0]['content']='Generate only type and breaking_evidence_ref for each finalized group.\n'+SEMANTIC+suffix+' The required JSON Schema is: '+schema
   assert req['schema']==source['schema'] and req['messages'][1]==source['messages'][1]
   assert {k:v for k,v in req.items() if k!='messages'}=={k:v for k,v in source.items() if k!='messages'}
   p=DATA/f'inputs/{case}.json';save(p,req);hashes[str(p.relative_to(SRC))]=h(p.read_bytes())
  for p in [Path(__file__),SRC/'tools/benchmark173/run.py',SRC/'tools/benchmark173/prepare.py',BASE/'gold.json',BASE/'category-results.jsonl']:
   hashes[str(p.relative_to(SRC))]=h(p.read_bytes())
  for pin in json.loads((BASE/'model-artifacts.json').read_text()):assert h((Path(runtime['model_path'])/pin['name']).read_bytes())==pin['sha256']
  save(freeze,{'comment':'https://github.com/neural-int/commiter-cli/issues/173#issuecomment-6188352282','cases':cases,'max_generation_calls':25,'retries':0,'repairs':0,'timeout_seconds':120,'comparison':'Iteration 1 candidate historical comparison; no contemporaneous control, known synthetic fixtures, not independent evaluation','model':old['model'],'helper_sha256':old['helper_sha256'],'hashes':hashes,'product_source_changed':False,'grouping_text_calls':0,'decision':'Report exact Gold, false feat/fix, correct/false update and change, side effects by type. Support only if existing Gold improves, boundary Gold does not regress, and false assertions do not rise. No production adoption or end-to-end success claim.'})
 pr=json.loads(freeze.read_text())
 assert not (DATA/'attempts.jsonl').exists(),'once-only run already attempted'
 for name,digest in pr['hashes'].items():assert h((SRC/name).read_bytes())==digest,name
 for i,case in enumerate(cases,1):
  before=prior.resources();assert before['pressure_level']<4
  with (DATA/'attempts.jsonl').open('a') as f:f.write(json.dumps({'number':i,'case':case})+'\n');f.flush();os.fsync(f.fileno())
  req=json.loads((DATA/f'inputs/{case}.json').read_text());req['model_path']=runtime['model_path'];start=time.monotonic()
  print('START',case,flush=True)
  try:
   p=subprocess.run(['/usr/bin/time','-l',runtime['helper']],input=json.dumps(req).encode(),capture_output=True,timeout=120)
   out,err=p.stdout,p.stderr;status='returned';code=p.returncode
  except subprocess.TimeoutExpired as e:out=e.stdout or b'';err=e.stderr or b'';status='timeout';code=None
  row={'case':case,'schema_valid':False,'stop_reason':status,'response':None,'input_tokens':None,'output_tokens':None,'exit_code':code}
  try:
   r=json.loads(out);row.update(helper_stop_reason=r.get('stop_reason'),error_class=r.get('error_class'))
   if status!='timeout':row['stop_reason']=r.get('stop_reason')
   if status!='timeout' and r.get('ok') and r.get('stop_reason')=='completed':
    assert r['model']==pr['model'] and r['generation_profile']=='bounded-category'
    row['response']=prior.validate(r['generated_json'],req);row['schema_valid']=True
  except (ValueError,KeyError,TypeError):row['validation_error']='invalid_response_or_schema'
  import re
  m=re.search(rb'(?m)^\s*(\d+)\s+maximum resident set size\s*$',err);rss=int(m[1]) if m else None;after=prior.resources()
  row.update(wall_seconds=round(time.monotonic()-start,3),helper_peak_rss_bytes=rss,resources_before=before,resources_after=after,input_bytes=sum(len(m['content'].encode()) for m in req['messages']))
  row['groups']=[{'group_id':g,'expected':t,'actual':(row['response'] or {}).get(g,{}).get('type'),'identifiability':gold[case]['identifiability']} for g,t in gold[case]['groups'].items()]
  local=root/'iteration-3'/case;local.mkdir(parents=True,exist_ok=True);(local/'stdout.json').write_bytes(out);(local/'stderr.txt').write_bytes(err)
  with (DATA/'results.jsonl').open('a') as f:f.write(json.dumps(row)+'\n');f.flush();os.fsync(f.fileno())
  print('RESULT',case,row['stop_reason'],row['groups'],row['wall_seconds'],flush=True)
  assert not prior.critical(before,after,rss),'resource stop'
 for name,digest in pr['hashes'].items():assert h((SRC/name).read_bytes())==digest,name
 save(DATA/'completion.json',{'generation_calls':25,'status':'completed','preregistration_sha256':h(freeze.read_bytes())})
if __name__=='__main__':main()
