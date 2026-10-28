"""Load pinned models and audit tokenizer/KV only; never run generation."""
import argparse,json,pathlib,os,subprocess
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--study',type=pathlib.Path,required=True)
parser.add_argument('--helper',type=pathlib.Path,required=True)
parser.add_argument('--requests',type=pathlib.Path,required=True)
args=parser.parse_args()
base=args.study.resolve();helper=args.helper.resolve()
assets={r['model']:r for r in json.loads((base/'assets.json').read_text())}
rows=[]
for row in json.loads((base/'matrix.json').read_text())['cases']:
 if row['output_constraint']!='json_schema':continue
 config=base/row['config'];case=json.loads(config.read_text());asset=assets[case['model']]
 requests=[]
 for phase in ['category','text']:
  r=json.loads((args.requests/f'bounded-{phase}.json').read_text())
  r.update(model=case['model'],model_path=asset['path'],output_tokens=0,context_tokens=min(r['context_tokens'],asset['max_context']))
  requests.append(r)
 env={k:v for k,v in os.environ.items() if not k.startswith(('COMMITER_180_','COMMITER_182_'))}
 env.update(COMMITER_182_CASE_CONFIG=str(config),COMMITER_180_KV_BITS='8')
 proc=subprocess.run([str(helper),'--benchmark-token-audit'],input=''.join(json.dumps(r)+'\n' for r in requests),env=env,text=True,capture_output=True)
 dest=base/('tokens-'+case['id']);dest.mkdir(mode=0o700,exist_ok=False)
 (dest/'stdout.jsonl').write_text(proc.stdout);(dest/'stderr.log').write_text(proc.stderr)
 assert proc.returncode==0,(case['id'],'helper exit')
 responses=[json.loads(line) for line in proc.stdout.splitlines()]
 assert len(responses)==2 and all(r['ok'] for r in responses),(case['id'],responses)
 for phase,r,reply in zip(['category','text'],requests,responses):
  audit=json.loads(reply['generated_json']);tokens=audit['token_ids']
  assert len(tokens)<=r['context_tokens'],(case['id'],phase,'input too large')
  assert audit['enable_thinking']==case['thinking']
  assert len(audit['planned_cache'])==case['attentionLayers']+case['otherLayers']
  assert sum(l['kind']=='affine' for l in audit['planned_cache'])==case['attentionLayers']
  rows.append(dict(case=case['id'],phase=phase,tokens=len(tokens),context=r['context_tokens'],cache_layers=len(audit['planned_cache']),thinking=audit['enable_thinking']))
 print(json.dumps({'case':case['id'],'status':'tokenizer/cache ready'}),flush=True)
(base/'model-audit.json').write_text(json.dumps(dict(rows=rows,generation_calls=0,model_cases=9),indent=2)+'\n')
