"""Portable private-fixture 004 runner. Never acquires history or mutates Git.
Manifest: {"cases":[{"name":"7","requests":".../requests","gold":"...json"}]}.
"""
import argparse,collections,copy,hashlib,json,os,pathlib,plistlib,signal,subprocess,threading,time
from multifile004 import adapt
from multifile import validate,groups_from_membership,payload_for,request_for,score_groups
PHASES=('grouping','category','text')
def save(path,value):path.write_text(json.dumps(value,ensure_ascii=False,indent=2));path.chmod(0o600)
def invoke(argv,**kwargs):return subprocess.run(argv,capture_output=True,text=True,check=True,**kwargs)
def host(validator,groups,categories,text):
 commits=[]
 for g in groups:
  k=g['group_id'];c=categories[k]
  if c['breaking_evidence_ref']=='unresolved':return {'accepted':False,'violations':['unresolved_breaking_evidence']}
  commits.append(dict(type=c['type'],scope=text[k]['scope'],summary=text[k]['summary'],breaking=c['breaking_evidence_ref']!='none',file_ids=[f['file_id'] for f in g['files']]))
 data=dict(plan=dict(schema_version=1,commits=commits),file_ids=[f for c in commits for f in c['file_ids']])
 return json.loads(invoke([str(validator)],input=json.dumps(data)).stdout)
def executable(path):
 if not path.is_file() or not os.access(path,os.X_OK):raise ValueError('required executable unavailable: '+path.name)
class Monitor:
 def __init__(self,out,probe):self.out=out;self.probe=probe;self.done=threading.Event();self.active=None;self.fatal=[];self.stage='preflight'
 def sample(self):
  row={'wall_time':time.time(),'stage':self.stage,'thermal':json.loads(invoke([str(self.probe)]).stdout)}
  for key,args in {'vm':['vm_stat'],'swap':['sysctl','vm.swapusage'],'pressure':['memory_pressure','-Q'],'power':['pmset','-g','batt'],'processes':['ps','-axo','pid=,ppid=,%cpu=,rss=,comm=']}.items():row[key]=invoke(args).stdout
  gpu=subprocess.run(['ioreg','-r','-c','AGXAccelerator','-a'],capture_output=True,check=True);row['gpu']=[x.get('PerformanceStatistics',{}) for x in plistlib.loads(gpu.stdout)];return row
 def loop(self):
  while not self.done.is_set():
   try:
    row=self.sample()
    with (self.out/'monitor.jsonl').open('a') as f:f.write(json.dumps(row)+'\n')
    if row['thermal']['thermal_state']=='critical':raise RuntimeError('thermal critical')
   except Exception as e:
    self.fatal.append(type(e).__name__)
    if self.active is not None:
     try:os.killpg(self.active.pid,signal.SIGTERM)
     except ProcessLookupError:pass
   self.done.wait(2)
 def cool(self):
  import re
  stable=None;start=time.monotonic();self.stage='cooling'
  while True:
   if self.fatal:raise RuntimeError('monitor failed')
   s=json.loads(invoke([str(self.probe)]).stdout);p=invoke(['memory_pressure','-Q']).stdout;m=re.search(r'free percentage:\s*(\d+)',p);assert m
   good=s['thermal_state']=='nominal' and not s['low_power_mode'] and os.getloadavg()[0]<=os.cpu_count()/2 and int(m[1])>=20
   now=time.monotonic();stable=(stable or now) if good else None
   if stable is not None and now-stable>=60:return
   if now-start>1800:raise RuntimeError('start conditions not recovered')
   time.sleep(5)
def run(args):
 os.umask(0o077)
 for p in [args.helper,args.validator,args.thermal_probe]:executable(p)
 manifest=json.loads(args.manifest.read_text());prepared=[];model_identity=None
 def manifest_path(value):
  path=pathlib.Path(value)
  return path if path.is_absolute() else args.manifest.resolve().parent/path
 if not manifest.get('cases'):raise ValueError('manifest contains no cases')
 # Preflight every source, adapter and real host validator before any model work.
 for case in manifest['cases']:
  directory=manifest_path(case['requests']);gold=json.loads(manifest_path(case['gold']).read_text());qs={p:json.loads((directory/(p+'.request.json')).read_text()) for p in PHASES};files=json.loads(qs['grouping']['messages'][1]['content'])['files']
  assert len({f['file_id'] for f in files})==len(files)
  for q in qs.values():
   assert q['context_tokens']==32768 and q['output_tokens']==4096
   identity=(q['model'],q['model_path'])
   if model_identity is None:model_identity=identity
   assert identity==model_identity,'model settings differ between requests'
   payload=json.loads(q['messages'][1]['content'])
   assert all(not h.get('user_requests') for h in payload['agent_history_contexts']),'expected final-only history'
  derived={p:adapt(q,p,shared_files=files) for p,q in qs.items()}
  for q in derived.values():q.update(input_layout='harness004',context_tokens=65536)
  assert derived['grouping']['messages'][:2]==derived['text']['messages'][:2]
  groups=json.loads(qs['text']['messages'][1]['content'])['groups'];cats={g['group_id']:{'type':g['type'],'breaking_evidence_ref':'none'} for g in groups};texts={g['group_id']:{'scope':'fixture','summary':'変更内容とテストを更新'} for g in groups};assert host(args.validator,groups,cats,texts)['accepted']
  groups_missing=copy.deepcopy(groups);groups_missing[-1]['files'].pop()
  # Use original selected IDs to verify the host rejects a missing assignment.
  good={'plan':{'schema_version':1,'commits':[dict(type=g['type'],scope='fixture',summary='変更内容とテストを更新',file_ids=[f['file_id'] for f in g['files']]) for g in groups_missing]},'file_ids':[f['file_id'] for f in files]}
  assert not json.loads(invoke([str(args.validator)],input=json.dumps(good)).stdout)['accepted']
  prepared.append((case,qs,derived,gold))
 args.output.mkdir(mode=0o700);save(args.output/'manifest.json',manifest)
 save(args.output/'provenance.json',{'helper_sha256':hashlib.sha256(args.helper.read_bytes()).hexdigest(),'prompt_sha256':hashlib.sha256((pathlib.Path(__file__).resolve().parents[2]/'internal/planning/prompts/harness004.json').read_bytes()).hexdigest(),'revision':'004-r3-issue187','prepared_only':not args.execute})
 for index,(case,qs,derived,gold) in enumerate(prepared):
  dest=args.output/f'{index+1:02d}';dest.mkdir();save(dest/'source.json',qs);save(dest/'gold.json',gold)
  for p,q in derived.items():save(dest/(p+'.request.json'),q)
 if not args.execute:return
 assert invoke([str(args.helper),'--smoke-metal']).stdout.strip()=='metal_ok'
 monitor=Monitor(args.output,args.thermal_probe);thread=threading.Thread(target=monitor.loop,daemon=True);thread.start();rows=[];case_results=[];state='stopped';error=None
 try:
  for index,(case,qs,derived,gold) in enumerate(prepared):
   monitor.cool();dest=args.output/f'{index+1:02d}';env={k:v for k,v in os.environ.items() if not k.startswith('COMMITER_')};result={'name':case['name'],'oracle':{},'propagated':{}}
   with (dest/'helper.stderr.log').open('w') as log:
    process=subprocess.Popen([str(args.helper),'--benchmark-multifile'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,text=True,env=env,start_new_session=True);monitor.active=process
    def call(track,phase,q):
     monitor.stage=f'{index+1}-{track}-{phase}';save(dest/(track+'-'+phase+'.request.json'),q);start=time.monotonic();process.stdin.write(json.dumps(q)+'\n');process.stdin.flush();line=process.stdout.readline()
     if not line:raise RuntimeError('helper terminated')
     r=json.loads(line);save(dest/(track+'-'+phase+'.response.json'),r);row={'name':case['name'],'track':track,'phase':phase,'wall_seconds':time.monotonic()-start,'response':r};rows.append(row);save(args.output/'results.json',rows)
     if not r['ok'] or r['stop_reason']!='completed':raise RuntimeError('incomplete phase')
     value=json.loads(r['generated_json']);validate(value,q['schema']);return value
    try:
     g=call('shared','grouping',derived['grouping']);result['grouping']=score_groups(g,gold);c=call('oracle','category',derived['category']);t=call('oracle','text',derived['text']);og=json.loads(qs['text']['messages'][1]['content'])['groups'];oc={x['group_id']:{'type':x['type'],'breaking_evidence_ref':'none'} for x in og};result['oracle']={'category':c,'text_validator':host(args.validator,og,oc,t)}
     groups=groups_from_membership(g,[f for x in og for f in x['files']]);ids=[x['group_id'] for x in groups];p=payload_for(json.loads(qs['category']['messages'][1]['content']),groups);q=adapt(request_for(qs['category'],p,ids),'category',shared_files=[f for x in og for f in x['files']]);q.update(input_layout='harness004',context_tokens=65536);c=call('propagated','category',q);p=payload_for(json.loads(qs['text']['messages'][1]['content']),groups,c);q=adapt(request_for(qs['text'],p,ids),'text',shared_files=[f for x in og for f in x['files']]);q.update(input_layout='harness004',context_tokens=65536);assert q['messages'][:2]==derived['grouping']['messages'][:2];t=call('propagated','text',q);result['propagated']={'category':c,'validator':host(args.validator,groups,c,t)}
    except (RuntimeError,ValueError,BrokenPipeError) as e:result['failure']=str(e)
    finally:process.stdin.close();process.wait();monitor.active=None
   save(dest/'evaluation.json',result);case_results.append(result)
   if monitor.fatal:raise RuntimeError('monitor failed')
  state='complete'
 except Exception as e:error=type(e).__name__+': '+str(e)
 finally:monitor.done.set();thread.join(25);save(args.output/'completion.json',{'state':state,'error':error,'phase_calls':len(rows),'maximum_phase_calls':5*len(prepared),'expected_cases':len(prepared),'recorded_cases':len(case_results),'failed_cases':sum('failure' in result for result in case_results),'monitor_errors':monitor.fatal})
if __name__=='__main__':
 p=argparse.ArgumentParser(description=__doc__)
 for name in ['manifest','helper','validator','thermal-probe','output']:p.add_argument('--'+name,type=pathlib.Path,required=True)
 p.add_argument('--execute',action='store_true',help='sample model outputs; omitted means preparation only');run(p.parse_args())
