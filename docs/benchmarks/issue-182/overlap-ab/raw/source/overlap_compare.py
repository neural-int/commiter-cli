"""Prepare/run H182-007 A/B; no generation during prepare."""
import json,pathlib,shutil,subprocess,sys
import prompt_compare as harness
ROOT=harness.ROOT
PARENT=pathlib.Path('<PRIVATE_TMP>/issue182-release-comparison')
OUT=pathlib.Path('<PRIVATE_TMP>/issue182-overlap-ab')
RUNTIME=OUT/'runtime'
harness.OUT=OUT;harness.RUNTIME=RUNTIME
original_environment=harness.environment

def environment(cell):
 env=original_environment(cell)
 env['COMMITER_182_PIPELINE']=cell['pipeline']
 return env
harness.environment=environment

def prepare():
 assert not (OUT/'contract.json').exists()
 parent=json.loads((PARENT/'contract.json').read_text())
 changed=ROOT/'mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift'
 assert all(harness.sha(p)==h for p,h in parent['hashes'].items() if p!=str(changed)),'reference asset drift'
 RUNTIME.mkdir(parents=True,exist_ok=True)
 shutil.copyfile('<PRIVATE_TMP>/issue182-ministral-runtime/swift-build/release/commiter-mlx-helper',RUNTIME/'commiter-mlx-helper')
 (RUNTIME/'commiter-mlx-helper').chmod(0o755)
 shutil.copyfile(PARENT/'runtime/mlx.metallib',RUNTIME/'mlx.metallib')
 (RUNTIME/'proxy').write_text('#!/bin/sh\nexec /opt/homebrew/bin/python3 '+str(pathlib.Path(__file__).resolve())+' proxy "$@"\n')
 (RUNTIME/'proxy').chmod(0o755)
 shutil.copyfile(PARENT/'repos.json',OUT/'repos.json');shutil.copytree(PARENT/'fixtures',OUT/'fixtures')
 cells=[]
 for old in parent['cells']:
  capture=dict(old,label='capture-'+old['label'],pipeline='eager')
  env=environment(capture);env['COMMITER_180_MODE']='capture'
  subprocess.run([str(harness.PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env,check=True,capture_output=True)
  requests=[]
  for i in [1,2,3]:
   name=f'{i:02d}-request.json';r=json.loads((OUT/capture['label']/name).read_text())
   assert r==json.loads((PARENT/capture['label']/name).read_text()),'live request drift'
   r['model_path']=old['path'];requests.append(harness.transform(r,old['arm']))
  for variant in (['A','B'] if old['case']=='C01' else ['B','A']):
   c=dict(old,label=old['label']+'-'+variant,pipeline='eager' if variant=='A' else 'overlap')
   result=subprocess.run([str(RUNTIME/'commiter-mlx-helper'),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests[1:]),text=True,capture_output=True,env=environment(c),check=True)
   replies=[json.loads(line) for line in result.stdout.splitlines()];assert len(replies)==2 and all(r['ok'] for r in replies),result.stdout
   native=[json.loads(r['generated_json']) for r in replies]
   assert native==json.loads((PARENT/'audit'/(old['label']+'.json')).read_text())['native'],'token/cache drift'
   harness.save(OUT/'audit'/(c['label']+'.json'),{'requests':requests,'native':native,'prefix':c['prefix'],'pipeline':c['pipeline']})
   cells.append(c)
 files=[pathlib.Path(p) for p in parent['hashes']]+[pathlib.Path(__file__),RUNTIME/'proxy',RUNTIME/'commiter-mlx-helper',RUNTIME/'mlx.metallib',OUT/'repos.json',ROOT/'mlx-helper/Sources/BoundedGeneration/OverlappedTokenProcessor.swift',ROOT/'mlx-helper/Tests/BoundedGenerationTests/OverlappedTokenProcessorTests.swift']
 files+=list((OUT/'audit').glob('*.json'))+list((OUT/'fixtures').glob('*.json'))
 harness.save(OUT/'contract.json',dict(parent,id='H182-007-overlap-ab',cells=cells,hashes={str(p):harness.sha(p) for p in files},max_calls=24,baseline='A eager is remeasured in same Release binary; historical timings are reference only',build='same Release binary and shaders for A and B',limitations='one A/B pair per case; serial C01 A-B and C02 B-A; no timing claims before generation; state semantics are committed before returned token, unreturned lookahead discarded'))
 print('READY: eight A/B cells audited; shared Release binary; no model generation performed',flush=True)

if __name__=='__main__':
 if sys.argv[1]=='prepare':prepare()
 elif sys.argv[1]=='proxy':harness.proxy()
 elif sys.argv[1]=='run':harness.run()
 else:raise ValueError('mode')
