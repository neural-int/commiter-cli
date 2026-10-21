import collections,hashlib,importlib.util,json,pathlib,re
P=pathlib.Path('/private/tmp/issue182-kv-native');ROOT=pathlib.Path('<LOCAL_PATH>')
spec=importlib.util.spec_from_file_location('h',ROOT/'tools/benchmark182/prompt_compare.py');h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)
c=json.loads((P/'contract.json').read_text());done=json.loads((P/'completion.json').read_text());rows=json.loads((P/'results.json').read_text())
assert done['cells']==len(rows)==8 and not done['monitor_failed']
assert all(h.sha(p)==v for p,v in c['hashes'].items())
summary=[]
for cell,row in zip(c['cells'],rows):
 assert row['cell']==cell['label'] and row['exit']==0 and row['interrupted'] is None and not row['result']['error'] and row['result']['calls']==3
 phases=[];contents=[]
 for i in [1,2,3]:
  d=P/cell['label'];req=json.loads((d/f'{i:02}-request.json').read_text());wire=json.loads((d/f'{i:02}-actual-request.json').read_text());wire.pop('model_path',None);assert wire==h.transform(req,cell['arm']);h.invariant(req,wire)
  response=json.loads((d/f'{i:02}-response.json').read_text());n=response['native'];assert n['ok'] and n['stop_reason']=='completed'
  m=n['benchmark_metrics'];assert m['benchmarkCacheMode']==cell['kv_mode'] and not m.get('diagnosticCPUSeconds')
  layers=m['phaseEndKVLayers'];assert layers
  types=collections.Counter(l['nativeType'] for l in layers)
  assert types['QuantizedKVCache' if cell['kv_mode']=='8' else 'KVCacheSimple']==(4 if cell['model'].startswith('gemma') or cell['label'].startswith('gemma') else 8)
  phases.append({'phase':['Grouping','Category','Text'][i-1],'generated_tokens':m['generatedTokens'],'decode_seconds':m['runtimeGenerationSeconds'],'decode_tps':m['generatedTokens']/m['runtimeGenerationSeconds'],'thinking_tokens':m['nativeThoughtTokens'],'forced_close':m['nativeThoughtForcedClose'],'kv_bytes':sum(l['bytes'] for l in layers),'peak_mlx_bytes':m['peakMemoryBytes'],'cache_types':dict(types),'cache_dtypes':sorted(set(a['dtype'] for l in layers for a in l['arrays']))})
  contents.append(response['response']['Content'])
 group=json.loads(contents[0]);fixed=row['result']['fixed_membership'];assert set(group)==set(fixed)
 equivalent=all((group[a]==group[b])==(fixed[a]==fixed[b]) for a in fixed for b in fixed)
 summary.append({'cell':cell['label'],'fixture':cell['case'],'mode':cell['kv_mode'],'wall_seconds':row['result']['wall_seconds'],'generated_tokens':sum(m['generated_tokens'] for m in phases),'decode_tps':sum(m['generated_tokens'] for m in phases)/sum(m['decode_seconds'] for m in phases),'actual_grouping_match':equivalent,'phase_contents':contents,'plan':row['result']['result']['Plan'],'phases':phases})
mon=[json.loads(s) for s in (P/'monitor.jsonl').read_text().splitlines()];assert all(m['valid'] for m in mon)
pressure=[int(re.search(r'percentage: (\d+)',m['pressure']['stdout'])[1]) for m in mon];swaps=[float(re.search(r'used = ([\d.]+)M',m['swap']['stdout'])[1]) for m in mon]
pairs=[]
for i in range(0,8,2):
 a,b=summary[i:i+2];kv=next(x for x in [a,b] if x['mode']=='8');native=next(x for x in [a,b] if x['mode']=='native')
 for phase in [1,2,3]:assert json.loads((P/a['cell']/f'{phase:02}-actual-request.json').read_text())==json.loads((P/b['cell']/f'{phase:02}-actual-request.json').read_text())
 pairs.append({'cells':[a['cell'],b['cell']],'native_decode_change_percent':100*(native['decode_tps']/kv['decode_tps']-1),'native_wall_change_percent':100*(native['wall_seconds']/kv['wall_seconds']-1),'phase_contents_identical':a['phase_contents']==b['phase_contents']})
h.save(P/'summary.json',{'id':'H182-008-kv','completion':done,'verified_requests':24,'paired_requests_identical':12,'monitor':{'samples':len(mon),'invalid':0,'pressure_free_percent_range':[min(pressure),max(pressure)],'swap_used_MiB_start_end':[swaps[0],swaps[-1]],'swap_used_MiB_range':[min(swaps),max(swaps)],'max_helper_rss_GiB':max(r['rss_kib']/2**20 for m in mon for r in m['helper_processes']),'thermal_warning_reported':any('No thermal warning' not in m['thermal']['stdout'] for m in mon)},'cells':summary,'pairs':pairs,'decision':'NO-GO for blanket native adoption; n=1 mixed speed and one actual Grouping mismatch; keep existing KV8 default','limitations':['single observation per arm, no statistical significance','output and Thinking counts differ so wall time is not pure kernel speed','fixed downstream partition; plan acceptance does not establish actual Grouping correctness','system swap grows during suite; cannot attribute system swap to helper or isolate memory pressure effects','no GPU trace or detailed CPU timers in performance cells']})
print(json.dumps({'pairs':pairs,'monitor_samples':len(mon),'group_matches':[s['actual_grouping_match'] for s in summary]},indent=2))
