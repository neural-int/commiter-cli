"""Read-only bounded scanner for explicitly selected Go symbol identities."""
import subprocess
from symbol_history import touches,summarize

def scan(root,selected,helper):
 if len(selected)>64: raise ValueError('history_budget')
 def git(*args):
  return subprocess.run(['git',*args],cwd=root,capture_output=True,timeout=5)
 def snapshot(commit,path):
  size=git('cat-file','-s',commit+':'+path)
  if size.returncode or int(size.stdout)>1048576:return None
  p=git('show',commit+':'+path);return p.stdout if not p.returncode else None
 log=git('log','-64','--format=%H %P','HEAD')
 if log.returncode:raise ValueError('history_unavailable')
 events=[]
 for line in log.stdout.decode().splitlines():
  parts=line.split();commit,parents=parts[0],parts[1:]
  if len(parents)!=1:
   events.append(dict(commit=commit,status='unknown_initial' if not parents else 'unknown_merge',symbols=[]));continue
  names=git('diff-tree','--no-commit-id','--name-status','-r','-M',parents[0],commit)
  if names.returncode:raise ValueError('history_unavailable')
  # Quoted paths are not decoded: mark any rename conservatively unknown.
  if any(x.startswith(b'R') for x in names.stdout.splitlines()):
   events.append(dict(commit=commit,status='unknown_rename',symbols=[]));continue
  touched=[];unknown=False
  for path in sorted({p for p,s in selected}):
   r=touches(snapshot(parents[0],path),snapshot(commit,path),helper)
   if r['status']!='observed_symbol_snapshot':unknown=True;continue
   touched.extend(path+'::'+s for p,s in selected if p==path and s in r['symbols'])
  events.append(dict(commit=commit,status='unknown_snapshot' if unknown else 'observed_symbol_snapshot',symbols=sorted(set(touched)) if not unknown else []))
 return dict(events=events,summary=summarize(events,[p+'::'+s for p,s in selected]))
