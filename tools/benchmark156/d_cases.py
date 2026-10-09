"""D independent set fixed before comparator inference, separate from C."""
import base64,copy,json,pathlib
from c_cases import make,controlled
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
def source(prefix):return json.loads((OUT/'corpus-d'/(prefix+'.json')).read_text())
def public(name,records,counts=None,tags=()):
 files=[];purposes={};commits=[]
 for i,r in enumerate(records):
  subset=r['files'] if counts is None else r['files'][:counts[i]]
  for f in subset:
   new=dict(f,id=f'F{len(files)+1:03}');files.append(new);purposes[new['id']]=i
  commits.append(r['repository']+'@'+r['commit'])
 if len({f['path'] for f in files})!=len(files):raise ValueError('public_superposition_overlap')
 return dict(name=name,files=files,gold_file_purposes=purposes,metric_granularity='line-operation',primary=True,gold_status='identifiable source-level public edit purposes; full-file gold independently audited from actual implementation/assertions or mechanical formatter edits',tags=list(tags),source_commits=commits,superposition=len(records)>1,correlated_family='testify-disjoint-history-components')
def cases():
 mock=source('5e8a21e6');assertion=source('15f6826f');fmt=source('3dce331f');doc=source('cce2851b');ci=source('2e4825f0');require=source('556f9edb')
 yield public('mock-sentinel',[mock],tags=['positive','source-test'])
 yield public('not-subset-format',[assertion],tags=['positive','source-test'])
 yield public('mock-plus-assert-negative',[mock,assertion],tags=['negative','cross-dir-independent'])
 yield public('formatter-eight',[fmt],counts=[8],tags=['positive','5-8','selected-projection','mechanical'])
 require=copy.deepcopy(require);require['files']=[f for f in require['files'] if f['path']=='require/require.go']
 yield public('public-sixteen-mixed',[fmt,mock,assertion,doc,ci,require],tags=['9-16','16-file','multiple-independent','controlled-public-superposition'])
 yield public('doc-only',[doc],tags=['single-file','documentation'])
 yield public('ci-only',[ci],tags=['single-file','unsupported-language-control'])
 for n,pairs in [(6,False),(16,True),(16,False)]:
  files=[];purpose={}
  for i in range(n):
   key=i//2 if pairs else i;path=('source' if i%2==0 else 'test')+f'/limit{i}.txt'
   f=dict(make(path,f'limit{key}=10\n',f'limit{key}=20\n'),id=f'F{i+1:03}');files.append(f);purpose[f['id']]=key
  yield dict(name=f'authored-{n}-'+('pairs' if pairs else 'independent'),files=files,gold_file_purposes=purpose,metric_granularity='line-operation',primary=True,gold_status='new controlled independent limit settings; pairs have corresponding changed assertions by construction',tags=['negative' if not pairs else 'source-test','cross-dir','16-file' if n==16 else '5-8'],source_commits=[],superposition=False,correlated_family='new-authored-boundary-controls')
 b='const maxUploadMB=8; const telemetryOn=false;\n';a=b.replace('8','16');c=b.replace('false','true');after=a.replace('false','true')
 yield controlled('d-same-line',[make('gateway/limits.ts',b,after)],{'upload':[a],'telemetry':[c]},['mixed-file','same-line','independent'])
 src1='func Read() int { return 1 }\n';dst1=src1.replace('1','2');src2='func Flush() int { return 5 }\n';dst2=src2.replace('5','8');test='assert(Read()==1)\nassert(Flush()==5)\n';ta=test.replace('1','2');tb=test.replace('5','8');aftertest=ta.replace('5','8')
 yield controlled('d-shared-test',[make('engine/read.go',src1,dst1),make('engine/flush.go',src2,dst2),make('engine/engine_test.go',test,aftertest)],{'read':[dst1,src2,ta],'flush':[src1,dst2,tb]},['shared-test','source-test','same-dir-independent','partial-cross-file','mixed-file'])
 for name,before,after,tags in [('opaque-unknown',b'alpha=17\n',b'alpha=29\n',['weak-missing-relation','unknown']),('nonutf8-unknown',b'\xff\x00\n',b'\xfe\x00\n',['nonutf8','unknown']),('interleaved-purpose',b'code=A\n',b'code=B\n',['interleaved','indivisible-operation','unknown'])]:
  files=[dict(make('opaque/'+name+'.txt',before,after),id='F001')]
  yield dict(name=name,files=files,primary=False,gold_status='unobservable purpose from source alone; interleaved hypothetical purposes cannot each own the same changed byte; no fabricated intent gold',tags=tags,source_commits=[],metric_granularity='line-operation',correlated_family=name)
