"""Opt-in Spark mlx-lm adapter. Local pinned assets only; no planner/schema changes."""
import json,os,sys,time,traceback
from pathlib import Path
import mlx.core as mx
import numpy as np
from mlx_lm import load,stream_generate
from mlx_lm.sample_utils import make_sampler
from llguidance import LLTokenizer,LLMatcher
from llguidance.numpy import allocate_token_bitmask,fill_next_token_bitmask
from llguidance.mlx import apply_token_bitmask

class BoundaryProcessor:
    def __init__(self,prompt_len,thinking,matcher,vocab=131072):
        self.seen=prompt_len;self.thinking=thinking;self.matcher=matcher
        self.mask=allocate_token_bitmask(1,vocab);self.timings={'state_update_seconds':0.,'mask_seconds':0.}
    def __call__(self,tokens,logits):
        start=time.perf_counter()
        if tokens.size>self.seen:
            for token in tokens[self.seen:].tolist():
                if self.thinking:
                    if token==4:self.thinking=False
                    elif token==1:raise ValueError('eos_before_thinking_close')
                elif self.matcher is not None:
                    if not self.matcher.consume_token(token):raise ValueError('grammar_token_rejected')
            self.seen=tokens.size
        self.timings['state_update_seconds']+=time.perf_counter()-start
        if self.thinking or self.matcher is None:return logits
        start=time.perf_counter();fill_next_token_bitmask(self.matcher,self.mask)
        if self.matcher.is_error():raise ValueError('grammar_mask_error: '+self.matcher.get_error())
        result=apply_token_bitmask(logits,self.mask);self.timings['mask_seconds']+=time.perf_counter()-start
        return result

def output_contract(request):
    # Reuse the exact field/type/schema instruction generator; it performs no model call.
    import subprocess
    helper=os.environ['COMMITER_SPARK_CONTRACT_HELPER'];env=dict(os.environ)
    env['COMMITER_182_CASE_CONFIG']=os.environ['COMMITER_SPARK_CONTRACT_CONFIG'];env['COMMITER_182_NEUTRAL_EXAMPLES']='1'
    result=json.loads(subprocess.check_output([helper,'--benchmark-output-contract'],input=json.dumps(request).encode()+b'\n',env=env))
    assert result['generation_calls']==0
    return result['messages']

def run(request):
    total=time.perf_counter();conf=json.loads(Path(os.environ['COMMITER_182_CASE_CONFIG']).read_text());metrics={};tokens=[];response='';reason='runtime_error';error=None
    try:
        mx.random.seed(144);start=time.perf_counter();model,tokenizer=load(request['model_path'],trust_remote_code=True);mx.synchronize();metrics['loadSeconds']=time.perf_counter()-start
        start=time.perf_counter();messages=output_contract(request);kwargs={'enable_thinking':conf['thinking']}
        if conf['thinking']:kwargs['reasoning_effort']=conf['thinking_effort']
        prompt=tokenizer.apply_chat_template(messages,tokenize=True,add_generation_prompt=True,**kwargs)
        assert prompt[-1]==(3 if conf['thinking'] else 4)
        assert tokenizer.eos_token_ids=={1} or set(tokenizer.eos_token_ids)=={1}
        if len(prompt)+request['output_tokens']>request['context_tokens']:raise ValueError('input_too_large')
        metrics['prepareSeconds']=time.perf_counter()-start;metrics['inputTokens']=len(prompt)
        start=time.perf_counter();matcher=None
        if conf['output_constraint']=='json_schema':
            vocab=LLTokenizer(Path(request['model_path'],'tokenizer.json').read_text(),n_vocab=131072,eos_token=1)
            matcher=LLMatcher(vocab,LLMatcher.grammar_from_json_schema(request['schema']),log_level=0)
            if matcher.is_error():raise ValueError(matcher.get_error())
        processor=BoundaryProcessor(len(prompt),conf['thinking'],matcher)
        metrics['grammarPreparationSeconds']=metrics['grammarSetupSeconds']=time.perf_counter()-start
        cache=model.make_cache();metrics['initial_cache_types']=[type(c).__name__ for c in cache]
        start=time.perf_counter();last=None
        for item in stream_generate(model,tokenizer,prompt,max_tokens=request['output_tokens'],sampler=make_sampler(temp=1.0,top_p=.95,top_k=0),logits_processors=[processor],prompt_cache=cache):
            last=item
            if item.token!=1:tokens.append(item.token)
            if len(tokens)==1 or len(tokens)%64==0:print(json.dumps({'event':'generation_progress','tokens':len(tokens),'seconds_including_prefill':time.perf_counter()-start}),file=sys.stderr,flush=True)
        mx.synchronize();metrics['generationWallSeconds']=time.perf_counter()-start
        if last is None:raise ValueError('no_generation_result')
        metrics.update(runtimePromptSeconds=last.prompt_tokens/last.prompt_tps,runtimeGenerationSeconds=last.generation_tokens/last.generation_tps,generatedTokens=len(tokens),libraryGenerationTokens=last.generation_tokens,libraryPrefillTokensPerSecond=last.prompt_tps,libraryDecodeTokensPerSecond=last.generation_tps)
        final=[];thought=[];markers=[];in_thought=conf['thinking'];closed=not in_thought
        for token in tokens:
            if in_thought and token==4:markers.append(token);in_thought=False;closed=True
            elif in_thought:thought.append(token)
            else:final.append(token)
        metrics.update(nativeThoughtTokens=len(thought),thinkingBoundaryTokens=len(markers),finalAnswerTokens=len(final),nativeThoughtForcedClose=False,thinkingClosed=closed,diagnosticCPUSeconds=processor.timings)
        response=tokenizer.decode(final)
        if last.finish_reason=='length':reason='max_tokens'
        elif not closed:reason='native_eos_before_close'
        else:
            # Schema validation/contract checks remain in Go; this only requires valid JSON.
            json.loads(response);reason='completed'
        snapshots=[]
        for i,c in enumerate(cache):
            state=[a for a in c.state if isinstance(a,mx.array)];snapshots.append({'index':i,'nativeType':type(c).__name__,'kind':'native','bytes':sum(a.nbytes for a in state),'arrays':[{'shape':list(a.shape),'dtype':str(a.dtype),'bytes':a.nbytes} for a in state]})
        metrics['phaseEndKVLayers']=snapshots
        metrics['cacheMemoryBytes']=mx.get_cache_memory();metrics['activeMemoryBytes']=mx.get_active_memory();metrics['peakMemoryBytes']=mx.get_peak_memory()
    except Exception as e:
        reason='runtime_error';error=type(e).__name__+': '+str(e);traceback.print_exc(file=sys.stderr)
    metrics['adapterWallSeconds']=time.perf_counter()-total
    metrics['thinkingEnabled']=conf['thinking'];metrics['thinkingEffort']=conf.get('thinking_effort');metrics['runtimeEngine']='Python mlx-lm 0.32.2';metrics['outputConstraint']=conf['output_constraint'];metrics['benchmarkCacheMode']='native';metrics['modelReused']=False;metrics['cacheReusedTokens']=0
    return {'ok':reason=='completed','model':request.get('model'),'generation_profile':request.get('generation_profile'),'stop_reason':reason,'generated_json':response if reason=='completed' else '', 'benchmark_metrics':metrics,'adapter_error':error}
if __name__=='__main__':
    request=json.loads(sys.stdin.buffer.readline());print(json.dumps(run(request),ensure_ascii=False),flush=True)
