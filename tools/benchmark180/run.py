"""Finite B/C experiment. No downloads, retries, timeout, repair, or D fallback."""
import argparse, datetime, hashlib, json, os, pathlib, signal, subprocess

ROOT=pathlib.Path(__file__).resolve().parents[2]
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def save(path,value):path.write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
def runtime_paths(helper):
    paths=[helper]
    if helper.name=='h180004-helper.sh':
        native=os.environ.get('COMMITER_180_H180004_NATIVE_HELPER')
        if not native:
            raise RuntimeError('COMMITER_180_H180004_NATIVE_HELPER is required when using h180004-helper.sh')
        binary=pathlib.Path(native).resolve()
        paths.extend([binary,binary.parent/'mlx.metallib'])
    else:
        paths.append(helper.parent/'mlx.metallib')
    return paths
def git_state(root,paths):
    return {'head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip(),
      'index_canonical_sha256':hashlib.sha256(subprocess.check_output(['git','ls-files','--stage','-z'],cwd=root)).hexdigest(),
      'files':{p:sha(root/p) for p in paths}}
def snapshot_resources():
    result={'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'scope':'system snapshot; other workloads uncontrolled'}
    for key,cmd in [('pressure',['memory_pressure','-Q']),('swap',['sysctl','vm.swapusage'])]:
        p=subprocess.run(cmd,capture_output=True,text=True);result[key]={'exit':p.returncode,'stdout':p.stdout,'stderr':p.stderr}
    return result

def freeze(a):
    a.bundle.mkdir(exist_ok=False,parents=True)
    sources={}
    files=subprocess.check_output(['git','ls-files','-z'],cwd=ROOT).decode().split('\0')
    for name in files:
        if name and (name.endswith(('.go','.swift','.py')) or name in ['go.mod','go.sum','mlx-helper/Package.resolved']):sources[name]=sha(ROOT/name)
    git_before={r['case_id']:git_state(pathlib.Path(r['root']),r['selected_paths']) for r in json.loads(a.repos.read_text())}
    import shutil
    shutil.copytree(a.capture,a.bundle/'capture');shutil.copytree(a.audit,a.bundle/'prefix-audit')
    shutil.copy(ROOT/'docs/benchmarks/issue-178/iteration-03/gold-frozen.json',a.bundle/'gold-frozen.json')
    audit=json.loads((a.audit/'prefix-audit.json').read_text())
    # The actual first4 IDs are only BOS/turn/system/newline; divergence is
    # before even the shared system text, hence before any repository facts.
    if not all(r['messages_shared'] and r['longest_common_prefix_tokens']==4 and r['first_token_ids'][0][4]==98 and r['first_token_ids'][1][4]!=98 for r in audit):
        raise RuntimeError('unexpected prefix gate: investigate before registering generation')
    save(a.bundle/'execution-contract.json',{
      'created_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),
      'source_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'source_hashes':sources,
      'runtime':{str(p.resolve()):sha(p) for p in [*runtime_paths(a.helper),a.binary,a.overlay]},
      'overlay_source_hashes':{p:sha(pathlib.Path(p)) for p in json.loads(a.overlay.read_text())['Replace'].values()},
      'artifact_hashes':{str(p.relative_to(a.bundle)):sha(p) for p in a.bundle.rglob('*') if p.is_file()},
      'model':'mlx-community/gemma-4-E4B-it-4bit','revision':'475b9088d29754a3379866cf5aeb6b41acd313c2',
      'binary':str(a.binary.resolve()),'helper':str(a.helper.resolve()),'repos':str(a.repos.resolve()),
      'gold':'unchanged agent-authored exploratory reference, not independent human Gold',
      'order':['C01-B','C01-C','C02-B','C02-C'],'repeats':1,'max_generation_requests':12,'retries':0,'repairs':0,'judge_calls':0,
      'timeout_seconds':None,'runner_timeout_seconds':None,'production_timeout_changed':False,
      'conditions':{'A':'reuse #178 iteration03 history-present results; no rerun; different timeout; C02 incomplete',
        'B':'new message placement, one helper/model load per phase, no KV reuse',
        'C':'same placement, one helper/model/tokenizer across Grouping/Category/Text, fresh generation/cache per phase',
        'D':'not_evaluated_prefix_gate_failed: actual LCP4tokens, thinking diverges before facts; no cache implementation or sampling'},
      'fixed_membership':'Gold partition injected ONLY after recording and validating actual Grouping output; not end-to-end grouping success',
      'history':'present, unchanged; phase requests retain full diffs and selected history',
      'generation':{'temperature':0,'seed':144,'top_p':1,'top_k':0,'native_budgets':[512,384,0],'output_budgets':[768,512,768],'context_ceiling':65536},
      'stop':'stop failed phase/cycle without retry; continue remaining registered cycles; abort on infrastructure failure/cancel; no further generation after4 cycles',
      'metrics':'load/prepare/grammar setup/generation wall; dependency promptTime includes prefill and first token; generation time/tokens; MLX allocator active/peak (cycle cumulative in C); not OS RSS/system peak',
      'git_before':git_before})
    print('frozen',a.bundle)

def verify(bundle,c):
    checks={**{str(ROOT/p):s for p,s in c['source_hashes'].items()},**c['runtime'],**c['overlay_source_hashes'],**{str(bundle/p):s for p,s in c['artifact_hashes'].items()}}
    bad=[p for p,s in checks.items() if sha(pathlib.Path(p))!=s]
    return bad

def run(a):
    c=json.loads((a.bundle/'execution-contract.json').read_text());bad=verify(a.bundle,c)
    if bad:raise RuntimeError('frozen content changed: '+repr(bad))
    out=a.bundle/'runs';out.mkdir(exist_ok=False)
    save(a.bundle/'resource-before.json',snapshot_resources())
    active=None
    def cancel(signum,frame):
        if active is not None:active.send_signal(signal.SIGTERM)
        raise KeyboardInterrupt
    signal.signal(signal.SIGTERM,cancel)
    try:
        for label in c['order']:
            cid,arm=label.split('-');env=os.environ.copy();env.update(COMMITER_180_MODE='run',COMMITER_180_CASE=cid,COMMITER_180_CONDITION=arm,COMMITER_180_OUT=str((out/label).resolve()),COMMITER_180_REPOS=c['repos'],COMMITER_180_HELPER=c['helper'])
            print('start',label,flush=True)
            with (a.bundle/f'{label}-terminal.log').open('x') as log:
                active=subprocess.Popen([c['binary'],'-test.run=^TestIssue180Evaluation$','-test.v','-test.timeout=0'],cwd=ROOT/'internal/cli',env=env,stdout=log,stderr=subprocess.STDOUT)
                code=active.wait();active=None
            if code:raise RuntimeError('runner infrastructure failure; no retry: '+label)
            count=len(list(out.glob('*/*-request.json')))-len(list(out.glob('*/*-original-request.json')))
            if count>c['max_generation_requests']:raise RuntimeError('global request cap')
            r=json.loads((out/label/'result.json').read_text());print(label,round(r['wall_seconds'],2),r['error'] or 'accepted',flush=True)
    finally:
        if active is not None:
            active.send_signal(signal.SIGTERM);active.wait()
        save(a.bundle/'resource-after.json',snapshot_resources())
        after={r['case_id']:git_state(pathlib.Path(r['root']),r['selected_paths']) for r in json.loads(pathlib.Path(c['repos']).read_text())}
        save(a.bundle/'completion-audit.json',{'content_mismatches':verify(a.bundle,c),'git_unchanged':after==c['git_before'],'git_after':after,'generation_requests':len(list(out.glob('*/*-response.json')))})

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('mode',choices=['freeze','run']);p.add_argument('--bundle',type=pathlib.Path,required=True)
    for name in ['helper','binary','overlay','repos','capture','audit']:p.add_argument('--'+name,type=pathlib.Path)
    a=p.parse_args();freeze(a) if a.mode=='freeze' else run(a)
