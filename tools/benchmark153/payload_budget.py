"""Compact stage-one proposals; byte guard is not an exact tokenizer."""
import base64,json,pathlib,sys
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent))
import refinement
MAX_MESSAGE_BYTES=8192

def build(snapshots):
 proposals=[];mapping={}
 for fid,(before,after) in sorted(snapshots.items()):
  h=refinement.contiguous.extract(before,after,fid)
  for p in h['operations']:
   alias=f'P{len(proposals)+1:03}';mapping[alias]=p['id']
   proposals.append(dict(id=alias,file=fid,before=base64.b64decode(p['before']).decode('utf8'),after=base64.b64decode(p['after']).decode('utf8')))
 if not 1<=len(proposals)<=8:raise ValueError('proposal_budget')
 schema=dict(type='object',properties={p['id']:dict(type='string',enum=['accept','refine','unresolved']) for p in proposals},required=[p['id'] for p in proposals],additionalProperties=False)
 system='For each proposal decide: accept if all changes serve one purpose; refine if independent purposes need internal splitting; unresolved if insufficient information. Proximity alone is not shared purpose. Output JSON matching: '+json.dumps(schema,sort_keys=True)
 messages=[dict(role='system',content=system),dict(role='user',content=json.dumps(dict(proposals=proposals),sort_keys=True))]
 size=len(json.dumps(messages,ensure_ascii=False).encode('utf8'))
 if size>MAX_MESSAGE_BYTES:raise ValueError('message_byte_budget')
 return messages,schema,mapping,size

def validate_decisions(answer,mapping):
 if not isinstance(answer,dict) or set(answer)!=set(mapping) or any(type(v) is not str or v not in ('accept','refine','unresolved') for v in answer.values()):raise ValueError('invalid_decisions')
 if 'unresolved' in answer.values():raise ValueError('unresolved')
 return {mapping[k]:v for k,v in answer.items()}
