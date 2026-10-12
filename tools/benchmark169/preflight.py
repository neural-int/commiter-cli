"""Native tokenize-only audit using existing #163 preflight helper. No generation."""
import argparse,hashlib,json,subprocess,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
DATA=ROOT/'docs/benchmarks/issue-169'
def main():
    p=argparse.ArgumentParser();p.add_argument('--runtime',type=Path,required=True);a=p.parse_args()
    local=json.loads(a.runtime.read_text());path=DATA/'frozen-inputs/manifest.json';rows=json.loads(path.read_text());log=DATA/'tokenization-preflight.jsonl'
    if log.exists():raise ValueError('never overwrite preflight receipts')
    tokens={}
    for row in rows:
        request=json.loads((path.parent/row['request_file']).read_text());request['model_path']=local['model_path'];request['preflight_only']=True
        started=time.monotonic()
        # Repeat identical phase/order byte payloads only when needed; each exact request hash binding is preserved.
        canonical=json.dumps(request,sort_keys=True,separators=(',',':'));digest=hashlib.sha256(canonical.encode()).hexdigest()
        if digest in tokens:response=tokens[digest];cached=True
        else:
            proc=subprocess.run([local['preflight_helper']],input=json.dumps(request).encode()+b'\n',capture_output=True,timeout=120)
            if proc.returncode:raise ValueError('tokenizer process failed')
            response=json.loads(proc.stdout);tokens[digest]=response;cached=False
        n=response.get('benchmark_input_tokens');admitted=bool(response.get('ok')) and response.get('stop_reason')=='completed' and isinstance(n,int) and n+768<=16384
        if response.get('benchmark_output_tokens') not in [None,0]:raise ValueError('preflight unexpectedly generated tokens')
        receipt={k:row[k] for k in ['fixture','arm','order','phase','request_sha256']}
        receipt.update(token_count=n,admitted=admitted,stop_reason=response.get('stop_reason'),error_class=response.get('error_class'),cached_exact_request=cached,wall_seconds=time.monotonic()-started,generated_tokens=0,generated_json_is_preflight_sentinel=response.get('generated_json')=='{}')
        row.update(token_count_before_inference=n,admitted=admitted,token_count_status='native processor prepare: existing #163 preflight-only helper; same Package.resolved, model and thinking chat template as measured generation helper')
        with log.open('a') as f:f.write(json.dumps(receipt)+'\n')
        print(json.dumps({k:receipt[k] for k in ['fixture','arm','order','phase','token_count','admitted']}),flush=True)
    path.write_text(json.dumps(rows,indent=2)+'\n')
if __name__=='__main__':main()
