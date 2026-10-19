"""Report all fixed cells, timing scopes, and contract failures without a judge."""
import argparse,json,pathlib

def main():
 p=argparse.ArgumentParser();p.add_argument('--bundle',type=pathlib.Path,required=True);a=p.parse_args();rows=[]
 gold=json.loads((a.bundle/'gold-frozen.json').read_text())
 for label in json.loads((a.bundle/'execution-contract.json').read_text())['order']:
  root=a.bundle/'runs'/label
  if not (root/'result.json').exists():rows.append({'cell':label,'status':'not_completed'});continue
  result=json.loads((root/'result.json').read_text());phases={}
  for path in sorted(root.glob('*-request.json')):
   if path.name.endswith('-original-request.json'):continue
   req=json.loads(path.read_text());r=json.loads(path.with_name(path.name.replace('-request','-response')).read_text())
   try:content=json.loads(r['response'].get('Content',''))
   except ValueError:content=None
   phases[req['generation_profile']]={'wall_seconds':r['wall_seconds'],'content':content,'error':r['error'],'metrics':(r.get('native') or {}).get('benchmark_metrics')}
  ref=next(c for c in gold['cases'] if c['case_id']==label[:3]);membership=phases.get('bounded-grouping',{}).get('content');partition=None
  if isinstance(membership,dict):
   partition={}
   for path,g in membership.items():partition.setdefault(g,[]).append(path.removeprefix('path:'))
   partition=sorted(sorted(v) for v in partition.values())
  grouping=partition==sorted(sorted(g) for g in ref['grouping']['allowed_partitions'][0]) if partition else None
  cat=phases.get('bounded-category',{}).get('content');category=(len(cat)==1 and all(v['type']==ref['category']['type'] and v['breaking_evidence_ref']==ref['category']['breaking_evidence_ref'] for v in cat.values())) if cat else None
  text=phases.get('bounded-text',{}).get('content')
  rows.append({'cell':label,'wall_seconds':result['wall_seconds'],'requests':result['calls'],'validator_accepted':not bool(result['error']),'error':result['error'],'observed_grouping_matches_reference':grouping,'observed_partition':partition,'fixed_group_injected':True,'category_matches_fixed_group_reference':category,'text':text,'text_review':'pending_agent_review_not_human' if text else 'not_evaluated','phases':phases})
 (a.bundle/'analysis.json').write_text(json.dumps({'observations':rows,'scope':'fixed-group downstream comparison, not end-to-end semantic success'},ensure_ascii=False,indent=2)+'\n')
 print(json.dumps([{k:r[k] for k in ['cell','wall_seconds','validator_accepted','observed_grouping_matches_reference','category_matches_fixed_group_reference','error'] if k in r} for r in rows],ensure_ascii=False,indent=2))
if __name__=='__main__':main()
