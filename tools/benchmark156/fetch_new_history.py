"""Minimal public snapshots; no author/contact metadata or external code execution."""
import base64,json,pathlib,subprocess,urllib.parse
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156/corpus-c'
SOURCES=[('spf13/afero','341307286fb77e1fbb6cfe6466c3eb34e9db14d0'),('spf13/afero','83e877f70c073bfc66eb03c9608caa813c8873f9'),('spf13/afero','3fa29445a95d478969fd6639f43ad8fb8b3a85bc')]
def api(route):return json.loads(subprocess.run(['gh','api',route],capture_output=True,check=True).stdout)
def run():
 OUT.mkdir(exist_ok=True)
 for repo,sha in SOURCES:
  target=OUT/(repo.replace('/','-')+'-'+sha[:8]+'.json')
  if target.exists():continue
  r=api(f'repos/{repo}/commits/{sha}');parent=r['parents'][0]['sha'];files=[]
  for i,item in enumerate(r['files']):
   path=item['filename']
   if item['status'] not in ('modified','added','removed'):raise ValueError('unsupported_status')
   versions={}
   for name,ref,absent in [('before',parent,item['status']=='added'),('after',sha,item['status']=='removed')]:
    v=b'' if absent else base64.b64decode(api(f'repos/{repo}/contents/{urllib.parse.quote(path,safe="/")}?ref={ref}')['content'])
    versions[name+'_b64']=base64.b64encode(v).decode()
   files.append(dict(id=f'F{i+1:03}',path=path,public_patch=item.get('patch'),before_exists=item['status']!='added',after_exists=item['status']!='removed',**versions))
  record=dict(repository=repo,commit=sha,parent=parent,files=files,message=r['commit']['message'].split('\n\nSigned-off-by:')[0])
  target.write_text(json.dumps(record,ensure_ascii=False,indent=2)+'\n');print(sha[:8],[f['path'] for f in files],flush=True)
 license=OUT/'spf13-afero-LICENSE.txt'
 if not license.exists():license.write_bytes(base64.b64decode(api('repos/spf13/afero/contents/LICENSE.txt')['content']))
if __name__=='__main__':run()
