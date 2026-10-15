"""Capture real planner requests without inference; freeze source/fixtures/Gold."""
import argparse, hashlib, json, os, shutil, subprocess
from pathlib import Path
SRC=Path(__file__).resolve().parents[2]; DATA=SRC/'docs/benchmarks/issue-173'
def h(b):return hashlib.sha256(b).hexdigest()
def save(p,x):p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(x,indent=2,ensure_ascii=False)+'\n')
def git(root,*args):return subprocess.check_output(['git',*args],cwd=root,env=dict(os.environ,GIT_CONFIG_GLOBAL='/dev/null',GIT_CONFIG_NOSYSTEM='1'))
def state(root):return {k:h(git(root,*a)) for k,a in {'head':['rev-parse','HEAD'],'index':['ls-files','--stage'],'diff':['diff','--binary'],'status':['status','--porcelain']}.items()}
def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',type=Path,required=True);ap.add_argument('--prior',type=Path,required=True);ap.add_argument('--helper',type=Path,required=True);ap.add_argument('--preflight-helper',type=Path,required=True);a=ap.parse_args();root=a.root.resolve();prior=a.prior.resolve()
    if (DATA/'preregistration.json').exists():raise ValueError('never overwrite a frozen suite')
    root.mkdir(parents=True,exist_ok=True);(root/'bin').mkdir();(root/'libexec').mkdir();(root/'fixtures').mkdir()
    subprocess.run(['go','build','-o',str(root/'bin/commiter'),'./cmd/commiter'],cwd=SRC,check=True)
    shutil.copy2(SRC/'tools/benchmark173/trace_helper.py',root/'libexec/commiter-mlx-helper');(root/'libexec/commiter-mlx-helper').chmod(0o755)
    (root/'config/commiter').mkdir(parents=True);shutil.copy2(prior/'config/commiter/config.toml',root/'config/commiter/config.toml')
    model=json.loads((prior/'capture/A/02.request.json').read_text())
    runtime={'helper':str(a.helper.resolve()),'preflight_helper':str(a.preflight_helper.resolve()),'model_path':model['model_path']}
    save(root/'runtime.json',runtime)
    env=dict(os.environ,XDG_CONFIG_HOME=str(root/'config'),XDG_STATE_HOME=str(root/'state'),GIT_CONFIG_GLOBAL='/dev/null',GIT_CONFIG_NOSYSTEM='1',BENCH173_ROOT=str(root),BENCH173_CAPTURE='1')
    gold={};boundary=json.loads((SRC/'tools/benchmark173/boundaries.json').read_text())
    for case in 'ABCDE':shutil.copytree(prior/'fixtures'/case,root/'fixtures'/case)
    for item in boundary:
        repo=root/'fixtures'/item['id'];repo.mkdir();git(repo,'init','-b','codex/fixture');git(repo,'config','user.name','Category Fixture');git(repo,'config','user.email','fixture@example.invalid')
        for name,data in {'.commiter.toml':'schema_version = 1\n[verification]\nautodetect = false\n',**item['before']}.items():
            p=repo/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(bytes.fromhex(data[4:]) if data.startswith('hex:') else data.encode())
        git(repo,'add','.');git(repo,'commit','-m','test: Category境界fixtureの初期状態を固定')
        for name,data in item['after'].items():
            p=repo/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(bytes.fromhex(data[4:]) if data.startswith('hex:') else data.encode())
        gold[item['id']]={'groups':{'G001':item['type']},'identifiability':item['class'],'reason':item['reason'],'independence':'agent-authored synthetic Gold, not independent human evaluation'}
    oldgold=json.loads((prior/'gold.json').read_text());save(DATA/'prior-gold.json',oldgold)
    rows=[]
    for case in [*'ABCDE',*[x['id'] for x in boundary]]:
        repo=root/'fixtures'/case;trace=root/'capture'/case
        before=state(repo)
        cli=subprocess.run([str(root/'bin/commiter'),'--json','--dry-run','--record-metrics'],cwd=repo,env=dict(env,BENCH173_CASE=case,BENCH173_TRACE=str(trace)),capture_output=True)
        trace.mkdir(parents=True,exist_ok=True);(trace/'cli.stdout.json').write_bytes(cli.stdout);(trace/'cli.stderr.jsonl').write_bytes(cli.stderr)
        if cli.returncode:raise ValueError('offline capture failed '+case+': '+cli.stderr.decode()[:150])
        assert before==state(repo),'capture changed Git'
        candidate=json.loads((trace/'02.request.json').read_text());groups=json.loads(candidate['messages'][1]['content'])['groups']
        if case in 'ABCDE':
            gold[case]={'groups':{f'G{i+1:03}':g['expected_type'] for i,g in enumerate(oldgold[case]['groups'])},'identifiability':'identifiable','reason':'Reuse exact PR #172 source-bound Gold; do not rewrite prior evidence.','independence':'known reused fixtures'}
            baseline=json.loads((prior/'capture'/case/'02.request.json').read_text());baseline.pop('model_path',None)
            save(DATA/f'inputs/{case}-baseline.json',baseline)
            # Verify unchanged Grouping/Text messages, schemas and budgets byte-for-byte semantically.
            for phase in ['01','03']:
                previous=json.loads((prior/'capture'/case/f'{phase}.request.json').read_text());current=json.loads((trace/f'{phase}.request.json').read_text());previous.pop('model_path',None)
                assert previous==current,(case,phase,'Grouping/Text drift')
        save(DATA/f'inputs/{case}-candidate.json',candidate)
        for phase in ['01','03']:save(DATA/f'integrated-inputs/{case}-{phase}.json',json.loads((trace/f'{phase}.request.json').read_text()))
        paths=git(repo,'diff','--name-only').decode().splitlines()
        rows.append({'case':case,'git_state':before,'paths':paths,'before_sha256':{p:h(git(repo,'show','HEAD:'+p)) if (repo/p).exists() else None for p in paths},'after_sha256':{p:h((repo/p).read_bytes()) if (repo/p).exists() else None for p in paths},'category_groups':[{ 'group_id':g['group_id'],'contents_paths':list(g['contents']),'related_test_paths':list(g['related_tests'] or {})} for g in groups]})
    save(DATA/'gold.json',gold);save(DATA/'fixtures.json',rows)
    # All model files are pinned. No paths or file bytes are published.
    modelpins=[]
    for p in sorted(Path(runtime['model_path']).iterdir()):
        if p.is_file():modelpins.append({'name':p.name,'bytes':p.stat().st_size,'sha256':h(p.read_bytes())})
    save(DATA/'model-artifacts.json',modelpins)
    order=[]
    for i,case in enumerate('ABCDE'):
        for arm in (['baseline','candidate'] if i%2==0 else ['candidate','baseline']):order.append([case,arm])
    order += [[x['id'],'candidate'] for x in boundary]
    pins={str(p.relative_to(SRC)):h(p.read_bytes()) for p in sorted((SRC/'internal').rglob('*.go'))}
    pins.update({str(p.relative_to(SRC)):h(p.read_bytes()) for p in sorted((SRC/'tools/benchmark173').glob('*')) if p.is_file()})
    pins.update({str(p.relative_to(SRC)):h(p.read_bytes()) for p in sorted(DATA.rglob('*.json'))})
    pr={'source_base':'0f9278c33c2f477372bc7dd07c7616a738dd6268','goal_blob':'ad92785ee8b4f4d04ded9c5755c4450cf02281ca','model':model['model'],'helper_sha256':h(Path(runtime['helper']).read_bytes()),'preflight_helper_sha256':h(Path(runtime['preflight_helper']).read_bytes()),'cli_sha256':h((root/'bin/commiter').read_bytes()),'source_fixture_request_sha256':pins,'category_order':order,'integrated_order':list('ABCDE'),'max_category_calls':30,'max_integrated_cycles':5,'max_integrated_calls':15,'max_generation_calls':45,'retries':0,'repairs':0,'temperature':0,'top_p':1,'top_k':0,'seed':144,'category_native_thought_limit':384,'context_ceiling':65536,'output_budgets':[768,512,768],'category_timeout_seconds':120,'shared_cycle_timeout_seconds':120,'gate':{'safety':'zero Git mutation, no Grouping/Text drift, no membership loss, no schema violation','paired':'candidate Gold exact >= 5/6 group decisions and strictly above baseline; no false definite feat/fix on boundary ambiguity','boundaries':'all 20 source-bound category decisions exact, including each of the two fallback types','integrated':'all 5 actual cycles accepted by final Validator within shared deadline; failure here is separate from Category and stops adoption','resources':'no critical pressure >=4, no helper RSS > 12 GiB, no per-call swap growth > 2 GiB'},'stop':'No prompt/Gold/fixture edits or additional inference after freeze. On quality failure retain evidence and stop adoption. Complete predeclared category and integrated measurements unless safety/resource stop. Text changes/deadline changes are out of scope.','evaluation_independence':'agent-authored, known-source synthetic benchmark; no generalization/16-file/80% production claim','token_counts':'native tokenize-only preflight per exact request; output token count unavailable from fixed production helper, never estimate as measured'}
    save(DATA/'preregistration.json',pr)
    print(json.dumps({'offline_captures':len(rows),'category_calls':len(order),'integrated_max_calls':15,'freeze_sha256':h((DATA/'preregistration.json').read_bytes())}))
if __name__=='__main__':main()
