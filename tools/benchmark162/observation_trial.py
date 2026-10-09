"""Iteration 3: fresh authored fixtures, test observability and measured probe cost."""
import argparse, json, pathlib, subprocess, tempfile, time
from partition_trial import ROOT, OUT, FACTS, VALIDATOR, file, go, facts, dependencies, plan, review_valid, audit, materialize, run_command, encode, sha


def dataset():
    rows = []
    def add(name, condition, kind, a, b, support_body='', assertion=None, rationale=''):
        support = {'go.mod': 'module example.test/observationfixture\n\ngo 1.27\n', 'support.go': go(support_body)}
        if assertion:
            support['support_test.go'] = go('import "testing"\nfunc TestResult(t *testing.T) { '+assertion+' }')
        fs = [a, b] + [file('E2','padding.go',go('const Padding = 2'),go('const Padding = 3')),file('E3','buffer.go',go('const Buffer = 8'),go('const Buffer = 16'))]
        rows.append({'name':name,'condition':condition,'kind':kind,'files':fs,'support':support,'rationale':rationale,'input_sha256':sha(encode({'files':fs,'support':support}))})
    add('absent-independent','absent','must_separate',file('EA','accept.go',go('func Accept(n int) bool { return n >= 0 }'),go('func Accept(n int) bool { return n > 0 }')),file('EB','limit.go',go('const Limit = 10'),go('const Limit = 20')),rationale='独立した受理条件と上限値の変更。テストなし。')
    add('absent-new-api','absent','dependency',file('EA','consumer.go',go('func Read(n int) int { return n }'),go('func Read(n int) int { return Normalize(n) }')),file('EB','provider.go',go('const Version = 1'),go('const Version = 1\nfunc Normalize(n int) int { if n < 0 { return 0 }; return n }')),rationale='新APIのprovider/consumer。統合と依存順分割を許容。テストなしでもコンパイル依存は観測可能。')
    for weak in (False,True):
        mode='weak' if weak else 'unchanged-strong'
        add(mode+'-offset','one-sided-unobserved' if weak else 'unchanged-observing','must_join',file('EA','compute.go',go('func Compute(n int) int { return n + 1 }'),go('func Compute(n int) int { return n + 2 }')),file('EB','offset.go',go('const Offset = 1'),go('const Offset = 2')),'func Result(n int) int { return Compute(n) - Offset }','if Result(3) < 0 { t.Fatal("negative") }' if weak else 'if got := Result(3); got != 3 { t.Fatal(got) }','Result(n)=nを維持する補償変更。片側だけではResult(3)が2または4になるため、本試験では共同レビューを要求。弱いassertionでは両方とも通る。')
        add(mode+'-scale','one-sided-unobserved' if weak else 'unchanged-observing','must_join',file('EA','scale.go',go('func Scale(n int) int { return n * 2 }'),go('func Scale(n int) int { return n * 4 }')),file('EB','divisor.go',go('const Divisor = 2'),go('const Divisor = 4')),'func Result(n int) int { return Scale(n) / Divisor }','if Result(8) < 0 { t.Fatal("negative") }' if weak else 'if got := Result(8); got != 8 { t.Fatal(got) }','Result(n)=nを維持する倍率・除数の補償変更。片側だけではResult(8)が4または16になる。本試験のレビュー規範はjoin。')
    add('weak-independent-first','one-sided-unobserved','must_separate',file('EA','first.go',go('func First(n int) int { return n + 1 }'),go('func First(n int) int { return n + 2 }')),file('EB','second.go',go('func Second(n int) int { return n * 2 }'),go('func Second(n int) int { return n * 4 }')),'','if First(3) < 0 { t.Fatal("negative") }','独立した関数変更。弱い既存テストはSecondを呼ばない。')
    add('weak-independent-second','one-sided-unobserved','must_separate',file('EA','enabled.go',go('func Enabled(n int) bool { return n >= 0 }'),go('func Enabled(n int) bool { return n > 0 }')),file('EB','capacity.go',go('func Capacity(n int) int { return n + 5 }'),go('func Capacity(n int) int { return n + 10 }')),'','if Capacity(3) < 0 { t.Fatal("negative") }','独立した条件と容量の変更。既存テストはEnabledを呼ばない。')
    return rows


def probe(c, ids, deadline):
    if time.monotonic() >= deadline: raise ValueError('total_budget')
    start = time.monotonic()
    with tempfile.TemporaryDirectory(prefix='benchmark162-observation-') as work:
        materialize(work,c,set(ids))
        p=run_command(['go','test','-count=1','-json','./...'],work,timeout=min(30,max(.1,deadline-time.monotonic())))
    events=[]
    for line in p.stdout.splitlines():
        try: events.append(json.loads(line))
        except json.JSONDecodeError: pass
    return {'selected':sorted(ids),'pass':p.returncode==0,'exit':p.returncode,'tests_run':sum(e.get('Action')=='run' and 'Test' in e for e in events),'seconds':time.monotonic()-start,'log':(p.stdout+p.stderr).decode(errors='replace'),'log_sha256':sha(p.stdout+p.stderr)}


def choose(observed, edges, cautious):
    # No review label, source rationale or fixture condition is passed to selector.
    if not observed['before']['pass'] or not observed['after']['pass'] or not observed['both']['pass']:
        return 'defer',edges,'endpoint_or_pair_invalid'
    a,b=observed['only_a']['pass'],observed['only_b']['pass']
    if not a and not b: return 'merge',edges,'both_partial_states_fail'
    if a != b:
        direction={'provider':'EA' if a else 'EB','consumer':'EB' if a else 'EA','basis':'partial-state failure ordering only; not purpose proof'}
        return 'keep_separate',edges+[direction],'one_partial_state_fails'
    if cautious: return 'defer',edges,'both_partial_states_pass_without_independence_proof'
    return 'keep_separate',edges,'both_partial_states_pass'


def freeze():
    cases=dataset()
    record={'contract':{'cases':8,'selected_files':4,'model_calls':0,'total_seconds':1200,'command_seconds':30,'retries':0,'test_command':'go test -count=1 -json ./...','conditions':['absent','unchanged-observing','one-sided-unobserved'],'methods':['file-only','static-new-api-order','test-assisted-v2','observability-cautious'],'evaluation':'record review validity separately from ordered tests; cautious defer is unresolved, never successful split; same all-pass observation for must_join and must_separate falsifies test-only independence inference','adoption_gate':'all8 partitions finalized, review valid and ordered tests/byte/authoritative valid; no forced wrong split or overmerge; total<=1200s. Study completion is separate from adoption GO','cost':'one measured probe set per case reused by test-assisted methods; report marginal observation cost and shared audit cost separately; no sum double counting; test execution cache disabled via -count=1; Go compilation cache normal; source and audit use existing helpers','limitations':'authored local pair fixtures, no public history or full4/16file grouping; cautious policy can abstain without proving review validity; assertion count does not prove coverage; must_join compensating pairs defined normatively before execution, not author intent','safety':'self-authored code only, temp dirs, no external code tests/new dependencies/models/production edits'},'cases':cases,'source_sha256':{str(p.relative_to(ROOT)):sha(p.read_bytes()) for p in [pathlib.Path(__file__),ROOT/'tools/benchmark162/partition_trial.py',ROOT/'tools/benchmark162/benchmark.py']},'facts_binary_sha256':sha(FACTS.read_bytes()),'validator_binary_sha256':sha(VALIDATOR.read_bytes())}
    with (OUT/'iteration-3-preregistered.json').open('x') as f:json.dump(record,f,ensure_ascii=False,indent=2);f.write('\n')
    print('Fixed 8 fresh cases; zero fixture/model execution before registration')


def run():
    fixed=json.loads((OUT/'iteration-3-preregistered.json').read_text())
    for path,digest in fixed['source_sha256'].items():
        if sha((ROOT/path).read_bytes())!=digest:raise ValueError('source_changed')
    assert sha(FACTS.read_bytes())==fixed['facts_binary_sha256']
    assert sha(VALIDATOR.read_bytes())==fixed['validator_binary_sha256']
    start=time.monotonic();deadline=start+1200;rows=[]
    with (OUT/'iteration-3-results.jsonl').open('x') as out:
        for c in fixed['cases']:
            t=time.monotonic();edges=dependencies(facts(c['files']));source_seconds=time.monotonic()-t
            observed={name:probe(c,ids,deadline) for name,ids in [('before',[]),('only_a',['EA']),('only_b',['EB']),('both',['EA','EB']),('after',[f['id'] for f in c['files']])]}
            if not observed['before']['pass'] or not observed['after']['pass']:raise ValueError('invalid_fixture_endpoint: '+c['name'])
            row={'case':c['name'],'kind':c['kind'],'condition':c['condition'],'source_seconds':source_seconds,'dependencies':edges,'observations':observed,'probe_seconds':sum(x['seconds'] for x in observed.values()),'methods':{}};cache={}
            specs=[('file-only','keep_separate',[], 'input_order'),('static-new-api-order','keep_separate',edges,'static_order')]
            for name,cautious in [('test-assisted-v2',False),('observability-cautious',True)]:
                d,e,reason=choose(observed,edges,cautious);specs.append((name,d,e,reason))
            for name,d,e,reason in specs:
                if d=='defer':row['methods'][name]={'decision':d,'reason':reason,'finalized':False,'review_valid':None};continue
                groups=[[f['id']] for f in c['files']] if name=='file-only' else plan(c,d,e)
                key=tuple(tuple(g) for g in groups)
                reused=key in cache
                if not reused:
                    t=time.monotonic();cache[key]=(audit(c,groups,deadline),time.monotonic()-t)
                result,cost=cache[key]
                row['methods'][name]={'decision':d,'reason':reason,'groups':groups,'finalized':True,'review_valid':review_valid(c,groups),'audit':result,'audit_seconds':cost,'audit_reused':reused}
            rows.append(row);out.write(json.dumps(row,ensure_ascii=False)+'\n');out.flush()
            print(json.dumps({'case':c['name'],'partial_pass':[observed['only_a']['pass'],observed['only_b']['pass']],'tests_run':[x['tests_run'] for x in observed.values()],'methods':{n:{'finalized':r['finalized'],'review_valid':r['review_valid']} for n,r in row['methods'].items()},'elapsed':time.monotonic()-start}),flush=True)
    summary={'cases':len(rows),'model_calls':0,'total_seconds':time.monotonic()-start,'budget_pass':time.monotonic()<=deadline,'source_seconds':sum(r['source_seconds'] for r in rows),'probe_seconds':sum(r['probe_seconds'] for r in rows),'actual_probe_commands':len(rows)*5,'audit_seconds_unique':sum(m['audit_seconds'] for r in rows for m in r['methods'].values() if m['finalized'] and not m['audit_reused']),'methods':{}}
    for n in rows[0]['methods']:
        ms=[r['methods'][n] for r in rows];done=[m for m in ms if m['finalized']]
        summary['methods'][n]={'finalized':len(done),'unresolved':len(ms)-len(done),'valid':sum(m['audit']['valid'] for m in done),'wrong_review':sum(not m['review_valid'] for m in done),'ordered_tests_pass':sum(m['audit']['all_ordered_states_pass'] for m in done),'adoption_go':len(done)==8 and all(m['audit']['valid'] for m in done) and summary['budget_pass']}
    summary['all_pass_signature_review_labels']=sorted({r['kind'] for r in rows if all(v['pass'] for v in r['observations'].values()) and all(v['tests_run']>0 for v in r['observations'].values())})
    with (OUT/'iteration-3-summary.json').open('x') as f:json.dump(summary,f,ensure_ascii=False,indent=2);f.write('\n')
    print(json.dumps(summary),flush=True)

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('mode',choices=['freeze','run']);a=p.parse_args();freeze() if a.mode=='freeze' else run()
