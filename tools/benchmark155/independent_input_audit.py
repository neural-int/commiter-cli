"""Apply frozen source binders to fixed unmeasured history; no semantic gold."""
import hashlib,json,subprocess
from attribution import ROOT,prepare
OUT=ROOT/'docs/benchmarks/issue-155'
selection=json.loads((OUT/'iteration-14-selection.json').read_text())
for path,digest in selection['frozen_source_sha256'].items():assert hashlib.sha256((ROOT/path).read_bytes()).hexdigest()==digest
processors={'loop':'/tmp/issue155-loop-binding','table':'/tmp/issue155-table-binding','error':'/tmp/issue155-error-guard-binding','column':'/tmp/issue155-column-guard-binding'}
rows=[]
for record in json.loads((OUT/'iteration-14-source-snapshots.json').read_text()):
    extraction={};candidates=[]
    for name,binary in processors.items():
        proc=subprocess.run([binary],input=json.dumps({'files':record['files']}).encode(),capture_output=True,check=False)
        if proc.returncode:extraction[name]=dict(status='rejected',exit_code=proc.returncode);continue
        bindings=json.loads(proc.stdout);extraction[name]=dict(status='completed',bindings=len(bindings))
        def ref(binding,key):return binding[key] if key in binding else binding[key.lower()]
        def signature(b):return (ref(b,'Row')['file'],b.get('Function'),ref(b,'Input')['text'] if 'Input'in b else b['actual_expression']['text'],ref(b,'Expected')['text'] if 'Expected'in b else b['expected_literal']['text'])
        before={signature(b) for b in bindings if ref(b,'Row')['version']=='before'}
        for b in bindings:
            r=ref(b,'Row')
            if r['version']!='after' or signature(b)in before:continue
            for value in b.values():
                if type(value)is dict and set(('file','version','span','text'))<=set(value):
                    source=next(f for f in record['files'] if f['id']==value['file'])[value['version']].encode();a,z=value['span'];assert source[a:z].decode()==value['text']
            candidates.append(dict(processor=name,binding=b,semantic_gold=None,implementation_effect=None))
    try:p,m=prepare(record['files']);units=len(m);input_reject=None
    except ValueError as exc:units=None;input_reject=str(exc)
    rows.append(dict(commit=record['commit'],parent=record['parent'],extractors=extraction,new_or_changed_source_bindings=candidates,units=units,attribution_input_reject=input_reject,model_calls=0,holdout='not_certified',positive_negative_unknown_gate='unresolved: no independent semantic gold or cross-boundary coverage'))
result=dict(selection=selection,rows=rows,model_calls=0)
with (OUT/'iteration-14-input-audit.json').open('x') as f:json.dump(result,f,ensure_ascii=False,indent=2);f.write('\n')
for r in rows:print(r['commit'],len(r['new_or_changed_source_bindings']),r['extractors'],r['attribution_input_reject'])
