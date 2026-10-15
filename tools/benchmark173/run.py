"""Run the frozen Category comparison or independent integrated cycles once."""
import argparse, hashlib, json, os, re, signal, subprocess, time
from pathlib import Path
from prepare import SRC, DATA, h, save, state

def resources():
    pressure=int(subprocess.check_output(['/usr/sbin/sysctl','-n','kern.memorystatus_vm_pressure_level']))
    swap=subprocess.check_output(['/usr/sbin/sysctl','-n','vm.swapusage'],text=True)
    m=re.search(r'used = ([0-9.]+)([MG])',swap)
    return {'pressure_level':pressure,'swap_used_mib':float(m[1])*(1024 if m[2]=='G' else 1)}
def check(pr,root,runtime):
    for name,digest in pr['source_fixture_request_sha256'].items():assert h((SRC/name).read_bytes())==digest,'frozen artifact drift: '+name
    assert h(Path(runtime['helper']).read_bytes())==pr['helper_sha256']
    assert h((root/'bin/commiter').read_bytes())==pr['cli_sha256']
    for row in json.loads((DATA/'fixtures.json').read_text()):assert state(root/'fixtures'/row['case'])==row['git_state'],'fixture Git drift: '+row['case']
def critical(before,after,rss):return after['pressure_level']>=4 or after['swap_used_mib']-before['swap_used_mib']>2048 or (rss is not None and rss>12*1024**3)
def unique(pairs):
    d={}
    for k,v in pairs:
        if k in d:raise ValueError('duplicate JSON key')
        d[k]=v
    return d

def validate(text,req):
    result=json.loads(text,object_pairs_hook=unique);props=req['schema']['properties']
    if not isinstance(result,dict) or set(result)!=set(props):raise ValueError('group keys')
    for key,value in result.items():
        if not isinstance(value,dict) or set(value)!={'type','breaking_evidence_ref'}:raise ValueError('category fields')
        for field in value:
            if value[field] not in props[key]['properties'][field]['enum']:raise ValueError('category enum')
    return result

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',type=Path,required=True);ap.add_argument('--phase',choices=['category','integrated'],required=True);a=ap.parse_args();root=a.root.resolve();pr=json.loads((DATA/'preregistration.json').read_text());runtime=json.loads((root/'runtime.json').read_text());gold=json.loads((DATA/'gold.json').read_text());check(pr,root,runtime)
    journal=DATA/f'{a.phase}-attempts.jsonl';results=DATA/f'{a.phase}-results.jsonl'
    if journal.exists():raise ValueError('once-only suite already attempted; retain receipts, no repeats')
    lock=json.loads((DATA/'execution-lock.json').read_text())
    for name,digest in lock['sha256'].items():assert h((SRC/name).read_bytes())==digest,'execution contract drift'
    order=pr['category_order'] if a.phase=='category' else [[x,'candidate'] for x in pr['integrated_order']]
    for number,(case,arm) in enumerate(order,1):
        check(pr,root,runtime);before=resources();assert before['pressure_level']<4,'critical resource stop before generation'
        directory=root/'runs'/a.phase/f'{number:02d}-{case}-{arm}';directory.mkdir(parents=True);repo=root/'fixtures'/case;git_before=state(repo)
        attempt={'number':number,'case':case,'arm':arm,'phase':a.phase}
        with journal.open('a') as f:f.write(json.dumps(attempt)+'\n');f.flush();os.fsync(f.fileno())
        started=time.monotonic();rss=None;sampled_peak=0
        print('START '+json.dumps(attempt),flush=True)
        if a.phase=='category':
            req=json.loads((DATA/f'inputs/{case}-{arm}.json').read_text());req['model_path']=runtime['model_path']
            proc=subprocess.Popen(['/usr/bin/time','-l',runtime['helper']],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,start_new_session=True)
            try:out,err=proc.communicate(json.dumps(req).encode()+b'\n',timeout=120);status='returned'
            except subprocess.TimeoutExpired:
                os.killpg(proc.pid,signal.SIGTERM)
                try:out,err=proc.communicate(timeout=5)
                except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);out,err=proc.communicate()
                status='timeout'
            row={**attempt,'stop_reason':status,'schema_valid':False,'response':None,'output_tokens':None}
            m=re.search(rb'(?m)^\s*(\d+)\s+maximum resident set size\s*$',err)
            if m:rss=int(m[1])
            try:
                r=json.loads(out);row.update(stop_reason=r.get('stop_reason'),error_class=r.get('error_class'),helper_ok=r.get('ok'))
                if status!='timeout' and r.get('ok') and r.get('stop_reason')=='completed':
                    assert r['model']==pr['model'] and r['generation_profile']=='bounded-category','protocol drift'
                    row['response']=validate(r['generated_json'],req);row['schema_valid']=True
                if status=='timeout':row['stop_reason']='timeout'
            except (ValueError,KeyError,TypeError):row['validation_error']='invalid_helper_or_category_schema'
            row['input_tokens']=json.loads((DATA/'token-counts.json').read_text()).get(f'{case}-{arm}',{}).get('input_tokens')
            row['input_bytes']=sum(len(x['content'].encode()) for x in req['messages'])
            row['groups']=[{'group_id':g,'expected':typ,'actual':(row['response'] or {}).get(g,{}).get('type'),'identifiability':gold[case]['identifiability']} for g,typ in gold[case]['groups'].items()]
        else:
            env=dict(os.environ,XDG_CONFIG_HOME=str(root/'config'),XDG_STATE_HOME=str(directory/'state'),GIT_CONFIG_GLOBAL='/dev/null',GIT_CONFIG_NOSYSTEM='1',BENCH173_ROOT=str(root),BENCH173_TRACE=str(directory/'trace'),BENCH173_CASE=case)
            env.pop('BENCH173_CAPTURE',None)
            with (directory/'stdout.json').open('wb') as stdout,(directory/'stderr.txt').open('wb') as stderr:
                proc=subprocess.Popen([str(root/'bin/commiter'),'--json','--dry-run','--record-metrics'],cwd=repo,env=env,stdin=subprocess.DEVNULL,stdout=stdout,stderr=stderr)
                while proc.poll() is None:
                    tree=subprocess.check_output(['/bin/ps','-axo','pid=,ppid=,rss='],text=True)
                    entries=[tuple(map(int,x.split())) for x in tree.splitlines() if len(x.split())==3];children={proc.pid};last=-1
                    while last!=len(children):
                        last=len(children);children.update(pid for pid,ppid,_ in entries if ppid in children)
                    sampled_peak=max(sampled_peak,max([rss_kib*1024 for pid,_,rss_kib in entries if pid in children]+[0]));time.sleep(1)
            out=(directory/'stdout.json').read_bytes();err=(directory/'stderr.txt').read_bytes()
            try:r=json.loads(out)
            except ValueError:r={'error':'unparseable CLI output'}
            metas=[json.loads(p.read_text()) for p in sorted((directory/'trace').glob('*.meta.json'))]
            assert len(metas)<=3,'extra model call'
            row={**attempt,'plan_accepted':proc.returncode==0 and 'plan' in r,'cli_error':r.get('error'),'generation_calls':len(metas),'phase_calls':metas,'sampled_largest_process_peak_rss_bytes':sampled_peak}
            # Preserve only sanitized actual requests publicly, with plan and errors.
            for p in sorted((directory/'trace').glob('*.request.json')):shutil_request=json.loads(p.read_text());save(DATA/f'integrated-receipts/{case}-{p.name}',shutil_request)
            save(DATA/f'integrated-receipts/{case}-cli.json',r)
        after=resources();row.update(exit_code=proc.returncode,wall_seconds=round(time.monotonic()-started,3),resources_before=before,resources_after=after,helper_peak_rss_bytes=rss,git_unchanged=git_before==state(repo))
        (directory/'helper.stdout.json').write_bytes(out);(directory/'helper.stderr.txt').write_bytes(err)
        save(directory/'result.json',row)
        with results.open('a') as f:f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush();os.fsync(f.fileno())
        print('RESULT '+json.dumps(row,ensure_ascii=False),flush=True)
        assert row['git_unchanged'],'Git changed; receipt retained'
        if critical(before,after,rss or sampled_peak):raise ValueError('resource stop; receipt retained')
    check(pr,root,runtime);save(DATA/f'{a.phase}-completion.json',{'status':'completed','attempts':len(order),'generation_calls':len(order) if a.phase=='category' else sum(json.loads(x)['generation_calls'] for x in results.read_text().splitlines()),'preregistration_sha256':h((DATA/'preregistration.json').read_bytes())})
if __name__=='__main__':main()
