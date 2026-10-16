"""Audit captured production requests; tokenize without generating any tokens."""
import argparse,hashlib,json,pathlib,subprocess,time

def main():
 p=argparse.ArgumentParser();p.add_argument('--capture',type=pathlib.Path,required=True);p.add_argument('--out',type=pathlib.Path,required=True);p.add_argument('--runtime',type=pathlib.Path,required=True);a=p.parse_args()
 runtime=json.loads(a.runtime.read_text());a.out.parent.mkdir(parents=True,exist_ok=True)
 with a.out.open('x') as receipt:
  for cid in ['C01','C02']:
   for phase in range(1,4):
    paths=[a.capture/f'{cid}-{arm}'/f'{phase:02}-request.json' for arm in ['A','B']]
    if not all(path.exists() for path in paths):continue
    pair=[json.loads(path.read_text()) for path in paths]
    assert pair[0]['messages'][0]==pair[1]['messages'][0]
    assert pair[0]['context_tokens']==pair[1]['context_tokens']
    assert pair[0]['schema']==pair[1]['schema']
    users=[json.loads(x['messages'][1]['content']) for x in pair];assert 'codex_context' not in users[0];users[1].pop('codex_context');assert users[0]==users[1]
    for arm,path,request in zip(['A','B'],paths,pair):
     actual_context=request['context_tokens']
     # The existing tokenize-only helper accepts 16k requests. Tokenization of
     # identical messages/profile is independent of the generation context ceiling.
     request['context_tokens']=16384
     request['model_path']=runtime['model_path'];request['preflight_only']=True;start=time.monotonic()
     proc=subprocess.run([runtime['preflight_helper']],input=json.dumps(request).encode()+b'\n',capture_output=True,timeout=120,check=True)
     response=json.loads(proc.stdout);n=response.get('benchmark_input_tokens')
     admitted=response.get('ok') is True and response.get('generated_json')=='{}' and isinstance(n,int) and n+request['output_tokens']<=actual_context
     assert response.get('benchmark_output_tokens') in [None,0]
     row={'case':cid,'arm':arm,'phase':phase,'request_sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'native_input_tokens':n,'context_tokens':actual_context,'tokenizer_helper_context_tokens':request['context_tokens'],'output_reservation':request['output_tokens'],'admitted':admitted,'generated_tokens':0,'wall_seconds':time.monotonic()-start,'stop_reason':response.get('stop_reason'),'error_class':response.get('error_class')}
     receipt.write(json.dumps(row)+'\n');receipt.flush();print(json.dumps(row),flush=True)
     if not admitted:raise RuntimeError('admission failed; stop before inference')
if __name__=='__main__':main()
