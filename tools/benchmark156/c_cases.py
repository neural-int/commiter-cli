"""New controlled intent states and independently audited public mixed changes."""
import base64,copy,difflib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def b64(b):return base64.b64encode(b).decode()
def make(path,before,after):
    if isinstance(before,str):before=before.encode()
    if isinstance(after,str):after=after.encode()
    patch=''.join(difflib.unified_diff(before.decode('utf8',errors='replace').splitlines(True),after.decode('utf8',errors='replace').splitlines(True),fromfile='a/'+path,tofile='b/'+path))
    return dict(path=path,before_b64=b64(before),after_b64=b64(after),public_patch=patch)
def controlled(name,records,states,tags):
    files=[dict(f,id=f'F{i+1:03}') for i,f in enumerate(records)]
    # Exact single-intent after states are evaluator-only, never model input.
    return dict(name=name,files=files,intent_states={intent:{files[i]['id']:b64(v.encode() if isinstance(v,str) else v) for i,v in enumerate(values)} for intent,values in states.items()},tags=tags,gold_status='identifiable controlled independent edits with evaluator-only single-intent states',source='new authored controlled workload; not natural repository history')
def cases():
    root=OUT/'corpus-c';r=json.loads(next(root.glob('*34130728.json')).read_text());files=copy.deepcopy(r['files']);states={'close':{},'cache':{}}
    for f in files:
        before=base64.b64decode(f['before_b64']);after=base64.b64decode(f['after_b64'])
        if f['path']=='composite_test.go':
            cache_start=after.index(b'func TestCacheOnReadFsWriteReader(');cache_end=after.index(b'func NewTempOsBaseFs(',cache_start)
            close=after[:cache_start]+after[cache_end:]
            cache=before.replace(b'func NewTempOsBaseFs(',after[cache_start:cache_end]+b'func NewTempOsBaseFs(',1)
        else:close=after;cache=before
        states['close'][f['id']]=b64(close);states['cache'][f['id']]=b64(cache)
    yield dict(name='afero-natural-shared-test',files=files,intent_states=states,tags=['public','mixed-file','shared-test','source-test','same-dir-independent'],gold_status='identifiable: UnionFile Close return fix/tests and independent CacheOnReadFs WriteReader test; commit title alone does not define gold',source=r['repository']+'@'+r['commit'])
    before='package retry\nfunc Wait() int { return 100 }\nfunc Log() bool { return false }\n';a=before.replace('100','250');b=before.replace('false','true');after=a.replace('false','true')
    yield controlled('separate-hunks',[make('retry/config.go',before,after)],{'wait':[a],'log':[b]},['mixed-file','separate-hunks'])
    before='const requestTimeout = 20; const traceEnabled = false;\n';a=before.replace('20','45');b=before.replace('false','true');after=a.replace('false','true')
    yield controlled('same-line-controls',[make('client/options.ts',before,after)],{'timeout':[a],'trace':[b]},['mixed-file','same-line','inline-required'])
    before='func Render(v int) string {\n if v == 0 { return "zero" }\n return fmt.Sprint(v)\n}\n';a=before.replace('v == 0','v <= 0');b=before.replace('fmt.Sprint(v)','strconv.Itoa(v)');after=a.replace('fmt.Sprint(v)','strconv.Itoa(v)')
    yield controlled('same-symbol-controls',[make('format/render.go',before,after)],{'negative':[a],'allocation':[b]},['mixed-file','same-symbol','adjacent'])
    src='最大待機=10\r\n通知色=赤\r\n';a=src.replace('10','30');b=src.replace('赤','青');after=a.replace('赤','青')
    test='assert(wait == 10)\n';newtest='assert(wait == 30)\n'
    yield controlled('utf8-cross-file',[make('ui/settings.txt',src,after),make('tests/wait.txt',test,newtest)],{'wait':[a,newtest],'color':[b,test]},['mixed-file','utf8','crlf','partial-cross-file','source-test'])
    before='retryLimit=3\ncacheBytes=256\n';a=before.replace('3','5');b=before.replace('256','512');after=a.replace('256','512')
    records=[make('runtime/config.txt',before,after),make('runtime/retry_test.txt','assert(retries==3)\n','assert(retries==5)\n'),make('runtime/cache_test.txt','assert(bytes==256)\n','assert(bytes==512)\n')]
    yield controlled('partial-cross-file',[*records],{'retry':[a,'assert(retries==5)\n','assert(bytes==256)\n'],'cache':[b,'assert(retries==3)\n','assert(bytes==512)\n']},['mixed-file','partial-cross-file','shared-config','source-test'])
    before='const batchSize=16\n';after='const batchSize=32\n';t='assert(batchSize==16)\n';ta='assert(batchSize==32)\n'
    yield controlled('single-purpose-control',[make('worker/batch.txt',before,after),make('test/batch.txt',t,ta)],{'batch':[after,ta]},['single-purpose','cross-dir','source-test'])
    yield controlled('independent-dir-control',[make('misc/rotation.txt','days=7\n','days=14\n'),make('misc/download.txt','threads=2\n','threads=4\n')],{'rotation':['days=14\n','threads=2\n'],'download':['days=7\n','threads=4\n']},['negative','same-dir-independent'])
