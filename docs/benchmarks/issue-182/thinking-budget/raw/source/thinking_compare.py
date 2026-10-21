"""H182-005: finite thought budget with model-specific adopted prompts."""
import json,pathlib,shutil,subprocess,sys
import prompt_compare as harness
ROOT=harness.ROOT
PARENT=pathlib.Path('/private/tmp/issue182-prompt-comparison')
OUT=pathlib.Path('/private/tmp/issue182-thinking-budget')
RUNTIME=OUT/'runtime'
harness.OUT=OUT
harness.RUNTIME=RUNTIME

def prepare():
 assert not (OUT/'contract.json').exists()
 parent=json.loads((PARENT/'contract.json').read_text())
 changed=ROOT/'mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift'
 assert all(harness.sha(p)==h for p,h in parent['hashes'].items() if p!=str(changed)), 'parent frozen drift'
 RUNTIME.mkdir(parents=True,exist_ok=True)
 shutil.copyfile('/private/tmp/issue182-ministral-runtime/swift-build/debug/commiter-mlx-helper',RUNTIME/'commiter-mlx-helper')
 (RUNTIME/'commiter-mlx-helper').chmod(0o755)
 shutil.copyfile(PARENT/'runtime/mlx.metallib',RUNTIME/'mlx.metallib')
 (RUNTIME/'proxy').write_text('#!/bin/sh\nexec /opt/homebrew/bin/python3 '+str(pathlib.Path(__file__).resolve())+' proxy "$@"\n')
 (RUNTIME/'proxy').chmod(0o755)
 shutil.copyfile(PARENT/'repos.json',OUT/'repos.json')
 shutil.copytree(PARENT/'fixtures',OUT/'fixtures')
 selected=[c for c in parent['cells'] if (c['label'].startswith('gemma-on-') and c['arm']=='compact') or (c['label'].startswith('qwen4-on-') and c['arm']=='current')]
 assert len(selected)==4
 cells=[];audits=[]
 for previous in selected:
  c=dict(previous);cfg=ROOT/f"docs/benchmarks/issue-182/thinking-budget/{c['label'].split('-C')[0]}-1024.json"
  old=json.loads(pathlib.Path(c['config']).read_text());new=json.loads(cfg.read_text())
  assert old['thinkingBudget'] is None and new['thinkingBudget']==1024
  assert {k:v for k,v in old.items() if k!='thinkingBudget'}=={k:v for k,v in new.items() if k!='thinkingBudget'}
  c.update(config=str(cfg),label=c['label']+'-thinking1024')
  capture=dict(c,label='capture-'+c['label']);env=harness.environment(capture);env['COMMITER_180_MODE']='capture'
  subprocess.run([str(harness.PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env,check=True,capture_output=True)
  requests=[]
  for i in [1,2,3]:
   name=f'{i:02d}-request.json';r=json.loads((OUT/capture['label']/name).read_text())
   assert r==json.loads((PARENT/('capture-'+c['label'].split('-C')[0]+'-'+c['case'])/name).read_text()),'live input drift'
   r['model_path']=c['path'];requests.append(harness.transform(r,c['arm']))
  result=subprocess.run([str(RUNTIME/'commiter-mlx-helper'),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests[1:]),text=True,capture_output=True,env=harness.environment(c),check=True)
  replies=[json.loads(line) for line in result.stdout.splitlines()];assert len(replies)==2 and all(r['ok'] for r in replies),result.stdout
  actual=[json.loads(r['generated_json']) for r in replies]
  oldaudit=json.loads((PARENT/'audit'/(previous['label']+'.json')).read_text())
  assert all(a['token_ids']==b['token_ids'] and a['planned_cache']==b['planned_cache'] for a,b in zip(actual,oldaudit['native'])),'token/cache drift'
  assert all(a['enable_thinking'] for a in actual)
  harness.save(OUT/'audit'/(c['label']+'.json'),{'requests':requests,'native':actual,'prefix':c['prefix'],'tokens_and_kv_equal_to_adopted_unlimited':True})
  cells.append(c);audits.append(c['label'])
 # Reject invalid budgets before model loading; these probes never generate tokens.
 probe=dict(cells[0]);bad=dict(new,thinkingBudget=0)
 invalid=OUT/'invalid-budget.json';harness.save(invalid,bad)
 probe['config']=str(invalid)
 result=subprocess.run([str(RUNTIME/'commiter-mlx-helper'),'--benchmark-token-audit'],input=json.dumps(requests[0])+'\n',text=True,capture_output=True,env=harness.environment(probe),check=True)
 assert not json.loads(result.stdout)['ok'],'zero thinking budget accepted'
 harness.save(OUT/'invalid-budget-audit.json',{'response':json.loads(result.stdout),'generated_tokens':0})
 files=[pathlib.Path(p) for p in parent['hashes']]+[pathlib.Path(__file__),RUNTIME/'proxy',RUNTIME/'commiter-mlx-helper',RUNTIME/'mlx.metallib',OUT/'repos.json',PARENT/'final-summary.json',ROOT/'docs/benchmarks/issue-182/prompt-adoption.md']
 files+=list((OUT/'audit').glob('*.json'))+list((OUT/'fixtures').glob('*.json'))+[pathlib.Path(c['config']) for c in cells]
 contract=dict(parent,id='H182-005-thinking1024',cells=cells,hashes={str(p):harness.sha(p) for p in files},max_calls=12,thinking_budget=1024,baseline='H182-004 adopted prompt cells; existing measurements, no unlimited reruns',parent_contract_sha256=harness.sha(PARENT/'contract.json'),limitations='single finite-budget repeat; historical unlimited reference uses earlier binary and time; model source change only connects optional budget; Qwen4 C02 unlimited was user-interrupted; no end-to-end quality claim')
 harness.save(OUT/'contract.json',contract)
 print('AUDIT PASSED: adopted prompts, sampling, live requests, native tokens and KV topology match; thinking1024 only; invalid zero rejected',flush=True)

if __name__=='__main__':
 if sys.argv[1]=='prepare':prepare()
 elif sys.argv[1]=='proxy':harness.proxy()
 elif sys.argv[1]=='run':harness.run()
 else:raise ValueError('mode')
