"""No-inference source-evidence audit. Evaluation labels never enter the observer."""
import hashlib
import json
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'tools/benchmark151'))
from inline import extract
from evaluate import reconstruct

OUT = ROOT / 'docs/benchmarks/issue-155'
OBSERVER = pathlib.Path('/tmp/issue155-anchor-observer')

def digest(raw):
    return hashlib.sha256(raw).hexdigest()

def encoded(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True).encode()

def records():
    used = json.loads((ROOT / 'tools/benchmark153/policy-fixtures.json').read_text())
    cases = [dict(name=r['name'], split='used_diagnostic', files=[
        {k:f[k] for k in ('id', 'path', 'before', 'after')} for f in r['files']]) for r in used]
    before = 'package scope\nimport "testing"\nfunc TestPair(t *testing.T) { if Left(2)!=3 || Right(2)!=4 {t.Fatal("日本語")} }\n'
    after = before.replace('Left(2)!=3', 'Left(2)!=4').replace('Right(2)!=4', 'Right(2)!=6')
    cases.append(dict(name='same-line-two-conditions', split='development_scope_probe', files=[dict(id='F001',path='pair_test.go',before=before,after=after)]))
    cases.append(dict(name='no-test-observation', split='development_scope_probe', files=[dict(id='F001',path='value.go',before='package scope\nfunc Value(n int) int { return n }\n',after='package scope\nfunc Value(n int) int { return n+1 }\n')]))
    return cases

def run_observer(files):
    return json.loads(subprocess.run([str(OBSERVER)],input=encoded({'files': files}),capture_output=True,check=True,timeout=10).stdout)

def main():
    cases=records()
    prereg=dict(kind='no_inference_input_audit', model_calls=0,
        fixture_sha256=digest(encoded(cases)),observer_sha256=digest(OBSERVER.read_bytes()),
        source_sha256=digest((ROOT/'tools/benchmark155/anchors.go').read_bytes()),
        checks=['exact UTF-8 source spans and source digests', 'repeat and reversed-file deterministic output',
                'all inline atoms reconstruct before/after without omission',
                'changed failure predicate counts are observations only; no relation quality or GO'],
        observer_scope='Go Test functions with explicit single Fatal/Fatalf/Error/Errorf body; nested callbacks unsupported; || leaves retain common condition span',
        model_gate='not registered; requires attribution schema, decoder audit and independent holdout before inference')
    with (OUT/'iteration-1-audit-contract.json').open('x') as stream:
        json.dump(prereg,stream,ensure_ascii=False,indent=2);stream.write('\n')
    rows=[]
    for case in cases:
        files=case['files'];facts=run_observer(files)
        assert facts==run_observer(files)==run_observer(list(reversed(files)))
        snapshots={f['id']: f for f in files}
        for e in facts['evidence']:
            source=snapshots[e['file']][e['version']].encode()
            start,end=e['span'];cs,ce=e['condition_span']
            assert 0<=cs<=start<=end<=ce<=len(source)
            assert source[start:end].decode()==e['text'] and digest(source)==e['source_sha256']
        unit_count=0
        for f in files:
            old,new=f['before'].encode(),f['after'].encode()
            atoms=extract(old,new,f['id']);ids=[a['id'] for a in atoms]
            assert reconstruct(old,atoms,ids)==new
            assert reconstruct(old,atoms,[])==old
            assert reconstruct(old,atoms,list(reversed(ids)))==new
            unit_count+=len(atoms)
        predicates=[e for e in facts['evidence'] if e['kind']=='failure_predicate']
        changed=[e for e in predicates if e['changed']]
        row=dict(name=case['name'],split=case['split'],input_sha256=digest(encoded({'files':files})),
                 file_count=len(files),inline_atoms=unit_count,predicate_observations=len(predicates),
                 changed_predicates_before=sum(e['version']=='before' for e in changed),
                 changed_predicates_after=sum(e['version']=='after' for e in changed),
                 source_mapping_valid=True,reconstruction_valid=True,deterministic=True,
                 evidence=facts['evidence'],source_status=facts['status'],semantic_quality=None)
        rows.append(row)
    result=dict(contract_sha256=digest((OUT/'iteration-1-audit-contract.json').read_bytes()),model_calls=0,rows=rows)
    with (OUT/'iteration-1-input-audit.json').open('x') as stream:
        json.dump(result,stream,ensure_ascii=False,indent=2);stream.write('\n')
    print(json.dumps([{k:v for k,v in r.items() if k not in ('evidence','source_status')} for r in rows],ensure_ascii=False,indent=2))

if __name__=='__main__':
    main()
