"""Fetch pinned public model assets without inference; two bounded download workers."""
import argparse,pathlib,json,urllib.request,hashlib,concurrent.futures
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--study',type=pathlib.Path,required=True)
study=parser.parse_args().study
root=study/'models';root.mkdir(mode=0o700,exist_ok=True)
configs={}
for p in study.glob('*.json'):
 d=json.loads(p.read_text())
 if 'model' in d:configs[d['model']]=d

def load(model):
 repo,rev=model.split('@');name=repo.split('/')[-1]
 dest=root/name;dest.mkdir(mode=0o700,exist_ok=True)
 info=json.load(urllib.request.urlopen(f'https://huggingface.co/api/models/{repo}/revision/{rev}?blobs=true',timeout=60))
 rows=[]
 allowed={'config.json','tokenizer.json','tokenizer_config.json','generation_config.json','special_tokens_map.json','added_tokens.json','vocab.json','merges.txt','tokenizer.model','chat_template.jinja','model.safetensors.index.json'}
 for f in info['siblings']:
  fn=f['rfilename']
  if fn not in allowed and not (fn.endswith('.safetensors') and '/' not in fn):continue
  target=dest/fn
  with urllib.request.urlopen(f'https://huggingface.co/{repo}/resolve/{rev}/{fn}',timeout=120) as r, target.with_suffix(target.suffix+'.part').open('wb') as w:
   h=hashlib.sha256();size=0
   while b:=r.read(4*1024*1024):h.update(b);size+=len(b);w.write(b)
  if f.get('lfs'):
   assert h.hexdigest()==f['lfs']['sha256'] and size==f['lfs']['size'],fn
  target.with_suffix(target.suffix+'.part').rename(target)
  rows.append({'file':fn,'bytes':size,'sha256':h.hexdigest()})
 config=json.loads((dest/'config.json').read_text());text=config.get('text_config',config)
 assert text.get('model_type',config.get('model_type')) in ['qwen2','qwen3_5','qwen3_5_text']
 assert text['num_hidden_layers']==configs[model]['attentionLayers']+configs[model]['otherLayers']
 entry={'model':model,'path':str(dest),'max_context':min(text.get('max_position_embeddings',32768),65536),'files':rows}
 (dest/'asset-manifest.json').write_text(json.dumps(entry,indent=2)+'\n')
 print(json.dumps({'model':name,'status':'ready','bytes':sum(r['bytes'] for r in rows)}),flush=True)
 return entry
with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
 rows=list(pool.map(load,configs))
(study/'assets.json').write_text(json.dumps(rows,indent=2)+'\n')
