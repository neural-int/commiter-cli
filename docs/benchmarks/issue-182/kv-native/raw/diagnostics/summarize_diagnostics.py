import collections,hashlib,importlib.util,json,pathlib,xml.etree.ElementTree as E
root=pathlib.Path('$HOME/.codex/worktrees/issue-182-model-comparison/commiter-cli');p=pathlib.Path('/private/tmp/issue182-kv-native/diagnostics')
spec=importlib.util.spec_from_file_location('h',root/'tools/benchmark182/prompt_compare.py');h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)
contract=json.loads((p/'contract.json').read_text())
for n,d in contract['hashes'].items():assert h.sha(n)==d,n
completion=json.loads((p/'completion.json').read_text());rows=json.loads((p/'results.json').read_text());assert completion['cells']==len(rows)==2 and not completion['monitor_failed']
traces=json.loads((p/'trace-results.json').read_text());assert len(traces)==2 and all(r['exit']==0 and r['trace_exists'] for r in traces)
summary=[];audits=0
for row in rows:
 c=next(c for c in contract['cells'] if c['label']==row['cell']);folder=p/c['label'];totals=collections.defaultdict(float);metrics=[];phases=[]
 for i in [1,2,3]:
  name=f'{i:02d}';req=json.loads((folder/(name+'-request.json')).read_text());wire=json.loads((folder/(name+'-actual-request.json')).read_text());wire.pop('model_path',None);assert wire==h.transform(req,c['arm']);h.invariant(req,wire);audits+=1
  r=json.loads((folder/(name+'-response.json')).read_text());assert r['native']['ok'] and r['native']['stop_reason']=='completed'
  m=r['native']['benchmark_metrics'];assert m['benchmarkCacheMode']=='8' and m['diagnosticCPUSeconds'] and m['diagnosticEventCounts']
  for k,v in m['diagnosticCPUSeconds'].items():totals[k]+=v
  metrics.append(m);phases.append(r['response']['Content'])
 assert not row['result']['error'];group=json.loads(phases[0]);assert len(set(group.values()))==1 and set(group)==set(row['result']['fixed_membership'])
 model='gemma' if c['label'].startswith('gemma') else 'qwen';toc=E.parse(p/(model+'-toc.xml')).getroot();info={n.tag:(n.text or '').strip() for n in toc.findall('.//run/info/summary/*')}
 trace={'duration_seconds':float(info['duration']),'end_reason':info['end-reason'],'scope':'Grouping and Category' if model=='gemma' else 'Text only','gpu_interval_rows':0,'target_gpu_interval_rows':0}
 proc={};pid=next(t['pid'] for t in traces if t['cell']==c['label'])
 for event,node in E.iterparse(p/(model+'-gpu.xml'),events=['end']):
  if node.tag=='process' and 'id' in node.attrib:proc[node.attrib['id']]=node.attrib.get('fmt','')+' '+''.join(node.itertext())
  if node.tag=='row':
   trace['gpu_interval_rows']+=1
   for child in node.findall('process'):
    text=proc.get(child.attrib.get('ref',''),child.attrib.get('fmt','')+' '+''.join(child.itertext()))
    if str(pid) in text:trace['target_gpu_interval_rows']+=1
   node.clear()
 summary.append({'cell':c['label'],'wall_seconds':row['result']['wall_seconds'],'generated_tokens':sum(m['generatedTokens'] for m in metrics),'thinking_tokens':[m['nativeThoughtTokens'] for m in metrics],'forced_closes':[m['nativeThoughtForcedClose'] for m in metrics],'cpu_seconds':dict(totals),'first_read_seconds':[m['diagnosticFirstReadSeconds'] for m in metrics],'phase_contents':phases,'trace':trace})
monitor=[json.loads(line) for line in (p/'monitor.jsonl').read_text().splitlines()];assert all(m['valid'] for m in monitor)
h.save(p/'summary.json',{'id':'H182-008-diagnostic','completion':completion,'audited_requests':audits,'monitor_samples':len(monitor),'cells':summary,'limitations':['instrumented run, not performance baseline','read wait includes GPU/runtime wait, not pure synchronization overhead','Qwen trace Text only; no Qwen Thinking GPU timeline','GPU interval row count not occupancy or overlap ratio','no direct causal attribution of model/KV/synchronization bottleneck']})
h.save(p/'review.json',{'proceed_to_kv_comparison':True,'findings':['Both diagnostic cells accepted, actual grouping reference matches; six completed phase responses include CPU metrics','Thinking state update/progress total 0.0456s Gemma, 0.1790s Qwen; token-read waiting dominates these CPU callback measurements','Both target-process Metal traces contain GPU intervals; Gemma covers G/C, delayed Qwen attach covers Text only','Qwen Thinking GPU/CPU overlap and exact GPU occupancy remain unconfirmed; no synchronization-waste claim','KV8/native comparison remains a useful isolated next experiment; no automatic run authorization inferred'],'trace_capture_issue':'D1 recorder serialization took253.68s including saving, delaying D2 attach; D2 recorded19.87s until target exit','kv_comparison_started':False})
print('diagnostics verified',audits,'requests',len(monitor),'monitor samples; KV run not started')
