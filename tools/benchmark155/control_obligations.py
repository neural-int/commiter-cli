"""Source-only inventory of obligations for rejected natural table bindings."""
import json,subprocess
from attribution import ROOT
OUT=ROOT/'docs/benchmarks/issue-155'
rejected=json.loads((OUT/'iteration-10-loop-bindings.json').read_text())['rows'];records=[]
for iteration in (6,7):
    snapshots=json.loads((OUT/f'iteration-{iteration}-source-snapshots.json').read_text())
    for record in snapshots:
        audit=next((r for r in rejected if r['commit']==record['commit']),None)
        if audit is None:continue
        entries=[e for e in audit['entries'] if e['binding']is None]
        if not entries:continue
        test=next(f for f in record['files'] if f['path'].endswith('_test.go'))
        nodes=json.loads(subprocess.check_output(['/tmp/issue155-assertion-scope'],input=json.dumps({'files':record['files']}).encode()))
        for e in entries:
            span=e['row_span'];parents=[n for n in nodes if n['kind']=='function' and n['file']==test['id'] and n['version']=='after' and n['span'][0]<=span[0]<=span[1]<=n['span'][1]];assert len(parents)==1
            parent=parents[0];source=[n for n in nodes if n['kind']!='function' and n['file']==test['id'] and n['version']=='after' and n['function']==parent['function']]
            for n in source:
                a,b=n['span'];assert test['after'].encode()[a:b].decode()==n['text']
            records.append(dict(commit=record['commit'],row_span=span,function=parent['function'],source_syntax=source,binding_status='unverified',effect=None,obligation='Validate version-specific argument definitions and path to failure predicate; syntactic proximity is insufficient.'))
with (OUT/'iteration-11-control-obligations.json').open('x') as f:json.dump(dict(model_calls=0,records=records),f,ensure_ascii=False,indent=2);f.write('\n')
for r in records:print(r['function'],[(n['kind'],n['text']) for n in r['source_syntax'] if n['kind']in ('control_condition_syntax','branch_syntax')])
