import hashlib,json,urllib.request
from pathlib import Path
root=Path('<PRIVATE_TMP>/issue182-ministral-runtime/assets');root.mkdir(exist_ok=True)
records=[]
for kind in ['Instruct','Reasoning']:
 repo=f'mistralai/Ministral-3-3B-{kind}-2512'
 with urllib.request.urlopen('https://huggingface.co/api/models/'+repo) as r:meta=json.load(r)
 rev=meta['sha'];names=[x['rfilename'] for x in meta['siblings']]
 selected=[n for n in names if n in ['tekken.json','SYSTEM_PROMPT.txt','README.md','params.json']]
 folder=root/kind.lower();folder.mkdir(exist_ok=True)
 for name in selected:
  with urllib.request.urlopen(f'https://huggingface.co/{repo}/resolve/{rev}/{name}') as r:data=r.read()
  (folder/name).write_bytes(data)
  records.append(dict(repo=repo,revision=rev,file=name,sha256=hashlib.sha256(data).hexdigest(),bytes=len(data)))
 print(kind,rev,selected)
(root/'manifest.json').write_text(json.dumps(records,indent=2)+'\n')
