"""Reference source reasoning with explicit state assumptions; no code execution."""
import hashlib,json,subprocess
from attribution import ROOT,OBSERVER
OUT=ROOT/'docs/benchmarks/issue-155'
records=json.loads((OUT/'iteration-15-closure-snapshots.json').read_text())
inputs=json.loads((OUT/'iteration-14-input-audit.json').read_text())['rows']
reference={
 ('d7993fd3','ToCamel','"CONSTANT_CASE"'):('CONSTANTCASE','ConstantCase','changed','Repeated capitals are retained before; the added prevIsCap path lowers capitals after the word initial.'),
 ('d7993fd3','ToLowerCamel','"CONSTANT_CASE"'):('cONSTANTCASE','constantCase','changed','Before only the first capital is lowered; after the prevIsCap path lowers later capitals while separator starts the next word.'),
 ('692d1b89','ToLowerCamel','"some string"'):('someString','someString','unchanged','Only the added TrimSpace statement changes; it preserves this input and subsequent state/source are identical.'),
 ('692d1b89','ToLowerCamel','" some string"'):('SomeString','someString','changed','Before the first space sets capNext before the s byte; after trim the s byte is first and initCase is false.'),
 ('692d1b89','ToSnake','"some string"'):('some_string','some_string','unchanged','Snake implementation and reachable package functions are unchanged and do not call the modified Camel implementation.'),
 ('692d1b89','ToSnake','" some string"'):('some_string','some_string','unchanged','Snake trims its input in the unchanged implementation; the Camel edit is outside its package-call path.')}
rows=[]
for r in records:
    facts=json.loads(subprocess.check_output([str(OBSERVER)],input=json.dumps({'files':r['files']}).encode()))
    a=next(x for x in inputs if x['commit']==r['commit'])
    for candidate in a['new_or_changed_source_bindings']:
        b=candidate['binding'];function=b['Call']['text'].split('(')[0];key=(r['commit'][:8],function,b['Input']['text']);before,after,effect,reason=reference[key];assert json.loads(b['Expected']['text'])==after
        root_files=['camel.go','acronyms.go'] if function!='ToSnake' else ['snake.go']
        sources=[dict(path=f['path'],before_sha256=hashlib.sha256(f['before'].encode()).hexdigest(),after_sha256=hashlib.sha256(f['after'].encode()).hexdigest()) for f in r['files'] if f['path']in root_files]
        relevant=[e for e in facts['evidence'] if e['kind']=='function_source' and e['function']in ([function,'toCamelInitCase'] if function!='ToSnake' else ['ToSnake','ToDelimited','ToScreamingDelimited'])]
        for e in relevant:
            source=next(f for f in r['files'] if f['id']==e['file'])[e['version']].encode();start,end=e['span'];assert source[start:end].decode()==e['text']
        rows.append(dict(commit=r['commit'],call=b['Call'],input=b['Input'],expected=b['Expected'],source_refs=relevant,source_hashes=sources,proposed_observation_effect=effect,reference_return_before=before,reference_return_after=after,reason=reason,status='source_reasoned_reference_not_runtime_measured',required_state='For Camel references: initialized uppercaseAcronym map with ID:id, no outside ConfigureAcronym mutation; same initial state in before and after. Snake refers to the unchanged supplied source.',whole_intent_gold=None,runtime_measurement=None))
result=dict(model_calls=0,rows=rows,scope='reference effects on concrete observations, not independent commit purposes; formal acceptance threshold/holdout still not registered')
target=OUT/'iteration-15-effect-reference.json'
if target.exists():assert json.loads(target.read_text())==result
else:target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print('source reasoned reference rows',len(rows),'changed',sum(x['proposed_observation_effect']=='changed' for x in rows),'unchanged',sum(x['proposed_observation_effect']=='unchanged' for x in rows))
