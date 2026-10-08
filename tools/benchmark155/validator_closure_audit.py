"""Reproduce strict source bindings for previously missing natural validators."""
import json,subprocess
from attribution import ROOT
OUT=ROOT/'docs/benchmarks/issue-155'
snapshots=json.loads((OUT/'iteration-9-closure-snapshots.json').read_text())
audits=json.loads((OUT/'iteration-6-natural-audit.json').read_text())['rows'];rows=[]
for record in snapshots:
    bindings=json.loads(subprocess.check_output(['/tmp/issue155-table-binding'],input=json.dumps({'files':record['files']}).encode()))
    old=next(r for r in audits if r['commit']==record['commit']);selected=[]
    for query in old['natural_queries']:
        matches=[b for b in bindings if b['row']['version']=='after' and b['row']['span']==query['source_span'] and b['row']['file']=='F002']
        assert len(matches)==1
        binding=matches[0]
        for ref in binding.values():
            source=next(f for f in record['files'] if f['id']==ref['file'])[ref['version']].encode();a,b=ref['span'];assert source[a:b].decode()==ref['text']
        selected.append(dict(source_binding=binding,behavior_value=None,implementation_attribution=None))
    rows.append(dict(commit=record['commit'],parent=record['parent'],new_table_rows=selected,model_calls=0,scope='strict direct composite literal and single-loop validator; source only, no value execution or purpose inference'))
result=dict(rows=rows);target=OUT/'iteration-9-source-bindings.json'
if target.exists():assert json.loads(target.read_text())==result
else:target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print('source bindings verified:',sum(len(r['new_table_rows']) for r in rows))
