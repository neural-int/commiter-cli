"""One preregistered B comparison. No threshold/model/prompt tuning."""
import copy,hashlib,json,pathlib,subprocess,time
from planner import prepare,fallback,encode,replay
from selective import proposals,payload,schema,finish,PROMPT
from evaluate_a import metrics
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
HELPER=pathlib.Path('/private/tmp/issue153-neutral-score-helper/.build/release/commiter-mlx-helper')
HELPER_SHA='bc61461f860957780394650f8b46fab2b737f9931917d0194a0f736e671b41a4'
MODEL='mlx-community/Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192'
MODEL_PATH='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'

def snapshots(prefix):return json.loads(next((OUT/'corpus').glob(prefix+'*-snapshots.json')).read_text())
def cases():
    unwrap=snapshots('BurntSushi-toml-9fe9ada4');doc=snapshots('BurntSushi-toml-b66ba2a0')
    for name,records in [('unwrap-positive',[unwrap]),('unwrap-doc-negative',[unwrap,doc])]:
        files=[];purpose={}
        for i,r in enumerate(records):
            for source in r['files']:
                f=dict(source,id=f'F{len(files)+1:03}');files.append(f);purpose[f['id']]=i
        units=prepare(files);gold={u['id']:purpose[u['file']] for u in units}
        yield name,files,units,gold,[r['commit'] for r in records]

def sha(b):return hashlib.sha256(b).hexdigest()
def strict(pairs):
    result={}
    for k,v in pairs:
        if k in result:raise ValueError('duplicate_json_key')
        result[k]=v
    return result

def preflight():
    if sha(HELPER.read_bytes())!=HELPER_SHA:raise ValueError('helper_digest')
    rows=[]
    for name,files,units,gold,commits in cases():
        candidates=proposals(files)
        for reverse in (False,True):
            data=payload(files,candidates,reverse);contract=schema(data)
            messages=[dict(role='system',content=PROMPT+encode(contract).decode()),dict(role='user',content=encode(data).decode())]
            if len(encode(messages))>32768:raise ValueError('message_budget')
            rows.append(dict(name=name,reverse=reverse,files=files,gold=gold,source_commits=commits,candidates=candidates,messages=messages,schema=contract,input_sha256=sha(encode(files)),prompt_sha256=sha(encode(messages))))
    record=dict(contract=dict(stage='B only; D holdout reserved',model=MODEL,helper_sha256=HELPER_SHA,profile='bounded-routed-grammar-neutral',context_tokens=16384,output_tokens=1536,temperature=0,top_p=1,top_k=0,seed=144,native_thought_tokens=0,max_candidates=8,max_calls=4,total_seconds=480,per_call_seconds=120,retries=0,margin=1,accept_score=3,full_member_coverage=True,threshold_tuning=False,
        capability_go='all responses valid; FS improves on both orders of positive; no FM introduced on negative; exact improves; order-invariant partition; staging valid',
        limitations='2 unseen-public-source workloads, one is controlled superposition of disjoint files from two historical changes; not population performance or D holdout. Single/mixed/unknown are model hypotheses, not author-intent facts.'),rows=rows,source_sha256={str(p.relative_to(ROOT)):sha(p.read_bytes()) for p in [pathlib.Path(__file__),ROOT/'tools/benchmark156/selective.py',ROOT/'tools/benchmark156/planner.py']})
    with (OUT/'iteration-2-preregistered.json').open('x') as f:json.dump(record,f,ensure_ascii=False,indent=2);f.write('\n')
    print(json.dumps(dict(calls=4,cases=[dict(name=r['name'],reverse=r['reverse'],candidates=len(r['candidates']),message_bytes=len(encode(r['messages']))) for r in rows])))

def run():
    fixed=json.loads((OUT/'iteration-2-preregistered.json').read_text())
    for path,digest in fixed['source_sha256'].items():
        if sha((ROOT/path).read_bytes())!=digest:raise ValueError('source_digest')
    if sha(HELPER.read_bytes())!=HELPER_SHA:raise ValueError('helper_digest')
    deadline=time.monotonic()+480
    with (OUT/'iteration-2-results.jsonl').open('x') as stream:
        for spec in fixed['rows']:
            files=spec['files'];units=prepare(files);baseline=fallback(files,units)
            row=dict(name=spec['name'],reverse=spec['reverse'],calls=1,baseline=metrics(units,baseline,spec['gold']),prompt_sha256=spec['prompt_sha256'],valid=False)
            start=time.perf_counter()
            try:
                request=dict(schema=spec['schema'],messages=spec['messages'],model=MODEL,model_path=MODEL_PATH,generation_profile='bounded-routed-grammar-neutral',context_tokens=16384,output_tokens=1536)
                remaining=deadline-time.monotonic()
                if remaining<=0:raise ValueError('total_budget')
                proc=subprocess.run([str(HELPER)],input=encode(request)+b'\n',capture_output=True,timeout=min(120,remaining))
                if proc.returncode:raise ValueError('helper_exit')
                response=json.loads(proc.stdout,object_pairs_hook=strict)
                row.update(stop=response.get('stop_reason'),input_tokens=response.get('benchmark_input_tokens'),output_tokens=response.get('benchmark_output_tokens'))
                if not response.get('ok') or row['stop']!='completed':raise ValueError('backend_not_completed')
                answer=json.loads(response['generated_json'],object_pairs_hook=strict);row['answer']=answer
                groups=finish(files,units,spec['candidates'],answer)
                row.update(valid=True,quality=metrics(units,groups,spec['gold']),naive_highest=metrics(units,finish(files,units,spec['candidates'],answer,True),spec['gold']),groups=groups,fallback_rate=sum(c['fallback'] for c in groups)/len(groups),staging=replay(files,units,groups),reason='accepted')
            except subprocess.TimeoutExpired:row.update(reason='timeout',stop='timeout')
            except (ValueError,KeyError,TypeError) as exc:row['reason']=str(exc)
            row['wall_seconds']=time.perf_counter()-start
            stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush()
            print(json.dumps({k:v for k,v in row.items() if k not in ('answer','groups')},ensure_ascii=False),flush=True)

if __name__=='__main__':
    import sys
    preflight() if sys.argv[1]=='preflight' else run()
