"""One frozen local-fact capability probe; no global grouping stage."""
import argparse
import base64
import hashlib
import json
import pathlib
import subprocess
import time
from attribution import ROOT
from context import encode
from local_facts import prepare,messages,validate,FACTUAL
from diagnostic_measure import HELPER,HELPER_SHA,MODEL,MODEL_PATH,strict

OUT=ROOT/'docs/benchmarks/issue-155'

def sha(raw):return hashlib.sha256(raw).hexdigest()
def file_slice(record):return [{k:f[k] for k in ('id','path','before','after')} for f in record['files']]
def q(function,*args):return dict(package='.',function=function,arguments=list(args))

def cases():
    used=json.loads((ROOT/'tools/benchmark153/policy-fixtures.json').read_text())
    by_name={r['name']:r for r in used}
    rows=[dict(name='fee-wrap',split='used_diagnostic',files=file_slice(by_name['fee-rounding-and-wrap']),queries=[q('Fee',101),q('Wrap','a')],query_origins=['observed_test_call','observed_test_call']),
          dict(name='shared-left-right',split='used_diagnostic',files=file_slice(by_name['shared-test-independent-assertions']),queries=[q('Left',2),q('Right',2)],query_origins=['observed_test_call','observed_test_call']),
          dict(name='count-observed-and-control',split='used_diagnostic_plus_control',files=file_slice(by_name['same-call-independent-diagnostic']),queries=[q('Count',3),q('Count',-1)],query_origins=['observed_test_call','constructed_negative_argument_control']),
          dict(name='no-observation',split='development_scope_probe',files=[dict(id='F001',path='value.go',before='package scope\nfunc Value(n int) int {return n}\n',after='package scope\nfunc Value(n int) int {return n+1}\n')],queries=[None],query_origins=['no_test_observation']),
          dict(name='missing-callee',split='development_scope_probe',files=[dict(id='F001',path='opaque.go',before='package scope\nfunc Opaque(n int) int {return Delegate(n)}\n',after='package scope\nfunc Opaque(n int) int {return Delegate(n)+1}\n')],queries=[q('Opaque',3)],query_origins=['missing_source_control'])]
    public=json.loads(pathlib.Path('/tmp/issue155-public-candidates.json').read_text())[0]
    typefile={}
    for version in ('before','after'):
        response=json.loads(pathlib.Path('/tmp/issue155-uuid-type-'+version+'.json').read_text())
        typefile[version]=base64.b64decode(response['content']).decode()
    files=[public['files'][0],dict(id='F003',path='uuid.go',**typefile)]
    low=[0]*15+[1];high=[0]*15+[2]
    rows.append(dict(name='public-uuid-compare',split='real_source_constructed_argument_probe',files=files,queries=[q('Compare',low,high),q('Compare',high,high),q('Compare',high,low)],query_origins=['constructed_lexicographic_inputs']*3,repository=public['repository'],commit=public['commit'],parent=public['parent']))
    return rows

def preflight():
    assert sha(HELPER.read_bytes())==HELPER_SHA
    fixed=[];wire=[]
    for record in cases():
        payload,oracle=prepare(record['files'],record['queries'])
        msg,contract=messages(payload)
        validate(payload,oracle,oracle)
        unknown={qid:{v:dict(known=False,kind='unknown',value=None,return_refs=[]) for v in ('before','after')} for qid in payload['queries']}
        bad=json.loads(json.dumps(oracle));first=next(iter(bad));bad[first]['before']['kind']='invented_kind'
        wire.append(dict(name=record['name'],schema=encode(contract).decode(),messages=msg,legal=[encode(oracle).decode(),encode(unknown).decode()],illegal=[encode(bad).decode()]))
        fixed.append(dict(name=record['name'],split=record['split'],payload=payload,oracle=oracle,query_origins=record['query_origins'],payload_sha256=sha(encode(payload)),message_bytes=len(encode(msg)),source_provenance=dict(repository=record.get('repository'),commit=record.get('commit'),parent=record.get('parent'),snapshots=[dict(file=f['id'],path=f['path'],before_sha256=sha(f['before'].encode()),after_sha256=sha(f['after'].encode())) for f in record['files']]),queries=record['queries']))
    protocol=dict(model=MODEL,helper_sha256=HELPER_SHA,factual_observer_sha256=sha(FACTUAL.read_bytes()),context_tokens=16384,output_tokens=1536,generation_profile='bounded-routed-grammar-neutral',native_thought_tokens=0,temperature=0,top_p=1,top_k=0,seed=144,calls=6,per_call_seconds=120,total_seconds=720,retries=0,repairs=0,max_message_bytes=32768,
        gate='all six host-valid: exact source-derived value/type/return path or correctly unknown; this permits a local-fact bridge audit only, not #155 bounded attribution GO',
        exclusions='no final groups, no arbitrary execution, no stage B/C, no natural intent holdout or production adoption',
        responsibility_difference='LLM emits exact values and return witnesses; host verifies facts and may derive local observation effects. #149 H5/H12/H10 text deltas/global grouping are not repeated.',
        source_sha256={p.name:sha(p.read_bytes()) for p in [ROOT/'tools/benchmark155/local_facts.py',pathlib.Path(__file__)]})
    for path,value in [(OUT/'iteration-4-preflight.json',dict(protocol=protocol,rows=fixed)),(OUT/'iteration-4-wire-input.json',wire)]:
        with path.open('x') as stream:json.dump(value,stream,ensure_ascii=False,indent=2);stream.write('\n')
    print(json.dumps([dict(name=r['name'],queries=len(r['queries']),bytes=r['message_bytes'],oracle=r['oracle']) for r in fixed],ensure_ascii=False,indent=2))

def run():
    fixed=json.loads((OUT/'iteration-4-preflight.json').read_text())
    assert sha(HELPER.read_bytes())==HELPER_SHA and sha(FACTUAL.read_bytes())==fixed['protocol']['factual_observer_sha256']
    for name,digest in fixed['protocol']['source_sha256'].items():assert sha((ROOT/'tools/benchmark155'/name).read_bytes())==digest
    audit=json.loads((OUT/'iteration-4-wire-audit.json').read_text())
    assert audit['legal_accepted']==12 and audit['illegal_rejected']==6 and audit['zero_penalty_replays']==12 and audit['all_prompt_tokens_fit'] and audit['model_calls']==0
    deadline=time.monotonic()+720
    with (OUT/'iteration-4-results.jsonl').open('x') as stream:
        for record in fixed['rows']:
            payload=record['payload'];msg,contract=messages(payload)
            req=dict(schema=contract,messages=msg,context_tokens=16384,output_tokens=1536,model=MODEL,model_path=MODEL_PATH,generation_profile='bounded-routed-grammar-neutral')
            row=dict(name=record['name'],split=record['split'],calls=1,backend_complete=False,host_valid=False,quality=None,payload_sha256=record['payload_sha256'],message_sha256=sha(encode(msg)),schema_sha256=sha(encode(contract)))
            start=time.perf_counter()
            try:
                remaining=deadline-time.monotonic()
                if remaining<=0:raise ValueError('total_budget')
                proc=subprocess.run([str(HELPER)],input=encode(req)+b'\n',capture_output=True,timeout=min(120,remaining))
                if proc.returncode:raise ValueError('helper_exit')
                response=json.loads(proc.stdout,object_pairs_hook=strict)
                row.update(stop=response.get('stop_reason'),input_tokens=response.get('benchmark_input_tokens'),output_tokens=response.get('benchmark_output_tokens'))
                if not response.get('ok') or row['stop']!='completed':raise ValueError('backend_not_completed')
                row['backend_complete']=True
                answer=json.loads(response['generated_json'],object_pairs_hook=strict);row['answer']=answer
                validate(payload,answer,record['oracle'])
                known=sum(state['known'] for values in answer.values() for state in values.values())
                row.update(host_valid=True,reason='accepted',quality=dict(exact=True,query_coverage=len(answer),known_states=known,unknown_states=2*len(answer)-known,source_verified=True))
            except subprocess.TimeoutExpired:row.update(stop='timeout',reason='timeout')
            except (ValueError,KeyError,TypeError) as exc:row['reason']=str(exc)
            row['wall_seconds']=time.perf_counter()-start
            stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush()
            print(json.dumps({k:v for k,v in row.items() if k!='answer'},ensure_ascii=False),flush=True)

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('mode',choices=['preflight','run']);args=parser.parse_args()
    preflight() if args.mode=='preflight' else run()
