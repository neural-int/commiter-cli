#!/usr/bin/env python3
"""Unchanged helper protocol with numeric process measurements; no thought/stderr export."""
import hashlib,json,os,pathlib,resource,subprocess,sys,time
request=sys.stdin.buffer.read();data=json.loads(request);trace=pathlib.Path(os.environ['COMMITER_175_NATIVE_ROOT']);trace.mkdir(parents=True,exist_ok=True)
index=len(list(trace.glob('*.json')))+1
if index>12:raise SystemExit('inference call cap exceeded')
start=time.monotonic();p=subprocess.run([os.environ['COMMITER_175_REAL_HELPER']],input=request,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
row={'call':index,'profile':data['generation_profile'],'request_sha256':hashlib.sha256(request).hexdigest(),'wall_seconds':time.monotonic()-start,'exit_code':p.returncode,'child_max_rss_bytes':resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss,'rss_scope':'native helper process maximum resident size on macOS; not GPU/Metal peak or system pressure'}
try:
 response=json.loads(p.stdout);row.update(stop_reason=response.get('stop_reason'),runtime=response.get('runtime'),model=response.get('model'),error_class=response.get('error_class'))
except ValueError:row['stop_reason']='invalid_protocol'
with (trace/f'{index:02}.json').open('x') as f:json.dump(row,f,indent=2)
sys.stdout.buffer.write(p.stdout);sys.stdout.buffer.flush();raise SystemExit(p.returncode if p.returncode>=0 else 1)
