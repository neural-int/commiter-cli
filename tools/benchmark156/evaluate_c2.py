"""Independent C semantic/compatibility audit; raw capability is not final quality."""
import base64,hashlib,json,pathlib,resource,subprocess,sys,time
from adaptive import request,finish
from c_cases import cases
from coarse import prepare,refine
from planner import prepare as fine_prepare,encode,fallback,replay,validate
from evaluate_a import metrics
from evaluate_b import HELPER,HELPER_SHA,MODEL,MODEL_PATH,strict
from remeasure import matching_subsets
from selective import proposals,payload,schema,finish as b_finish,PROMPT as B_PROMPT
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def sha(data):return hashlib.sha256(data).hexdigest()
def gold(case,fine):
    import itertools
    alternatives=[]
    for intent,states in case['intent_states'].items():
        for f in case['files']:
            own=[u for u in fine if u['file']==f['id']]
            hits=matching_subsets(base64.b64decode(f['before_b64']),own,base64.b64decode(states[f['id']]))
            if not hits:raise ValueError('gold_unrepresentable')
            alternatives.append([(intent,hit) for hit in hits])
    variants=[]
    for combination in itertools.product(*alternatives):
        result={};valid=True
        for intent,hit in combination:
            for uid in hit:
                if uid in result:valid=False
                result[uid]=intent
        if valid and set(result)=={u['id'] for u in fine} and result not in variants:variants.append(result)
    if not variants and case['name']=='afero-natural-shared-test':
        reference={}
        for f in case['files']:
            after=base64.b64decode(f['after_b64']);start=end=-1
            if f['path']=='composite_test.go':
                start=after.index(b'func TestCacheOnReadFsWriteReader(');end=after.index(b'func NewTempOsBaseFs(',start)
            for u in fine:
                if u['file']==f['id']:reference[u['id']]='cache' if start<=u['new_span'][0]<end else 'close'
        case['gold_status']='non-unique repeated braces/blank-line ownership: bounded two-witness DP cannot establish complete compatible intent states. Source-span reference partition is diagnostic only; excluded from exact primary'
        case['gold_variants']=[reference];case['primary']=False
        return reference
    if not variants:raise ValueError('gold_incomplete')
    if len(variants)>1:
        case['gold_status']='multiple-valid blank-line ownership in public mixed test additions; excluded from unique-gold primary; all fixed complete variants retained'
    case['gold_variants']=variants;case['primary']=len(variants)==1
    return variants[0]

def canonical(units,groups,fine):
    by_id={u['id']:u for u in units};output=[]
    for group in groups:
        ids=[]
        for uid in group['unit_ids']:
            parent=by_id[uid]
            ids.extend(v['id'] for v in fine if v['file']==parent['file'] and all(parent[k][0]<=v[k][0]<=v[k][1]<=parent[k][1] for k in ('old_span','new_span')))
        output.append(dict(file_ids=group['file_ids'],unit_ids=ids))
    validate(fine,output)
    return output

def authoritative(files,groups):
    plan=dict(schema_version=1,commits=[{k:c[k] for k in ('type','scope','breaking','summary','file_ids')} for c in groups])
    return json.loads(subprocess.run(['/private/tmp/benchmark156-validate'],input=encode([dict(candidate=plan,file_ids=[f['id'] for f in files])]),capture_output=True,check=True).stdout)[0]

def preflight():
    rows=[]
    for case in cases():
        fine=fine_prepare(case['files']);case['gold']=gold(case,fine);case['input_sha256']=sha(encode(case['files']))
        coarse=prepare(case['files'])
        for reverse in (False,True):
            s,m,_=request(case['files'],coarse,reverse)
            rows.append(dict(case=case,reverse=reverse,coarse_units=len(coarse),fine_units=len(fine),first_prompt_sha256=sha(encode(m)),first_message_bytes=len(encode(m))))
    sources=['adaptive.py','c_cases.py','coarse.py','evaluate_c2.py','planner.py','selective.py','vendor151/evaluate.py','vendor151/inline.py','vendor151/remeasure.py']
    fixed=dict(contract=dict(candidate='A2+C; B remains NO-GO. Global ownership assignment is a distinct hypothesis from ordinal group proposal ranking',primary='final plan exact, not raw split; fallback must actually match gold',model=MODEL,helper_sha256=HELPER_SHA,profile='bounded-routed-grammar-neutral',context_tokens=16384,output_tokens=1536,thought_tokens=0,temperature=0,top_p=1,top_k=0,seed=144,retries=0,max_calls=36,max_calls_per_adaptive=2,per_call_seconds=60,total_seconds=1200,max_units_per_file=256,max_files=16,refinement='only first response status=mixed triggers bounded whole-file inline refinement; no gold trigger; unknown/refinement refusal preserves coarse file fallback',compatibility='raw partial-file plan is tested by unchanged planning.Validate. Repeated file ownership rejected; final candidate then A2 fallback, no CU-as-file validator bypass',b_comparators=['afero-natural-shared-test','partial-cross-file'],acceptance='C GO only if mixed FM falls without FS rise, both orders stable, complete byte ownership/staging and authoritative compatibility. Overall exact>=.8 is final D requirement',dataset='1 new public natural mixed workload separately classified for multiple-valid whitespace ownership plus 7 new authored controlled workloads; controlled states define independent purposes, not population estimate; reserved D sources not used'),rows=rows,source_sha256={p:sha((ROOT/'tools/benchmark156'/p).read_bytes()) for p in sources})
    if sha(HELPER.read_bytes())!=HELPER_SHA:raise ValueError('helper_digest')
    with (OUT/'iteration-6-preregistered.json').open('x') as f:json.dump(fixed,f,ensure_ascii=False,indent=2);f.write('\n')
    print(json.dumps(dict(observations=len(rows),units=[(r['case']['name'],r['coarse_units'],r['fine_units']) for r in rows[::2]],max_calls=36)))

def run():
    fixed=json.loads((OUT/'iteration-6-preregistered.json').read_text())
    for p,d in fixed['source_sha256'].items():
        if sha((ROOT/'tools/benchmark156'/p).read_bytes())!=d:raise ValueError('source_digest')
    if sha(HELPER.read_bytes())!=HELPER_SHA:raise ValueError('helper_digest')
    deadline=time.monotonic()+1200;count=0
    def call(schema,messages,trace):
        nonlocal count
        if count>=36 or time.monotonic()>=deadline:raise ValueError('total_budget')
        count+=1;start=time.perf_counter()
        req=dict(schema=schema,messages=messages,model=MODEL,model_path=MODEL_PATH,generation_profile='bounded-routed-grammar-neutral',context_tokens=16384,output_tokens=1536)
        r=subprocess.run([str(HELPER)],input=encode(req)+b'\n',capture_output=True,timeout=min(60,deadline-time.monotonic()))
        if r.returncode:raise ValueError('helper_exit')
        response=json.loads(r.stdout,object_pairs_hook=strict);trace.append(dict(prompt_sha256=sha(encode(messages)),wall_seconds=time.perf_counter()-start,response=response))
        if not response.get('ok') or response.get('stop_reason')!='completed':raise ValueError('backend_not_completed')
        return json.loads(response['generated_json'],object_pairs_hook=strict)
    with (OUT/'iteration-6-results.jsonl').open('x') as out:
        for spec in fixed['rows']:
            case=spec['case'];files=case['files'];fine=fine_prepare(files);coarse=prepare(files);baseline=fallback(files,coarse);truth=case['gold']
            architectures=['A2+C']+(['B'] if case['name'] in fixed['contract']['b_comparators'] else [])
            for architecture in architectures:
                start=time.perf_counter();calls_before=count;row=dict(name=case['name'],architecture=architecture,reverse=spec['reverse'],files=len(files),input_sha256=case['input_sha256'],baseline=metrics(fine,canonical(coarse,baseline,fine),truth),complete=False,exact=False,false_merge=None,false_split=None,calls=[],reason=None,refinement=[])
                units=coarse;groups=baseline
                try:
                    if architecture=='A2+C':
                        s,m,labels=request(files,units,spec['reverse']);answer=call(s,m,row['calls']);groups=finish(files,units,labels,answer);row['coarse_answer']=answer
                        mixed={fid for fid,status in answer['status'].items() if status=='mixed'}
                        if mixed:
                            expanded=[]
                            for f in files:
                                own=[u for u in units if u['file']==f['id']]
                                if f['id'] in mixed:
                                    newer,reason=refine(f);row['refinement'].append(dict(file=f['id'],reason=reason,before=len(own),after=len(newer)))
                                    if reason:row.setdefault('unsupported_refinement',[]).append(f['id'])
                                    own=newer
                                expanded.extend(own)
                            if {u['id'] for u in expanded}!={u['id'] for u in units}:
                                units=expanded;s,m,labels=request(files,units,spec['reverse']);answer=call(s,m,row['calls']);row['fine_answer']=answer;groups=finish(files,units,labels,answer)
                        row['raw_quality']=metrics(fine,canonical(units,groups,fine),truth);row['raw_validation']=authoritative(files,groups);row['raw_staging']=replay(files,units,groups)
                        if not row['raw_validation']['valid']:
                            units=coarse;groups=baseline;row['reason']='partial_file_contract_fallback'
                    else:
                        units=fine;candidates=proposals(files);data=payload(files,candidates,spec['reverse']);s=schema(data);m=[dict(role='system',content=B_PROMPT+encode(s).decode()),dict(role='user',content=encode(data).decode())]
                        answer=call(s,m,row['calls']);row['answer']=answer;groups=b_finish(files,units,candidates,answer)
                    final=authoritative(files,groups)
                    if not final['valid']:raise ValueError('final_invalid_plan')
                    row.update(metrics(fine,canonical(units,groups,fine),truth));row.update(complete=True,final_validation=final,final_staging=replay(files,units,groups),final_groups=groups,fallback_rate=sum(c['fallback'] for c in groups)/len(groups),compose_rate=0.,unknown_continuation=True)
                    row['reason']=row['reason'] or 'accepted'
                except (ValueError,KeyError,TypeError,subprocess.TimeoutExpired) as exc:
                    # Invalid model drafts do not mutate Git. A2 is the predetermined safe fallback.
                    row['draft_failure']=type(exc).__name__+':'+str(exc);units=coarse;groups=baseline
                    row.update(metrics(fine,canonical(units,groups,fine),truth));row.update(complete=True,final_validation=authoritative(files,groups),final_staging=replay(files,units,groups),final_groups=groups,fallback_rate=1.,compose_rate=0.,unknown_continuation=True,reason='model_failure_fallback')
                row['primary']=case['primary'];row['gold_status']=case['gold_status'];row['final_quality_variants']=[metrics(fine,canonical(units,groups,fine),v) for v in case['gold_variants']];row['reference_exact']=any(v['exact'] for v in row['final_quality_variants']);row['backend_calls']=count-calls_before;row['wall_seconds']=time.perf_counter()-start;row['peak_rss_bytes_cumulative']=resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss
                out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush();print(json.dumps({k:v for k,v in row.items() if k in ('name','architecture','reverse','baseline','raw_quality','raw_validation','exact','false_merge','false_split','complete','reason','draft_failure','backend_calls','wall_seconds','refinement')},ensure_ascii=False),flush=True)
if __name__=='__main__':preflight() if sys.argv[1]=='preflight' else run()
