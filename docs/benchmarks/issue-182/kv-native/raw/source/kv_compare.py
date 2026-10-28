"""H182-008: eager pipeline, diagnostic gate then KV8/native comparison."""
import json,pathlib,shutil,subprocess,sys,threading,time
import prompt_compare as harness
ROOT=harness.ROOT
PARENT=pathlib.Path('<PRIVATE_TMP>/issue182-release-comparison')
OUT=pathlib.Path('<PRIVATE_TMP>/issue182-kv-native')
RUNTIME=OUT/'runtime'
harness.OUT=OUT;harness.RUNTIME=RUNTIME
original_environment=harness.environment

def environment(cell):
 env=original_environment(cell)
 env.update(COMMITER_182_PIPELINE='eager',COMMITER_180_KV_BITS=cell['kv_mode'])
 if cell.get('diagnostic'):env['COMMITER_182_DIAGNOSTICS']='1'
 return env
harness.environment=environment

def prepare():
 assert not (OUT/'contract.json').exists()
 parent=json.loads((PARENT/'contract.json').read_text())
 changed=ROOT/'mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift'
 assert all(harness.sha(p)==h for p,h in parent['hashes'].items() if p!=str(changed)),'reference asset drift'
 RUNTIME.mkdir(parents=True,exist_ok=True)
 shutil.copyfile('<PRIVATE_TMP>/issue182-ministral-runtime/swift-build/release/commiter-mlx-helper',RUNTIME/'commiter-mlx-helper');(RUNTIME/'commiter-mlx-helper').chmod(0o755)
 shutil.copyfile(PARENT/'runtime/mlx.metallib',RUNTIME/'mlx.metallib')
 (RUNTIME/'proxy').write_text('#!/bin/sh\nexec /opt/homebrew/bin/python3 '+str(pathlib.Path(__file__).resolve())+' proxy "$@"\n');(RUNTIME/'proxy').chmod(0o755)
 shutil.copyfile(PARENT/'repos.json',OUT/'repos.json');shutil.copytree(PARENT/'fixtures',OUT/'fixtures')
 cells=[];diagnostics=[]
 for old in parent['cells']:
  capture=dict(old,label='capture-'+old['label'],kv_mode='8')
  env=environment(capture);env['COMMITER_180_MODE']='capture'
  subprocess.run([str(harness.PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env,check=True,capture_output=True)
  requests=[]
  for i in [1,2,3]:
   name=f'{i:02d}-request.json';r=json.loads((OUT/capture['label']/name).read_text())
   assert r==json.loads((PARENT/capture['label']/name).read_text()),'live input drift'
   r['model_path']=old['path'];requests.append(harness.transform(r,old['arm']))
  oldaudit=json.loads((PARENT/'audit'/(old['label']+'.json')).read_text())['native']
  for mode in (['8','native'] if old['case']=='C01' else ['native','8']):
   c=dict(old,label=old['label']+'-kv'+mode,kv_mode=mode,diagnostic=False)
   result=subprocess.run([str(RUNTIME/'commiter-mlx-helper'),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests[1:]),text=True,capture_output=True,env=environment(c),check=True)
   replies=[json.loads(line) for line in result.stdout.splitlines()];assert len(replies)==2 and all(r['ok'] for r in replies),result.stdout
   native=[json.loads(r['generated_json']) for r in replies]
   for actual,previous in zip(native,oldaudit):
    assert actual['token_ids']==previous['token_ids'] and actual['enable_thinking']==previous['enable_thinking']
    if mode=='8':assert actual['planned_cache']==previous['planned_cache']
    else:
     assert len(actual['planned_cache'])==len(previous['planned_cache'])
     for a,b in zip(actual['planned_cache'],previous['planned_cache']):
      if b['kind']=='affine':assert a['nativeType']=='KVCacheSimple' and a['kind']!='affine'
      else:assert a==b,'non-attention cache changed'
   harness.save(OUT/'audit'/(c['label']+'.json'),{'requests':requests,'native':native,'prefix':c['prefix'],'kv_mode':mode})
   cells.append(c)
  if old['case']=='C01':diagnostics.append(dict(old,label=old['label']+'-diagnostic',kv_mode='8',diagnostic=True))
 (OUT/'diagnostics').mkdir();shutil.copyfile(OUT/'repos.json',OUT/'diagnostics/repos.json');shutil.copytree(OUT/'fixtures',OUT/'diagnostics/fixtures')
 templates=subprocess.run(['xcrun','xctrace','list','templates'],capture_output=True,text=True,check=True)
 assert 'Metal System Trace' in templates.stdout
 (OUT/'trace-templates.log').write_text(templates.stdout)
 files=[pathlib.Path(p) for p in parent['hashes']]+[pathlib.Path(__file__),RUNTIME/'proxy',RUNTIME/'commiter-mlx-helper',RUNTIME/'mlx.metallib',OUT/'repos.json',OUT/'diagnostics/repos.json',ROOT/'mlx-helper/Sources/BoundedGeneration/NativeThoughtBudget.swift',ROOT/'mlx-helper/Tests/BoundedGenerationTests/NativeThoughtBudgetTests.swift',ROOT/'docs/benchmarks/issue-182/kv-native/plan.md']
 files+=list((OUT/'audit').glob('*.json'))+list((OUT/'fixtures').glob('*.json'))+list((OUT/'diagnostics/fixtures').glob('*.json'))
 hashes={str(p):harness.sha(p) for p in files}
 common=dict(parent,hashes=hashes,baseline='KV8 is remeasured; no historical timings reused',build='same Release binary/shader; eager only',limitations='single pair per fixture; native KV may increase memory and change numerical results; CPU timers include waits and trace affects timing; no GPU overlap claim without trace analysis')
 harness.save(OUT/'contract.json',dict(common,id='H182-008-kv',cells=cells,max_calls=24))
 harness.save(OUT/'diagnostics/contract.json',dict(common,id='H182-008-diagnostic',cells=diagnostics,max_calls=6))
 print('READY: 8 performance cells + 2 diagnostic cells; tokens/cache policy audited; no generation',flush=True)

def trace_watcher(done,records):
 contract=json.loads((OUT/'diagnostics/contract.json').read_text())
 for c in contract['cells']:
  d=OUT/'diagnostics'/c['label']
  while not done.is_set():
   f=d/'01-helper-stderr.log'
   if f.exists() and 'generation_progress' in f.read_text():break
   done.wait(0.2)
  if done.is_set():return
  inventory=subprocess.check_output(['ps','-ww','-axo','pid=,comm='],text=True)
  found=[int(v[0]) for line in inventory.splitlines() if len(v:=line.strip().split(None,1))==2 and v[1]==str(RUNTIME/'commiter-mlx-helper')]
  if len(found)!=1:
   records.append({'cell':c['label'],'error':'helper PID not unique'});continue
  started=time.monotonic()
  with (d/'metal-trace.log').open('x') as log:
   result=subprocess.run(['xcrun','xctrace','record','--template','Metal System Trace','--attach',str(found[0]),'--time-limit','30s','--output',str(d/'metal.trace')],stdout=log,stderr=subprocess.STDOUT)
  records.append({'cell':c['label'],'pid':found[0],'exit':result.returncode,'seconds':time.monotonic()-started,'trace_exists':(d/'metal.trace').exists()})
  harness.save(OUT/'diagnostics/trace-results.json',records)

def diagnose():
 done=threading.Event();records=[]
 def worker():
  try:trace_watcher(done,records)
  except Exception as error:records.append({'error':str(error)})
 thread=threading.Thread(target=worker);thread.start();harness.OUT=OUT/'diagnostics'
 try:harness.run()
 finally:
  done.set();thread.join();harness.save(OUT/'diagnostics/trace-results.json',records)

def run():
 d=OUT/'diagnostics';completion=json.loads((d/'completion.json').read_text())
 assert completion['cells']==2 and not completion['monitor_failed']
 assert len(json.loads((d/'results.json').read_text()))==2
 traces=json.loads((d/'trace-results.json').read_text());assert len(traces)==2 and all(r.get('exit')==0 and r.get('trace_exists') for r in traces),'diagnostic trace incomplete'
 review=json.loads((d/'review.json').read_text());assert review.get('proceed_to_kv_comparison') is True and review.get('findings'),'diagnostics must be reviewed before performance run'
 harness.run()

if __name__=='__main__':
 if sys.argv[1]=='prepare':prepare()
 elif sys.argv[1]=='proxy':harness.proxy()
 elif sys.argv[1]=='diagnose':diagnose()
 elif sys.argv[1]=='run':run()
 else:raise ValueError('mode')
