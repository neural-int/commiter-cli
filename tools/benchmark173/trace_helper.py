#!/usr/bin/env python3
"""Local capture/forward helper. Capture mode performs zero generation."""
import json, os, signal, subprocess, sys, time
from pathlib import Path
root=Path(os.environ['BENCH173_ROOT']); runtime=json.loads((root/'runtime.json').read_text())
trace=Path(os.environ['BENCH173_TRACE']);trace.mkdir(parents=True,exist_ok=True)
data=sys.stdin.buffer.read(); req=json.loads(data); stem=trace/f'{len(list(trace.glob("*.request.json")))+1:02d}'
public={k:v for k,v in req.items() if k!='model_path'}
stem.with_suffix('.request.json').write_text(json.dumps(public,sort_keys=True)+'\n')
started=time.monotonic(); child=None
meta={'profile':req['generation_profile'],'context_tokens':req['context_tokens'],'output_budget':req['output_tokens']}
def finish(status,code):
    meta.update(status=status,wall_seconds=time.monotonic()-started,exit_code=code)
    stem.with_suffix('.meta.json').write_text(json.dumps(meta)+'\n')
def interrupt(sig,frame):
    if child is not None: child.terminate()
    finish('interrupted_by_product',128+sig);sys.exit(128+sig)
signal.signal(signal.SIGTERM,interrupt);signal.signal(signal.SIGINT,interrupt)
if os.environ.get('BENCH173_CAPTURE')=='1':
    props=req['schema']['properties']; profile=req['generation_profile']; case=os.environ['BENCH173_CASE']
    if profile=='bounded-grouping':
        result={k:('G002' if case=='C' and 'retry' in k else 'G001') for k in props}
    elif profile=='bounded-category':result={k:{'type':('test' if v['properties']['type']['enum']==['test'] else 'fix'),'breaking_evidence_ref':'none'} for k,v in props.items()}
    else: result={k:{'scope':'fixture','summary':'correct fixture behavior'} for k in props}
    out=json.dumps({'ok':True,'stop_reason':'completed','model':req['model'],'runtime':'mlx','generation_profile':profile,'generated_json':json.dumps(result)}).encode()+b'\n';err=b'';code=0
    finish('capture_only_zero_generation',0)
else:
    child=subprocess.Popen([runtime['helper']],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    out,err=child.communicate(data);code=child.returncode
    try:
        response=json.loads(out);meta.update(stop_reason=response.get('stop_reason'),error_class=response.get('error_class'))
    except ValueError:pass
    finish('returned',code)
stem.with_suffix('.response.json').write_bytes(out);stem.with_suffix('.stderr.txt').write_bytes(err)
sys.stdout.buffer.write(out);sys.stderr.buffer.write(err);sys.exit(code if code>=0 else 1)
