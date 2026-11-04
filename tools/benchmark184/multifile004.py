"""Adapt retained canonical fixtures to adopted H184-HARNESS-004 display.
No generation, acquisition, history editing, or token-budget reduction.
"""
import copy,json,re
from pathlib import Path
from multiline_view import multiline_view
EXACT='Exact JSON Schema (including enum values, lengths, patterns, and other format constraints):\n'
FIELDS='Field names, types, and required/optional fields:\n'
END='\nChoose values independently from the current Git evidence; enum order is not a preference.'
PHASE_INPUT='\n\nPhase input:\n'
def prune(v):
 if isinstance(v,list):return [prune(x) for x in v]
 if isinstance(v,dict):return {k:prune(x) for k,x in v.items() if k not in {'block_id','execution_ref','user_requests'}}
 return v
def describe(s,path='$',required=True):
 lines=[f'{path}: type={s["type"]}; '+('required' if required else 'optional')]
 if s['type']=='object':
  for key,v in sorted(s['properties'].items(),key=lambda item:natural_key(item[0])):lines+=describe(v,path+'.'+key,key in s.get('required',[]))
 return lines
def natural_key(value):
 return tuple((1,int(p),len(p)) if p.isdigit() else (0,p) for p in re.split(r'(\d+)',value))
def ordered(value):
 if isinstance(value,dict):return {k:ordered(value[k]) for k in sorted(value,key=natural_key)}
 if isinstance(value,list):return [ordered(x) for x in value]
 return value
def load_templates():
 data=json.loads((Path(__file__).resolve().parents[2]/'internal/planning/prompts/harness004.json').read_text())
 return {p:{'messages':[{'role':'system','content':v['system']},{'role':'user','content':'{}'},{'role':'user','content':v['instructions_prefix']+FIELDS+EXACT+'{}'+END+PHASE_INPUT+'{}'}]} for p,v in data.items()}
def adapt(canonical,phase,templates=None,shared_files=None):
 if phase not in ['grouping','category','text']:raise ValueError('phase')
 templates=load_templates() if templates is None else templates
 r=copy.deepcopy(canonical);payload=json.loads(r['messages'][1]['content']);template=templates[phase]
 assert len(template['messages'])==3
 history=payload.pop('agent_history_contexts')
 if phase=='grouping':files=payload['files']
 else:files=[f for g in payload['groups'] for f in g['files']]
 byid={f['file_id']:f for f in files};assert len(byid)==len(files),'duplicate files'
 if shared_files is not None:
  expected={f['file_id']:f for f in shared_files};assert byid==expected,'file evidence differs between phases'
 if phase=='category':shared={'agent_history_contexts':history}
 else:
  shared={'files':sorted(files,key=lambda f:natural_key(f['file_id'])),'agent_history_contexts':history}
  if phase=='grouping':payload['files']=[{'file_id':f['file_id']} for f in payload['files']]
  else:
   for g in payload['groups']:g['files']=[{'file_id':f['file_id']} for f in g['files']]
 payload.pop('task',None);payload.pop('schema_version',None)
 instruction=template['messages'][2]['content'].split(PHASE_INPUT)[0]
 fixed=instruction.split(FIELDS)[0]
 assert fixed.endswith('When thinking is enabled, think freely in the model\'s native thinking section; apply this contract only after that section ends. When thinking is disabled, begin the JSON answer directly.\n')
 instruction=fixed+FIELDS+'\n'.join(describe(r['schema']))+'\n'+EXACT+json.dumps(ordered(r['schema']),ensure_ascii=False,separators=(',',':'))+END
 render=lambda x:multiline_view(json.dumps(ordered(prune(x)),ensure_ascii=False,separators=(',',':')))
 r['messages']=[copy.deepcopy(template['messages'][0]),{'role':'user','content':render(shared)},{'role':'user','content':instruction+PHASE_INPUT+render(payload)}]
 # Every full schema occurs exactly once, plus runtime's separate schema object.
 for key in ['"block_id":','"execution_ref":','"user_requests":[]']:
  assert all(key not in m['content'] for m in r['messages'][1:])
 assert json.loads(r['messages'][1]['content'],strict=False)==prune(shared)
 assert json.loads(r['messages'][2]['content'].split(PHASE_INPUT)[1],strict=False)==prune(payload)
 return r

OLD_SUMMARY_INSTRUCTION = 'Summary must be 1-48 characters; count characters, not tokens or bytes.'
NEW_SUMMARY_INSTRUCTION = ('Write a concrete, self-contained summary within 48 characters.\n'
    'Prioritize the affected behavior and what changed.\n'
    'Omit nonessential identifiers and details while preserving the meaning.\n'
    'Choose wording that fits completely; do not end with an unfinished identifier or phrase.')

def summary_instruction_variant(request):
 """Reproduce the historical Text-only comparison from a revision-1 input."""
 result=copy.deepcopy(request)
 if result.get('generation_profile')!='bounded-text' or len(result['messages'])!=3:raise ValueError('expected 004 Text request')
 instruction,payload=result['messages'][2]['content'].split(PHASE_INPUT,1)
 if instruction.count(OLD_SUMMARY_INSTRUCTION)!=1:raise ValueError('unexpected baseline summary instruction')
 if any(p['properties']['summary'].get('maxLength')!=48 for p in result['schema']['properties'].values()):raise ValueError('expected unchanged 48-character schema')
 result['messages'][2]['content']=instruction.replace(OLD_SUMMARY_INSTRUCTION,NEW_SUMMARY_INSTRUCTION)+PHASE_INPUT+payload
 return result
