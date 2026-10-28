"""H182-011 fixed-history real three-phase comparison. Explicit run only; no retries."""
import os, tempfile
import argparse,ctypes,datetime,hashlib,importlib.util,itertools,json,os,pathlib,plistlib,resource,shutil,signal,statistics,subprocess,threading,time
ROOT=pathlib.Path(__file__).resolve().parents[2]
BENCHMARK_ROOT=pathlib.Path(os.environ.get('COMMITER_BENCHMARK_ROOT', tempfile.gettempdir()))
GOLD=[{'name':'A','type':'fix','files':['deadlinewidget/mail/store_reader/normalization.py','tests/unit/mail/store_reader/test_normalization.py']},
      {'name':'B','type':'feat','files':['deadlinewidget/app/ui/widget_view.py','tests/unit/app/ui/test_widget_view.py']},
      {'name':'C','type':'docs','files':['README.md']}]

def save(p,v):
    p.write_text(json.dumps(v,ensure_ascii=False,indent=2)+'\n');p.chmod(0o600)
def sha(p):
    h=hashlib.sha256()
    with pathlib.Path(p).open('rb') as f:
        for block in iter(lambda:f.read(4*1024*1024),b''):h.update(block)
    return h.hexdigest()
def command(argv):
    start=time.monotonic()
    try:
        p=subprocess.run(argv,text=True,capture_output=True,timeout=4)
        return dict(exit=p.returncode,stdout=p.stdout,stderr=p.stderr,seconds=time.monotonic()-start)
    except Exception as e:return dict(error=type(e).__name__,seconds=time.monotonic()-start)

class Monitor:
    def __init__(self,out):
        spec=importlib.util.spec_from_file_location('sampler',ROOT/'tools/benchmark182/monitor_v2.py')
        self.sampler=importlib.util.module_from_spec(spec);spec.loader.exec_module(self.sampler)
        self.out=out;self.done=threading.Event();self.fatal=[];self.active=None;self.samples=0;self.durations=[];self.stage='preparation';self.previous_cpu={}
    def loop(self):
        while not self.done.is_set():
            start=time.monotonic()
            try:
                row=self.sampler.sample();row['stage']=self.stage
                process=command(['ps','-axo','pid=,ppid=,%cpu=,rss=,vsz=,time=,comm='])
                allowed=[]
                for line in process.get('stdout','').splitlines():
                    fields=line.split(None,6)
                    if len(fields)==7 and (fields[6].endswith('/commiter-mlx-helper') or fields[6].endswith('/evaluation.test') or fields[6].endswith('/codex')):
                        allowed.append(dict(pid=int(fields[0]),ppid=int(fields[1]),cpu_lifetime_percent=float(fields[2]),rss_kib=int(fields[3]),vsz_kib=int(fields[4]),cpu_time=fields[5],executable=pathlib.Path(fields[6]).name))
                now=time.monotonic()
                for item in allowed:
                    parts=item['cpu_time'].replace('-',':').split(':')
                    seconds=0.0
                    for part in parts:seconds=seconds*60+float(part)
                    previous=self.previous_cpu.get(item['pid'])
                    item['cpu_interval_percent']=100*(seconds-previous[1])/(now-previous[0]) if previous and now>previous[0] else None
                    self.previous_cpu[item['pid']]=(now,seconds)
                row['processes']=allowed;row['process_probe_seconds']=process['seconds']
                probe=os.environ.get('COMMITER_SYSTEM_PROBE')
                if probe:
                    result=command([probe]);row['system_status']=result
                    if result.get('exit')==0:row['thermal_and_power_mode']=json.loads(result['stdout'])
                row['load_average']=command(['sysctl','vm.loadavg'])
                row['disk_io_cumulative']=command(['iostat','-Id','-c','1'])
                gpu=command(['ioreg','-r','-c','AGXAccelerator','-a'])
                if gpu.get('exit')==0:
                    row['gpu_performance_statistics']=[i.get('PerformanceStatistics',{}) for i in plistlib.loads(gpu['stdout'].encode())]
                row['gpu_available']=bool(row.get('gpu'))
                if any(p['rss_kib']>=12*1024*1024 for p in allowed if p['executable']=='commiter-mlx-helper'):self.fatal.append('helper_rss_12GiB')
                required=['cpu','vm','pressure','swap','power','thermal']
                row['valid']=all(row.get(k,{}).get('exit')==0 for k in required) and 'gpu' in row
                if not row['valid']:self.fatal.append('monitor_failure')
                self.samples+=1;self.durations.append(time.monotonic()-start)
                with (self.out/'monitor.jsonl').open('a') as f:f.write(json.dumps(row)+'\n')
            except Exception as e:self.fatal.append('monitor_exception_'+type(e).__name__)
            if self.fatal and self.active:
                try:os.killpg(self.active.pid,signal.SIGTERM)
                except ProcessLookupError:pass
            self.done.wait(max(0,2-(time.monotonic()-start)))
    def start(self):self.thread=threading.Thread(target=self.loop);self.thread.start()
    def stop(self):self.done.set();self.thread.join(timeout=10)

def score(cell):
    expected=[frozenset(g['files']) for g in GOLD]
    first=cell/'01-response.json';request=cell/'01-request.json'
    report={'validator_accepted':False,'grouping_available':False,'text_semantic_review':'pending if text exists; no model judge'}
    if not first.exists() or not request.exists():return report
    try:
        native=json.loads(first.read_text());answer=json.loads(native['generated_json'])
        files=json.loads(json.loads(request.read_text())['messages'][1]['content'])['files']
        paths={f['file_id']:f.get('new_path',f.get('old_path')) for f in files}
        if set(answer)!=set(paths):return dict(report,grouping_failure='missing_or_unknown_ids')
        groups={}
        for key,label in answer.items():groups.setdefault(label,set()).add(paths[key])
        actual=[frozenset(v) for v in groups.values()]
        truth={p:i for i,g in enumerate(expected) for p in g};pred={p:i for i,g in enumerate(actual) for p in g}
        tp=fp=fn=tn=0
        for a,b in itertools.combinations(sorted(truth),2):
            same=truth[a]==truth[b];got=pred[a]==pred[b]
            if same and got:tp+=1
            elif same:fn+=1
            elif got:fp+=1
            else:tn+=1
        exact=sum(g in expected for g in actual)
        precision=tp/(tp+fp) if tp+fp else None;recall=tp/(tp+fn) if tp+fn else None
        report.update(grouping_available=True,exact_groups=exact,reference_groups=3,predicted_groups=len(actual),exact_partition=set(actual)==set(expected),exact_group_recall=exact/3,
                      pairs={'tp':tp,'fp':fp,'fn':fn,'tn':tn,'precision':precision,'recall':recall},actual_groups=[sorted(g) for g in actual])
        result=json.loads((cell/'result.json').read_text());report['validator_accepted']=not result['error'];report['git_unchanged']=result['git_unchanged']
        if (cell/'02-request.json').exists() and (cell/'02-response.json').exists():
            inputs=json.loads(json.loads((cell/'02-request.json').read_text())['messages'][1]['content'])['groups']
            response=json.loads((cell/'02-response.json').read_text());category=json.loads(response['generated_json'])
            correct=denominator=0;refs=[]
            for group in inputs:
                members=frozenset(f.get('new_path',f.get('old_path')) for f in group['files'])
                if members in expected:
                    index=expected.index(members);denominator+=1
                    value=category[group['group_id']];correct+=value['type']==GOLD[index]['type'];refs.append(value['breaking_evidence_ref'])
            report.update(category_correct_on_exact_groups=correct,category_denominator=denominator,breaking_refs_on_exact_groups=refs)
    except Exception as e:report['evaluation_note']=type(e).__name__
    return report

def run(out,helper,study,fixture_root,frozen_from=None):
    if (out/'execution-contract.json').exists():raise RuntimeError('no repeat or overwrite')
    matrix=json.loads((study/'matrix.json').read_text())['cases'];assets={x['model']:x for x in json.loads((study/'assets.json').read_text())}
    # Pair constraint arms in alternating order to limit a single fixed order bias.
    keys=list(dict.fromkeys(x['case'] for x in matrix));cells=[]
    for index,key in enumerate(keys):
        modes=['json_schema','none'] if index%2==0 else ['none','json_schema']
        for mode in modes:cells.append(next(x for x in matrix if x['case']==key and x['output_constraint']==mode))
    frozen_files=[helper,helper.parent/'mlx.metallib',out/'build/evaluation.test',out/'build/overlay.json',pathlib.Path(__file__),ROOT/'tools/benchmark182/testdata/inference_e2e_test.go.txt']
    frozen_files += [p for p in (out/'build').iterdir() if p.suffix=='.go']
    frozen_files += [study/c['config'] for c in cells]
    frozen_files += [ROOT/'tools/benchmark182/mistral_tokenize.py',ROOT/'docs/benchmarks/issue-182/prompt-candidate.md']
    for asset in assets.values():
        frozen_files += [pathlib.Path(asset['path'])/f['file'] for f in asset['files']]
    frozen_files += [ROOT/'tools/benchmark182/monitor_v2.py',ROOT/'tools/benchmark182/run.py']
    if os.environ.get('COMMITER_SYSTEM_PROBE'):frozen_files.append(pathlib.Path(os.environ['COMMITER_SYSTEM_PROBE']))
    contract=dict(id='H182-015',context_policy=json.loads((out/'build/context-policy.json').read_text()) if (out/'build/context-policy.json').exists() else 'automatic',fixture=str(fixture_root),models=1,cells=cells,repeats=1,warmup=0,retry=0,repair=0,seed=144,
                  thinking_budget=None,output_budget=4096,cycle_timeout=None,history_timeout=None,kv='native',pipeline='eager',
                  output_modes=['json_schema','none'],sampling_note='Gemma 1/.95/64; Ministral Instruct .1/.95/20; Reasoning .7/.95/20; prior adopted per-family settings',prompt_note='Gemma adopted compact rules plus docs definition and explicit forbidden scopes; remove automatically constructed assignment/type examples; retain full field descriptions and schema; native KV; two simultaneous interventions',
                  max_calls=12,gold=GOLD,scoring='exact file partition, pair TP/FP/FN/TN, category on exact groups; text factual/manual rubric separately; no injected partitions',
                  source_hashes={str(p):sha(p) for p in frozen_files},asset_manifest_hash=sha(study/'assets.json'),
                  monitor=dict(interval_seconds=2,cpu='Mach system ticks and process ps CPU/time',memory='vm, pressure, swap, RSS/VSZ, MLX metrics, KV arrays',gpu='system ioreg AGX utilization; per-process GPU unavailable',thermal='ProcessInfo thermalState plus pmset thermal/performance limits and low-power mode',temperature_celsius='unavailable',power_watts='unavailable: powermetrics requires root',power_source='pmset battery/AC',safety='12GiB helper RSS or monitoring failure stops entire study; time/output unlimited'))
    save(out/'execution-contract.json',contract)
    save(out/'gold.json',GOLD)
    metadata={name:command(cmd) for name,cmd in [('os',['sw_vers']),('hardware',['sysctl','hw.model','hw.memsize','hw.ncpu','hw.physicalcpu','hw.logicalcpu']),('power',['pmset','-g','batt']),('powermetrics',['/usr/bin/powermetrics','--samplers','cpu_power,gpu_power,thermal','-n','1','-i','1000'])]}
    save(out/'environment.json',metadata)
    monitor=Monitor(out);monitor.start();started=time.monotonic();rows=[];state='running';reason=None
    env={k:v for k,v in os.environ.items() if not k.startswith(('COMMITER_180_','COMMITER_182_','COMMITER_STUDY_'))}
    try:
        freeze=out/'freeze';monitor.stage='freeze'
        if frozen_from is not None:
            shutil.copytree(frozen_from,freeze)
            contract['freeze_reused_from']=str(frozen_from)
            print('REUSING exact frozen Git/history input; prior infrastructure failures preserved',flush=True)
        else:
            freeze.mkdir(mode=0o700)
            ev=dict(env,COMMITER_STUDY_ACTION='freeze',COMMITER_STUDY_ROOT=str(fixture_root),COMMITER_STUDY_OUT=str(freeze))
            with (out/'freeze.log').open('x') as log:
                p=subprocess.Popen([str(out/'build/evaluation.test'),'-test.run=^TestIssue183Freeze$','-test.v','-test.timeout=0'],cwd=ROOT,env=ev,stdout=log,stderr=subprocess.STDOUT,start_new_session=True);monitor.active=p;code=p.wait();monitor.active=None
            if code:raise RuntimeError('freeze failed; no inference started')
        contract['fixture_sha256']=sha(freeze/'fixture.json');save(out/'execution-contract.json',contract)
        print('FROZEN; starting registered inference cells',flush=True)
        for case in cells:
            if monitor.fatal:raise RuntimeError('monitor safety stop')
            if any(sha(p)!=h for p,h in contract['source_hashes'].items()):raise RuntimeError('source/runtime drift')
            label=case['case']+'-'+case['output_constraint'];dest=out/label;dest.mkdir(mode=0o700,exist_ok=False);monitor.stage=label
            conf=json.loads((study/case['config']).read_text());asset=assets[conf['model']]
            ev=dict(env,COMMITER_STUDY_ACTION='infer',COMMITER_STUDY_OUT=str(dest),COMMITER_STUDY_FIXTURE=str(freeze/'fixture.json'),COMMITER_STUDY_HELPER=str(helper),COMMITER_STUDY_MODEL_PATH=asset['path'],COMMITER_182_CASE_CONFIG=str(study/case['config']),COMMITER_182_PIPELINE='eager',COMMITER_182_PROGRESS='1',COMMITER_182_DIAGNOSTICS='1',COMMITER_180_KV_BITS='native',COMMITER_182_NEUTRAL_EXAMPLES='1')
            if conf['family']=='ministral':
                runtime=pathlib.Path(str(BENCHMARK_ROOT / 'issue182-ministral-runtime'))
                ev.update(COMMITER_182_MISTRAL_PYTHON=str(BENCHMARK_ROOT / 'issue183-ministral-restored-venv/bin/python'),COMMITER_182_MISTRAL_BRIDGE=str(ROOT/'tools/benchmark182/mistral_tokenize.py'),COMMITER_182_MISTRAL_ASSETS=str(runtime/'assets'/('reasoning' if conf['thinking'] else 'instruct')))
            print('START',label,datetime.datetime.now(datetime.timezone.utc).isoformat(),flush=True)
            begin=time.monotonic()
            with (out/(label+'.log')).open('x') as log:
                p=subprocess.Popen([str(out/'build/evaluation.test'),'-test.run=^TestIssue183Inference$','-test.v','-test.timeout=0'],cwd=ROOT,env=ev,stdout=log,stderr=subprocess.STDOUT,start_new_session=True);monitor.active=p;code=p.wait();monitor.active=None
            if code or not (dest/'result.json').exists():raise RuntimeError('runner stopped; no retries')
            evaluation=score(dest);save(dest/'evaluation.json',evaluation)
            result=json.loads((dest/'result.json').read_text())
            row=dict(case=label,wall_seconds=time.monotonic()-begin,calls=result['calls'],error=result['error'],evaluation=evaluation)
            rows.append(row);save(out/'results.json',rows);print('END',label,'accepted',evaluation['validator_accepted'],'seconds',round(row['wall_seconds'],2),flush=True)
            if not result['git_unchanged']:raise RuntimeError('fixture changed')
        state='complete'
    except BaseException as e:
        state='stopped';reason=str(e) or type(e).__name__;print('STOP',type(e).__name__,reason,flush=True)
    finally:
        if monitor.active is not None and monitor.active.poll() is None:
            try: os.killpg(monitor.active.pid,signal.SIGTERM)
            except ProcessLookupError: pass
            monitor.active.wait()
            monitor.active=None
        monitor.stop()
        if monitor.fatal:state='stopped';reason=str(monitor.fatal)
        save(out/'completion.json',dict(state=state,reason=reason,registered_cells=len(cells),finished_cells=len(rows),calls=sum(r['calls'] for r in rows),elapsed_seconds=time.monotonic()-started,monitor_samples=monitor.samples,monitor_fatal=monitor.fatal,sampler_mean_seconds=statistics.mean(monitor.durations) if monitor.durations else None,sampler_max_seconds=max(monitor.durations,default=None)))
    if state!='complete':raise SystemExit(1)

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ['out','helper','study','fixture-root']:parser.add_argument('--'+name,type=pathlib.Path,required=True)
    parser.add_argument('--frozen-from',type=pathlib.Path)
    args=parser.parse_args();run(args.out.resolve(),args.helper.resolve(),args.study.resolve(),args.fixture_root.resolve(),args.frozen_from.resolve() if args.frozen_from else None)
