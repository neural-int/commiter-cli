"""Fixed unit/symbol evidence ablation; gold excluded from all requests."""
import argparse
import importlib.util
import itertools
import json
import pathlib
import subprocess
import sys
import time

HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE))
spec=importlib.util.spec_from_file_location('api152', HERE/'evaluate.py')
api=importlib.util.module_from_spec(spec)
spec.loader.exec_module(api)
from inline import extract


def relations(units, graph):
    adjacency={}
    for e in graph['edges']:
        a,b=(e['from'],e['caller']),(e['to'],e['callee'])
        adjacency.setdefault(a,set()).add(b)
    nodes={}
    for u in units:
        for symbol in u['symbols']:
            nodes.setdefault((u['file'],symbol),[]).append(u['id'])
    pairs={}
    for node, ids in nodes.items():
        frontier={node}
        visited={node}
        for depth in (1,2):
            next_nodes={b for a in frontier for b in adjacency.get(a,())} - visited
            for target in sorted(next_nodes):
                for a,b in itertools.product(ids,nodes.get(target,())):
                    if a!=b:
                        pairs[a,b]=dict(from_unit=a,to_unit=b,hops=depth,kind='soft_directed_call_path')
                        if len(pairs)>256: raise ValueError('relation_budget')
            visited|=next_nodes
            frontier=next_nodes
    return [pairs[p] for p in sorted(pairs)]


def payload(record, graph_tool, symbols_tool, mode):
    start=time.perf_counter();units=[];gold={}
    labels={tuple(node):i for i,g in enumerate(record.get('SymbolGold',[])) for node in g['symbols']}
    file_labels={fid:i for i,g in enumerate(record.get('Gold',[])) for fid in g}
    for f in record['Files']:
        symbols=json.loads(subprocess.run([symbols_tool],input=f['After'].encode(),capture_output=True,check=True).stdout)
        raw=extract(f['Before'].encode(),f['After'].encode(),f['ID'],symbols)
        for u in raw:
            uid=f'U{len(units)+1:03}'
            # Refine annotations to actual atom location, never file-level scope.
            ns,ne=u['new_span']
            names=[s['name'] for s in symbols if s['start']<ne and ns<s['end']]
            units.append(dict(id=uid,file=f['ID'],old_span=u['old_span'],new_span=u['new_span'],symbols=names,before=u['before'],after=u['after']))
            if 'SymbolGold' in record:
                matches={labels[(f['ID'],s)] for s in names if (f['ID'],s) in labels}
                if len(matches)!=1: raise ValueError('invalid_evaluator_gold_mapping')
                gold[uid]=matches.pop()
            else:
                gold[uid]=file_labels[f['ID']]
    if not 1<=len(units)<=64: raise ValueError('unit_budget')
    data=dict(change_units=units,selected_file_context=[dict(id=f['ID'],path=f['Path'],before=f['Before'],after=f['After']) for f in record['Files']])
    evidence=None
    if mode=='repository':
        sources=[dict(ID=f['ID'],Path=f['Path'],Content=f['After']) for f in record['Files']]+record.get('Repository',[])
        graph=json.loads(subprocess.run([graph_tool],input=json.dumps(sources).encode(),capture_output=True,check=True).stdout)
        evidence=dict(unit_relations=relations(units,graph),syntax_graph=graph,absence='unknown_no_default_merge',
                      unchanged_sources=record.get('Repository',[]),max_hops=2)
        data['repository_evidence']=evidence
    if len(json.dumps(data).encode())>api.MAX_CONTEXT_BYTES: raise ValueError('context_byte_budget')
    return data,gold,time.perf_counter()-start,evidence


def main():
    p=argparse.ArgumentParser()
    for name in ('fixtures','regression','graph','symbols','helper','model-path','output'):
        p.add_argument('--'+name,required=True)
    args=p.parse_args()
    records=json.loads(pathlib.Path(args.fixtures).read_text())
    regression=json.loads(pathlib.Path(args.regression).read_text())
    names=['same-directory-independent-5','cross-directory-single-intent-9','multiple-intents-boundary-12','shared-callee-independent-6']
    records += [dict(r,Split='regression') for r in regression if r['Name'] in names]
    assert len(records)==9
    with pathlib.Path(args.output).open('w') as output:
        for r in records:
            for mode in ('A-only','repository'):
                try:
                    data,gold,wall,evidence=payload(r,args.graph,args.symbols,mode)
                except ValueError as error:
                    if str(error) not in ('unit_budget','relation_budget','context_byte_budget'):
                        raise
                    row=dict(fixture=r['Name'],split=r['Split'],source=mode,files=len(r['Files']),units=None,
                             exact=None,false_merge=None,false_split=None,complete=False,unresolved=True,
                             calls=0,stop=str(error),validation_reason='preflight_rejected',
                             input_tokens=0,output_tokens=0)
                    output.write(json.dumps(row,sort_keys=True)+'\n');output.flush()
                    print(json.dumps(row),flush=True)
                    continue
                membership,meta=api.invoke(data,args.helper,args.model_path)
                exact,fm,fs=api.quality(membership,gold) if membership else (None,None,None)
                row=dict(fixture=r['Name'],split=r['Split'],source=mode,files=len(r['Files']),units=len(gold),
                         exact=exact,false_merge=fm,false_split=fs,complete=membership is not None,
                         unresolved=membership is None,calls=1,membership=membership,evidence=evidence,
                         extraction_seconds=wall,context_bytes=len(json.dumps(data).encode()),**meta)
                output.write(json.dumps(row,sort_keys=True)+'\n');output.flush()
                print(json.dumps({k:row[k] for k in ('fixture','source','exact','false_merge','false_split','complete','stop','wall_seconds')}),flush=True)


if __name__=='__main__':
    main()
