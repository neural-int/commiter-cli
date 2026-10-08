"""Fixed synthetic ablation; raw prompts/responses never persisted."""
import argparse
import collections
import hashlib
import json
import pathlib
import subprocess
import sys
import tempfile
import time
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[1]/'benchmark151'))
from evaluate import extract
from fixtures import fresh

MODEL='mlx-community/Qwen3-8B-4bit'
REVISION='545dc4251c05440727734bcd94334791f6ab0192'
MAX_CONTEXT_BYTES=1024*1024


def strict(pairs):
    out={}
    for k,v in pairs:
        if k in out: raise ValueError('duplicate_json_key')
        out[k]=v
    return out


def history(record):
    with tempfile.TemporaryDirectory(prefix='benchmark152-history-') as tmp:
        root=pathlib.Path(tmp)
        def git(*args):
            return subprocess.run(['git',*args],cwd=root,capture_output=True,check=True).stdout
        git('init','-q')
        git('config','user.name','Synthetic benchmark')
        git('config','user.email','benchmark@example.invalid')
        originals={}
        for f in record['Files']:
            dest=root/f['Path'];dest.parent.mkdir(parents=True,exist_ok=True)
            dest.write_text(f['Before']);originals[f['Path']]=f['Before']
        git('add','.')
        git('commit','-qm','initial synthetic sources')
        schedule=record.get('History',[[i for i in range(len(record['Files'])) if i%2==0],[i for i in range(len(record['Files'])) if i%2==1]])
        for round_id, indices in enumerate(schedule):
            for i in indices:
                dest=root/record['Files'][i]['Path']
                dest.write_text(dest.read_text()+f'\n// housekeeping {round_id}\n')
            git('add','.')
            git('commit','-qm',f'synthetic housekeeping {round_id}')
        for name,content in originals.items(): (root/name).write_text(content)
        git('add','.')
        git('commit','-qm','restore source snapshot')
        start=time.perf_counter()
        hashes=git('log','-64','--format=%H').decode().splitlines()
        ids={f['Path']:f['ID'] for f in record['Files']}
        touches=collections.Counter();pairs=collections.Counter()
        for h in hashes:
            paths=git('diff-tree','--root','--no-commit-id','--name-only','-r','-z',h).decode().split('\0')
            selected=sorted({ids[p] for p in paths if p in ids})
            touches.update(selected)
            for i,a in enumerate(selected):
                for b in selected[i+1:]: pairs[a,b]+=1
        evidence=[dict(files=list(pair),cochange=n,touches=[touches[pair[0]],touches[pair[1]]]) for pair,n in sorted(pairs.items())]
        return dict(relations=evidence,commits=len(hashes),absence='unknown_never_default_merge'),time.perf_counter()-start


def payload(record,graph_executable,mode,symbol_executable=None):
    start=time.perf_counter()
    units=[];gold_for={}
    # Gold exists only in this evaluation map, excluded from model request.
    fg={fid:i for i,g in enumerate(record['Gold']) for fid in g}
    for f in record['Files']:
        symbols=json.loads(subprocess.run([symbol_executable],input=f['After'].encode(),capture_output=True,check=True).stdout) if symbol_executable else []
        extracted=extract(f['Before'].encode(),f['After'].encode(),f['ID'],symbols)
        for u in extracted:
            uid=f'U{len(units)+1:03}'
            units.append(dict(id=uid,source_unit_id=u['id'],file=f['ID'],old_span=u['old_span'],new_span=u['new_span'],symbols=u['symbols'],before=u['before'],after=u['after']))
            gold_for[uid]=fg[f['ID']]
    if not 1<=len(units)<=32: raise ValueError('unit_budget')
    selected=[dict(id=f['ID'],path=f['Path'],before=f['Before'],after=f['After']) for f in record['Files']]
    data=dict(change_units=units,selected_file_context=selected)
    extraction_wall=time.perf_counter()-start
    if mode=='repository':
        start=time.perf_counter()
        sources=[dict(ID=f['ID'],Path=f['Path'],Content=f['After']) for f in record['Files']]+record.get('Repository',[])
        graph=json.loads(subprocess.run([graph_executable],input=json.dumps(sources).encode(),capture_output=True,check=True).stdout)
        data['repository_evidence']=dict(unchanged_sources=record.get('Repository',[]),syntax_graph=graph)
        extraction_wall+=time.perf_counter()-start
    elif mode=='history':
        h,wall=history(record);data['history_evidence']=h;extraction_wall+=wall
    raw=json.dumps(data,sort_keys=True)
    if len(raw.encode())>MAX_CONTEXT_BYTES: raise ValueError('context_byte_budget')
    return data,gold_for,extraction_wall


def validate(answer,ids):
    if set(answer)!={'membership','unresolved'} or not isinstance(answer['unresolved'],bool): raise ValueError('invalid_schema')
    if answer['unresolved']: raise ValueError('unresolved')
    m=answer['membership']
    if not isinstance(m,dict) or set(m)!=set(ids): raise ValueError('invalid_assignment')
    allowed={f'G{i+1:03}' for i in range(len(ids))}
    if any(v not in allowed for v in m.values()): raise ValueError('invalid_group')
    return m


def quality(m,gold):
    fm=fs=0;ids=sorted(gold)
    for i,a in enumerate(ids):
        for b in ids[i+1:]:
            actual=m[a]==m[b];expected=gold[a]==gold[b]
            fm+=actual and not expected;fs+=expected and not actual
    return fm==fs==0,fm,fs


def invoke(data,helper,model_path):
    ids=[u['id'] for u in data['change_units']]
    groups=[f'G{i+1:03}' for i in range(len(ids))]
    schema=dict(type='object',properties=dict(membership=dict(type='object',properties={i:dict(type='string',enum=groups) for i in ids},required=ids,additionalProperties=False),unresolved=dict(type='boolean')),required=['membership','unresolved'],additionalProperties=False)
    system='Assign every selected change unit to one semantic commit intent. Group by changed purpose, not directory or syntax connectivity. A shared dependency or historical co-change is soft evidence and never proves shared intent. Unchanged repository files are context only and must not be assigned. Preserve independent changes. If the evidence cannot determine a complete coherent assignment, set unresolved true. Output JSON matching the supplied schema.'
    # The registered native0 helper does not enforce the request schema.
    # Its schema is therefore supplied to the model, as in benchmark149.invoke.
    system += ' Output JSON matching this schema: ' + json.dumps(schema,sort_keys=True)
    request=dict(schema=schema,messages=[dict(role='system',content=system),dict(role='user',content=json.dumps(data,sort_keys=True))],context_tokens=16384,output_tokens=1536,model=MODEL+'@'+REVISION,model_path=model_path,generation_profile='bounded-routed-grouping')
    start=time.perf_counter()
    try:
        p=subprocess.run([helper],input=(json.dumps(request)+'\n').encode(),capture_output=True,timeout=120)
    except subprocess.TimeoutExpired:
        return None,dict(stop='timeout',wall_seconds=time.perf_counter()-start,input_tokens=None,output_tokens=None)
    meta=dict(validation_reason='helper_failure',stop='helper_failure',wall_seconds=time.perf_counter()-start,input_tokens=None,output_tokens=None)
    if p.returncode: return None,meta
    try:
        response=json.loads(p.stdout,object_pairs_hook=strict)
        meta.update(stop=response.get('stop_reason','unknown'),input_tokens=response.get('benchmark_input_tokens'),output_tokens=response.get('benchmark_output_tokens'))
        if meta['stop']!='completed' or not response.get('ok'): return None,meta
        answer=json.loads(response.get('generated_json',''),object_pairs_hook=strict)
        membership=validate(answer,ids)
        meta['validation_reason']='accepted'
        return membership,meta
    except (ValueError,TypeError,KeyError) as err:
        code=str(err)
        allowed={'duplicate_json_key','invalid_schema','unresolved','invalid_assignment','invalid_group'}
        meta['validation_reason']=code if code in allowed else 'invalid_json'
        return None,meta


def main():
    parser=argparse.ArgumentParser();parser.add_argument('--fixtures',required=True);parser.add_argument('--graph',required=True);parser.add_argument('--symbols',required=True);parser.add_argument('--helper',required=True);parser.add_argument('--model-path',required=True);parser.add_argument('--output',required=True)
    args=parser.parse_args()
    all_records=json.loads(pathlib.Path(args.fixtures).read_text())
    names=['same-directory-independent-5','cross-directory-single-intent-9','multiple-intents-boundary-12','shared-callee-independent-6']
    records=[r for r in all_records if r['Name'] in names]+fresh()
    assert len(records)==6
    output=pathlib.Path(args.output)
    with output.open('w') as stream:
        for r in records:
            # Sequential ablations; never run build/tests during local inference.
            for mode in ('A-only','repository','history'):
                data,gold,extract_wall=payload(r,args.graph,mode,args.symbols)
                m,meta=invoke(data,args.helper,args.model_path)
                exact,fm,fs=quality(m,gold) if m else (None,None,None)
                row=dict(fixture=r['Name'],source=mode,files=len(r['Files']),units=len(gold),exact=exact,false_merge=fm,false_split=fs,complete=m is not None,unresolved=m is None,calls=1,membership=m,extraction_seconds=extract_wall,context_bytes=len(json.dumps(data).encode()),**meta)
                stream.write(json.dumps(row,sort_keys=True)+'\n');stream.flush()
                print(json.dumps({k:row[k] for k in ('fixture','source','exact','false_merge','false_split','complete','stop','wall_seconds')}),flush=True)

if __name__=='__main__': main()
