"""Evaluator-only purpose recall/precision derived from frozen final partitions."""
import json,pathlib
from coarse import prepare
from planner import prepare as fine_prepare
from evaluate_c2 import canonical
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def clusters(mapping):return {frozenset(k for k,v in mapping.items() if v==value) for value in set(mapping.values())}
def measure(truth,groups):
 expected=clusters(truth);predicted={frozenset(g['unit_ids']) for g in groups};correct=len(expected & predicted)
 return dict(exact_purpose_true_positive=correct,gold_purposes=len(expected),predicted_purposes=len(predicted),exact_purpose_recall=correct/len(expected),exact_purpose_precision=correct/len(predicted) if predicted else None)
def main(iteration):
 spec=json.loads((OUT/f'iteration-{iteration}-preregistered.json').read_text());rows=[json.loads(s) for s in (OUT/f'iteration-{iteration}-results.jsonl').read_text().splitlines()];records=[]
 if iteration==6:cases={r['case']['name']:r['case'] for r in spec['rows']}
 else:cases={r['name']:r for r in spec['rows']}
 for r in rows:
  c=cases[r['name']]
  if not c['primary']:continue
  ref=fine_prepare(c['files']) if 'intent_states' in c else prepare(c['files'])
  if iteration==6:
   groups=r['final_groups'];by_id={u['id']:u for u in prepare(c['files'])+fine_prepare(c['files'])};mapped=canonical(list(by_id.values()),groups,ref)
  elif r['complete']:
   if 'groups' in r:groups=r['groups']
   else:groups=[dict(file_ids=g,unit_ids=[u['id'] for u in ref if u['file'] in g]) for g in r['observed']['groups']]
   mapped=canonical(prepare(c['files']),groups,ref)
  else:mapped=[]
  records.append(dict(name=r['name'],architecture=r['architecture'],reverse=r.get('reverse'),complete=r['complete'],**measure(c['gold'],mapped)))
 output=dict(definition='exact purpose cluster precision/recall; complete gold cluster must equal entire predicted cluster, not pair accuracy; stopped plans have recall0 and precision null',rows=records)
 with (OUT/f'iteration-{iteration}-secondary.json').open('x') as f:json.dump(output,f,ensure_ascii=False,indent=2);f.write('\n')
 print('saved',len(records),'primary secondary observations')
if __name__=='__main__':
 import sys
 main(int(sys.argv[1]))
