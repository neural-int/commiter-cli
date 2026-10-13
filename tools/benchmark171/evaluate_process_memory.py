"""Frozen file-partition scoring; malformed outputs remain in exact denominator."""
import itertools,json,math,statistics,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2];DATA=ROOT/'docs/benchmarks/issue-171'
sys.path.insert(0,str(ROOT/'tools/benchmark169'));from run import validate_grouping

def partition(groups):return tuple(sorted(tuple(sorted(g)) for g in groups))
def predicted(obj):
 groups={}
 for k,v in obj.items():groups.setdefault(v,[]).append(k.removeprefix('path:'))
 return partition(groups.values())
def score(f,obj):
 if f['gold_status']=='undetermined':return {'exact':None,'fm':None,'fs':None,'unscorable':True,'reason':'gold_undetermined'}
 ids={x['path'] for x in f['files'] if x['before']!=x['after']}
 if obj is None or set(obj)!={'path:'+x for x in ids}:return {'exact':0,'fm':None,'fs':None,'unscorable':True,'reason':'invalid_output'}
 pred=predicted(obj);allowed=[partition(g) for g in f['allowed_partitions']]
 # Multiple-valid scores describe all fixed permissible partitions; never select
 # a favorable FM from one solution and FS from another.
 choices=[]
 for gold in allowed:
  membership={p:i for i,g in enumerate(gold) for p in g};fm=fs=0;neg=pos=0
  for a,b in itertools.combinations(sorted(ids),2):
   gs=membership[a]==membership[b];ps=obj['path:'+a]==obj['path:'+b];pos+=gs;neg+=not gs;fm+=not gs and ps;fs+=gs and not ps
  choices.append({'exact':int(pred==gold),'fm':fm,'fs':fs,'gold_same_pairs':pos,'gold_different_pairs':neg})
 r={**choices[0],'unscorable':False,'predicted_partition':[list(g) for g in pred],'all_merge':len(pred)==1,'all_singleton':len(pred)==len(ids),'fallback_correct':sum(any(g==(p,) for g in pred) for p in f['fallback_files']),'gold_fallback_files':len(f['fallback_files']),'gold_fallback_fragmented_purposes':len(f['fallback_purpose_fragmentation'])}
 if f['gold_status']=='multiple_valid':r.update(exact=int(any(c['exact'] for c in choices)),per_allowed_partition=choices)
 return r

def quantile(xs,p):
 if not xs:return None
 xs=sorted(xs);return xs[max(0,math.ceil(p*len(xs))-1)]
def main():
 fx={f['name']:f for f in json.loads((ROOT/'tools/benchmark171/fixtures.json').read_text())};manifest=json.loads((DATA/'frozen-inputs/manifest.json').read_text());records={}
 path=DATA/'measurements.jsonl'
 if path.exists():
  for line in path.read_text().splitlines():
   r=json.loads(line);key=(r['fixture'],r['arm'],r['order']);assert key not in records;records[key]=r
 scored=[]
 for row in manifest:
  r=records.get((row['fixture'],row['arm'],row['order']),{**row,'status':'not_attempted','schema_valid':False,'response':None})
  f=fx[row['fixture']];s=score(f,r.get('response') if r.get('schema_valid') else None);scored.append({**r,'gold_status':f['gold_status'],'score':s})
 arms={}
 for arm in 'ABCD':
  allrows=[r for r in scored if r['arm']==arm];rs=[r for r in allrows if r['gold_status']=='confirmed'];valid=[r for r in rs if not r['score']['unscorable']]
  stable=[]
  for f in fx.values():
   if f['gold_status']!='confirmed':continue
   a=next(r for r in rs if r['fixture']==f['name'] and r['order']=='normal');b=next(r for r in rs if r['fixture']==f['name'] and r['order']=='reverse')
   stable.append(bool(a.get('schema_valid') and b.get('schema_valid') and predicted(a['response'])==predicted(b['response'])))
  walls=[r['wall_seconds'] for r in allrows if 'wall_seconds' in r];rss=[r['helper_peak_rss_bytes'] for r in allrows if r.get('helper_peak_rss_bytes') is not None];tok=[r['input_tokens'] for r in allrows if r.get('input_tokens') is not None];out=[r['output_tokens'] for r in allrows if r.get('output_tokens') is not None]
  arms[arm]={'exact':sum(r['score']['exact'] for r in rs),'denominator':len(rs),'normal_exact':sum(r['score']['exact'] for r in rs if r['order']=='normal'),'reverse_exact':sum(r['score']['exact'] for r in rs if r['order']=='reverse'),'fm_pairs':sum(r['score']['fm'] for r in valid),'fs_pairs':sum(r['score']['fs'] for r in valid),'fm_fixtures':sum(r['score']['fm']>0 for r in valid),'fs_fixtures':sum(r['score']['fs']>0 for r in valid),'fm_fixture_rate':sum(r['score']['fm']>0 for r in valid)/len(rs),'fs_fixture_rate':sum(r['score']['fs']>0 for r in valid)/len(rs),'fm_pair_rate_scorable':sum(r['score']['fm'] for r in valid)/sum(r['score']['gold_different_pairs'] for r in valid) if sum(r['score']['gold_different_pairs'] for r in valid) else None,'fs_pair_rate_scorable':sum(r['score']['fs'] for r in valid)/sum(r['score']['gold_same_pairs'] for r in valid) if sum(r['score']['gold_same_pairs'] for r in valid) else None,'unscorable':len(rs)-len(valid),'schema_valid':sum(bool(r.get('schema_valid')) for r in allrows),'total_rows':len(allrows),'order_stable':sum(stable),'order_denominator':len(stable),'all_merge':sum(r['score']['all_merge'] for r in valid),'all_singleton':sum(r['score']['all_singleton'] for r in valid),'fallback_correct':sum(r['score']['fallback_correct'] for r in valid),'gold_fallback_files':sum(len(fx[r['fixture']]['fallback_files']) for r in rs),'input_tokens':tok,'output_tokens':out,'wall_p50':quantile(walls,.5),'wall_p95':quantile(walls,.95),'max_rss_bytes':max(rss,default=None)}
 comparisons={}
 for a,b,label in [('A','B','input_at_existing'),('A','C','prompt_at_existing'),('B','D','prompt_at_raw'),('C','D','input_at_new_prompt')]:
  comparisons[label]={k:arms[b][k]-arms[a][k] for k in ['exact','fm_pairs','fs_pairs','order_stable']}
 interaction={k:arms['D'][k]-arms['B'][k]-arms['C'][k]+arms['A'][k] for k in ['exact','fm_pairs','fs_pairs','order_stable']}
 pr=json.loads((DATA/'preregistration.json').read_text());g=pr['quality_gate'];a=arms['A'];d=arms['D'];resource_ok=all(r.get('git_unchanged') and r.get('helper_peak_rss_bytes') is not None and r['helper_peak_rss_bytes']<=pr['max_rss_bytes'] and r.get('resource_after',{}).get('pressure_level') in [1,2] and isinstance(r.get('peak_bytes'),int) and r['peak_bytes']<=pr['max_mlx_peak_bytes'] for r in scored)
 checks={'complete_execution':len(records)==len(manifest),'all_arms_structural_valid':all(v['schema_valid']==v['total_rows'] for v in arms.values()),'D_exact_minimum':d['exact']/d['denominator']>=g['D_exact_min_rate'],'D_exact_improvement':d['exact']-a['exact']>=g['D_exact_min_improvement_rows'],'D_FM_no_regression':d['fm_pairs']<=a['fm_pairs'],'D_FS_no_regression':d['fs_pairs']<=a['fs_pairs'],'D_order_minimum':d['order_stable']/d['order_denominator']>=g['D_order_min_rate'],'D_nontrivial_mixed_cases':all(r['score'].get('exact')==1 for r in scored if r['arm']=='D' and r['fixture'] in g['required_mixed_cases']),'resources_and_Git':resource_ok}
 result={'gate':'GO' if all(checks.values()) else 'NO-GO','checks':checks,'arms':arms,'simple_effects':comparisons,'interaction_difference_in_differences':interaction,'confirmed_fixtures':sum(f['gold_status']=='confirmed' for f in fx.values()),'undetermined_fixtures':[f['name'] for f in fx.values() if f['gold_status']=='undetermined'],'multiple_valid_fixtures':[f['name'] for f in fx.values() if f['gold_status']=='multiple_valid'],'excluded_fraction':sum(f['gold_status']=='undetermined' for f in fx.values())/len(fx),'records':len(records),'production_go':False,'system_swap_policy':'context_only_excluded_from_stop_and_acceptance_by_user_instruction','scored_rows':scored}
 (DATA/'evaluation.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps({k:v for k,v in result.items() if k!='scored_rows'},ensure_ascii=False))
if __name__=='__main__':main()
