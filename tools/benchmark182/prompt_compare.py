"""H182-004 prompt-only comparison; prepare audits before any generation."""
import os, tempfile
import argparse,copy,hashlib,json,os,pathlib,re,signal,subprocess,sys,time
ROOT=pathlib.Path(__file__).resolve().parents[2]
BENCHMARK_ROOT=pathlib.Path(os.environ.get('COMMITER_BENCHMARK_ROOT', tempfile.gettempdir()))
OUT=pathlib.Path(str(BENCHMARK_ROOT / 'issue182-prompt-comparison'))
PREV=pathlib.Path(str(BENCHMARK_ROOT / 'issue182-suite-v3'))
RUNTIME=OUT/'runtime'
def save(p,d):p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
def sha(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
def prompts():
 s=(ROOT/'docs/benchmarks/issue-182/prompt-candidate.md').read_text()
 blocks=re.findall(r'```text\n(.*?)\n```',s,re.S);assert len(blocks)==4
 return blocks

def transform(request,arm):
 r=copy.deepcopy(request)
 if arm=='current':return r
 common,grouping,category,text=prompts()
 phase=r['generation_profile']
 schema=' The required JSON Schema is: '+json.dumps(r['schema'],separators=(',',':'),sort_keys=True)
 if phase=='bounded-grouping':r['messages'][0]['content']=common+'\n\n'+grouping+schema
 else:
  r['messages'][0]['content']=common
  instruction=json.loads(r['messages'][-1]['content'])
  instruction['phase_instructions']=(category if phase=='bounded-category' else text)+schema
  r['messages'][-1]['content']=json.dumps(instruction,ensure_ascii=False,separators=(',',':'),sort_keys=True)
 return r

def invariant(a,b):
 assert {k:v for k,v in a.items() if k!='messages'}=={k:v for k,v in b.items() if k!='messages'}
 assert [m['role'] for m in a['messages']]==[m['role'] for m in b['messages']]
 if len(a['messages'])==2:assert a['messages'][1]==b['messages'][1]
 else:
  assert a['messages'][1]==b['messages'][1]
  x=json.loads(a['messages'][2]['content']);y=json.loads(b['messages'][2]['content'])
  x.pop('phase_instructions');y.pop('phase_instructions');assert x==y

def proxy():
 arm=os.environ['COMMITER_182_PROMPT_ARM'];target=pathlib.Path(os.environ['COMMITER_180_OUT'])
 process=subprocess.Popen([str(RUNTIME/'commiter-mlx-helper'),*sys.argv[2:]],stdin=subprocess.PIPE,stdout=subprocess.PIPE,text=True)
 try:
  for n,line in enumerate(sys.stdin,1):
   original=json.loads(line);wire=transform(original,arm);invariant(original,wire)
   save(target/f'{n:02d}-actual-request.json',wire)
   process.stdin.write(json.dumps(wire)+'\n');process.stdin.flush()
   response=process.stdout.readline()
   if not response:break
   print(response.strip(),flush=True)
 finally:
  process.stdin.close();process.wait()

def environment(cell):
 e={k:v for k,v in os.environ.items() if not k.startswith(('COMMITER_180_','COMMITER_182_'))}
 e.update(COMMITER_182_CASE_CONFIG=cell['config'],COMMITER_182_MODEL=cell['model'],COMMITER_182_MODEL_PATH=cell['path'],COMMITER_182_PROMPT_ARM=cell['arm'],COMMITER_182_PROGRESS='1',COMMITER_180_MODE='run',COMMITER_180_CONDITION='D',COMMITER_180_THINKING='T',COMMITER_180_CASE=cell['case'],COMMITER_180_OUT=str(OUT/cell['label']),COMMITER_180_REPOS=str(OUT/'repos.json'),COMMITER_180_FIXTURE_DIR=str(OUT/'fixtures'),COMMITER_180_HELPER=str(RUNTIME/'proxy'),COMMITER_180_KV_BITS='8',COMMITER_180_PREFIX_TOKENS=str(cell.get('prefix',0)))
 return e

def prepare():
 assert not (OUT/'contract.json').exists()
 OUT.mkdir(parents=True,exist_ok=True);RUNTIME.mkdir(exist_ok=True)
 (RUNTIME/'proxy').write_text('#!/bin/sh\nexec /opt/homebrew/bin/python3 '+str(pathlib.Path(__file__).resolve())+' proxy "$@"\n');(RUNTIME/'proxy').chmod(0o755)
 fixtures=pathlib.Path(os.environ.get('COMMITER_182_REFERENCE_FIXTURES', str(ROOT/'docs/benchmarks/issue-175/semantic-evaluation')))
 (OUT/'fixtures').mkdir(exist_ok=True)
 import shutil
 for case in ['c01','c02']:shutil.copyfile(fixtures/(case+'-fixture.json'),OUT/'fixtures'/(case+'-fixture.json'))
 shutil.copyfile(str(BENCHMARK_ROOT / 'issue182-ministral-runtime/recovered-fixtures-v2/repos.json'),OUT/'repos.json')
 models=[('gemma-on',pathlib.Path(os.environ['COMMITER_182_GEMMA_MODEL'])),('qwen4-on',pathlib.Path(str(BENCHMARK_ROOT / 'issue182-models/Qwen3.5-4B-4bit'))),('qwen2-off',pathlib.Path(str(BENCHMARK_ROOT / 'issue182-models/Qwen3.5-2B-4bit')))]
 cells=[];files=[pathlib.Path(__file__),ROOT/'docs/benchmarks/issue-182/prompt-candidate.md',ROOT/'mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift',RUNTIME/'commiter-mlx-helper',RUNTIME/'mlx.metallib',RUNTIME/'proxy',PREV/'evaluation.test',OUT/'repos.json']
 for model,path in models:
  cfg=ROOT/f'docs/benchmarks/issue-182/suite-v2/{model}.json';config=json.loads(cfg.read_text());files.append(cfg)
  for name in ['model.safetensors','config.json','tokenizer.json','tokenizer_config.json','chat_template.jinja']:files.append(path/name)
  for case in ['C01','C02']:
   # Capture from real fixture without inference and compare all phase inputs.
   capture=OUT/f'capture-{model}-{case}';c=dict(config=str(cfg),model=config['model'],path=str(path),arm='current',case=case,label=capture.name)
   env=environment(c);env['COMMITER_180_MODE']='capture'
   subprocess.run([str(PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env,check=True,capture_output=True)
   requests=[]
   for n in [1,2,3]:
    r=json.loads((capture/f'{n:02d}-request.json').read_text());base=json.loads((PREV/('capture-'+case)/f'{n:02d}-request.json').read_text())
    x=copy.deepcopy(r);x.pop('model',None);base.pop('model',None);assert x==base,'fixture/request drift'
    r['model_path']=str(path);requests.append(r);files.append(capture/f'{n:02d}-request.json')
   for arm in (['current','compact'] if case=='C01' else ['compact','current']):
    wire=[transform(r,arm) for r in requests]
    for a,b in zip(requests,wire):invariant(a,b)
    env=environment(c)
    p=subprocess.run([str(RUNTIME/'commiter-mlx-helper'),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in wire[1:]),text=True,capture_output=True,env=env,check=True)
    replies=[json.loads(line) for line in p.stdout.splitlines()];assert len(replies)==2 and all(r['ok'] for r in replies),p.stdout
    audits=[json.loads(r['generated_json']) for r in replies];a,b=[r['token_ids'] for r in audits]
    prefix=next((i for i,(x,y) in enumerate(zip(a,b)) if x!=y),min(len(a),len(b)));assert 4<prefix<min(len(a),len(b))
    assert all(sum(layer['kind']=='affine' for layer in row['planned_cache'])==config['attentionLayers'] for row in audits)
    label=f'{model}-{case}-{arm}';save(OUT/'audit'/f'{label}.json',{'requests':wire,'native':audits,'prefix':prefix,'input_tokens':[len(a),len(b)]})
    cells.append(dict(c,arm=arm,label=label,prefix=prefix))
 files+=list((OUT/'audit').glob('*.json'))+list((OUT/'fixtures').glob('*.json'))+[ROOT/'tools/benchmark182/monitor_v2.py',ROOT/'tools/benchmark182/run.py']
 save(OUT/'contract.json',{'id':'H182-004-prompt-only','cells':cells,'hashes':{str(p):sha(p) for p in files},'repeats':1,'max_calls':36,'warmup':0,'retry':0,'repair':0,'thinking_budget':None,'output_budget':None,'cycle_timeout':None,'seed':144,'sampling':'unchanged case configs within each pair','safety':'RSS12GiB, monitor failure, no token progress300s; interruption is not success','limitations':'exploratory C01/C02, fixed partition downstream, no independent human Gold; original and compact share same runtime; historical results not reused'})
 print('AUDIT PASSED: 12 cells; facts/schema/types/configs invariant; real token prefixes verified',flush=True)

def run():
 from monitor_v2 import Monitor
 contract=json.loads((OUT/'contract.json').read_text());assert not (OUT/'completion.json').exists()
 monitor=Monitor(OUT/'monitor.jsonl');monitor.start();started=time.monotonic();rows=[]
 try:
  for c in contract['cells']:
   assert all(sha(p)==h for p,h in contract['hashes'].items()),'frozen drift'
   assert not monitor.failed
   target=OUT/c['label'];assert not target.exists()
   print('START',c['label'],flush=True)
   with (OUT/(c['label']+'.log')).open('x') as log:
    proc=subprocess.Popen([str(PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=environment(c),stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
    last=time.monotonic();size=0;interrupted=None
    while proc.poll() is None:
     current=sum(p.stat().st_size for p in target.glob('*helper-stderr.log')) if target.exists() else 0
     if current!=size:last=time.monotonic();size=current
     if monitor.failed or time.monotonic()-last>300:
      interrupted='monitor_failure' if monitor.failed else 'no_progress_300s'
      inventory=subprocess.check_output(['ps','-axo','pid=,pgid=,comm='],text=True)
      for line in inventory.splitlines():
       values=line.split(None,2)
       if len(values)==3 and values[2]==str(RUNTIME/'commiter-mlx-helper'):os.kill(int(values[0]),signal.SIGTERM)
      if proc.poll() is None:os.killpg(proc.pid,signal.SIGTERM)
      proc.wait(timeout=10);break
     time.sleep(1)
   f=target/'result.json';result=json.loads(f.read_text()) if f.exists() else {'error':'infrastructure failure'}
   rows.append({'cell':c['label'],'exit':proc.returncode,'interrupted':interrupted,'result':result});save(OUT/'results.json',rows)
   print('END',c['label'],result.get('error'),result.get('wall_seconds'),flush=True)
   if not f.exists() or interrupted:raise RuntimeError('suite stopped; no retry')
 finally:
  monitor.stop();save(OUT/'completion.json',{'cells':len(rows),'seconds':time.monotonic()-started,'monitor_failed':monitor.failed})

if __name__=='__main__':
 if sys.argv[1]=='proxy':proxy()
 elif sys.argv[1]=='prepare':prepare()
 elif sys.argv[1]=='run':run()
 else:raise ValueError('mode')
