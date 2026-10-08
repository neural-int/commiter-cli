"""Audit source intersections only; never infer causal attribution or intent."""
import json
from pathlib import Path
from attribution import ROOT,prepare,overlap
from local_measure import cases
from local_facts import prepare as facts_prepare


def audit(record):
    source,manifest=prepare(record['files'])
    payload,oracle=facts_prepare(record['files'],record['queries'])
    nodes={n['id']:n for n in payload['witnesses']}
    intersections=[]
    for qid,versions in oracle.items():
        for version,fact in versions.items():
            refs=[]
            for wid in fact['return_refs']:
                node=nodes[wid]
                raw=next(f[version] for f in record['files'] if f['id']==node['file']).encode()
                a,b=node['span']
                assert raw[a:b].decode()==node['text']
                units=[u['id'] for u in source['change_units'] if u['file']==node['file'] and overlap(u['old_span'] if version=='before' else u['new_span'],node['span'])]
                refs.append(dict(witness=wid,changed_unit_intersections=units))
            intersections.append(dict(query=qid,version=version,fact=fact,return_source_intersections=refs))
    deltas={q:('unknown' if not all(f['known'] for f in v.values()) else 'observed_value_changed' if (v['before']['kind'],v['before']['value'])!=(v['after']['kind'],v['after']['value']) else 'observed_value_unchanged') for q,v in oracle.items()}
    return dict(name=record['name'],split=record['split'],query_origins=record['query_origins'],units=len(manifest),queries=len(oracle),fact_deltas=deltas,source_mapping_verified=True,intersections=intersections,semantic_attribution=None,commit_partition=None,warning='Return overlap is syntax only. No overlap does not prove independence: conditions, arguments and callees can affect the return. Same observation does not prove shared intent.')

if __name__=='__main__':
    out=ROOT/'docs/benchmarks/issue-155/iteration-5-host-audit.json'
    rows=[audit(r) for r in cases()]
    with out.open('x') as f:json.dump(dict(model_calls=0,rows=rows,scope='Used diagnostics and constructed-input probes; not independent attribution holdout'),f,ensure_ascii=False,indent=2);f.write('\n')
    for r in rows:print(r['name'],r['units'],r['fact_deltas'])
