"""Source paths for natural integer-precision tests; no implementation execution."""
import json,subprocess
from attribution import ROOT
OUT=ROOT/'docs/benchmarks/issue-155'
record=next(r for r in json.loads((OUT/'iteration-6-source-snapshots.json').read_text()) if r['commit'].startswith('71f653b2'))
audit=next(r for r in json.loads((OUT/'iteration-6-natural-audit.json').read_text())['rows'] if r['commit']==record['commit'])
bindings=json.loads(subprocess.check_output(['/tmp/issue155-error-guard-binding'],input=json.dumps({'files':record['files']}).encode()))
selected=[];test=next(f for f in record['files'] if f['path']=='bytes_test.go')
for q in audit['natural_queries']:
    matches=[b for b in bindings if b['Row']['version']=='after' and b['Row']['span']==q['source_span'] and b['Row']['file']==test['id']];assert len(matches)==1
    b=matches[0]
    for key in ('Row','Input','Expected','Call','Condition','Guard','Continue'):
        ref=b[key];a,z=ref['span'];assert test[ref['version']].encode()[a:z].decode()==ref['text']
    selected.append(dict(binding=b,actual_value=None,implementation_attribution=None))
result=dict(commit=record['commit'],parent=record['parent'],model_calls=0,bindings=selected,scope='source path and literal field binding; no call execution, whole-repository typecheck or intent judgment')
target=OUT/'iteration-12-error-bindings.json'
if target.exists():assert json.loads(target.read_text())==result
else:target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print('source error guard bindings:',len(selected))
