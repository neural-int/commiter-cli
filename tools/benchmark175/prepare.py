"""Materialize local-only historical fixtures without touching source repositories."""
import argparse,hashlib,json,pathlib,subprocess
def git(repo,*args,input=None,check=True):
 return subprocess.run(['git',*args],cwd=repo,input=input,stdout=subprocess.PIPE,stderr=subprocess.PIPE,check=check).stdout
def main():
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);p.add_argument('--fixture-dir',type=pathlib.Path,required=True);p.add_argument('--source-c01',type=pathlib.Path,required=True);p.add_argument('--source-c02',type=pathlib.Path,required=True);a=p.parse_args();a.out.mkdir(exist_ok=False,parents=True)
 rows=[]
 for cid in ['C01','C02']:
  fixture_path=a.fixture_dir/(cid.lower()+'-fixture.json');fixture=json.loads(fixture_path.read_text());repo=a.out/cid;repo.mkdir()
  git(repo,'init','-b','codex/fixture');git(repo,'config','user.name','Evaluation Fixture');git(repo,'config','user.email','fixture@example.invalid')
  selected=[f['path'] for f in fixture['files']]
  if cid=='C01':
   source=a.source_c01;revision='ba57e5c';patch=''
   for f in fixture['files']:
    path=f['path'];dest=repo/path;dest.parent.mkdir(parents=True,exist_ok=True)
    dest.write_bytes(git(source,'show',revision+':'+path))
    added=path=='src/lib/staticAsset.js'
    patch+=f'diff --git a/{path} b/{path}\n'+('new file mode 100644\n' if added else '')+f'--- {"/dev/null" if added else "a/"+path}\n+++ b/{path}\n'+f['diff']
   after={path:(repo/path).read_bytes() for path in selected}
   git(repo,'apply','--reverse','--unidiff-zero','-',input=patch.encode())
   base_ref='reverse logged operation from historical final blobs ba57e5c (unchanged surrounding content retained)'
  else:
   source=a.source_c02;base=fixture['snapshot']['base'];head=fixture['snapshot']['head'];after={}
   for path in selected:
    dest=repo/path;dest.parent.mkdir(parents=True,exist_ok=True)
    prior=git(source,'show',base+':'+path,check=False)
    exists=subprocess.run(['git','cat-file','-e',base+':'+path],cwd=source,capture_output=True).returncode==0
    if exists:dest.write_bytes(prior)
    after[path]=git(source,'show',head+':'+path)
   base_ref=base+'..'+head+' selected paths'
  # Empty base is permitted; source repositories are only read.
  existing=[path for path in selected if (repo/path).is_file()]
  if existing:git(repo,'add','--',*existing)
  git(repo,'commit','--allow-empty','-m','fixture base')
  for path,content in after.items():
   dest=repo/path;dest.parent.mkdir(parents=True,exist_ok=True);dest.write_bytes(content)
  before={path:hashlib.sha256(git(repo,'show','HEAD:'+path,check=False)).hexdigest() for path in selected}
  rows.append({'case_id':cid,'root':str(repo.resolve()),'source':base_ref,'selected_paths':selected,'before_sha256':before,'after_sha256':{p:hashlib.sha256(c).hexdigest() for p,c in after.items()},'fixture_sha256':hashlib.sha256(fixture_path.read_bytes()).hexdigest()})
 (a.out/'repos.json').write_text(json.dumps(rows,indent=2)+'\n');print(a.out/'repos.json')
if __name__=='__main__':main()
