"""H182-006: Release helper, otherwise H182-005 conditions unchanged."""
import json,pathlib,shutil,subprocess,sys
import prompt_compare as harness
ROOT=harness.ROOT
PARENT=pathlib.Path('/private/tmp/issue182-thinking-budget')
OUT=pathlib.Path('/private/tmp/issue182-release-comparison')
RUNTIME=OUT/'runtime'
harness.OUT=OUT;harness.RUNTIME=RUNTIME

def prepare():
 assert not (OUT/'contract.json').exists()
 parent=json.loads((PARENT/'contract.json').read_text())
 assert all(harness.sha(p)==h for p,h in parent['hashes'].items()),'Debug reference drift'
 RUNTIME.mkdir(parents=True,exist_ok=True)
 shutil.copyfile('/private/tmp/issue182-ministral-runtime/swift-build/release/commiter-mlx-helper',RUNTIME/'commiter-mlx-helper')
 (RUNTIME/'commiter-mlx-helper').chmod(0o755)
 # Keep the same GPU shader library; change host Swift compilation only.
 shutil.copyfile(PARENT/'runtime/mlx.metallib',RUNTIME/'mlx.metallib')
 (RUNTIME/'proxy').write_text('#!/bin/sh\nexec /opt/homebrew/bin/python3 '+str(pathlib.Path(__file__).resolve())+' proxy "$@"\n')
 (RUNTIME/'proxy').chmod(0o755)
 shutil.copyfile(PARENT/'repos.json',OUT/'repos.json');shutil.copytree(PARENT/'fixtures',OUT/'fixtures')
 cells=parent['cells'];assert len(cells)==4
 for c in cells:
  capture=dict(c,label='capture-'+c['label']);env=harness.environment(capture);env['COMMITER_180_MODE']='capture'
  subprocess.run([str(harness.PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env,check=True,capture_output=True)
  requests=[]
  for i in [1,2,3]:
   name=f'{i:02d}-request.json';r=json.loads((OUT/capture['label']/name).read_text())
   assert r==json.loads((PARENT/capture['label']/name).read_text()),'live input drift'
   r['model_path']=c['path'];requests.append(harness.transform(r,c['arm']))
  result=subprocess.run([str(RUNTIME/'commiter-mlx-helper'),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests[1:]),text=True,capture_output=True,env=harness.environment(c),check=True)
  replies=[json.loads(line) for line in result.stdout.splitlines()];assert len(replies)==2 and all(r['ok'] for r in replies),result.stdout
  actual=[json.loads(r['generated_json']) for r in replies];old=json.loads((PARENT/'audit'/(c['label']+'.json')).read_text())
  assert actual==old['native'],'token/cache/thinking drift'
  harness.save(OUT/'audit'/(c['label']+'.json'),{'requests':requests,'native':actual,'prefix':c['prefix'],'equal_to_debug':True})
 files=[pathlib.Path(p) for p in parent['hashes']]+[pathlib.Path(__file__),RUNTIME/'proxy',RUNTIME/'commiter-mlx-helper',RUNTIME/'mlx.metallib',OUT/'repos.json',PARENT/'summary.json']
 files+=list((OUT/'audit').glob('*.json'))+list((OUT/'fixtures').glob('*.json'))
 harness.save(OUT/'contract.json',dict(parent,id='H182-006-release',hashes={str(p):harness.sha(p) for p in files},baseline='H182-005 Debug, saved measurements; no Debug rerun',parent_contract_sha256=harness.sha(PARENT/'contract.json'),build='swift test -c release --build-system native; same source/dependencies and identical mlx.metallib',limitations='one repeat per case; Debug and Release measured at different times; same fixed downstream partition; no independent human quality Gold'))
 print('AUDIT PASSED: four Release cells; all requests, tokens, KV topology, configs and shaders match Debug',flush=True)

if __name__=='__main__':
 if sys.argv[1]=='prepare':prepare()
 elif sys.argv[1]=='proxy':harness.proxy()
 elif sys.argv[1]=='run':harness.run()
 else:raise ValueError('mode')
