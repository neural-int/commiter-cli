"""Read-only source checks and isolated temporary-index/tree reconstruction."""
import hashlib,json,os,subprocess,tempfile
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2];DATA=ROOT/'docs/benchmarks/issue-171'

def git(root,*args,env=None):return subprocess.run(['git','-C',str(root),*args],env=env or {**os.environ,'GIT_OPTIONAL_LOCKS':'0','GIT_CONFIG_GLOBAL':'/dev/null','GIT_CONFIG_NOSYSTEM':'1'},capture_output=True,check=True).stdout

def source_state(root):
 index=Path(git(root,'rev-parse','--path-format=absolute','--git-path','index').decode().strip())
 return {'head':git(root,'rev-parse','HEAD').decode().strip(),'index_sha256':hashlib.sha256(index.read_bytes()).hexdigest(),'status':git(root,'status','--porcelain=v1','--untracked-files=all').decode(),'tracked_diff_sha256':hashlib.sha256(git(root,'diff','--binary','HEAD')).hexdigest()}

def audit(local):
 sm=json.loads((DATA/'source-mapping.json').read_text());fx=json.loads((ROOT/'tools/benchmark171/fixtures.json').read_text());rows=[]
 for f in fx:
  root=Path(local['fixture_roots'][f['name']]);before=source_state(root);mapping=sm[f['name']]['files'];checks=[]
  objects=Path(git(root,'rev-parse','--path-format=absolute','--git-path','objects').decode().strip())
  with tempfile.TemporaryDirectory(prefix='commiter-171-index-') as tmp:
   tmp=Path(tmp);(tmp/'objects').mkdir();env={**os.environ,'GIT_INDEX_FILE':str(tmp/'index'),'GIT_OBJECT_DIRECTORY':str(tmp/'objects'),'GIT_ALTERNATE_OBJECT_DIRECTORIES':str(objects),'GIT_CONFIG_GLOBAL':'/dev/null','GIT_CONFIG_NOSYSTEM':'1','GIT_OPTIONAL_LOCKS':'0'}
   git(root,'read-tree','HEAD',env=env)
   groups=f['gold_groups'] if f['gold_status']!='undetermined' else [[v['path'] for v in mapping.values()]]
   for g in groups:git(root,'add','--',*g,env=env)
   tree=git(root,'write-tree',env=env).decode().strip()
   expected={v['path']:v for v in mapping.values()}
   for path,m in expected.items():
    blob=git(root,'show',tree+':'+path,env=env)
    actual=(root/path).read_bytes();assert blob==actual and hashlib.sha256(blob).hexdigest()==m['after_sha256']
    original=git(root,'show','HEAD:'+path);assert hashlib.sha256(original).hexdigest()==m['before_sha256']
    checks.append(path)
   # Whole-tree equality: only selected changed files replace HEAD entries.
   entries=lambda raw:{r.split(b'\t',1)[1]:r.split(b'\t',1)[0] for r in raw.split(b'\x00') if r}
   head=entries(git(root,'ls-tree','-r','-z','HEAD'));rebuilt=entries(git(root,'ls-tree','-r','-z',tree,env=env));assert head.keys()==rebuilt.keys()
   for path in head:
    if path.decode() not in expected:assert head[path]==rebuilt[path]
   assert set(git(root,'diff','--name-only','HEAD').decode().splitlines())==set(expected)
  assert source_state(root)==before
  rows.append({'fixture':f['name'],'temporary_index_only':True,'original_head_index_worktree_unchanged':True,'selected_bytes_equal':checks,'whole_tree_equal':True,'reconstructed_tree':tree})
 result={'fixtures':len(rows),'rows':rows,'original_source_unchanged':True};(DATA/'integrity-audit.json').write_text(json.dumps(result,indent=2)+'\n');return result
if __name__=='__main__':
 import sys
 print(json.dumps(audit(json.loads(Path(sys.argv[1]).read_text()))))
