"""H182-009: prepare only by default; explicit stage1/stage2 commands generate."""
import hashlib,json,os,pathlib,shutil,signal,subprocess,sys,time
import prompt_compare as h
ROOT=h.ROOT
PARENT=pathlib.Path('/private/tmp/issue182-kv-native')
OUT=pathlib.Path('/private/tmp/issue182-path-comparison')
RUNTIME=OUT/'runtime'
h.OUT=OUT;h.RUNTIME=RUNTIME
base_environment=h.environment

def environment(cell):
 e=base_environment(cell)
 e.update(COMMITER_182_PIPELINE='eager',COMMITER_180_KV_BITS='8',COMMITER_182_PATH=cell['path_mode'])
 e.pop('COMMITER_182_DIAGNOSTICS',None)
 return e
h.environment=environment

def prepare():
 assert not (OUT/'contract.json').exists(),'do not overwrite evidence'
 parent=json.loads((PARENT/'contract.json').read_text())
 changed=str(ROOT/'mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift')
 assert all(h.sha(p)==v for p,v in parent['hashes'].items() if p!=changed),'parent asset drift'
 RUNTIME.mkdir(parents=True,exist_ok=True)
 shutil.copyfile('/private/tmp/issue182-ministral-runtime/swift-build/release/commiter-mlx-helper',RUNTIME/'commiter-mlx-helper');(RUNTIME/'commiter-mlx-helper').chmod(0o755)
 shutil.copyfile(PARENT/'runtime/mlx.metallib',RUNTIME/'mlx.metallib')
 (RUNTIME/'proxy').write_text('#!/bin/sh\nexec /opt/homebrew/bin/python3 '+str(pathlib.Path(__file__).resolve())+' proxy "$@"\n');(RUNTIME/'proxy').chmod(0o755)
 shutil.copyfile(PARENT/'repos.json',OUT/'repos.json');shutil.copytree(PARENT/'fixtures',OUT/'fixtures')
 cells=[];audits=[]
 for old in parent['cells']:
  if old['kv_mode']!='8':continue
  old=dict(old);stem=old['label'].removesuffix('-kv8');capture=dict(old,label='capture-'+stem,path_mode='full')
  e=environment(capture);e['COMMITER_180_MODE']='capture'
  subprocess.run([str(h.PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=e,check=True,capture_output=True)
  requests=[]
  for i in [1,2,3]:
   r=json.loads((OUT/capture['label']/f'{i:02}-request.json').read_text());assert r==json.loads((PARENT/('capture-'+stem)/f'{i:02}-request.json').read_text())
   # Capture uses a fix stub; real C02 Text carries the generated feat type.
   # Replay saved actual requests, not the synthetic capture Text request.
   wire=json.loads((PARENT/old['label']/f'{i:02}-actual-request.json').read_text());requests.append(wire)
  reqfile=OUT/'requests'/(stem+'.jsonl');reqfile.parent.mkdir(exist_ok=True);reqfile.write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in requests))
  result=subprocess.run([str(RUNTIME/'commiter-mlx-helper'),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests[1:]),text=True,capture_output=True,check=True,env=e)
  replies=[json.loads(s) for s in result.stdout.splitlines()];assert len(replies)==2 and all(r['ok'] for r in replies)
  native=[json.loads(r['generated_json']) for r in replies]
  baseline=subprocess.run([str(PARENT/'runtime/commiter-mlx-helper'),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests[1:]),text=True,capture_output=True,check=True,env=e)
  assert native==[json.loads(json.loads(line)['generated_json']) for line in baseline.stdout.splitlines()]
  h.save(OUT/'audit'/(stem+'.json'),{'native':native,'requests_identical':True});audits.append(stem)
  for stage,modes in [('stage1',['go','direct'] if old['case']=='C01' else ['direct','go']),('stage2',['bounded-cap','standard-cap'] if old['case']=='C01' else ['standard-cap','bounded-cap'])]:
   for mode in modes:cells.append(dict(old,label=stage+'-'+stem+'-'+mode,stage=stage,transport='go' if mode=='go' else 'direct',path_mode='full' if stage=='stage1' else mode,requests=str(reqfile)))
 files=[pathlib.Path(p) for p in parent['hashes']]+[pathlib.Path(__file__),RUNTIME/'commiter-mlx-helper',RUNTIME/'mlx.metallib',RUNTIME/'proxy',h.PREV/'evaluation.test',ROOT/'docs/benchmarks/issue-182/path-overhead/goal.md']+list((OUT/'requests').glob('*'))+list((OUT/'audit').glob('*'))
 h.save(OUT/'contract.json',{'id':'H182-009','cells':cells,'hashes':{str(p):h.sha(p) for p in files},'seed':144,'kv_bits':8,'stage1_output_cap':None,'stage2_output_cap':1024,'thinking_budget':1024,'warmup':0,'retry':0,'max_calls_per_stage':24,'audited_cases':audits,'stage2_limitations':'common maximum, natural EOS may stop earlier; outputs/paths diverge; no task validator or Thinking control in standard-cap; shared cache/prefill and consumer remain; not standalone mlx-lm CLI'})
 print('READY: stage1 8 cells + stage2 8 cells; capture/token/cache audits PASS; no generation')

def run(stage):
 from monitor_v2 import Monitor
 assert stage in ['stage1','stage2']
 contract=json.loads((OUT/'contract.json').read_text());folder=OUT/stage;folder.mkdir(exist_ok=False)
 monitor=Monitor(folder/'monitor.jsonl');monitor.start();rows=[];started=time.monotonic()
 try:
  for cell in [c for c in contract['cells'] if c['stage']==stage]:
   assert all(h.sha(p)==v for p,v in contract['hashes'].items()),'frozen drift'
   assert not monitor.failed
   target=OUT/cell['label'];assert not target.exists()
   logdir=target if cell['transport']=='direct' else folder/cell['label'];logdir.mkdir(exist_ok=False);e=environment(cell)
   command=[str(h.PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'] if cell['transport']=='go' else [str(RUNTIME/'commiter-mlx-helper'),'--benchmark-cache']
   print('START',cell['label'],flush=True);begin=time.monotonic()
   with (logdir/'driver.log').open('x') as err,(logdir/'direct-responses.jsonl').open('x') as output,open(cell['requests']) as requests:
    proc=subprocess.Popen(command,cwd=ROOT/'internal/cli',env=e,stdin=requests if cell['transport']=='direct' else subprocess.DEVNULL,stdout=output if cell['transport']=='direct' else err,stderr=err,start_new_session=True)
    last=time.monotonic();size=0;interrupted=None
    while proc.poll() is None:
     current=sum(p.stat().st_size for p in target.glob('*.log'))+output.tell()
     if current!=size:size=current;last=time.monotonic()
     if monitor.failed or time.monotonic()-last>300:
      interrupted='monitor_failure' if monitor.failed else 'no_progress_300s'
      inventory=subprocess.check_output(['ps','-axo','pid=,comm='],text=True)
      for line in inventory.splitlines():
       values=line.strip().split(None,1)
       if len(values)==2 and values[1]==str(RUNTIME/'commiter-mlx-helper'):os.kill(int(values[0]),signal.SIGTERM)
      if proc.poll() is None:os.killpg(proc.pid,signal.SIGTERM)
      proc.wait(timeout=10);break
     try:proc.wait(timeout=1)
     except subprocess.TimeoutExpired:pass
   elapsed=time.monotonic()-begin
   row={'cell':cell['label'],'exit':proc.returncode,'interrupted':interrupted,'driver_wall_seconds':elapsed}
   if cell['transport']=='direct':
    responses=[json.loads(s) for s in (target/'direct-responses.jsonl').read_text().splitlines()];row['responses']=responses
    row['valid']=len(responses)==3 and all(r['ok'] for r in responses)
   else:
    resultfile=target/'result.json';row['result']=json.loads(resultfile.read_text()) if resultfile.exists() else {'error':'missing result'};row['valid']=not row['result'].get('error')
   rows.append(row);h.save(folder/'results.json',rows);print('END',cell['label'],elapsed,flush=True)
   assert not interrupted and proc.returncode==0 and row['valid'],'suite stopped; no retry'
 finally:
  monitor.stop();h.save(folder/'completion.json',{'cells':len(rows),'monitor_failed':monitor.failed,'seconds':time.monotonic()-started})

if __name__=='__main__':
 mode=sys.argv[1]
 if mode=='prepare':prepare()
 elif mode=='proxy':h.proxy()
 elif mode in ['stage1','stage2']:run(mode)
 else:raise ValueError('prepare/proxy/stage1/stage2 only')
