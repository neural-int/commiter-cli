"""Prepare local tokenizer/grammar replay vectors; never load model weights."""
import hashlib, json, pathlib
import policy_ablation as policy
ROOT=policy.ROOT
fixtures=json.loads((policy.HERE/'policy-fixtures.json').read_text())
measured=[json.loads(x) for x in (ROOT/'iteration-21-results.jsonl').read_text().splitlines()]
cases=[]
for record in fixtures:
 data,*_=policy.prepare(record)
 original,schema=policy.messages(data,'original');changed,changed_schema=policy.messages(data,'policy')
 assert schema==changed_schema
 tokens={r['condition']:r['input_tokens'] for r in measured if r['fixture']==record['name']}
 pairs=list(schema['properties']['scores']['properties'])
 vectors=[]
 for value in [-2,-1,0,1,2]:
  vectors.append(dict(legal=True,score=value,output=json.dumps(dict(scores={p:value for p in pairs},unresolved=False))))
 vectors.append(dict(legal=True,score=99,output=json.dumps(dict(scores={p:(2 if i%2 else -2) for i,p in enumerate(pairs)},unresolved=True))))
 for value in [-3,3]:
  vectors.append(dict(legal=False,score=value,output=json.dumps(dict(scores={p:value for p in pairs},unresolved=False))))
 cases.append(dict(name=record['name'],schema=json.dumps(schema,sort_keys=True),original=original,policy=changed,expected_original_tokens=tokens['original'],expected_policy_tokens=tokens['policy'],criterion=policy.POLICY.strip(),vectors=vectors))
output=pathlib.Path('/tmp/issue153-score-wire-input.json')
output.write_text(json.dumps(cases,indent=2)+'\n')
print(json.dumps(dict(cases=len(cases),legal_vectors=sum(v['legal'] for c in cases for v in c['vectors']),illegal_vectors=sum(not v['legal'] for c in cases for v in c['vectors']),input_sha256=hashlib.sha256(output.read_bytes()).hexdigest(),model_calls=0)))
