"""Summarize fixed-reference structural/category observations without model judging."""
import argparse,hashlib,json,pathlib,subprocess

def main():
 p=argparse.ArgumentParser();p.add_argument('--runs',type=pathlib.Path,required=True);p.add_argument('--gold',type=pathlib.Path,required=True);p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();gold=json.loads(a.gold.read_text());rows=[]
 for case in gold['cases']:
  cid=case['case_id']
  for arm in ['A','B']:
   run=a.runs/f'{cid}-{arm}';result=json.loads((run/'result.json').read_text());phases={}
   for path in sorted(run.glob('*-request.json')):
    req=json.loads(path.read_text());response=json.loads(path.with_name(path.name.replace('-request','-response')).read_text());native=response['response'];content=native.get('Content','');parsed=None
    try:parsed=json.loads(content)
    except ValueError:pass
    phases[req['generation_profile']]={'stop':native.get('StopReason'),'content':parsed,'backend_error':response['error'],'wall_seconds':response['wall_seconds']}
   membership=phases.get('bounded-grouping',{}).get('content');partition=None;group_correct=None
   if isinstance(membership,dict) and all(isinstance(v,str) for v in membership.values()):
    groups={}
    for path,label in membership.items():groups.setdefault(label,[]).append(path.removeprefix('path:'))
    partition=sorted(sorted(v) for v in groups.values());group_correct=any(partition==sorted(sorted(g) for g in allowed) for allowed in case['grouping']['allowed_partitions'])
   category=phases.get('bounded-category',{}).get('content');type_correct=None
   if group_correct and isinstance(category,dict):type_correct=len(category)==1 and all(v.get('type')==case['category']['type'] and v.get('breaking_evidence_ref')==case['category']['breaking_evidence_ref'] for v in category.values())
   text=phases.get('bounded-text',{}).get('content')
   rows.append({'case':cid,'arm':arm,'model_calls':len(phases),'accepted_plan':not bool(result['error']),'pipeline_error':result['error'],'grouping_exact':group_correct,'partition':partition,'category_matches_reference':type_correct,'category':category,'text':text,'text_semantic_review':'pending' if text else 'not_evaluated','phases':phases,'wall_seconds':result['wall_seconds']})
 a.out.write_text(json.dumps({'gold_sha256':hashlib.sha256(a.gold.read_bytes()).hexdigest(),'observations':rows},ensure_ascii=False,indent=2)+'\n')
 print(json.dumps([{k:r[k] for k in ['case','arm','model_calls','accepted_plan','grouping_exact','category_matches_reference','pipeline_error']} for r in rows],ensure_ascii=False,indent=2))
if __name__=='__main__':main()
