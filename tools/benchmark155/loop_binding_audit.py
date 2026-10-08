"""Audit remaining natural source rows, retaining unsupported bindings."""
import json,subprocess
from attribution import ROOT
OUT=ROOT/'docs/benchmarks/issue-155'
rows=[]
for iteration in (6,7):
    snapshots=json.loads((OUT/f'iteration-{iteration}-source-snapshots.json').read_text())
    audits=json.loads((OUT/f'iteration-{iteration}-natural-audit.json').read_text())['rows']
    for record,audit in zip(snapshots,audits):
        if iteration==6 and len(audit['natural_queries'])!=4:continue
        bindings=json.loads(subprocess.check_output(['/tmp/issue155-loop-binding'],input=json.dumps({'files':record['files']}).encode()))
        test=next(f for f in record['files'] if f['path'].endswith('_test.go'))
        natural=audit['natural_queries'] if iteration==6 else [o['versions']['after']['test_source'] for o in audit['observations']]
        entries=[]
        for item in natural:
            span=item['source_span'] if iteration==6 else item['span']
            found=[b for b in bindings if b['Row']['file']==test['id'] and b['Row']['version']=='after' and b['Row']['span']==span]
            assert len(found)<=1
            if found:
                for key in ('Row','Input','Expected','Call','Condition'):
                    ref=found[0][key];a,b=ref['span'];assert test[ref['version']].encode()[a:b].decode()==ref['text']
            entries.append(dict(row_span=span,status='source_binding_verified' if found else 'unsupported_loop_shape',binding=found[0] if found else None,implementation_attribution=None))
        rows.append(dict(commit=record['commit'],entries=entries))
result=dict(model_calls=0,rows=rows,scope='strict straight-line table loop; source mapping only, not actual values, runnable test validation or causal attribution')
target=OUT/'iteration-10-loop-bindings.json'
if target.exists():assert json.loads(target.read_text())==result
else:target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
for r in rows:print(r['commit'],len(r['entries']),sum(e['binding']is not None for e in r['entries']))
