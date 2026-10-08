"""Fixed whitespace-bias ablation with shared binary and original scorer prompt."""
import hashlib, itertools, json, pathlib, subprocess, time
import policy_ablation as base
from partition import partition
ROOT=base.ROOT
HELPER=pathlib.Path('/tmp/issue153-neutral-score-helper/.build/release/commiter-mlx-helper')
PROFILES={'biased':'bounded-routed-grammar','neutral':'bounded-routed-grammar-neutral'}
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def main():
 fixed=json.loads((ROOT/'iteration-23-setup.json').read_text())
 assert sha(HELPER)==fixed['helper_sha256']
 path=base.HERE/'policy-fixtures.json';assert sha(path)==fixed['fixture_sha256']
 records=json.loads(path.read_text());output=ROOT/'iteration-23-results.jsonl'
 if output.exists():raise ValueError('results_already_exist')
 deadline=time.monotonic()+960
 with output.open('x') as stream:
  for i,record in enumerate(records):
   data,gold,*_=base.prepare(record);messages,schema=base.messages(data,'original')
   size=len(json.dumps(messages,ensure_ascii=False).encode())
   if size>8192:raise ValueError('message_byte_budget')
   for condition in (['biased','neutral'] if i%2==0 else ['neutral','biased']):
    row=dict(fixture=record['name'],split='used_decoder_diagnostic',condition=condition,profile=PROFILES[condition],complete=False,exact=None,false_merge=None,false_split=None,calls=1,payload_sha256=base.digest(data),schema_sha256=base.digest(schema),prompt_sha256=base.digest(messages),message_bytes=size)
    request=dict(schema=schema,messages=messages,context_tokens=16384,output_tokens=1536,model=base.base.api.MODEL+'@'+base.base.api.REVISION,model_path=base.MODEL,generation_profile=PROFILES[condition])
    start=time.perf_counter()
    try:
     remaining=deadline-time.monotonic()
     if remaining<=0:raise ValueError('whole_budget')
     proc=subprocess.run([str(HELPER)],input=(json.dumps(request)+'\n').encode(),capture_output=True,timeout=min(120,remaining))
     if proc.returncode:raise ValueError('helper_failure')
     result=json.loads(proc.stdout,object_pairs_hook=base.base.api.strict)
     row.update(stop=result.get('stop_reason'),input_tokens=result.get('benchmark_input_tokens'),output_tokens=result.get('benchmark_output_tokens'))
     if not result.get('ok') or row['stop']!='completed':raise ValueError('backend_not_completed')
     answer=json.loads(result['generated_json'],object_pairs_hook=base.base.api.strict)
     if set(answer)!={'scores','unresolved'} or type(answer['unresolved']) is not bool:raise ValueError('invalid_schema')
     row['scores']=answer['scores'];row['unresolved']=answer['unresolved']
     if answer['unresolved']:raise ValueError('unresolved')
     membership,solver=partition(sorted(gold),answer['scores'])
     pairs=list(itertools.combinations(gold,2));fm=sum(membership[a]==membership[b] and gold[a]!=gold[b] for a,b in pairs);fs=sum(membership[a]!=membership[b] and gold[a]==gold[b] for a,b in pairs)
     row.update(complete=True,exact=fm==fs==0,false_merge=fm,false_split=fs,membership=membership,solver=solver,reason='accepted')
    except subprocess.TimeoutExpired:row.update(reason='timeout',stop='timeout')
    except (ValueError,TypeError,KeyError) as exc:row['reason']=str(exc)
    row['wall_seconds']=time.perf_counter()-start;stream.write(json.dumps(row)+'\n');stream.flush();print(json.dumps(row),flush=True)
if __name__=='__main__':main()
