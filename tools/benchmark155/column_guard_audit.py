"""Natural argument selection source proof, not implementation semantics."""
import json,subprocess
from attribution import ROOT
OUT=ROOT/'docs/benchmarks/issue-155';rows=[]
snapshots=json.loads((OUT/'iteration-7-source-snapshots.json').read_text());audits=json.loads((OUT/'iteration-7-natural-audit.json').read_text())['rows']
for record,audit in zip(snapshots,audits):
    if record['commit'].startswith('6fab'):continue
    bindings=json.loads(subprocess.check_output(['/tmp/issue155-column-guard-binding'],input=json.dumps({'files':record['files']}).encode()));entries=[]
    for observed in audit['observations']:
        versions={}
        for version,old in observed['versions'].items():
            if old['status']!='observed':versions[version]=dict(status='no_natural_row',binding=None);continue
            span=old['test_source']['span'];found=[b for b in bindings if b['Row']['version']==version and b['Row']['span']==span and b['Row']['file']=='F002'];assert len(found)==1
            b=found[0];test=next(f for f in record['files'] if f['id']=='F002')
            for key in ('Row','Input','Expected','Call','Condition','Guard','ArgumentDefinition','ArgumentAssignment'):
                ref=b[key];a,z=ref['span'];assert test[version].encode()[a:z].decode()==ref['text']
            expected=old['query']['arguments'][1]
            actual=int(b['SelectedArgument']) if b['ArgumentType']=='uint8' else b['SelectedArgument'];assert actual==expected
            versions[version]=dict(status='source_binding_verified',binding=b,actual_value=None,implementation_attribution=None)
        entries.append(dict(versions=versions))
    rows.append(dict(commit=record['commit'],parent=record['parent'],entries=entries))
result=dict(model_calls=0,rows=rows,scope='strict table column guard; string/first-byte argument selection only; no implementation call execution or intent classification')
target=OUT/'iteration-13-column-bindings.json'
if target.exists():assert json.loads(target.read_text())==result
else:target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
for r in rows:print(r['commit'],len(r['entries']),[(v['binding']['ArgumentType'],v['binding']['SelectedArgument']) for e in r['entries'] for v in e['versions'].values() if v['binding']])
