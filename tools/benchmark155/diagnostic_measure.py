"""One frozen diagnostic comparison; these cases cannot establish holdout GO."""
import argparse
import hashlib
import itertools
import json
import os
import pathlib
import subprocess
import time
from attribution import ROOT,prepare,messages,validate,validate_direct,encode
from input_audit import records

OUT=ROOT/'docs/benchmarks/issue-155'
HELPER=pathlib.Path('/tmp/issue153-neutral-score-helper/.build/release/commiter-mlx-helper')
HELPER_SHA='bc61461f860957780394650f8b46fab2b737f9931917d0194a0f736e671b41a4'
MODEL='mlx-community/Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192'
MODEL_PATH='/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8'

def sha(raw):return hashlib.sha256(raw).hexdigest()
def strict(pairs):
    result={}
    for k,v in pairs:
        if k in result:raise ValueError('duplicate_json_key')
        result[k]=v
    return result

def select_cases():
    used=json.loads((ROOT/'tools/benchmark153/policy-fixtures.json').read_text())
    names=['fee-rounding-and-wrap','shared-test-independent-assertions','same-call-independent-diagnostic']
    cases=[r for name in names for r in used if r['name']==name]
    for r in cases:r['split']='used_diagnostic'
    cases.append(next(r for r in records() if r['name']=='no-test-observation'))
    return cases

def gold(case,payload):
    files={f['id']:f for f in case['files']}
    purposes={}
    for u in payload['change_units']:
        f=files[u['file']]
        if 'eval_line_intents' in f:
            line=f['before'].encode()[:u['old_span'][0]].count(b'\n')+1
            purposes[u['id']]=f['eval_line_intents'][str(line)]
        else:purposes[u['id']]=f.get('eval_intent',0)
    anchor_purpose={}
    for a in payload['observations']:
        f=files[a['file']]
        line=f['after'].encode()[:a['span'][0]].count(b'\n')+1
        anchor_purpose[a['id']]=f['eval_line_intents'][str(line)] if 'eval_line_intents' in f else f.get('eval_intent',0)
    expected={}
    for u in payload['change_units']:
        if case['name']=='no-test-observation':role,contracts='unknown',[]
        elif case['name']=='same-call-independent-diagnostic':role,contracts='independent',[]
        else:
            role='asserts_changed_behavior' if files[u['file']]['path'].endswith('_test.go') else 'implements_changed_behavior'
            contracts=[cid for cid,purpose in anchor_purpose.items() if purpose==purposes[u['id']]]
        expected[u['id']]=dict(role=role,contracts=contracts)
    return dict(attribution=expected,partition=purposes,rationale=(
        'Fee ceiling versus Wrap delimiters are independently changed and verified; shared-test Left increment versus Right multiplier have distinct changed assertions; Count(3) stays 3 before/after while only negative inputs and diagnostic wording change; no-test Value has no supplied observation. This gold describes local attribution to supplied observations, not arbitrary author intent.'))

def golden_output(payload,expected):
    buckets={}
    for uid,value in expected.items():buckets.setdefault((value['role'],tuple(value['contracts'])),[]).append(uid)
    refs=[e['id'] for e in payload['source_context']]
    rows=[]
    for (role,contracts),units in sorted(buckets.items()):
        reason={'unknown':'no_supported_observation','independent':'independent_of_supplied_observations'}.get(role,'direct_source_evidence')
        rows.append(dict(role=role,unit_ids=units,observed_contract_ids=list(contracts),evidence_refs=[] if role=='unknown' else refs,reason=reason))
    return dict(assignments=rows)

def preflight():
    assert sha(HELPER.read_bytes())==HELPER_SHA
    rows=[];wire=[]
    for case in select_cases():
        files=[{k:f[k] for k in ('id','path','before','after')} for f in case['files']]
        payload,manifest=prepare(files);expected=gold(case,payload)
        msg_a,sch_a=messages(payload,'attribution');msg_d,sch_d=messages(payload,'direct')
        good=golden_output(payload,expected['attribution']);validate(payload,good)
        groups=[]
        for purpose in sorted(set(expected['partition'].values())):groups.append([uid for uid,p in expected['partition'].items() if p==purpose])
        direct=dict(groups=groups,unresolved=False);validate_direct(payload,direct)
        row=dict(name=case['name'],split=case['split'],payload=payload,manifest=manifest,gold=expected,payload_sha256=sha(encode(payload)),message_bytes={c:len(encode(m)) for c,m in [('attribution',msg_a),('direct',msg_d)]})
        rows.append(row)
        # Legal role alternatives are schema vectors only, not scored outputs.
        unknown=dict(assignments=[dict(role='unknown',unit_ids=[u['id'] for u in payload['change_units']],observed_contract_ids=[],evidence_refs=[],reason='ambiguous_or_missing_context')])
        invalid=json.loads(json.dumps(good));invalid['assignments'][0]['role']='invented_role'
        for condition,msg,sch,legal in [('attribution',msg_a,sch_a,[good,unknown]),('direct',msg_d,sch_d,[direct,dict(groups=[[u['id'] for u in payload['change_units']]],unresolved=True)])]:
            bad=invalid if condition=='attribution' else dict(groups=[['fake']],unresolved=False)
            wire.append(dict(name=case['name']+':'+condition,schema=encode(sch).decode(),messages=msg,legal=[encode(x).decode() for x in legal],illegal=[encode(bad).decode()]))
    contract=dict(model=MODEL,helper_sha256=HELPER_SHA,generation_profile='bounded-routed-grammar-neutral',context_tokens=16384,output_tokens=1536,native_thought_tokens=0,temperature=0,top_p=1,top_k=0,seed=144,per_call_seconds=120,total_seconds=960,max_calls=8,retries=0,repairs=0,max_message_bytes=32768,max_units=128,
        purpose='Diagnostic attribution/host contract contrast, not independent holdout or production GO',
        diagnostic_gate='all attribution responses host-valid; all known attribution role/contract labels exact; unknown case explicitly unknown; all positive/negative relations correct. Failures reject this diagnostic candidate; passing only permits independent holdout.',
        full_go='not eligible: holdout absent; cross-boundary and larger-range evaluation unproven',
        conditions='same payload and local model/helper/budget; different output tasks; no wording/threshold/model sweep',
        source_sha256={p.name:sha(p.read_bytes()) for p in [ROOT/'tools/benchmark155/attribution.py',pathlib.Path(__file__)]})
    result=dict(contract=contract,rows=rows)
    for path,value in [(OUT/'iteration-3-diagnostic-preflight.json',result),(OUT/'iteration-3-wire-input.json',wire)]:
        with path.open('x') as stream:json.dump(value,stream,ensure_ascii=False,indent=2);stream.write('\n')
    print(json.dumps([dict(name=r['name'],units=len(r['payload']['change_units']),anchors=len(r['payload']['observations']),message_bytes=r['message_bytes']) for r in rows],indent=2))

def quality(expected,answer):
    roles=sum(answer[u]['role']!=g['role'] for u,g in expected.items())
    edges={(u,c) for u,v in answer.items() for c in v['contracts']}
    truth={(u,c) for u,v in expected.items() for c in v['contracts']}
    unknown=sum(v['role']=='unknown' for v in answer.values())
    independent=sum(v['role']=='independent' for v in answer.values())
    return dict(exact=roles==0 and edges==truth,role_errors=roles,false_positive_relations=len(edges-truth),false_negative_relations=len(truth-edges),true_positive_relations=len(edges&truth),expected_positive_relations=len(truth),unknown_units=unknown,independent_units=independent,unit_coverage=len(answer))

def run():
    fixed=json.loads((OUT/'iteration-3-diagnostic-preflight.json').read_text())
    assert sha(HELPER.read_bytes())==HELPER_SHA
    for name,digest in fixed['contract']['source_sha256'].items():assert sha((ROOT/'tools/benchmark155'/name).read_bytes())==digest
    wire=json.loads((OUT/'iteration-3-wire-audit.json').read_text())
    assert wire['model_calls']==0 and wire['legal_accepted']==16 and wire['illegal_rejected']==8 and wire['zero_penalty_replays']==16 and wire['all_prompt_tokens_fit']
    deadline=time.monotonic()+960
    with (OUT/'iteration-3-results.jsonl').open('x') as stream:
        for index,case in enumerate(fixed['rows']):
            for condition in (['attribution','direct'] if index%2==0 else ['direct','attribution']):
                payload=case['payload'];msg,sch=messages(payload,condition)
                row=dict(name=case['name'],split=case['split'],condition=condition,payload_sha256=case['payload_sha256'],schema_sha256=sha(encode(sch)),message_sha256=sha(encode(msg)),calls=1,backend_complete=False,host_valid=False,quality=None)
                req=dict(schema=sch,messages=msg,context_tokens=16384,output_tokens=1536,model=MODEL,model_path=MODEL_PATH,generation_profile='bounded-routed-grammar-neutral')
                started=time.perf_counter()
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
                    if condition=='attribution':
                        resolved=validate(payload,answer);metrics=quality(case['gold']['attribution'],resolved)
                    else:
                        resolved=validate_direct(payload,answer);g=case['gold']['partition'];pairs=list(itertools.combinations(g,2))
                        fm=sum(resolved[a]==resolved[b] and g[a]!=g[b] for a,b in pairs);fs=sum(resolved[a]!=resolved[b] and g[a]==g[b] for a,b in pairs)
                        metrics=dict(exact=fm==fs==0,false_merge=fm,false_split=fs,unit_coverage=len(resolved))
                    row.update(host_valid=True,quality=metrics,reason='accepted')
                except subprocess.TimeoutExpired:row.update(reason='timeout',stop='timeout')
                except (ValueError,KeyError,TypeError) as exc:row['reason']=str(exc)
                row['wall_seconds']=time.perf_counter()-started
                stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush()
                print(json.dumps({k:v for k,v in row.items() if k!='answer'},ensure_ascii=False),flush=True)

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('mode',choices=['preflight','run']);args=parser.parse_args()
    preflight() if args.mode=='preflight' else run()
