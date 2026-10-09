"""Frozen B rule transfer across 6/8/16-file D inputs; no threshold fitting."""
import hashlib,json,pathlib,subprocess,sys,time
from coarse import prepare
from planner import encode,fallback,replay
from evaluate_a import metrics
from evaluate_b import HELPER,HELPER_SHA,MODEL,MODEL_PATH,strict
from selective import proposals,payload,schema,finish,PROMPT
from evaluate_c2 import canonical,authoritative
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def sha(b):return hashlib.sha256(b).hexdigest()
def preflight():
 d=json.loads((OUT/'iteration-7-preregistered.json').read_text());names=['authored-6-independent','formatter-eight','public-sixteen-mixed','authored-16-independent'];rows=[]
 for c in d['rows']:
  if c['name'] not in names:continue
  units=prepare(c['files']);cand=proposals(c['files'])
  for reverse in (False,True):
   data=payload(c['files'],cand,reverse);s=schema(data);m=[dict(role='system',content=PROMPT+encode(s).decode()),dict(role='user',content=encode(data).decode())]
   rows.append(dict(case=c,reverse=reverse,candidates=cand,schema=s,messages=m,prompt_sha256=sha(encode(m)),presentation_bytes=len(encode(m)),presentation_supported=len(encode(m))<=32768))
 sources=['selective.py','coarse.py','planner.py','evaluate_b2.py','vendor151/evaluate.py']
 fixed=dict(contract=dict(hypothesis='unchanged B semantic rules transferred onto verified A2 coarse ownership, no threshold/model/prompt changes',model=MODEL,helper_sha256=HELPER_SHA,profile='bounded-routed-grammar-neutral',context_tokens=16384,output_tokens=1536,temperature=0,top_p=1,top_k=0,seed=144,native_thought_tokens=0,max_candidates=8,max_calls=8,per_call_seconds=60,total_seconds=600,retries=0,accept_score=3,margin=1,presentation_bytes=32768,full_member_coverage=True,transfer_dataset='fixed D inputs: unseen by B model; repeated D components remain correlated diagnostic transfer, not a second independent population',acceptance='positive both orders FS improvement with no negative FM increase, complete validated stage, order-stable exact. Previous NO-GO is not erased by larger transfer',failure='noncompleted/schema/presentation rejection returns prevalidated A2 fallback; final invalid assignment/stage must stop'),rows=rows,source_sha256={p:sha((ROOT/'tools/benchmark156'/p).read_bytes()) for p in sources})
 with (OUT/'iteration-8-preregistered.json').open('x') as f:json.dump(fixed,f,ensure_ascii=False,indent=2);f.write('\n')
 print(json.dumps(dict(calls=len(rows),rows=[dict(name=r['case']['name'],reverse=r['reverse'],proposals=len(r['candidates']),bytes=r['presentation_bytes'],supported=r['presentation_supported']) for r in rows])))
def run():
 fixed=json.loads((OUT/'iteration-8-preregistered.json').read_text())
 for p,d in fixed['source_sha256'].items():
  if sha((ROOT/'tools/benchmark156'/p).read_bytes())!=d:raise ValueError('source_digest')
 if sha(HELPER.read_bytes())!=HELPER_SHA:raise ValueError('helper_digest')
 deadline=time.monotonic()+600;count=0
 with (OUT/'iteration-8-results.jsonl').open('x') as out:
  for spec in fixed['rows']:
   c=spec['case'];files=c['files'];units=prepare(files);base=fallback(files,units);truth=c['gold'];start=time.perf_counter();groups=base
   row=dict(name=c['name'],reverse=spec['reverse'],files=len(files),input_sha256=c['input_sha256'],metric_granularity='line-operation',baseline=metrics(units,base,truth),calls=0,input_tokens=None,output_tokens=None)
   try:
    if not spec['presentation_supported']:raise ValueError('presentation_budget')
    if count>=8 or time.monotonic()>=deadline:raise ValueError('total_budget')
    count+=1;row['calls']=1
    req=dict(schema=spec['schema'],messages=spec['messages'],model=MODEL,model_path=MODEL_PATH,generation_profile='bounded-routed-grammar-neutral',context_tokens=16384,output_tokens=1536)
    proc=subprocess.run([str(HELPER)],input=encode(req)+b'\n',capture_output=True,timeout=min(60,deadline-time.monotonic()))
    if proc.returncode:raise ValueError('helper_exit')
    response=json.loads(proc.stdout,object_pairs_hook=strict);row['response']=response;row.update(input_tokens=response.get('benchmark_input_tokens'),output_tokens=response.get('benchmark_output_tokens'))
    if not response.get('ok') or response.get('stop_reason')!='completed':raise ValueError('backend_not_completed')
    answer=json.loads(response['generated_json'],object_pairs_hook=strict);row['answer']=answer;groups=finish(files,units,spec['candidates'],answer);naive=finish(files,units,spec['candidates'],answer,True);row['naive_highest']=metrics(units,naive,truth);row['reason']='accepted'
   except (ValueError,KeyError,TypeError,subprocess.TimeoutExpired) as exc:row.update(reason=type(exc).__name__+':'+str(exc));groups=base
   validation=authoritative(files,groups)
   if not validation['valid']:raise ValueError('final_invalid_plan')
   row.update(metrics(units,groups,truth));row.update(complete=True,validation=validation,staging=replay(files,units,groups),groups=groups,fallback_rate=sum(g['fallback'] for g in groups)/len(groups),compose_rate=0.,unknown_continuation=True,wall_seconds=time.perf_counter()-start)
   out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush();print(json.dumps({k:v for k,v in row.items() if k not in ('response','answer','groups','staging')},ensure_ascii=False),flush=True)
if __name__=='__main__':preflight() if sys.argv[1]=='preflight' else run()
