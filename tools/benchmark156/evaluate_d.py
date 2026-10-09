"""Final reachable-subarchitecture audit. Invalid runs cannot count as exact."""
import base64,copy,hashlib,json,pathlib,resource,subprocess,time,sys
from planner import encode,prepare,fallback,replay
from evaluate_a import metrics
from evaluate_b import HELPER,HELPER_SHA
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
BINARY=pathlib.Path('/private/tmp/benchmark156-history')

def sha(data):return hashlib.sha256(data).hexdigest()
def snapshots(prefix):return json.loads(next((OUT/'corpus').glob(prefix+'*-snapshots.json')).read_text())
def cases():
    migration=snapshots('google-go-cmp-b133f1f1')
    for n in (1,4,8,16):
        files=copy.deepcopy(migration['files'][:n])
        yield dict(name='any-migration-'+str(n),files=files,source_commits=[migration['commit']],gold_file_groups=[[f['id'] for f in files]],
            gold_status='identifiable: all selected diffs replace the same interface{} type with its any alias; corresponding examples/tests are updated; no unrelated edited feature is present',
            selected_projection=True,correlated_family='same-public-commit-projections')
    for prefix,name in [('jmoiron-sqlx-421d1cdb','select-reset'),('jmoiron-sqlx-25a51134','azuresql-bind')]:
        r=snapshots(prefix);files=copy.deepcopy(r['files'])
        yield dict(name=name,files=files,source_commits=[r['commit']],gold_file_groups=[[f['id'] for f in files]],
            gold_status='identifiable from implementation change and corresponding changed assertions (Select resets destination length), or one driver binding addition; author commit only corroborates source',selected_projection=False,correlated_family=name)

def preflight():
    rows=list(cases());sources=[pathlib.Path(__file__),ROOT/'tools/benchmark156/history_adapter.go.txt',ROOT/'tools/benchmark156/build_history_adapter.py',ROOT/'tools/benchmark156/planner.py']
    contract=dict(primary='end-to-end exact partition: failed/invalid/timeout/budget-rejected plan is false; FM/FS null if no complete prediction',target_exact=.80,
        integrated_candidate='A only; B NO-GO, C semantic ability unproven. Never integrate B/C',
        strata=['1-4','5-8','9-16'],a_cases=6,a_llm_calls=0,comparators=dict(baseline=['any-migration-4','select-reset'],h23=['any-migration-4','any-migration-8','any-migration-16']),orders=['forward','reverse'],
        baseline_revision='f275d953f0c15452e9281d1c678cb9bb20adcbbb',h23_revision='e88f61bb06d8e609d5ad7fbcde1a40281f997e89',
        helper_sha256=HELPER_SHA,historical_binary_sha256=sha(BINARY.read_bytes()),
        baseline_model='mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2',h23_group_model='mlx-community/Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192',
        baseline_profiles=['bounded-grouping','bounded-category','bounded-text'],h23_profiles=['bounded-routed-grouping','bounded-category','bounded-text'],
        context_tokens=16384,phase_output_tokens={'bounded-grouping':768,'bounded-category':512,'bounded-text':768,'bounded-routed-grouping':1536},
        temperature=0,top_p=1,top_k=0,seed=144,baseline_cycle_seconds=120,h23_cycle_seconds_large=240,max_backend_calls=30,total_comparator_seconds=1200,retries=0,
        budgets='A retains 1MiB/20kline/256CU/file, 4096inline byte/4096total CU/16files; no relaxation after rejection',
        measurement='all A cases; comparators in representative strata only, not full factorial. Two presentation orders; one model sample per order. No population confidence claimed.',
        metadata='A unknown chore text preserved; compose unsupported and not injected. H23/baseline must pass their unchanged final metadata and planning.Validate.',
        excluded_gold='toml upstream test import selected16: multiple plausible intent partitions; separately recorded, never optimized into single gold',
        acceptance='>=80% overall and 9-16 plus safety/time gates. Failure is NO-GO, never promotion by coverage or oracle',
        safety='temporary index only; no source repo tests; no source repo/index mutation; no cloud transfer; local cached models only')
    for r in rows:r['input_sha256']=sha(encode(r['files']))
    fixed=dict(contract=contract,rows=rows,source_sha256={str(p.relative_to(ROOT)):sha(p.read_bytes()) for p in sources})
    with (OUT/'iteration-4-preregistered.json').open('x') as f:json.dump(fixed,f,ensure_ascii=False,indent=2);f.write('\n')
    print(json.dumps(dict(cases=len(rows),comparators=contract['comparators'],max_backend_calls=30)))

def strat(n):return '1-4' if n<=4 else '5-8' if n<=8 else '9-16'
def unit_gold(units,groups):
    by_file={fid:i for i,g in enumerate(groups) for fid in g}
    return {u['id']:by_file[u['file']] for u in units}
def group_units(units,groups):return [dict(file_ids=g,unit_ids=[u['id'] for u in units if u['file'] in g]) for g in groups]

def run():
    fixed=json.loads((OUT/'iteration-4-preregistered.json').read_text())
    for path,digest in fixed['source_sha256'].items():
        if sha((ROOT/path).read_bytes())!=digest:raise ValueError('source_digest')
    if sha(HELPER.read_bytes())!=HELPER_SHA or sha(BINARY.read_bytes())!=fixed['contract']['historical_binary_sha256']:raise ValueError('binary_digest')
    requests=[]
    with (OUT/'iteration-4-results.jsonl').open('x') as out:
        for case in fixed['rows']:
            start=time.perf_counter();files=case['files'];row=dict(name=case['name'],architecture='A',stratum=strat(len(files)),files=len(files),input_sha256=case['input_sha256'],exact=False,complete=False,valid_staging=False,false_merge=None,false_split=None,stop_kind=None,calls=0,input_tokens=0,output_tokens=0)
            try:
                units=prepare(files);commits=fallback(files,units);gold=unit_gold(units,case['gold_file_groups'])
                row.update(metrics(units,commits,gold));row.update(complete=True,valid_staging=True,staging=replay(files,units,commits),units=len(units),fallback_rate=1.0,compose_rate=0.0,file_singleton_commit_rate=1.0,unknown_continuation=True,groups=[c['file_ids'] for c in commits],reason='accepted')
                plan=dict(schema_version=1,commits=[{k:c[k] for k in ('type','scope','breaking','summary','file_ids')} for c in commits])
                requests.append(dict(candidate=plan,file_ids=[f['id'] for f in files]))
                row['authoritative_validation']=json.loads(subprocess.run(['/private/tmp/benchmark156-validate'],input=encode([requests[-1]]),capture_output=True,check=True).stdout)[0]
                if not row['authoritative_validation']['valid']:raise ValueError('invalid_plan')
            except ValueError as exc:row.update(reason=str(exc),stop_kind='structural_resource',complete=False,exact=False,valid_staging=False)
            row['wall_seconds']=time.perf_counter()-start
            out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush();print(json.dumps(row,ensure_ascii=False),flush=True)
        deadline=time.monotonic()+1200;calls=0
        for architecture,names in fixed['contract']['comparators'].items():
            for case in fixed['rows']:
                if case['name'] not in names:continue
                for reverse in (False,True):
                    req=dict(name=case['name'],architecture=architecture,reverse=reverse,files=case['files'],expected=case['gold_file_groups'])
                    row=dict(name=case['name'],architecture=architecture,reverse=reverse,stratum=strat(len(case['files'])),files=len(case['files']),input_sha256=case['input_sha256'],exact=False,complete=False,valid_staging=False,false_merge=None,false_split=None)
                    start=time.perf_counter()
                    try:
                        remaining=deadline-time.monotonic()
                        if remaining<=0 or calls>=30:raise ValueError('total_budget')
                        proc=subprocess.run([str(BINARY)],input=encode(req),capture_output=True,timeout=min(245,remaining))
                        if proc.returncode:raise ValueError('historical_adapter_exit')
                        observed=json.loads(proc.stdout);row['observed']=observed
                        backend_calls=observed.get('calls') or [];calls+=len(backend_calls)
                        row.update(calls=len(backend_calls),input_tokens=sum(c.get('input_tokens') or 0 for c in backend_calls),output_tokens=sum(c.get('output_tokens') or 0 for c in backend_calls),token_telemetry_complete=all(c.get('input_tokens') is not None and c.get('output_tokens') is not None for c in backend_calls),reason=observed.get('plan_stop') or observed.get('reason') or 'accepted')
                        row['complete']=observed.get('complete_assignment',False) and observed.get('plan_valid',False)
                        row['exact']=row['complete'] and observed.get('exact') is True
                        row['file_pairs']={k:observed.get(k) for k in ('false_merge_pairs','false_split_pairs')}
                        if row['complete']:
                            try:
                                units=prepare(case['files']);groups=group_units(units,observed['groups']);q=metrics(units,groups,unit_gold(units,case['gold_file_groups']))
                                row.update(false_merge=q['false_merge'],false_split=q['false_split'],commits=q['commits'],units=len(units),staging=replay(case['files'],units,groups),valid_staging=True)
                            except ValueError as exc:
                                # Comparator can retain a complete file plan even when A's leaf extractor cannot represent it.
                                row.update(unit_stage_audit_reason=str(exc),valid_staging=False)
                        if not row['complete']:row['stop_kind']='semantic_or_model' if row['reason']!='total_budget' else 'resource'
                    except subprocess.TimeoutExpired:row.update(reason='timeout',stop_kind='resource')
                    except (ValueError,KeyError,TypeError) as exc:row.update(reason=str(exc),stop_kind='structural_or_resource')
                    row['wall_seconds']=time.perf_counter()-start
                    row['process_peak_rss_bytes_cumulative']=resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss
                    out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush()
                    print(json.dumps({k:v for k,v in row.items() if k not in ('observed','staging')},ensure_ascii=False),flush=True)
if __name__=='__main__':preflight() if sys.argv[1]=='preflight' else run()
