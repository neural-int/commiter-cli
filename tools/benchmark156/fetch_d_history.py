"""D reserved public histories. Never executes downloaded source."""
import base64,json,pathlib
from fetch_new_history import api
import urllib.parse
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156/corpus-d'
SOURCES=['5e8a21e61529ac2c2846c543fefa1bed6d2fc72c','15f6826fe4614cc225c16903d2e9a1c966fb2332','3dce331fc4bbbd9585fe7431d86b3553b6e4729d','cce2851b990e0421118880449b437ff11912625b','2e4825f00fdbe8e0be4544f38e7fef8060d26467','556f9edbcca9b16c1e208f174d99008cc9e8cc5a']
def run():
 OUT.mkdir(exist_ok=True);repo='stretchr/testify'
 for sha in SOURCES:
  target=OUT/(sha[:8]+'.json')
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
  target.write_text(json.dumps(dict(repository=repo,commit=sha,parent=parent,files=files,message=r['commit']['message'].split('\n\nSigned-off-by:')[0]),ensure_ascii=False,indent=2)+'\n');print(sha[:8],[f['path'] for f in files],flush=True)
 license=OUT/'LICENSE.txt'
 if not license.exists():license.write_bytes(base64.b64decode(api(f'repos/{repo}/contents/LICENSE')['content']))
if __name__=='__main__':run()
