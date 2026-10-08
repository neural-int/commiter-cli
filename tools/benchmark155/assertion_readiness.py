"""Source evidence readiness, separate from behavior truth or causal attribution."""
import json,subprocess
from attribution import ROOT
OUT=ROOT/'docs/benchmarks/issue-155'
rows=[]
for iteration in (6,7):
    snapshots=json.loads((OUT/f'iteration-{iteration}-source-snapshots.json').read_text())
    audits=json.loads((OUT/f'iteration-{iteration}-natural-audit.json').read_text())['rows']
    for record,audit in zip(snapshots,audits):
        assert record['commit']==audit['commit']
        nodes=json.loads(subprocess.check_output(['/tmp/issue155-assertion-scope'],input=json.dumps({'files':record['files']}).encode()))
        test=next(f for f in record['files'] if f['path'].endswith('_test.go'))
        natural=audit['natural_queries'] if iteration==6 else [o['versions']['after']['test_source'] for o in audit['observations']]
        entries=[]
        for item in natural:
            span=item['source_span'] if iteration==6 else item['span']
            parents=[n for n in nodes if n['kind']=='function' and n['file']==test['id'] and n['version']=='after' and n['span'][0]<=span[0]<=span[1]<=n['span'][1]]
            assert len(parents)==1
            parent=parents[0];conditions=[n for n in nodes if n['kind']=='failure_condition_syntax' and n['file']==test['id'] and n['version']=='after' and n['function']==parent['function']]
            for n in [parent]+conditions:
                a,b=n['span'];assert test['after'].encode()[a:b].decode()==n['text']
            entries.append(dict(table_row_span=span,function=parent['function'],failure_conditions=conditions,status='source_evidence_only_binding_unverified' if conditions else 'missing_validator_body',behavior_value=None,causal_attribution=None))
        rows.append(dict(iteration=iteration,commit=record['commit'],entries=entries,semantic_gold=None))
with (OUT/'iteration-8-assertion-readiness.json').open('x') as f:json.dump(dict(model_calls=0,rows=rows,warning='Errorf/Fatal selector syntax is not a verified testing receiver or a complete table binding. No semantic success is claimed.'),f,ensure_ascii=False,indent=2);f.write('\n')
for r in rows:print(r['commit'],len(r['entries']),sum(bool(x['failure_conditions']) for x in r['entries']))
