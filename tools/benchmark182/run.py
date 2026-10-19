"""Finite model-default study. Freeze before inference; no retries or repairs."""
import argparse,datetime,hashlib,json,os,pathlib,plistlib,resource,statistics,subprocess,threading,time
ROOT=pathlib.Path(__file__).resolve().parents[2]
def sha(p):
 h=hashlib.sha256()
 with pathlib.Path(p).open('rb') as f:
  for b in iter(lambda:f.read(4*1024*1024),b''):h.update(b)
 return h.hexdigest()
def save(p,v):p.write_text(json.dumps(v,ensure_ascii=False,indent=2)+'\n')
def command(argv):
 p=subprocess.run(argv,capture_output=True,timeout=4)
 return {'exit':p.returncode,'stdout':p.stdout.decode(errors='replace'),'stderr':p.stderr.decode(errors='replace')}
def sample():
 start=time.monotonic();r={'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'monotonic':start}
 for key,args in [('cpu',['top','-l','1','-n','0']),('vm',['vm_stat']),('pressure',['memory_pressure','-Q']),('swap',['sysctl','vm.swapusage']),('thermal',['pmset','-g','therm']),('power',['pmset','-g','batt'])]:
  try:r[key]=command(args)
  except Exception as e:r[key]={'error':type(e).__name__}
 try:
  p=subprocess.run(['ioreg','-r','-c','AGXAccelerator','-a'],capture_output=True,timeout=4)
  r['gpu']=[{k:v for k,v in item.get('PerformanceStatistics',{}).items() if 'Utilization' in k or k in ['Alloc system memory','In use system memory']} for item in plistlib.loads(p.stdout)]
 except Exception as e:r['gpu_error']=type(e).__name__
 try:
  p=subprocess.run(['ps','-axo','pid=,%cpu=,rss=,comm='],capture_output=True,text=True,timeout=4)
  r['helper_processes']=[{'pid':int(v[0]),'cpu_percent':float(v[1]),'rss_kib':int(v[2])} for line in p.stdout.splitlines() if len(v:=line.split(None,3))==4 and v[3].endswith('/commiter-mlx-helper')]
 except Exception as e:r['process_error']=type(e).__name__
 r['collection_seconds']=time.monotonic()-start
 return r

def score(result,fixture):
 expected=[frozenset(g['files']) for g in fixture['groups']]
 plan=result['result']['Plan']['commits'] or []; paths=result['paths']
 actual=[frozenset(paths[f] for f in c['file_ids']) for c in plan]
 matched=[(i,expected.index(files)) for i,files in enumerate(actual) if files in expected]
 return {'exact_group_recall':len(matched)/len(expected),'exact_groups':len(matched),'reference_groups':len(expected),'predicted_groups':len(actual),'category_correct_on_exact_groups':sum(plan[i]['type']==fixture['groups'][j]['type'] for i,j in matched),'category_denominator':len(matched),'validator_accepted':not result['error'],'fallback':False}

def freeze(a):
 a.out.mkdir(parents=True,exist_ok=False)
 models=json.loads(a.models.read_text());fixture=json.loads((a.fixture/'fixture.json').read_text())
 assert len(models)==4 and len(fixture['files_sha256'])==16
 files=[ROOT/'mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift',ROOT/'mlx-helper/Sources/BoundedGeneration/NativeThoughtBudget.swift',ROOT/'internal/cli/issue182_evaluation_test.go',ROOT/'internal/planning/three_phase.go',pathlib.Path(__file__),ROOT/'tools/benchmark182/fixture.py',a.binary,a.helper,a.helper.parent/'mlx.metallib',a.fixture/'fixture.json']
 files.extend(a.fixture/'repo'/p for p in fixture['files_sha256'])
 files.append(a.overlay)
 files.extend(pathlib.Path(p) for p in json.loads(a.overlay.read_text())['Replace'].values())
 for m in models:
  files.extend(p for p in pathlib.Path(m['path']).iterdir() if p.is_file())
 print('Hashing runtime, weights and fixed fixture before inference',flush=True)
 contract={'harness_id':'H182-001-model-defaults','baseline_harness_id':'H180-004','baseline_note':'new Gemma measurements in H182-001; historical H180-004 scores are not reused','source_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'hashes':{str(p.resolve()):sha(p) for p in files},'models':models,'helper':str(a.helper.resolve()),'binary':str(a.binary.resolve()),'fixture':str(a.fixture.resolve()),'warmups':1,'measured_repeats':2,'order':[(m,i) for i in [0,1,2] for m in (range(4) if i!=2 else reversed(range(4)))],'max_cycles':12,'max_generation_requests':36,'shared_cycle_timeout_seconds':120,'outer_process_timeout_seconds':140,'retry':0,'repair':0,'judge_calls':0,'thinking':{'gemma':'established bounded profile: 512/384/0 maximum native thought tokens','Qwen3.5-4B':'template default ON; 512/384/384 maximum native thought tokens','Qwen3.5-2B':'template default OFF','Phi-4-mini-instruct':'normal inference; no dedicated thinking mode'},'generation':{'temperature':0,'seed':144,'top_p':1,'top_k':0,'output_budgets':[768,512,768],'context_ceiling':65536,'weights':'4bit, group64','kv':'model-native default dtype/cache classes; no quantization override or prefix reuse','model_and_grammar_retention':'within cycle only','build':'Swift Debug, same binary for all four; no production speed claim','file_limit':'research-only Go overlay increases four-file cap to16; no production source change'},'scoring':fixture['scoring'],'gold_human_approved':False,'monitor':{'interval_seconds':5,'commands':'top/vm_stat/memory_pressure/sysctl/pmset/ioreg/ps','temperature':'unavailable','power_watts':'unavailable','per_process_gpu':'unavailable','cpu_note':'top single sample system CPU is OS instantaneous/averaged estimate; ps process CPU is lifetime-average percentage','overhead':'measure sampler wall time and driver CPU time; causal latency overhead not separately measured'},'stop':'phase/validator failure ends that cycle; continue registered cells; source/weight/fixture drift, monitor gap >15s, helper RSS >=12GiB or monitor infrastructure failure stops entire study; no retries/repairs/tuning/extra inference'}
 save(a.out/'execution-contract.json',contract)
 save(a.out/'fixture-reference.json',fixture)
 baseline=[sample() for _ in range(3)]
 save(a.out/'monitor-preflight.json',baseline)
 assert all('gpu' in r and all(r[k].get('exit')==0 for k in ['cpu','vm','pressure','swap','power','thermal']) for r in baseline),'monitor unavailable'
 print('FROZEN: 12 cycles / at most 36 phase requests',flush=True)

def run(a):
 c=json.loads((a.out/'execution-contract.json').read_text());fixture=json.loads((a.out/'fixture-reference.json').read_text())
 assert all(sha(p)==h for p,h in c['hashes'].items()),'frozen drift'
 done=threading.Event();samples=[];fatal=[];active=[None]
 def monitor():
  while not done.is_set():
   r=sample();samples.append(r)
   with (a.out/'monitor.jsonl').open('a') as f:f.write(json.dumps(r)+'\n')
   if len(samples)>1 and r['monotonic']-samples[-2]['monotonic']>15:fatal.append('monitor_gap')
   if any(p['rss_kib']>=12*1024*1024 for p in r.get('helper_processes',[])):fatal.append('helper_rss_limit')
   if 'gpu' not in r or any(r[k].get('exit')!=0 for k in ['cpu','vm','pressure','swap','power','thermal']):fatal.append('monitor_failure')
   if fatal and active[0] is not None:active[0].terminate()
   done.wait(max(0,5-r['collection_seconds']))
 thread=threading.Thread(target=monitor);thread.start();rows=[];cpu_before=resource.getrusage(resource.RUSAGE_SELF);started=time.monotonic()
 try:
  for index,repeat in c['order']:
   if fatal:break
   m=c['models'][index];label=f"m{index}-r{repeat}";dst=a.out/label
   assert not dst.exists(),'do not rerun cells'
   env=os.environ.copy()
   for k in list(env):
    if k.startswith('COMMITER_180_') or k.startswith('COMMITER_182_'):del env[k]
   env.update(COMMITER_182_RUN='1',COMMITER_182_REPO=str(pathlib.Path(c['fixture'])/'repo'),COMMITER_182_FIXTURE=str(pathlib.Path(c['fixture'])/'fixture.json'),COMMITER_182_OUT=str(dst),COMMITER_182_MODEL=m['repo']+'@'+m['revision'],COMMITER_182_MODEL_PATH=m['path'],COMMITER_180_HELPER=c['helper'])
   print('START',label,m['repo'],'warmup' if repeat==0 else 'measured',flush=True)
   with (a.out/(label+'.log')).open('x') as log:
    proc=subprocess.Popen([c['binary'],'-test.run=^TestIssue182Evaluation$','-test.v','-test.timeout=135s'],cwd=ROOT/'internal/cli',env=env,stdout=log,stderr=subprocess.STDOUT);active[0]=proc
    try:code=proc.wait(timeout=140)
    except subprocess.TimeoutExpired:proc.terminate();code=proc.wait(timeout=5);fatal.append('outer_timeout')
    finally:active[0]=None
   row={'label':label,'model':m['repo'],'repeat':repeat,'warmup':repeat==0,'exit':code}
   if not (dst/'result.json').exists():fatal.append('runner_failure');rows.append(row);break
   result=json.loads((dst/'result.json').read_text());row.update(result)
   row['score']=score(result,fixture)
   row['metrics']=[json.loads(p.read_text())['native'].get('benchmark_metrics',{}) for p in sorted(dst.glob('*-response.json'))]
   rows.append(row);save(a.out/'results.json',rows)
   print('END',label,'accepted',not result['error'],'seconds',round(result['wall_seconds'],2),'score',row['score']['exact_group_recall'],'error',result['error'],flush=True)
 finally:
  done.set();thread.join(timeout=10)
  cpu_after=resource.getrusage(resource.RUSAGE_SELF)
  save(a.out/'completion.json',{'fatal':fatal,'cycles':len(rows),'generation_requests':sum(r.get('calls',0) for r in rows),'elapsed_seconds':time.monotonic()-started,'monitor_samples':len(samples),'sampler_wall_mean_seconds':statistics.mean(r['collection_seconds'] for r in samples) if samples else None,'sampler_wall_max_seconds':max((r['collection_seconds'] for r in samples),default=None),'driver_cpu_seconds':cpu_after.ru_utime+cpu_after.ru_stime-cpu_before.ru_utime-cpu_before.ru_stime,'content_mismatches':[p for p,h in c['hashes'].items() if sha(p)!=h]})
 if fatal:raise RuntimeError(fatal)

if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('mode',choices=['freeze','run']);p.add_argument('--out',type=pathlib.Path,required=True)
 for k in ['models','fixture','helper','binary','overlay']:p.add_argument('--'+k,type=pathlib.Path)
 a=p.parse_args();freeze(a) if a.mode=='freeze' else run(a)
