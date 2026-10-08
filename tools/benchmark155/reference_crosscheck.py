"""Bounded hand-authored reference model, not execution or interpretation of Go."""
import json
from attribution import ROOT
OUT=ROOT/'docs/benchmarks/issue-155'

def camel(text,init_case,trim,repeated_capitals,acronyms):
    if acronyms is None:raise ValueError('state_unknown')
    if not text.isascii() or len(text)>64:raise ValueError('reference_scope')
    if trim:text=text.strip(' \t\n\r\v\f')
    if not text:return '',[]
    has=text in acronyms
    if has:text=acronyms[text]
    if not text.isascii() or len(text)>64:raise ValueError('reference_scope')
    output=[];cap_next=init_case;previous_capital=False;trace=[]
    for index,original in enumerate(text):
        capital='A'<=original<='Z';lower='a'<=original<='z';value=original;branch='keep'
        if cap_next:
            if lower:value=chr(ord(original)-32);branch='capitalize'
        elif index==0:
            if capital:value=chr(ord(original)+32);branch='lower_initial'
        elif repeated_capitals and previous_capital and capital and not has:value=chr(ord(original)+32);branch='lower_repeated_capital'
        previous_capital=capital
        if capital or lower:output.append(value);cap_next=False
        elif '0'<=original<='9':output.append(value);cap_next=True
        else:cap_next=original in '_ -.'
        trace.append(dict(index=index,input=original,branch=branch,output=''.join(output),cap_next=cap_next))
    return ''.join(output),trace

if __name__=='__main__':
    source=json.loads((OUT/'iteration-15-effect-reference.json').read_text())['rows'];rows=[]
    for r in source:
        function=r['call']['text'].split('(')[0];text=json.loads(r['input']['text'])
        if function=='ToSnake':
            rows.append(dict(commit=r['commit'],input=text,function=function,status='effect_only_unchanged_source_path',value_checked=False,runtime_measurement=None));continue
        newer=r['commit'].startswith('d799');init=function=='ToCamel'
        before,tb=camel(text,init,newer,False,{'ID':'id'});after,ta=camel(text,init,True,newer,{'ID':'id'})
        assert (before,after)==(r['reference_return_before'],r['reference_return_after'])
        rows.append(dict(commit=r['commit'],input=text,function=function,before=before,after=after,trace_before=tb,trace_after=ta,status='bounded_reference_agrees',runtime_measurement=None))
    # Explicit constructed-state counterexamples, never natural intent holdout.
    controls=[]
    for text,init,trim,repeated,state in [('CONSTANT_CASE',True,True,True,{'CONSTANT_CASE':'same'}),('CONSTANT_CASE',False,True,True,{'CONSTANT_CASE':'same'}),(' some string',False,False,False,{' some string':'same','some string':'same'})]:
        before,_=camel(text,init,trim,False,state);after,_=camel(text,init,True,repeated,state);assert before==after
        controls.append(dict(input=text,init_case=init,state=state,before=before,after=after,origin='constructed_state_counterexample',meaning='previous default-state changed reference becomes unchanged; effect is not determined if state is unspecified'))
    result=dict(model_calls=0,scope='hand-authored ASCII reference model, bounded64 bytes, explicit map; not an independent Go runtime oracle',rows=rows,state_counterexamples=controls)
    with (OUT/'iteration-16-reference-crosscheck.json').open('x') as f:json.dump(result,f,ensure_ascii=False,indent=2);f.write('\n')
    print('reference values matched',sum(r['status']=='bounded_reference_agrees' for r in rows),'effect-only paths',sum(not r.get('value_checked',True) for r in rows),'constructed state controls',len(controls))
