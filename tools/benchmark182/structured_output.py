#!/usr/bin/env python3
"""Prepare the pinned Qwen study and proxy real phase requests; no inference in prepare/audit.

Raw requests/audits remain in the caller's private output directory, never in Git.
"""
import argparse, hashlib, json, os, pathlib, subprocess, sys
ROOT = pathlib.Path(__file__).resolve().parents[2]
PINS = [
 ('qwen25-05', 'Qwen2.5-0.5B-Instruct-4bit', 'a5339a4131f135d0fdc6a5c8b5bbed2753bbe0f3', 24),
 ('qwen25-15', 'Qwen2.5-1.5B-Instruct-4bit', '8b403126fc14f14cfc99bb4cfa72ecbc129ea677', 28),
 ('qwen25-3', 'Qwen2.5-3B-Instruct-4bit', '4f83f8f146fdf28b512a06562b671d7af4fab457', 36),
 ('coder25-05', 'Qwen2.5-Coder-0.5B-Instruct-4bit', '6b16732e5af5cd9bd600186ad59fa618867ef7a4', 24),
 ('coder25-15', 'Qwen2.5-Coder-1.5B-Instruct-4bit', 'b3252a2f97102b1fb1571fec2c9b27219a8536be', 28),
 ('coder25-3', 'Qwen2.5-Coder-3B-Instruct-4bit', '3dd939c621c08e5753d5b89f35a2642cd83b98ca', 36),
]

def write(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n')
    path.chmod(0o600)

def prepare(out):
    out.mkdir(mode=0o700, parents=True, exist_ok=False)
    configs = []
    for key, name, revision, layers in PINS:
        configs.append(dict(id=key, model=f'mlx-community/{name}@{revision}', family='qwen', thinking=False,
                            thinkingBudget=None, temperature=0.7, topP=0.8, topK=20,
                            attentionLayers=layers, otherLayers=0))
    for key in ['qwen2-off', 'qwen4-off', 'qwen4-on']:
        config = json.loads((ROOT/f'docs/benchmarks/issue-182/suite-v2/{key}.json').read_text())
        configs.append(config)
    rows=[]
    for config in configs:
        for mode in ['json_schema', 'none']:
            value = dict(config, output_constraint=mode)
            name = f'{config["id"]}-{mode}.json'
            write(out/name, value)
            rows.append(dict(case=config['id'], output_constraint=mode, config=name))
    write(out/'matrix.json', dict(model_count=8, model_thinking_cases=9, constraint_cases=18, cases=rows,
        generation_calls=0, note='Qwen2.5 sampling 0.7/0.8/20 is a preparation default, not an approved comparison condition. No new thinking/output budget or timeout adopted. Qwen3.5 retains prior sampling.'))

def environment(config):
    env={k:v for k,v in os.environ.items() if not k.startswith(('COMMITER_180_', 'COMMITER_182_'))}
    env.update(COMMITER_182_CASE_CONFIG=str(config.resolve()), COMMITER_182_PIPELINE='eager',
               COMMITER_182_PROGRESS='1', COMMITER_180_KV_BITS='8')
    return env

def audit(out, helper, requests):
    # Real captured current schemas and prompts, with no model loading or generation.
    rows=[]
    for case in json.loads((out/'matrix.json').read_text())['cases']:
        config=out/case['config']
        for phase in ['grouping','category','text']:
            request=json.loads((requests/f'bounded-{phase}.json').read_text())
            proc=subprocess.run([str(helper),'--benchmark-output-contract'], input=json.dumps(request)+'\n',
                                text=True,capture_output=True,env=environment(config),check=True)
            actual=json.loads(proc.stdout)
            assert actual['schema']==request['schema'] and actual['generation_calls']==0
            assert actual['output_constraint']==case['output_constraint']
            systems=[m['content'] for m in actual['messages'] if m['role']=='system']
            assert any('Structural example only' in s and 'required/optional' in s for s in systems)
            # Both modes receive byte-identical prompts. Only runtime token constraints differ.
            digest=hashlib.sha256(json.dumps(actual['messages'],sort_keys=True).encode()).hexdigest()
            rows.append(dict(case=case['case'],mode=case['output_constraint'],phase=phase,prompt_sha256=digest))
    by={ (r['case'],r['mode'],r['phase']):r['prompt_sha256'] for r in rows }
    for case in {r['case'] for r in rows}:
        for phase in ['grouping','category','text']:
            assert by[case,'none',phase]==by[case,'json_schema',phase]
    write(out/'contract-audit.json',dict(checks=rows,phase_count=len(rows),generation_calls=0,model_loads=0,
        exact_schema_preserved=True,paired_prompts_identical=True))

def proxy(helper, config, model_path):
    # Called by mlx.Client, once per actual phase. Host validators remain authoritative.
    value=json.loads(config.read_text())
    request=json.load(sys.stdin)
    if request.get('generation_profile') not in ['bounded-grouping','bounded-category','bounded-text']:
        raise ValueError('unexpected generation profile')
    manifest=json.loads((model_path/'asset-manifest.json').read_text())
    if manifest['model'] != value['model']:
        raise ValueError('model revision does not match the prepared assets')
    request.update(model=value['model'],model_path=str(model_path.resolve()),output_tokens=0,
                   context_tokens=min(request['context_tokens'],manifest['max_context']))
    env=environment(config)
    proc=subprocess.run([str(helper),'--benchmark-single'],input=json.dumps(request)+'\n',text=True,env=env)
    raise SystemExit(proc.returncode)

if __name__=='__main__':
    if len(sys.argv)==1:
        proxy(pathlib.Path(os.environ['COMMITER_STUDY_HELPER']).resolve(),
              pathlib.Path(os.environ['COMMITER_STUDY_CONFIG']).resolve(),
              pathlib.Path(os.environ['COMMITER_STUDY_MODEL_PATH']).resolve())
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('action',choices=['prepare','audit','proxy'])
    p.add_argument('--out',type=pathlib.Path)
    p.add_argument('--helper',type=pathlib.Path)
    p.add_argument('--requests',type=pathlib.Path)
    p.add_argument('--config',type=pathlib.Path)
    p.add_argument('--model-path',type=pathlib.Path)
    a=p.parse_args()
    if a.action=='prepare': prepare(a.out)
    elif a.action=='audit': audit(a.out,a.helper.resolve(),a.requests)
    else: proxy(a.helper.resolve(),a.config,a.model_path)
