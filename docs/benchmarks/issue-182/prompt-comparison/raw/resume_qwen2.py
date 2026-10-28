"""Resume only the four preregistered, unstarted Qwen2 OFF cells."""
import hashlib,importlib.util,json,pathlib,shutil,subprocess,sys
from datetime import datetime,timezone
parent=pathlib.Path('<PRIVATE_TMP>/issue182-prompt-comparison')
source=pathlib.Path('$HOME/.codex/worktrees/issue-182-model-comparison/commiter-cli/tools/benchmark182/prompt_compare.py')
sys.path.insert(0,str(source.parent))
spec=importlib.util.spec_from_file_location('prompt_comparison',source)
harness=importlib.util.module_from_spec(spec);spec.loader.exec_module(harness)
original=json.loads((parent/'contract.json').read_text())
assert all(harness.sha(p)==h for p,h in original['hashes'].items()),'frozen asset drift'
selected=[cell for cell in original['cells'] if cell['label'].startswith('qwen2-off-')]
assert len(selected)==4
assert all(not (parent/cell['label']).exists() for cell in selected),'previous outputs exist'
resume=parent/'qwen2-resume';resume.mkdir()
shutil.copyfile(parent/'repos.json',resume/'repos.json')
shutil.copytree(parent/'fixtures',resume/'fixtures')
harness.OUT=resume
for case in ['C01','C02']:
 cell=next(c for c in selected if c['case']==case)
 capture=dict(cell,label='capture-qwen2-off-'+case)
 env=harness.environment(capture);env['COMMITER_180_MODE']='capture'
 subprocess.run([str(harness.PREV/'evaluation.test'),'-test.run=^TestIssue180Evaluation$','-test.timeout=0'],cwd=harness.ROOT/'internal/cli',env=env,check=True,capture_output=True)
 for phase in [1,2,3]:
  name=f'{phase:02d}-request.json'
  assert json.loads((resume/capture['label']/name).read_text())==json.loads((parent/capture['label']/name).read_text()),'live input drift'
contract=dict(original,cells=selected,resumed_utc=datetime.now(timezone.utc).isoformat(),parent_contract_sha256=harness.sha(parent/'contract.json'),resume_reason='user_authorized_remaining_qwen2_off_only',max_calls=12)
contract['hashes'].update({str(resume/'repos.json'):harness.sha(resume/'repos.json'),str(pathlib.Path(__file__)):harness.sha(__file__)})
for f in (resume/'fixtures').glob('*.json'):contract['hashes'][str(f)]=harness.sha(f)
harness.save(resume/'contract.json',contract)
harness.save(resume/'preflight.json',{'all_original_hashes_match':True,'all_live_phase_requests_match':True,'thinking':False,'cells':[c['label'] for c in selected],'sampling':{'temperature':1,'top_p':1,'top_k':20},'seed':144,'output_budget':None,'thinking_budget':None,'timeout_seconds':None,'original_runtime_unchanged':True})
print('RESUME PREFLIGHT PASSED: original hashes and live requests match; four Qwen2 OFF cells only',flush=True)
harness.run()
