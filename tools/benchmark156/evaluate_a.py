"""Previously used diagnostic gold lives here, never in planner inputs."""
import base64
import hashlib
import itertools
import json
import pathlib
import subprocess
import sys
import time
from planner import prepare, fallback, replay, encode
from remeasure import matching_subsets
from fixture_bytes import read as fixture_read, paths as fixture_paths

ROOT=pathlib.Path(__file__).resolve().parents[2]
OUT=ROOT/'docs/benchmarks/issue-156'

def metrics(units, groups, gold):
    assigned={u:i for i,c in enumerate(groups) for u in c['unit_ids']}
    if set(assigned)!=set(gold):raise ValueError('metric_incomplete')
    fm=fs=0
    for a,b in itertools.combinations(gold,2):
        fm+=assigned[a]==assigned[b] and gold[a]!=gold[b]
        fs+=assigned[a]!=assigned[b] and gold[a]==gold[b]
    return dict(exact=fm==fs==0, false_merge=fm, false_split=fs, commits=len(groups))

def diagnostic_cases():
    root=pathlib.Path(__file__).parent/'fixtures151'
    for folder in ('line','inline'):
        manifest=json.loads((root/folder/'manifest.json').read_text())
        for path,digest in manifest.items():
            if hashlib.sha256(fixture_read(root/folder/path)).hexdigest()!=digest:raise ValueError('fixture_manifest')
        for gp in sorted((root/folder).glob('*/*/gold.json')):
            label=json.loads(gp.read_text());base=gp.parent;files=[]
            for path in fixture_paths(base/'before'):
                files.append(dict(id=f'F{len(files)+1:03}',path=path,
                    before_b64=base64.b64encode(fixture_read(base/'before'/path)).decode(),after_b64=base64.b64encode(fixture_read(base/'after'/path)).decode()))
            units=prepare(files);gold={}
            for intent in label['requirements']:
                state=next(s for s in label['expected_states'] if s['selected_intents']==[intent])
                for f in files:
                    atoms=[a for a in units if a['file']==f['id']]
                    matches=matching_subsets(base64.b64decode(f['before_b64']),atoms,fixture_read(base/'expected'/state['directory']/f['path']))
                    if len(matches)!=1:raise ValueError('diagnostic_gold_unrepresentable')
                    for uid in matches[0]:
                        if uid in gold:raise ValueError('gold_duplicate')
                        gold[uid]=intent
            yield folder+'/'+label['split']+'/'+label['name'],files,units,gold

def generated():
    for n in (1,4,8,16):
        files=[]
        for i in range(n):
            files.append(dict(id=f'F{i+1:03}',path=f'src/f{i}.txt',before_b64=base64.b64encode(b'left=1 right=2\n').decode(),after_b64=base64.b64encode(b'left=3 right=4\n').decode()))
        yield 'structural-'+str(n),files
    yield 'unsupported-inline-nonutf8',[dict(id='F001',path='opaque.bin',before_b64=base64.b64encode(b'\xff x\n').decode(),after_b64=base64.b64encode(b'\xfe y\n').decode())]
    yield 'create-delete',[dict(id='F001',path='new.txt',before_b64='',after_b64=base64.b64encode(b'new\n').decode(),before_exists=False),dict(id='F002',path='old.txt',before_b64=base64.b64encode(b'old\n').decode(),after_b64='',after_exists=False)]

if __name__=='__main__':
    rows=[];requests=[]
    for name,files,units,gold in diagnostic_cases():
        start=time.perf_counter();groups=fallback(files,units)
        cu=[dict(file_ids=[u['file']],unit_ids=[u['id']]) for u in units]
        row=dict(name=name,split='used_diagnostic',files=len(files),units=len(units),input_sha256=hashlib.sha256(encode(files)).hexdigest(),calls=0,input_tokens=0,output_tokens=0,unknown_continuation=True,compose=0,fallback_rate=1.0,coverage=True,
            candidate=metrics(units,groups,gold),naive_file=metrics(units,groups,gold),naive_cu=metrics(units,cu,gold),semantic_stop_baseline_coverage=0,
            singleton_file_commit_rate=1.0,staging=replay(files,units,groups))
        row['wall_seconds']=time.perf_counter()-start;rows.append(row)
        plan=dict(schema_version=1,commits=[{k:c[k] for k in ('type','scope','breaking','summary','file_ids')} for c in groups])
        requests.append(dict(candidate=plan,file_ids=[f['id'] for f in files]))
    for name,files in generated():
        start=time.perf_counter();units=prepare(files);groups=fallback(files,units)
        rows.append(dict(name=name,split='structural_only',files=len(files),units=len(units),input_sha256=hashlib.sha256(encode(files)).hexdigest(),calls=0,input_tokens=0,output_tokens=0,unknown_continuation=True,compose=0,coverage=True,staging=replay(files,units,groups),wall_seconds=time.perf_counter()-start,exact=None,false_merge=None,false_split=None))
        plan=dict(schema_version=1,commits=[{k:c[k] for k in ('type','scope','breaking','summary','file_ids')} for c in groups])
        requests.append(dict(candidate=plan,file_ids=[f['id'] for f in files]))
    compose=dict(schema_version=1,commits=[dict(type='compose',scope='changes',breaking=False,summary='複数目的の変更を保存',file_ids=['F001'])])
    requests.append(dict(candidate=compose,file_ids=['F001']))
    result=json.loads(subprocess.run(['/private/tmp/benchmark156-validate'],input=encode(requests),capture_output=True,check=True).stdout)
    for row,validation in zip(rows,result):row['authoritative_validation']=validation
    for row in rows:
        if not row['authoritative_validation']['valid']:raise ValueError('fallback_plan_invalid')
    summary=dict(cases=len(rows),files=sum(r['files'] for r in rows),units=sum(r['units'] for r in rows),complete=sum(r['coverage'] for r in rows),reconstruction_staging=sum(r['staging']['final_tree_equality'] for r in rows),unknown_continuation=sum(r['unknown_continuation'] for r in rows),compose_validation=result[-1],model_calls=0,independent_semantic_holdout=False)
    sources={str(p.relative_to(ROOT)):hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted((ROOT/'tools/benchmark156').rglob('*')) if p.is_file() and '__pycache__' not in p.parts}
    with (OUT/'iteration-1-results.json').open('x') as f:json.dump(dict(summary=summary,rows=rows,source_sha256=sources),f,ensure_ascii=False,indent=2);f.write('\n')
    print(json.dumps(summary,ensure_ascii=False))
