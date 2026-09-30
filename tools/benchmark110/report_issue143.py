# Recompute the fixed 240-call report; standard library only.
import json,statistics,collections,sys
from pathlib import Path
root=Path(sys.argv[1])
rows=[json.loads(x) for x in (root/'docs/benchmarks/issue-143-model-comparison-2026-09-30.jsonl').read_text().splitlines()]
manifest=json.loads((root/'docs/benchmarks/issue-143-manifest-2026-09-30.json').read_text())
assert len(rows)==240
assert [r['sequence'] for r in rows]==list(range(1,241))
assert len({(r['model_index'],r['fixture'],r['run']) for r in rows})==240
contracts={c['fixture']:c for c in manifest['contracts']}
for r in rows:
 c=contracts[r['fixture']];order=int(r['reverse'])
 assert r['prompt_sha256']==c['prompt_hashes'][order] and r['schema_sha256']==c['schema_hashes'][order]
 assert r['context_tokens']==c['context_tokens']==8192
 assert r['helper_sha256']==manifest['helper_sha256']
 assert r['calls']==1 and r['output_budget']==2048 and r['candidate_count']==2
 assert r['model']==manifest['models'][r['model_index']]['repo']+'@'+manifest['models'][r['model_index']]['revision']
 assert r['reverse']==manifest['reverse'][r['run']-1]
 assert r['valid_candidate'] or ('false_merge' not in r and 'false_split' not in r)
 assert not any(k in r for k in ['prompt','response','raw_response','summary','diff','path'])
structural={'invalid_json','invalid_schema','unknown_candidate_id','duplicate_key','forbidden_none','grammar_failure'}
def aggregate(rs):
 valid=[r for r in rs if r['valid_candidate']]
 join=[r for r in rs if r['category']=='join']
 guard=[r for r in rs if r['guardrail']]
 return dict(n=len(rs),gold=sum(r['correct_selection'] for r in rs),valid=len(valid),complete=sum(r['complete_assignment'] for r in rs),
  false_merge=sum(r.get('false_merge',0) for r in valid) if valid else None,false_split=sum(r.get('false_split',0) for r in valid) if valid else None,
  join_gold=sum(r['correct_selection'] for r in join),join_n=len(join),guardrail_gold=sum(r['correct_selection'] for r in guard),guardrail_n=len(guard),
  guardrail_valid=sum(r['valid_candidate'] for r in guard),guardrail_false_merge=sum(r.get('false_merge',0) for r in guard if r['valid_candidate']) if any(r['valid_candidate'] for r in guard) else None,
  structural=sum(r.get('failure') in structural or (r['stop_reason']=='completed' and not r['valid_candidate']) for r in rs),stops=dict(collections.Counter(r['stop_reason'] for r in rs)),failures=dict(collections.Counter(r.get('failure','') for r in rs)),
  failure_kinds=dict(collections.Counter(r.get('failure_kind','') for r in rs)),calls=sum(r['calls'] for r in rs),wall_ms=sum(r['wall_ms'] for r in rs),
  median_wall_ms=statistics.median(r['wall_ms'] for r in rs),max_wall_ms=max(r['wall_ms'] for r in rs),prompt_bytes=sum(r['prompt_bytes'] for r in rs),output_bytes=sum(r['output_bytes'] for r in rs),
  output_tokens=dict(collections.Counter(str(r['output_tokens']) for r in rs)))
results=[]
for i,spec in enumerate(manifest['models']):
 rs=[r for r in rows if r['model_index']==i]
 assert len(rs)==48
 hold=aggregate([r for r in rs if r['set']=='holdout']);assert hold['n']==32
 known=aggregate([r for r in rs if r['set']=='known']);assert known['n']==16
 fixtures={name:aggregate([r for r in rs if r['fixture']==name]) for name in contracts}
 variation={}
 for name in contracts:
  fr=sorted([r for r in rs if r['fixture']==name],key=lambda r:r['run'])
  values=[(r['stop_reason'],r.get('selected_id')) for r in fr]
  variation[name]=dict(values=values,repeat_variation=(values[0]!=values[3] or values[1]!=values[2]),direction_disagreement=(values[0]==values[3] and values[1]==values[2] and values[0]!=values[1]))
 results.append(dict(model_index=i,spec=spec,known=known,holdout=hold,all=aggregate(rs),fixtures=fixtures,variation=variation))
base=results[0]['holdout']
for result in results:
 h=result['holdout']
 gates=dict(gold_at_least_28=h['gold']>=28,gold_gain_at_least_4=h['gold']>=base['gold']+4,guardrail_false_merge_zero=h['guardrail_valid']==8 and h['guardrail_false_merge']==0,
  join_no_worse=h['join_gold']>=base['join_gold'],false_split_no_worse=h['false_split'] is not None and base['false_split'] is not None and h['false_split']<=base['false_split'],completion_no_worse=h['valid']>=base['valid'],
  structural_zero=h['structural']==0,wall_median_at_most_2x=h['median_wall_ms']<=base['median_wall_ms']*2)
 result['gates']=gates;result['eligible']=result['model_index']!=0 and all(gates.values())
summary=dict(total=aggregate(rows),models=results)
(root/'docs/benchmarks/issue-143-summary-2026-09-30.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
print(json.dumps([dict(model=r['spec']['repo'],known=r['known'],holdout=r['holdout'],gates=r['gates'],eligible=r['eligible']) for r in results],ensure_ascii=False,indent=2))
