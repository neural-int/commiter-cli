"""D final reachable A2 audit on reserved public and controlled input families."""
import hashlib,json,pathlib,resource,subprocess,sys,time
from d_cases import cases
from coarse import prepare
from planner import prepare as fine_prepare,encode,fallback,replay
from evaluate_c2 import gold,canonical,authoritative
from evaluate_a import metrics
from evaluate_d import strat,BINARY
from evaluate_b import HELPER,HELPER_SHA
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def sha(b):return hashlib.sha256(b).hexdigest()
def eval_units(case):return fine_prepare(case['files']) if 'intent_states' in case else prepare(case['files'])
def quality(case,units,groups,reference):
 if not case['primary']:return dict(exact=None,false_merge=None,false_split=None,commits=len(groups))
 return metrics(reference,canonical(units,groups,reference),case['gold'])
def preflight():
 rows=[]
 for c in cases():
  if 'intent_states' in c:
   ref=eval_units(c);c['gold']=gold(c,ref);c['metric_granularity']='utf8-edit-atom'
  elif c['primary']:
   ref=eval_units(c);c['gold']={u['id']:c['gold_file_purposes'][u['file']] for u in ref}
  c['input_sha256']=sha(encode(c['files']));rows.append(c)
 sources=['d_cases.py','evaluate_d2.py','coarse.py','planner.py','evaluate_c2.py','c_cases.py','adaptive.py','vendor151/evaluate.py','vendor151/inline.py','vendor151/remeasure.py']
 fixed=dict(contract=dict(candidate='A2 only. B NO-GO, C raw split cannot pass unchanged authoritative file assignment; no automatic promotion',dataset='reserved stretchr/testify public histories (not used in A/B/C), disjoint-history superpositions, plus new authored controls. Components/projections correlated, no independent-population confidence claimed',primary='unique-identifiable fixed complete intent partitions only, unknown/non-observable excluded BEFORE inference; exact never counts fallback unless actual gold match',target=.8,target_large=.8,rows=len(rows),primary_rows=sum(c['primary'] for c in rows),orders=['forward','reverse'],a_order_check='compare unchanged final plan hash after input reversal; replay both commit orders once per case, deterministic A2 has no model samples',comparators=[dict(architecture=a,name=n,reverse=r) for a,n,orders in [('baseline','mock-sentinel',[False,True]),('baseline','mock-plus-assert-negative',[False,True]),('h23','mock-plus-assert-negative',[False,True]),('h23','formatter-eight',[False]),('h23','public-sixteen-mixed',[False])] for r in orders],
 baseline_revision='f275d953f0c15452e9281d1c678cb9bb20adcbbb',h23_revision='e88f61bb06d8e609d5ad7fbcde1a40281f997e89',binary_sha256=sha(BINARY.read_bytes()),helper_sha256=HELPER_SHA,baseline_model='mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2',h23_model='mlx-community/Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192',decoder='unchanged historical adapter and bounded profiles from iteration-4; temp0/topP1/topK0/seed144, context16384',budgets='unchanged A2 16 files,1MiB/20kline/256line or fine units per file,4096total; no post-hoc relaxations',max_comparator_calls=64,total_comparator_seconds=1200,per_process_seconds=245,retries=0,stops='invalid assignment/source/staging: fail closed; unknown: legal file fallback; model resource stops recorded not exact. Token null never imputed as actual zero',safety='all staging disposable; no downloaded source tests; no secrets/external inference; no production change'),rows=rows,source_sha256={p:sha((ROOT/'tools/benchmark156'/p).read_bytes()) for p in sources})
 if sha(HELPER.read_bytes())!=HELPER_SHA:raise ValueError('helper_digest')
 with (OUT/'iteration-7-preregistered.json').open('x') as f:json.dump(fixed,f,ensure_ascii=False,indent=2);f.write('\n')
 print(json.dumps(dict(cases=len(rows),primary=sum(c['primary'] for c in rows),strata={s:sum(c['primary'] and strat(len(c['files']))==s for c in rows) for s in ('1-4','5-8','9-16')},comparators=len(fixed['contract']['comparators']))))
def run():
 fixed=json.loads((OUT/'iteration-7-preregistered.json').read_text())
 for p,d in fixed['source_sha256'].items():
  if sha((ROOT/'tools/benchmark156'/p).read_bytes())!=d:raise ValueError('source_digest')
 if sha(BINARY.read_bytes())!=fixed['contract']['binary_sha256'] or sha(HELPER.read_bytes())!=HELPER_SHA:raise ValueError('binary_digest')
 with (OUT/'iteration-7-results.jsonl').open('x') as out:
  for c in fixed['rows']:
   start=time.perf_counter();files=c['files'];row=dict(name=c['name'],architecture='A2',files=len(files),stratum=strat(len(files)),primary=c['primary'],gold_status=c['gold_status'],tags=c['tags'],input_sha256=c['input_sha256'],metric_granularity=c['metric_granularity'],complete=False,exact=False if c['primary'] else None,false_merge=None,false_split=None,calls=0,input_tokens=0,output_tokens=0)
   try:
    units=prepare(files);groups=fallback(files,units);reverse_groups=fallback(list(reversed(files)),prepare(list(reversed(files))))
    if encode(groups)!=encode(reverse_groups):raise ValueError('order_instability')
    reference=eval_units(c);row.update(quality(c,units,groups,reference));row.update(complete=True,units=len(units),evaluation_units=len(reference),order_stable=True,staging=replay(files,units,groups),validation=authoritative(files,groups),groups=groups,fallback_rate=1.,compose_rate=0.,file_singleton_commit_rate=1.,unknown_continuation=True,reason='accepted')
    if not row['validation']['valid']:raise ValueError('invalid_plan')
   except (ValueError,KeyError,TypeError) as exc:row.update(complete=False,exact=False if c['primary'] else None,reason=str(exc),stop_kind='structural_resource')
   row['wall_seconds']=time.perf_counter()-start
   out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush();print(json.dumps({k:v for k,v in row.items() if k not in ('groups','staging')},ensure_ascii=False),flush=True)
  deadline=time.monotonic()+1200;count=0
  for spec in fixed['contract']['comparators']:
   c=next(c for c in fixed['rows'] if c['name']==spec['name']);files=c['files'];start=time.perf_counter();row=dict(**spec,files=len(files),stratum=strat(len(files)),primary=c['primary'],input_sha256=c['input_sha256'],metric_granularity=c['metric_granularity'],exact=False,complete=False,false_merge=None,false_split=None)
   try:
    if count>=64 or time.monotonic()>=deadline:raise ValueError('total_budget')
    expected=[[fid for fid,p in c['gold_file_purposes'].items() if p==purpose] for purpose in sorted(set(c['gold_file_purposes'].values()))]
    req=dict(**spec,files=files,expected=expected)
    proc=subprocess.run([str(BINARY)],input=encode(req),capture_output=True,timeout=min(245,deadline-time.monotonic()))
    if proc.returncode:raise ValueError('adapter_exit')
    observed=json.loads(proc.stdout);row['observed']=observed;calls=observed.get('calls') or [];count+=len(calls)
    row.update(calls=len(calls),input_tokens_lower_bound=sum(v.get('input_tokens') or 0 for v in calls),output_tokens_lower_bound=sum(v.get('output_tokens') or 0 for v in calls),token_telemetry_complete=all(v.get('input_tokens') is not None and v.get('output_tokens') is not None for v in calls),reason=observed.get('plan_stop') or observed.get('reason') or 'accepted')
    if observed.get('complete_assignment') and observed.get('plan_valid'):
     units=prepare(files);groups=[dict(file_ids=g,unit_ids=[u['id'] for u in units if u['file'] in g]) for g in observed['groups']]
     row.update(quality(c,units,groups,eval_units(c)));row.update(complete=True,staging=replay(files,units,groups))
    else:row['stop_kind']='recorded_model_or_metadata_failure'
   except (ValueError,KeyError,TypeError,subprocess.TimeoutExpired) as exc:row.update(reason=type(exc).__name__+':'+str(exc),stop_kind='resource_or_structural')
   row['wall_seconds']=time.perf_counter()-start;row['peak_rss_bytes_cumulative']=resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss
   out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush();print(json.dumps({k:v for k,v in row.items() if k not in ('observed','staging')},ensure_ascii=False),flush=True)
if __name__=='__main__':preflight() if sys.argv[1]=='preflight' else run()
