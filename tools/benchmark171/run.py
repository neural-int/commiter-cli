"""Once-only #171 grouping calls using unchanged #169 helper/validators/resources."""
import argparse,hashlib,json,os,re,signal,subprocess,sys,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2];DATA=ROOT/'docs/benchmarks/issue-171'
sys.path.insert(0,str(ROOT/'tools/benchmark169'))
# A distinct module name avoids importing this same filename as the prior runner.
import importlib.util
spec=importlib.util.spec_from_file_location('prior169',ROOT/'tools/benchmark169/run.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior);prior.ROOT=ROOT

def check_model(pr, local, full=False):
 for artifact in pr['model_artifacts']:
  path=Path(local[artifact['runtime_key']]);st=path.stat()
  if st.st_size!=artifact['bytes'] or st.st_mtime_ns!=artifact['mtime_ns']:raise ValueError('model artifact signature drift: '+artifact['name'])
  if full:
   h=hashlib.sha256()
   with path.open('rb') as f:
    for block in iter(lambda:f.read(4*1024*1024),b''):h.update(block)
   if h.hexdigest()!=artifact['sha256']:raise ValueError('model artifact hash drift: '+artifact['name'])

def main():
 ap=argparse.ArgumentParser();ap.add_argument('--runtime',type=Path,required=True);a=ap.parse_args();local=json.loads(a.runtime.read_text());pr=json.loads((DATA/'preregistration.json').read_text());prior.check_pins(pr,local);check_model(pr,local,full=True)
 rows=json.loads((DATA/'frozen-inputs/manifest.json').read_text());journal=DATA/'attempts.jsonl';result=DATA/'measurements.jsonl';attempted=[json.loads(s) for s in journal.read_text().splitlines()] if journal.exists() else []
 key=lambda r:(r['fixture'],r['order'],r['arm']);seen={key(r) for r in attempted}
 # Results cannot be replayed or removed to recover call budget.
 for row in rows:
  if key(row) in seen:continue
  if len(attempted)>=pr['max_generation_calls']:raise ValueError('call cap')
  prior.check_pins(pr,local);check_model(pr,local);request=json.loads((DATA/'frozen-inputs'/row['request_file']).read_text());request['model_path']=local['model_path'];root=Path(local['fixture_roots'][row['fixture']])
  if prior.repo_digest(root)!=row['repository_sha256']:raise ValueError('Git changed before generation')
  before=prior.resources()
  if None in before.values() or before['pressure_level']>=4:raise ValueError('resource observer unavailable/critical')
  with journal.open('a') as f:f.write(json.dumps({**row,'attempt_number':len(attempted)+1})+'\n');f.flush();os.fsync(f.fileno())
  attempted.append(row);seen.add(key(row));start=time.monotonic();proc=subprocess.Popen(['/usr/bin/time','-l',local['helper']],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,start_new_session=True)
  try:out,err=proc.communicate(json.dumps(request).encode()+b'\n',timeout=pr['call_timeout_seconds']);stop='process_failure' if proc.returncode else None
  except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);out,err=proc.communicate();stop='timeout'
  after=prior.resources();rec={**row,'calls':1,'status':stop,'schema_valid':False,'response':None,'wall_seconds':time.monotonic()-start,'resource_before':before,'resource_after':after,'git_unchanged':prior.repo_digest(root)==row['repository_sha256'],'helper_peak_rss_bytes':None,'input_tokens':None,'output_tokens':None,'generated_text':None}
  rss=re.search(rb'(?m)^\s*(\d+)\s+maximum resident set size\s*$',err)
  if rss:rec['helper_peak_rss_bytes']=int(rss[1])
  if stop is None and len(out)<=4*1024*1024:
   try:
    response=json.loads(out,object_pairs_hook=prior.decode_unique);rec['status']=response['stop_reason'];rec['helper_stop_reason']=response['stop_reason'];rec['helper_error_class']=response.get('error_class');rec['generated_text']=response.get('generated_json')
    for k in ['input_tokens','output_tokens','load_seconds','ttft_seconds','peak_bytes']:rec[k]=response.get('benchmark_'+k)
    if response.get('ok') and (response.get('generation_profile')!=pr['profile'] or response.get('model')!=pr['model']+'@'+pr['revision']):rec['status']='protocol_mismatch'
    elif response.get('ok') and rec['status']=='completed':
     try:json.loads(response['generated_json'],object_pairs_hook=prior.decode_unique)
     except (ValueError,TypeError):rec['status']='invalid_json'
     if rec['status']=='completed':
      try:rec['response']=prior.validate_grouping(response['generated_json'],request);rec['schema_valid']=True
      except (ValueError,KeyError,TypeError):rec['status']='invalid_assignment_or_schema'
   except (ValueError,KeyError,TypeError):rec['status']='invalid_helper_protocol'
  with result.open('a') as f:f.write(json.dumps(rec,ensure_ascii=False)+'\n');f.flush();os.fsync(f.fileno())
  print(json.dumps({k:rec[k] for k in ['fixture','arm','order','status','wall_seconds','input_tokens','output_tokens','helper_peak_rss_bytes','git_unchanged']}),flush=True)
  if not rec['git_unchanged']:raise ValueError('Git changed: receipt preserved')
  if None in after.values() or after['pressure_level']>=4 or after['swap_used_mib']-before['swap_used_mib']>pr['max_swap_delta_mib'] or rec['helper_peak_rss_bytes'] is None or rec['helper_peak_rss_bytes']>pr['max_rss_bytes']:raise ValueError('hard resource stop: receipt preserved')
 prior.check_pins(pr,local);check_model(pr,local,full=True)
if __name__=='__main__':main()
