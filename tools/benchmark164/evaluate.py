"""Issue164 prompt-only paired evaluation, reusing the immutable Issue163 harness."""
import argparse
import copy
import json
import pathlib
import statistics
import subprocess
import sys
import time

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'tools/benchmark163'))
from harness import MODEL_PINS, LOCAL_PROMPT, PROFILE, encode, sha, loads, messages, request, classify, command, preflight
from local_trial import file, go, edges, split, allowed, state, plan_audit, swap_bytes
from resource import observed_call

OUT = ROOT / 'docs/benchmarks/issue-164'
BASE_REV = '3d514898144f00e37152148e3464f6fb44e71cc5'
POLICY = '''
Additional commit boundary decision policy (apply to both changed files):
1. Implementation and assertions of the same changed behavior strongly support merge. Merely sharing a test file or directory does not.
2. Separately reviewable behaviors support keep_separate. Different files or absent tests alone do not establish independence.
3. Sharing a helper, package, import or call relationship establishes possible relatedness, not a common change purpose or a must-merge requirement.
4. Prioritize source-confirmed necessary adaptations, compensating changes and dependencies. A provider-before-consumer split can be valid; do not mechanically merge every dependency.
5. If supplied evidence cannot determine a defensible boundary, use defer instead of inventing author intent. Increasing defer alone does not improve quality.
6. Recheck behavior and independent reviewability, not just changed-location relationships. Do not chain merges through transitive relatedness.
Decision order: inspect necessary adaptation/compensation evidence, then independence evidence, then defer if evidence is insufficient. When both merge and an ordered split are defensible, select either without forcing a unique author boundary.
'''


def write(name, data):
    with (OUT / name).open('x') as out:
        json.dump(data, out, ensure_ascii=False, indent=2); out.write('\n')


def corpus():
    specs = [
        ('nil-slice-corresponding-assertion', 'join',
         file('copy.go', 'func Clone(v []int) []int { return append([]int{}, v...) }', 'func Clone(v []int) []int { if v == nil { return nil }; return append([]int{}, v...) }'),
         file('copy_test.go', 'import "testing"\nfunc TestNil(t *testing.T) { if Clone(nil) == nil { t.Fatal("nil") } }', 'import "testing"\nfunc TestNil(t *testing.T) { if Clone(nil) != nil { t.Fatal("non-nil") } }'), {},
         'Clone(nil) changes observable nilness and its directly corresponding assertion changes; either partial tree fails.'),
        ('shared-bound-independent-ui-and-batch', 'separate',
         file('display.go', 'func DisplayWidth(n int) int { return Bound(n, 80) }', 'func DisplayWidth(n int) int { return Bound(n, 80) + 2 }'),
         file('batch.go', 'func BatchSize(n int) int { return Bound(n, 32) }', 'func BatchSize(n int) int { return Bound(n, 16) }'),
         {'bound.go': go('func Bound(n, max int) int { if n > max { return max }; return n }')},
         'Display padding and batch capacity are distinct observable behaviors. Unchanged Bound alone creates no required adaptation.'),
        ('same-package-independent-time-and-permission', 'separate',
         file('time.go', 'func Expired(now, deadline int) bool { return now > deadline }', 'func Expired(now, deadline int) bool { return now >= deadline }'),
         file('permission.go', 'func CanEdit(role string) bool { return role == "owner" }', 'func CanEdit(role string) bool { return role == "owner" || role == "editor" }'), {},
         'Deadline equality and role permission are separately reviewable behaviors with no shared contract or call relationship.'),
        ('shared-import-independent-search-and-sort', 'separate',
         file('find.go', 'import "strings"\nfunc Contains(s, q string) bool { return strings.Contains(s, q) }', 'import "strings"\nfunc Contains(s, q string) bool { return strings.Contains(strings.ToLower(s), strings.ToLower(q)) }'),
         file('order.go', 'import "strings"\nfunc Before(a, b string) bool { return strings.Compare(a, b) < 0 }', 'import "strings"\nfunc Before(a, b string) bool { return strings.Compare(a, b) > 0 }'), {},
         'Case-insensitive search and descending ordering change separate outputs; sharing strings import establishes no compensation.'),
        ('boolean-polarity-compensation', 'join',
         file('encode.go', 'func EncodeFlag(v bool) int { if v { return 1 }; return 0 }', 'func EncodeFlag(v bool) int { if v { return 0 }; return 1 }'),
         file('decode.go', 'func DecodeFlag(n int) bool { return n == 1 }', 'func DecodeFlag(n int) bool { return n == 0 }'),
         {'contract.go': go('// The local flag codec contract requires FlagRoundtrip for both boolean values.\nfunc FlagRoundtrip(v bool) bool { return DecodeFlag(EncodeFlag(v)) == v }'),
          'codec_test.go': go('import "testing"\nfunc TestCodec(t *testing.T) { if !FlagRoundtrip(false) || !FlagRoundtrip(true) { t.Fatal("roundtrip") } }')},
         'Encoding/decoding polarity changes compensate to preserve an explicit source contract; both partial trees fail the invariant test.'),
        ('new-round-provider-eb', 'dependency',
         file('total.go', 'func Total(n int) int { return n }', 'func Total(n int) int { return RoundToTen(n) }'),
         file('round.go', 'func Unchanged() int { return 10 }', 'func Unchanged() int { return 10 }\nfunc RoundToTen(n int) int { return (n + 9) / 10 * 10 }'), {},
         'New RoundToTen provider EB must precede consumer EA if split; merge and provider-first separation are both allowed.'),
        ('cache-expiry-multiple-boundaries', 'multiple',
         file('image.go', 'const ImageTTL = 60', 'const ImageTTL = 120'),
         file('document.go', 'const DocumentTTL = 90', 'const DocumentTTL = 180'), {},
         'Two cache policies may be jointly reviewed as one expiry adjustment or separately; no unique hidden intent or mandatory linkage is asserted.'),
        ('unknown-admission-contract', 'insufficient',
         file('cpu.go', 'func CPUReservation() int { return 2 }', 'func CPUReservation() int { return 3 }'),
         file('memory.go', 'func MemoryReservation() int { return 4 }', 'func MemoryReservation() int { return 6 }'),
         {'admission.go': go('// Admission is supplied by clients; its cross-resource constraints are unavailable.\ntype Admission func(cpu, memory int) bool\nfunc Admitted(check Admission) bool { return check(CPUReservation(), MemoryReservation()) }')},
         'The unknown client admission policy may couple reservations; input cannot identify a mandatory adaptation or independence. Diagnostic requires defer.'),
    ]
    result = []
    for name, kind, a, b, support, basis in specs:
        a = dict(a, id='EA'); b = dict(b, id='EB')
        support = dict(support, **{'go.mod': 'module example.test/issue164fixture\n\ngo 1.23.0\n'})
        result.append(dict(name=name, kind=kind, files=[a,b], support=support, gold_basis=basis,
                           input_sha256=sha(encode(dict(files=[a,b], support=support))), used_before_registration=False))
    return result


def identity(args):
    fixed = loads((ROOT / 'docs/benchmarks/issue-163/iteration-2-preregistered.json').read_bytes())
    binaries = {'helper_sha256': args.helper, 'facts_sha256': args.facts, 'validator_sha256': args.validator,
                'metal_library_sha256': str(pathlib.Path(args.helper).parent / 'mlx.metallib')}
    for key, path in binaries.items():
        assert sha(pathlib.Path(path).read_bytes()) == fixed[key], key
    downloaded = loads((ROOT / 'docs/benchmarks/issue-163/iteration-2-download.json').read_bytes())
    model = {f['file']: sha((pathlib.Path(args.coder) / f['file']).read_bytes()) for f in downloaded['files']}
    assert all(model[f['file']] == f['sha256'] for f in downloaded['files'])
    return dict(binaries={k:fixed[k] for k in binaries}, model_files_sha256=model,
                model_pin=MODEL_PINS['coder'], sources={str(p.relative_to(ROOT)): sha(p.read_bytes()) for p in
                sorted((ROOT / 'tools/benchmark163').rglob('*')) + [pathlib.Path(__file__)] if p.is_file() and '__pycache__' not in str(p)})


def prepare(args):
    start = time.monotonic(); OUT.mkdir(parents=True, exist_ok=True)
    ident = identity(args)
    old = loads((ROOT / 'docs/benchmarks/issue-163/iteration-2-preregistered.json').read_bytes())
    known = copy.deepcopy(old['cases']); cases = corpus(); audits = []
    prior = loads(command(['git','show','b8a7fe14a798ff36680efffdbd879c9a02cb37f2:docs/benchmarks/issue-162/iteration-2-preregistered.json']).stdout)
    prior_hashes = {sha(f[s].encode()) for c in known + prior['cases'] for f in c['files'] for s in ('before','after')}
    for c in cases:
        c['prior_changed_source_matches'] = [f['id']+':'+s for f in c['files'] for s in ('before','after') if sha(f[s].encode()) in prior_hashes]
        assert not c['prior_changed_source_matches']
        c['facts'] = loads(command([args.facts], encode(c['files'])).stdout); c['edges'] = edges(c['facts'])
        probes = {name:state(c, selected) for name,selected in [('before',set()),('only_a',{'EA'}),('only_b',{'EB'}),('after',{'EA','EB'})]}
        assert probes['before']['pass'] and probes['after']['pass']
        if c['kind'] == 'join': assert not probes['only_a']['pass'] and not probes['only_b']['pass']
        if c['kind'] in ('separate','multiple','insufficient'): assert all(p['pass'] for p in probes.values())
        if c['kind'] == 'dependency': assert len(c['edges']) == 1 and not probes['only_a']['pass'] and probes['only_b']['pass']
        audits.append(dict(case=c['name'],probes=probes, plans={name:plan_audit(c,g,args.validator) for name,g in [('merge',[['EA','EB']]),('ordered_split',split(c))]}))
    check = preflight(args.validator, args.facts); assert check['all_pass']; write('preflight.json',check)
    rows = []
    for cohort, dataset in [('fresh',cases),('known',known)]:
        for c in dataset:
            for reverse in (False,True):
                msg = messages('local',c['files'],c['support'],c['facts'],reverse)
                if cohort == 'known':
                    previous = next(r for r in old['rows'] if r['case']==c['name'] and r['reverse']==reverse)
                    assert msg == previous['messages']
                for arm in ('baseline','policy') if cohort == 'fresh' else ('policy',):
                    m = copy.deepcopy(msg)
                    if arm == 'policy': m[0]['content'] = LOCAL_PROMPT + POLICY + '\nSchema: ' + m[0]['content'].split('\nSchema: ',1)[1]
                    rows.append(dict(cohort=cohort,case=c['name'],reverse=reverse,arm=arm,messages=m,prompt_sha256=sha(encode(m))))
    tokens = []
    for row in rows:
        c = next(c for c in cases+known if c['name']==row['case'])
        req = request('coder',args.coder,args.helper,ident['binaries']['helper_sha256'],'local',c['files'],c['support'],c['facts'],row['reverse'])
        req.update(messages=row['messages'], preflight_only=True)
        p = subprocess.run(['/usr/bin/sandbox-exec','-p','(version 1)(allow default)(deny network*)',args.helper],input=encode(req)+b'\n',capture_output=True,timeout=30)
        response = loads(p.stdout); n = response.get('benchmark_input_tokens')
        assert p.returncode==0 and response.get('ok') is True and response.get('stop_reason')=='completed' and response.get('generated_json')=='{}' and type(n) is int and n<=4096 and n+1536<=16384 and response.get('benchmark_output_tokens') is None
        row['native_input_tokens']=n
        tokens.append(dict(cohort=row['cohort'],case=row['case'],arm=row['arm'],reverse=row['reverse'],input_tokens=n,raw=response,stdout_sha256=sha(p.stdout),stderr_sha256=sha(p.stderr)))
        print(json.dumps({k:row[k] for k in ('cohort','case','arm','reverse','native_input_tokens')}),flush=True)
    write('input-audit.json',audits); write('native-token-audit.json',tokens)
    fixed = dict(issue=164,base_revision=BASE_REV,identity=ident,cases=cases,known_cases=known,rows=rows,
                 baseline_prompt=LOCAL_PROMPT,additional_policy=POLICY,schema_in_all_messages=True,
                 context_tokens=16384,output_tokens=1536,temperature=0,top_p=1,top_k=0,seed=144,native_thought_tokens=0,
                 profile=PROFILE,decoder='pinned neutral JSON grammar, no preference bias',per_call_seconds=30,whole_seconds=1800,audit_reserve_seconds=400,
                 max_fresh_generation_calls=32,max_known_generation_calls=20,retries=0,max_native_input_tokens=4096,memory_gate_bytes=5000000000,
                 preparation_seconds=time.monotonic()-start, freshness='self-authored never inferred before registration; not independent author/repository adoption holdout',
                 family_correlation='nil/assertion, roundtrip compensation, new API, TTL/multiple, unknown admission reuse task families from #162/#163; 3 independent cases have distinct behaviors but shared negative-category correlation. No exact changed-source hash overlap; rename invariance not claimed.',
                 gate='fresh: both arms16/16 schema/source-ref valid; policy16/16 allowed; policy FM0/6 and strictly less than baseline; required FS0/4 and no increase; insufficient defer2/2, other defer0/14; policy finalized coverage >= baseline on 14 decidable rows; order8/8; authoritative/byte/tree/ordered all pass for finalized plans in both arms; all process footprint<=5e9, measured TTFT, pressure all1, swap delta<=0; policy median latency<=2x baseline; native input tokens<=4096, total input+output<=3x baseline; preparation+all52generation+audits<=1800sec. Known diagnosis excluded from qualification.',
                 comparison_limit='baseline already includes grouping rules; estimates additional explicit policy, not absence vs presence of all rules. Known baseline reused historical rows, non-concurrent exploratory comparison only.',
                 token_audit_sha256=sha((OUT/'native-token-audit.json').read_bytes()),input_audit_sha256=sha((OUT/'input-audit.json').read_bytes()),production_changed=False)
    write('preregistered.json',fixed)
    print(json.dumps(dict(generation_calls=0,tokenizer_calls=len(rows),preparation_seconds=fixed['preparation_seconds'])),flush=True)


def summarize(rows,cases):
    bycase={c['name']:c for c in cases}; n=len(cases)
    decisions=[r.get('answer',{}).get('decision') for r in rows]
    pairs=[[r for r in rows if r['case']==c['name']] for c in cases]
    resources=[r.get('resource',{}) for r in rows]
    return dict(total=len(rows),valid=sum(r['response_valid'] for r in rows),allowed=sum(r['allowed'] for r in rows),
                false_merge=sum(d=='merge' and bycase[r['case']]['kind']=='separate' for r,d in zip(rows,decisions)),
                false_split=sum(d=='keep_separate' and bycase[r['case']]['kind']=='join' for r,d in zip(rows,decisions)),
                defer=sum(d=='defer' for d in decisions),insufficient_defer=sum(d=='defer' and bycase[r['case']]['kind']=='insufficient' for r,d in zip(rows,decisions)),
                decidable_coverage=sum(d in ('merge','keep_separate') and bycase[r['case']]['kind']!='insufficient' for r,d in zip(rows,decisions)),
                order_stable=sum(len(pair)==2 and all(r['response_valid'] for r in pair) and pair[0]['answer']['decision']==pair[1]['answer']['decision'] for pair in pairs),cases=n,
                statuses=dict(__import__('collections').Counter(r['status'] for r in rows)),
                safe=all(r.get('audit',{}).get('authoritative',{}).get('valid') and r['audit']['reconstruction']['exact'] and r['audit']['all_ordered_pass'] for r,d in zip(rows,decisions) if d in ('merge','keep_separate')),
                input_tokens=sum(r['input_tokens'] for r in rows) if all(r['input_tokens'] is not None for r in rows) else None,
                output_tokens=sum(r['output_tokens'] for r in rows) if all(r['output_tokens'] is not None for r in rows) else None,
                walltime_median=statistics.median(r['total_seconds'] for r in resources) if len(resources)==len(rows) and rows else None,
                ttft_median=statistics.median(r['ttft_seconds'] for r in resources) if rows and all(r.get('ttft_seconds') is not None for r in resources) else None,
                footprint_max=max(r['sampled_peak_footprint_bytes'] for r in resources) if resources and all(r.get('sampled_peak_footprint_bytes') is not None for r in resources) else None,
                rss_max=max(r['sampled_peak_rss_bytes'] for r in resources) if resources and all(r.get('sampled_peak_rss_bytes') is not None for r in resources) else None,
                mlx_peak_max=max(r['mlx_peak_bytes'] for r in resources) if resources and all(r.get('mlx_peak_bytes') is not None for r in resources) else None,
                resource_pass=sum(r.get('resource_gate',False) for r in resources))


def run(args):
    start=time.monotonic()
    fixed=loads((OUT/'preregistered.json').read_bytes()); assert identity(args)==fixed['identity']
    for name in ('native-token-audit','input-audit'): assert sha((OUT/(name+'.json')).read_bytes())==fixed[('token_audit' if name=='native-token-audit' else 'input_audit')+'_sha256']
    cases={c['name']:c for c in fixed['cases']+fixed['known_cases']}; cache={}; results=[]
    # Interleave arms, alternating first arm for each presentation; fresh always precedes known.
    specs=[]
    for i in range(0,32,2): specs.extend(fixed['rows'][i:i+2] if i//2%2==0 else reversed(fixed['rows'][i:i+2]))
    specs.extend(fixed['rows'][32:])
    with (OUT/'results.jsonl').open('x') as out:
        for spec in specs:
            c=cases[spec['case']]; remaining=1800-fixed['preparation_seconds']-(time.monotonic()-start)-400
            row={k:spec[k] for k in ('cohort','case','arm','reverse','prompt_sha256')}; row.update(calls=0,response_valid=False,allowed=False,input_tokens=None,output_tokens=None)
            if remaining<=0: row['status']='total_budget'
            else:
                req=request('coder',args.coder,args.helper,fixed['identity']['binaries']['helper_sha256'],'local',c['files'],c['support'],c['facts'],spec['reverse']); req['messages']=spec['messages']
                row['calls']=1; res=observed_call(['/usr/bin/sandbox-exec','-p','(version 1)(allow default)(deny network*)',args.helper],encode(req)+b'\n',min(30,remaining))
                response=classify(res['stdout'].encode(),'local',['EA','EB'])
                if res['timed_out']: response.update(status='timeout',accepted=False)
                elif res['exit']!=0: response.update(status='helper_exit',accepted=False)
                row.update(resource=res,response=response,status=response['status'],response_valid=response['status'] in ('completed','unresolved'),input_tokens=response['input_tokens'],output_tokens=response['output_tokens'])
                try:
                    raw=loads(res['stdout'])
                    for name in ('ttft_seconds','load_seconds','prepared_seconds','first_chunk_seconds','prefill_seconds','decode_seconds','mlx_active_bytes','mlx_peak_bytes','mlx_cache_bytes'): res[name]=raw.get('benchmark_'+name)
                    row['model_pin_matches']=raw.get('model')==MODEL_PINS['coder']
                except ValueError: row['model_pin_matches']=False
                before=swap_bytes(res['system_before']); after=swap_bytes(res['system_after']); res['swap_delta_bytes']=after-before if before is not None and after is not None else None
                levels=[p['level'] for p in res['pressure_samples']]+[int(res[m]['pressure_level']['stdout'].strip()) if res[m]['pressure_level'].get('stdout','').strip().isdigit() else None for m in ('system_before','system_after')]
                res['pressure_levels']=levels
                res['resource_gate']=bool(res['sampled_peak_footprint_bytes'] is not None and res['sampled_peak_footprint_bytes']<=5000000000 and res['ttft_seconds'] is not None and levels and all(v==1 for v in levels) and res['swap_delta_bytes'] is not None and res['swap_delta_bytes']<=0)
                row['native_tokens_match']=row['input_tokens']==spec['native_input_tokens']
                if row['response_valid']:
                    answer=response['answer']; row.update(answer=answer,allowed=allowed(c,answer['decision']))
                    if answer['decision']!='defer':
                        groups=[['EA','EB']] if answer['decision']=='merge' else split(c); key=(c['name'],encode(groups))
                        if key not in cache: cache[key]=plan_audit(c,groups,args.validator)
                        row.update(groups=groups,audit=cache[key])
            results.append(row); out.write(json.dumps(row,ensure_ascii=False)+'\n'); out.flush()
            print(json.dumps({k:row.get(k) for k in ('cohort','case','arm','reverse','status','allowed')}),flush=True)
    previous=[loads(l) for l in (ROOT/'docs/benchmarks/issue-163/iteration-2-results.jsonl').read_text().splitlines() if loads(l)['model']=='coder']
    fresh={arm:summarize([r for r in results if r['cohort']=='fresh' and r['arm']==arm],fixed['cases']) for arm in ('baseline','policy')}
    known=dict(baseline=summarize(previous,fixed['known_cases']),policy=summarize([r for r in results if r['cohort']=='known'],fixed['known_cases']))
    a=fresh['baseline']; b=fresh['policy']; elapsed=time.monotonic()-start+fixed['preparation_seconds']
    gates=dict(structure=a['valid']==b['valid']==16,policy_allowed=b['allowed']==16,false_merge=b['false_merge']==0 and b['false_merge']<a['false_merge'],
               false_split=b['false_split']==0 and b['false_split']<=a['false_split'],defer=b['insufficient_defer']==b['defer']==2,coverage=b['decidable_coverage']>=a['decidable_coverage'],
               order=b['order_stable']==8,safety=a['safe'] and b['safe'],resource=a['resource_pass']==b['resource_pass']==16,
               cost=a['walltime_median'] is not None and b['walltime_median'] is not None and b['walltime_median']<=2*a['walltime_median'] and a['input_tokens'] is not None and b['input_tokens'] is not None and a['output_tokens'] is not None and b['output_tokens'] is not None and b['input_tokens']+b['output_tokens']<=3*(a['input_tokens']+a['output_tokens']),
               identity=all(r.get('model_pin_matches') and r.get('native_tokens_match') for r in results),whole_budget=elapsed<=1800,
               all_generation_rows=sum(r['calls'] for r in results)==52)
    write('summary.json',dict(issue=164,fresh=fresh,known_exploratory=known,gates=gates,decision='GO for next local trial only' if all(gates.values()) else 'NO-GO',
                              generation_calls=sum(r['calls'] for r in results),elapsed_seconds=elapsed,preparation_seconds=fixed['preparation_seconds'],post_identity_match=identity(args)==fixed['identity'],
                              limits='self-authored correlated 8case paired comparison, not 16file/80% exact/metadata/production qualification; known historical baseline is not concurrent'))
    print(json.dumps(dict(gates=gates,elapsed_seconds=elapsed)),flush=True)


if __name__=='__main__':
    parser=argparse.ArgumentParser(); parser.add_argument('action',choices=('prepare','run'))
    for name in ('helper','coder','facts','validator'): parser.add_argument('--'+name,required=True)
    args=parser.parse_args(); prepare(args) if args.action=='prepare' else run(args)
