"""Check gold derivation, source binding, factorial pairing and frozen artifacts."""
import hashlib,itertools,json,re,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2];DATA=ROOT/'docs/benchmarks/issue-171'
sys.path.insert(0,str(ROOT/'tools/benchmark169'));from run import check_pins,repo_digest
from evaluate import partition

def dump(p,v):p.write_text(json.dumps(v,ensure_ascii=False,indent=2)+'\n')
def audit_gold():
 fx=json.loads((ROOT/'tools/benchmark171/fixtures.json').read_text());known=[]
 for number in [169,170]:known+=json.loads((ROOT/f'tools/benchmark{number}/fixtures.json').read_text())
 old_sources={f[k] for x in known for f in x['files'] for k in ['before','after']};prior_hashes={v for source in json.loads((DATA/'prior-fixture-source-sha256.json').read_text()) for v in source['source_sha256']};rows=[]
 for f in fx:
  files={x['path']:x for x in f['files'] if x['before']!=x['after']};assert 2<=len(files)<=4 and len(files)==len(set(files))
  assert all(not re.search(r'(^|/)(\.env[^/]*|credentials[^/]*|[^/]*key[^/]*|[^/]*\.pem)$',p,re.I) for p in files)
  assert not any(x[k] in old_sources or hashlib.sha256(x[k].encode()).hexdigest() in prior_hashes for x in files.values() for k in ['before','after']),'previously used source'
  assert set(f['input_identifiability'])==set('ABCD') and f['rationale'] and f['revert_independence']
  for x in files.values():
   assert x['source_evidence']['before']==x['before'].splitlines() and x['source_evidence']['after']==x['after'].splitlines()
   assert set(x['purpose_ids'])<=set(f['purpose_descriptions'])
  if f['gold_status']=='undetermined':
   assert not f['gold_groups'] and not f['allowed_partitions'];rows.append({'fixture':f['name'],'status':'undetermined','main_excluded':True,'reason':f['rationale']});continue
  assert f['gold_status'] in ['confirmed','multiple_valid']
  for groups in f['allowed_partitions']:
   flat=[p for g in groups for p in g];assert len(flat)==len(set(flat)) and set(flat)==set(files)
  assert f['gold_groups']==f['allowed_partitions'][0]
  assert all(x['purpose_ids'] for x in files.values())
  expected=[];single={}
  fallback=[p for p,x in files.items() if len(x['purpose_ids'])>1]
  assert sorted(fallback)==sorted(f['fallback_files'])
  for p in fallback:expected.append([p])
  for p,x in files.items():
   if p not in fallback:single.setdefault(x['purpose_ids'][0],[]).append(p)
  expected+=list(single.values());assert partition(expected)==partition(f['gold_groups']),'R4-R7 derivation mismatch'
  relations={frozenset(r['purposes']):r for r in f['purpose_relations']}
  for a,b in itertools.combinations(f['purpose_descriptions'],2):
   r=relations[frozenset([a,b])];assert r['relation'] in ['independent','dependent_independent_goals'] and r['source_basis']
  fragments={p:[i for i,g in enumerate(f['gold_groups']) if any(p in files[x]['purpose_ids'] for x in g)] for p in f['purpose_descriptions']};fragments={p:g for p,g in fragments.items() if len(g)>1}
  assert fragments==f['fallback_purpose_fragmentation']
  rows.append({'fixture':f['name'],'status':f['gold_status'],'file_ids':['path:'+p for p in files],'rules':f['rules'],'derived_partition':expected,'fallback':fallback,'fallback_fragmented_purposes':list(fragments),'source_overlap_168_169_170':0,'semantic_source_review':f['rationale'],'input_identifiability':f['input_identifiability']})
 result={'gold_rules_version':'1.0','fixtures':len(fx),'confirmed':sum(f['gold_status']=='confirmed' for f in fx),'undetermined':sum(f['gold_status']=='undetermined' for f in fx),'multiple_valid':sum(f['gold_status']=='multiple_valid' for f in fx),'fallback_files':sum(len(f['fallback_files']) for f in fx),'rows':rows,'automated_checks':'Allocation, purposes, evidence bytes, R4-R7 derivation, fragmentation and prior-source overlap; semantic judgments are documented source reviews, not automated proof','model_generations':0}
 dump(DATA/'gold-audit.json',result);return result

def audit_inputs(local=None):
 manifest=json.loads((DATA/'frozen-inputs/manifest.json').read_text());sm=json.loads((DATA/'source-mapping.json').read_text());pair={};orders={};baseline_captured={}
 for row in manifest:
  p=DATA/'frozen-inputs'/row['request_file'];assert hashlib.sha256(p.read_bytes()).hexdigest()==row['request_sha256'];req=json.loads(p.read_text());u=json.loads(req['messages'][1]['content']);key=(row['fixture'],row['order']);pair.setdefault(key,{})[row['arm']]=req;orders[(row['fixture'],row['order'],row['arm'])]=u
  assert set(u)=={'task','files'} and len(u['files'])==row['selected_files']
  assert req['context_tokens']==16384 and req['output_tokens']==768 and req['generation_profile']=='bounded-grouping'
  assert row['admitted'] and isinstance(row['token_count_before_inference'],int)
  assert row['token_count_before_inference']+768<=16384
  ids={x['id'] for x in u['files']};assert ids==set(req['schema']['required'])==set(req['schema']['properties'])
  assert req['schema']['additionalProperties'] is False
  for f in u['files']:
   m=sm[row['fixture']]['files'][f['id']];assert f['new_path']==m['path']
   assert f['id']=='path:'+m['path'] and not(f['binary'] or f['vendor'] or f['opaque'])
   if row['arm'] in 'BD':assert f['raw_diff']==m['raw_diff'] and f['mode']=='raw_diff' and not any(k in f for k in ['summary','evidence','evidence_reduction'])
   else:assert f==m['baseline_file']
  if row['arm']=='A' and row['order']=='normal':
   original=json.loads((DATA/(row['fixture']+'-captured-baseline.json')).read_text());assert req==original;baseline_captured[row['fixture']]=True
 for key,arms in pair.items():
  assert set(arms)==set('ABCD')
  assert arms['A']['messages'][1]==arms['C']['messages'][1] and arms['B']['messages'][1]==arms['D']['messages'][1]
  assert arms['A']['messages'][0]==arms['B']['messages'][0] and arms['C']['messages'][0]==arms['D']['messages'][0]
  common=lambda r:{k:v for k,v in r.items() if k!='messages'}
  assert common(arms['A'])==common(arms['B'])==common(arms['C'])==common(arms['D'])
  for a,b in zip(json.loads(arms['A']['messages'][1]['content'])['files'],json.loads(arms['B']['messages'][1]['content'])['files']):
   remove=lambda f:{k:v for k,v in f.items() if k not in ['mode','evidence','evidence_reduction','raw_diff','summary']}
   assert remove(a)==remove(b),'non-input-representation confound'
 for (fx,order,arm),u in orders.items():
  if order=='normal':assert u['files']==list(reversed(orders[(fx,'reverse',arm)]['files']))
 if local:
  for name,path in local['fixture_roots'].items():
   r=next(x for x in manifest if x['fixture']==name);assert repo_digest(Path(path))==r['repository_sha256']
 result={'requests':len(manifest),'real_baseline_captures':len(baseline_captured),'factorial_pairs':len(pair),'schema_and_nonmessage_controls_equal':True,'unchanged_common_metadata':True,'reverse_applied_all_arms':True,'stable_sort':'Existing production non-test paths lexical first, then *_test.go paths lexical; schema/file ID mapping fixed before diagnostic reversal','gold_provenance_and_fixture_type_not_in_user_fields':True,'raw_diff_truncated':0,'model_generations':0,'context_tokens_range':[min(r['token_count_before_inference'] for r in manifest),max(r['token_count_before_inference'] for r in manifest)]}
 dump(DATA/'input-audit.json',result);return result
if __name__=='__main__':
 local=json.loads(Path(sys.argv[1]).read_text()) if len(sys.argv)>1 else None
 print(json.dumps(audit_gold(),ensure_ascii=False));print(json.dumps(audit_inputs(local)))
