"""Fixed context selection and provenance triage, without model inference."""
import hashlib
import json
import pathlib
import subprocess
from context import encode, select, MAX_INPUT_BYTES
from input_audit import records

ROOT=pathlib.Path(__file__).resolve().parents[2]
OUT=ROOT/'docs/benchmarks/issue-155'
OBSERVER=pathlib.Path('/tmp/issue155-context-observer')

def sha(raw):return hashlib.sha256(raw).hexdigest()
def git(*args):return subprocess.check_output(['git',*args],cwd=ROOT)
def observe(files):return json.loads(subprocess.run([str(OBSERVER)],input=encode({'files':files}),capture_output=True,check=True,timeout=10).stdout)

def main():
    cases=records()
    for commit,paths in [('8092acd2',['internal/relation/graph.go','internal/relation/graph_test.go']),('c643154b',['internal/relation/extract.go','internal/relation/extract_test.go'])]:
        full=git('rev-parse',commit).decode().strip();parent=git('rev-parse',commit+'^').decode().strip()
        files=[dict(id=f'F{i+1:03}',path=p,before=git('show',parent+':'+p).decode(),after=git('show',full+':'+p).decode()) for i,p in enumerate(paths)]
        cases.append(dict(name=commit,split='provisional_real_candidate',commit=full,parent=parent,files=files))
    # Fix the protocol before observing its outputs. Byte limit is this new task's
    # bound, not an assertion that it shares #153's old 8192-byte cost.
    contract=dict(model_calls=0,context_method='changed_top_level_declarations_plus_one_same_package_plain_call_step_exact_text_dedup',
        max_context_payload_bytes=MAX_INPUT_BYTES,fixture_sha256=sha(encode(cases)),observer_sha256=sha(OBSERVER.read_bytes()),
        context_source_sha256=sha((ROOT/'tools/benchmark155/context.py').read_bytes()),
        gate='Source/budget/provenance preflight only; attribution schema+message budget+holdout+numerical capability gate still required',
        prior_scan_scope=['docs/benchmarks/issue-149','docs/benchmarks/issue-151','docs/benchmarks/issue-152','docs/benchmarks/issue-153','tools/benchmark149','tools/benchmark151','tools/benchmark152','tools/benchmark153'])
    with (OUT/'iteration-2-context-contract.json').open('x') as s:json.dump(contract,s,indent=2);s.write('\n')
    corpus=[p for base in contract['prior_scan_scope'] for p in (ROOT/base).rglob('*') if p.is_file() and p.suffix in ('.json','.jsonl','.md','.py','.go') and not any(x in p.name.lower() for x in ('credential','secret','key','.env'))]
    rows=[]
    for case in cases:
        files=case['files'];facts=observe(files);context=select(files,facts)
        assert context==select(list(reversed(files)),observe(list(reversed(files))))
        raw=encode(context);changed_tests=[e for e in facts['evidence'] if e['kind']=='function_source' and e['changed'] and e['version']=='after' and next(f['path'] for f in files if f['id']==e['file']).endswith('_test.go')]
        hits={e['function']: [str(p.relative_to(ROOT)) for p in corpus if e['function'].encode() in p.read_bytes()] for e in changed_tests}
        result=dict(name=case['name'],split=case['split'],commit=case.get('commit'),parent=case.get('parent'),
            full_snapshot_bytes=len(encode({'files':files})),context_payload_bytes=len(raw),within_context_byte_bound=len(raw)<=MAX_INPUT_BYTES,
            context_sha256=sha(raw),context=context,deterministic=True,prior_test_symbol_hits=hits,
            model_calls=0,holdout_certified=False,semantic_quality=None)
        rows.append(result)
    with (OUT/'iteration-2-context-audit.json').open('x') as s:json.dump(dict(contract=contract,rows=rows),s,ensure_ascii=False,indent=2);s.write('\n')
    print(json.dumps([dict(name=r['name'],snapshot_bytes=r['full_snapshot_bytes'],context_bytes=r['context_payload_bytes'],context_bound=r['within_context_byte_bound'],prior_test_symbol_hits=r['prior_test_symbol_hits']) for r in rows],ensure_ascii=False,indent=2))

if __name__=='__main__':main()
