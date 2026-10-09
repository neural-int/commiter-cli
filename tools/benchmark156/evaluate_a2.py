"""A2 structural diagnosis on used D inputs; never a new independent holdout."""
import hashlib,json,pathlib,subprocess,sys,time
from planner import encode,fallback,replay
from coarse import prepare,refine
from evaluate_d import cases,strat
from evaluate_a import metrics
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'

def preflight():
    sources=['coarse.py','evaluate_a2.py','planner.py','vendor151/evaluate.py','vendor151/inline.py']
    fixed=dict(contract=dict(hypothesis='line ownership before optional inline refinement removes eager atomization rejection without increasing resource caps',dataset='used iteration-4 structural diagnosis only',limits=dict(files=16,bytes_per_file=1048576,lines_per_file=20000,coarse_operations_per_file=256,fine_units_per_file=256,inline_bytes=4096,total_units=4096),calls=0,retries=0,acceptance='all six reconstruct/stage/Validate; 16 selected files accepted; pair metrics only at file level, never compared across line/inline granularity',semantic='unknown fallback unchanged; no claim of exact improvement or mixed detection'),rows=list(cases()),source_sha256={p:hashlib.sha256((ROOT/'tools/benchmark156'/p).read_bytes()).hexdigest() for p in sources})
    for r in fixed['rows']:r['input_sha256']=hashlib.sha256(encode(r['files'])).hexdigest()
    with (OUT/'iteration-5-preregistered.json').open('x') as f:json.dump(fixed,f,ensure_ascii=False,indent=2);f.write('\n')
    print('preregistered six used diagnostic cases; no model calls')

def run():
    fixed=json.loads((OUT/'iteration-5-preregistered.json').read_text())
    for p,d in fixed['source_sha256'].items():
        if hashlib.sha256((ROOT/'tools/benchmark156'/p).read_bytes()).hexdigest()!=d:raise ValueError('source_digest')
    with (OUT/'iteration-5-results.jsonl').open('x') as out:
        for case in fixed['rows']:
            start=time.perf_counter();files=case['files'];units=prepare(files);commits=fallback(files,units)
            groups=[dict(unit_ids=c['file_ids']) for c in commits]
            gold={fid:i for i,g in enumerate(case['gold_file_groups']) for fid in g}
            row=dict(name=case['name'],stratum=strat(len(files)),files=len(files),coarse_operations=len(units),calls=0,input_tokens=0,output_tokens=0,input_sha256=case['input_sha256'],complete=True,fallback_rate=1.,compose_rate=0.,metric_granularity='files',**metrics([],groups,gold))
            row['staging']=replay(files,units,commits)
            plan=dict(schema_version=1,commits=[{k:c[k] for k in ('type','scope','breaking','summary','file_ids')} for c in commits])
            row['validation']=json.loads(subprocess.run(['/private/tmp/benchmark156-validate'],input=encode([dict(candidate=plan,file_ids=[f['id'] for f in files])]),capture_output=True,check=True).stdout)[0]
            if not row['validation']['valid']:raise ValueError('invalid_plan')
            if case['name']=='any-migration-16':
                f=next(f for f in files if f['path']=='cmp/compare_test.go');refined,reason=refine(f)
                row['optional_refinement']=dict(path=f['path'],reason=reason,retained_units=len(refined),coarse_fallback=reason is not None)
            row['wall_seconds']=time.perf_counter()-start
            out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush();print(json.dumps(row,ensure_ascii=False),flush=True)
if __name__=='__main__':preflight() if sys.argv[1]=='preflight' else run()
