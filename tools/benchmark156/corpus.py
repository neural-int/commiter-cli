"""Public history snapshots. Fetches only public source, never runs that source."""
import base64
import hashlib
import json
import pathlib
import subprocess
import urllib.parse
from planner import encode, prepare
ROOT=pathlib.Path(__file__).resolve().parents[2]
OUT=ROOT/'docs/benchmarks/issue-156/corpus'

def selected(record):
    files=[f for f in record['files'] if f['status'] in ('modified','added','removed') and f['changes']>0]
    if record['repository']=='google/go-cmp':
        return sorted(files,key=lambda f:f['filename'])[:16]
    if len(files)>16:
        # Predeclared selected-file slice; not a claim about the entire 61-file commit.
        files=sorted((f for f in files if f['filename'].endswith('.toml') and '/key/' not in f['filename']),key=lambda f:f['filename'])[:16]
    return files

if __name__=='__main__':
    rows=[]
    for p in sorted(OUT.glob('*-*.json')):
        if p.name.endswith('-snapshots.json'):continue
        record=json.loads(p.read_text());snapshot=OUT/(p.stem+'-snapshots.json')
        if snapshot.exists():continue
        repo,sha,parent=record['repository'],record['sha'],record['parents'][0]['sha']
        files=[]
        for i,item in enumerate(selected(record)):
            path=item['filename'];versions={}
            for name,ref,absent in [('before',parent,item['status']=='added'),('after',sha,item['status']=='removed')]:
                if absent:versions[name]='';continue
                request=f'repos/{repo}/contents/{urllib.parse.quote(path,safe="/")}?ref={ref}'
                r=json.loads(subprocess.run(['gh','api',request],capture_output=True,check=True).stdout)
                if r.get('type')!='file' or r.get('encoding')!='base64':raise ValueError('unsupported_public_content')
                versions[name]=base64.b64encode(base64.b64decode(r['content'])).decode()
            files.append(dict(id=f'F{i+1:03}',path=path,before_b64=versions['before'],after_b64=versions['after'],before_exists=item['status']!='added',after_exists=item['status']!='removed',public_patch=item.get('patch')))
        clean=[{k:v for k,v in f.items() if k!='public_patch'} for f in files]
        try:units=prepare(clean);audit=dict(structurally_supported=True,units=len(units),reason=None)
        except ValueError as exc:audit=dict(structurally_supported=False,units=None,reason=str(exc))
        output=dict(repository=repo,commit=sha,parent=parent,files=files,input_sha256=hashlib.sha256(encode(clean)).hexdigest(),source_commit_message=record['commit']['message'],audit=audit,
            classification='unseen_public_repository_source; gold requires separate audit before inference',whole_commit_files=len(record['files']),selected_files=len(files))
        snapshot.write_text(json.dumps(output,ensure_ascii=False,indent=2)+'\n')
        print(repo,sha[:8],len(files),audit,flush=True)
