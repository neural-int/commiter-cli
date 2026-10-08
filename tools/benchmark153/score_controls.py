"""Diagnostic controls only; never counted as semantic quality fixtures."""
import importlib.util,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'benchmark152'))
import inline_evaluate
sys.path.insert(0,str(HERE))
spec=importlib.util.spec_from_file_location('c_score_evaluate',HERE/'evaluate.py')
c=importlib.util.module_from_spec(spec);spec.loader.exec_module(c)
output=HERE.parents[1]/'docs/benchmarks/issue-153/iteration-3-results.jsonl'
with output.open('w') as out:
 for control in json.loads((HERE/'score-controls.json').read_text()):
  membership,meta=c.score(control['data'])
  scores=meta.get('scores',{})
  value=scores.get('U001__U002')
  expected=control['expected']
  passed=(type(value) is int and value>0) if expected=='positive' else (type(value) is int and value<0) if expected=='negative' else (meta['validation_reason']=='unresolved' or (value==0 and meta['validation_reason']=='ambiguous_optimum'))
  row=dict(control=control['name'],expected=expected,diagnostic_pass=passed,membership=membership,**meta)
  out.write(json.dumps(row)+'\n');out.flush();print(json.dumps(row),flush=True)
