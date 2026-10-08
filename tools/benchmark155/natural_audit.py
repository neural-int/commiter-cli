"""Natural test input coverage audit; no inference, gold intent or source execution."""
import hashlib,json,re
from attribution import ROOT,prepare,extract,reconstruct
from local_facts import observe

OUT=ROOT/'docs/benchmarks/issue-155'
def sha(s):return hashlib.sha256(s.encode()).hexdigest()
def queries(test):
    result=[]
    # Only literal natural call/expected rows. Dynamic loop inputs are not guessed.
    for m in re.finditer(r'\{"[^"\n]*",\s*(Bytes|BytesN|IBytesN)\((\d+)(?:,\s*(\d+))?\),\s*"([^"\n]*)"\}',test):
        args=[int(m.group(2))]
        if m.group(3):args.append(int(m.group(3)))
        result.append(dict(query=dict(package='.',function=m.group(1),arguments=args),source_span=[len(test[:m.start()].encode()),len(test[:m.end()].encode())],text=m.group(),expected=m.group(4)))
    # Table literal strings passed to ParseBytes by a loop are source inputs,
    # not direct AST call observations; retain their distinct origin.
    for m in re.finditer(r'\{"(\d+(?: ?B)?)",\s*(\d+)\}',test):
        result.append(dict(query=dict(package='.',function='ParseBytes',arguments=[m.group(1)]),source_span=[len(test[:m.start()].encode()),len(test[:m.end()].encode())],text=m.group(),expected=int(m.group(2)),origin='table_input_requires_loop_mapping'))
    return result

if __name__=='__main__':
    snapshots=json.loads((OUT/'iteration-6-source-snapshots.json').read_text());rows=[]
    for record in snapshots:
        test=next(f for f in record['files'] if f['path']=='bytes_test.go')
        old={(x['query']['function'],tuple(x['query']['arguments']),str(x['expected'])) for x in queries(test['before'])}
        selected=[x for x in queries(test['after']) if (x['query']['function'],tuple(x['query']['arguments']),str(x['expected'])) not in old]
        host=[]
        for x in selected:
            a,b=x['source_span'];assert test['after'].encode()[a:b].decode()==x['text']
            facts=observe(record['files'],x['query'])
            host.append(dict(**x,facts=facts['facts'],observation_origin=x.get('origin','direct_call_table_row'),semantic_attribution=None))
        try:
            payload,manifest=prepare(record['files']);input_reject=None;anchors=len(payload['observations']);units=len(manifest)
        except ValueError as exc:
            input_reject=str(exc);anchors=None;units=None
        rows.append(dict(repository=record['repository'],commit=record['commit'],parent=record['parent'],source_hashes=[dict(path=f['path'],before=sha(f['before']),after=sha(f['after'])) for f in record['files']],units=units,supported_failure_predicates=anchors,input_reject=input_reject,natural_queries=host,model_calls=0,semantic_gold=None,partition=None,holdout_status='input_audit_only_not_certified',warning='Expected table values do not establish developer intent; unsupported evaluation is unknown, not success.'))
    with (OUT/'iteration-6-natural-audit.json').open('x') as f:json.dump(dict(selection='latest six bytes.go commits; first three also changing bytes_test.go, before inspecting source',rows=rows),f,ensure_ascii=False,indent=2);f.write('\n')
    for r in rows:print(r['commit'],'units',r['units'],'anchors',r['supported_failure_predicates'],'new natural inputs',len(r['natural_queries']),'known states',sum(s['known'] for q in r['natural_queries'] for s in q['facts'].values()))
