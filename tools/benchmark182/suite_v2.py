"""Prepare/audit H180-derived model-specific suite. Never generates in prepare/check/audit."""
import argparse,hashlib,json,os,pathlib,subprocess
ROOT=pathlib.Path(__file__).resolve().parents[2]
CONFIG=ROOT/'docs/benchmarks/issue-182/suite-v2'
def save(p,d):p.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
def sha(p):
 h=hashlib.sha256()
 with pathlib.Path(p).open('rb') as source:
  for block in iter(lambda:source.read(4*1024*1024),b''):h.update(block)
 return h.hexdigest()
def cases():
 order=['gemma-on','gemma-off','qwen4-on','qwen4-off','qwen2-off','ministral-instruct','ministral-reasoning']
 return [(CONFIG/(key+'.json'),json.loads((CONFIG/(key+'.json')).read_text())) for key in order]
def environment(config,case,out,a,mode):
 env={k:v for k,v in os.environ.items() if not k.startswith(('COMMITER_180_','COMMITER_182_'))}
 env.update(COMMITER_182_CASE_CONFIG=str(config),COMMITER_180_MODE=mode,COMMITER_180_CONDITION='D',COMMITER_180_THINKING='T',COMMITER_180_CASE=case,COMMITER_180_OUT=str(out),COMMITER_180_REPOS=str(a.repos),COMMITER_180_FIXTURE_DIR=str(a.fixtures),COMMITER_180_HELPER=str(a.helper),COMMITER_180_KV_BITS='8')
 return env

def prepare(a):
 a.out.mkdir(parents=True,exist_ok=False)
 source=ROOT/'internal/cli/issue180_evaluation_test.go';text=source.read_text()
 start=text.index('\tmodelPath := ""');end=text.index('\t// run must be built',start)
 text=text[:start]+'\tmodelPath := os.Getenv("COMMITER_182_MODEL_PATH")\n'+text[end:]
 text=text.replace('"github.com/natsuki0413/commiter-cli/internal/mlxmodel"\n','').replace('"github.com/natsuki0413/commiter-cli/internal/config"\n','')
 text=text.replace('config.CandidateModel + "@" + config.CandidateRevision','os.Getenv("COMMITER_182_MODEL")')
 text=text.replace('request := mlx.Request{Messages: m, Schema: s, ContextTokens: o.ContextTokens, OutputTokens: o.OutputTokens','request := mlx.Request{Messages: m, Schema: s, ContextTokens: o.ContextTokens, OutputTokens: 0')
 replacement=a.out/'evaluation.go';replacement.write_text(text)
 planner=ROOT/'internal/planning/three_phase.go';ptext=planner.read_text();old='ctx, cancel := context.WithTimeout(parent, CandidateCycleTimeout)';assert old in ptext
 adjusted=a.out/'three_phase.go';adjusted.write_text(ptext.replace(old,'ctx, cancel := context.WithCancel(parent)'))
 overlay=a.out/'overlay.json';save(overlay,{'Replace':{str(source):str(replacement),str(planner):str(adjusted)}})
 subprocess.run(['go','test','-overlay='+str(overlay),'-c','-o',str(a.out/'evaluation.test'),'./internal/cli'],cwd=ROOT,check=True)
 for cid in ['C01','C02']:
  env=environment(cases()[0][0],cid,a.out/('capture-'+cid),a,'capture')
  subprocess.run([str(a.out/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env,check=True)
 save(a.out/'preparation.json',{'source_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'fixtures':str(a.fixtures),'repos':str(a.repos),'helper':str(a.helper),'cases':[v for _,v in cases()],'generation_calls':0,'baseline':'H180-004: same C01/C02, reference partition injection retained for downstream comparison, chunk512 sync eval, reclaim every4 + end, 8bit affine group64','changes':'7 model/mode cells; configured sampling; unbounded native thought and output; no cycle deadline; model-native cache topology','context_ceiling':65536,'transport_response_bytes':4194304})

def audit(a):
 paths=json.loads(a.paths.read_text());rows=[]
 for config,c in cases():
  for cid in ['C01','C02']:
   requests=[]
   for n in [2,3]:
    r=json.loads((a.out/('capture-'+cid)/f'{n:02d}-request.json').read_text());r.update(model=c['model'],model_path=paths[c['model']]);requests.append(r)
   env=environment(config,cid,a.out,a,'capture')
   p=subprocess.run([str(a.helper),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests),capture_output=True,text=True,env=env,check=True)
   replies=[json.loads(s) for s in p.stdout.splitlines()];assert len(replies)==2 and all(r['ok'] for r in replies)
   tokens=[json.loads(r['generated_json'])['token_ids'] for r in replies];lcp=0
   for x,y in zip(*tokens):
    if x!=y:break
    lcp+=1
   assert lcp>4 and lcp<min(map(len,tokens)), 'no reusable shared facts prefix'
   layouts=[json.loads(r['generated_json'])['planned_cache'] for r in replies]
   assert all(sum(layer['kind']=='affine' for layer in layout)==c['attentionLayers'] for layout in layouts)
   rows.append({'cache_types':[layer['nativeType'] for layer in layouts[0]],'id':c['id'],'case':cid,'prefix_tokens':lcp,'input_tokens':list(map(len,tokens))})
 save(a.out/'prefix-audit.json',rows)

def run(a):
 # Explicit run only, after the tokenizer audit and runtime hash freeze.
 paths=json.loads(a.paths.read_text());audit_rows=json.loads((a.out/'prefix-audit.json').read_text())
 assert not (a.out/'execution-contract.json').exists(), 'a suite can only be executed once'
 runtime=[pathlib.Path(__file__).with_name('monitor_v2.py'),pathlib.Path(__file__).with_name('run.py'),a.out/'prefix-audit.json',a.helper,a.helper.parent/'mlx.metallib',a.out/'evaluation.test',a.out/'overlay.json',a.out/'evaluation.go',a.out/'three_phase.go',pathlib.Path(__file__)]
 for directory in set(paths.values()):
  runtime.extend(p for p in pathlib.Path(directory).iterdir() if p.name in ['config.json','tokenizer.json','tokenizer_config.json','chat_template.jinja','model.safetensors'])
 contract={'cases':[v for _,v in cases()],'runtime_hashes':{str(p):sha(p) for p in runtime},'case_hashes':{str(p):sha(p) for p,_ in cases()},'prefix':audit_rows,'repeats':1,'maximum_calls':42,'retry':0,'output_tokens':None,'thinking_tokens':None,'cycle_timeout':None,'warmup':0,'fixture_and_reference':'same H180 C01/C02 exploratory reference; fixed partition injected after real Grouping validation','note':'OS monitoring must be arranged and documented before an explicit run; this command does not start inference without --monitor-ready'}
 if not a.monitor_ready:raise RuntimeError('configure OS monitoring before execution; no inference started')
 save(a.out/'execution-contract.json',contract)
 from monitor_v2 import Monitor
 monitor=Monitor(a.out/'monitor.jsonl');monitor.start()
 try:
  for config,c in cases():
   for cid in ['C01','C02']:
    if monitor.failed:raise RuntimeError('OS monitoring gate failed; no retry')
    env=environment(config,cid,a.out/(c['id']+'-'+cid),a,'run')
    env.update(COMMITER_182_MODEL=c['model'],COMMITER_182_MODEL_PATH=paths[c['model']],COMMITER_180_PREFIX_TOKENS=str(next(r['prefix_tokens'] for r in audit_rows if r['id']==c['id'] and r['case']==cid)))
    proc=subprocess.Popen([str(a.out/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env);monitor.active=proc
    code=proc.wait();monitor.active=None
    if code:raise RuntimeError('runner failure; no retry')
 finally:monitor.stop()

if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('mode',choices=['prepare','audit','run']);p.add_argument('--out',type=pathlib.Path,required=True);p.add_argument('--helper',type=pathlib.Path,required=True);p.add_argument('--repos',type=pathlib.Path,required=True);p.add_argument('--fixtures',type=pathlib.Path,required=True);p.add_argument('--paths',type=pathlib.Path);p.add_argument('--monitor-ready',action='store_true');a=p.parse_args()
 globals()[a.mode](a)
