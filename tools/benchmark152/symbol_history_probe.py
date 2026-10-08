"""Actual temporary Git commits with fixed content-independent touch extraction."""
import json,pathlib,tempfile,subprocess
from symbol_history import touches
HERE=pathlib.Path(__file__).resolve().parent
helper='/tmp/issue151-remeasure-symbols'
with tempfile.TemporaryDirectory(prefix='issue152-history-symbol-') as tmp:
 root=pathlib.Path(tmp)
 def git(*args): return subprocess.check_output(['git',*args],cwd=root)
 git('init','-q');git('config','user.name','Synthetic benchmark');git('config','user.email','benchmark@example.invalid')
 source='package p\nfunc Left() int { return 2 }\nfunc Right() int { return 3 }\n'
 p=root/'value.go';p.write_text(source);git('add','.');git('commit','-qm','initial')
 rows=[dict(case='initial',result=touches(None,source.encode(),helper))]
 for name,new in [('left',source.replace('return 2','return 4')),('housekeeping',source.replace('return 2','return 4')+'// housekeeping\n'),('both',source.replace('return 2','return 6').replace('return 3','return 5'))]:
  before=git('show','HEAD:value.go');p.write_text(new);git('add','.');git('commit','-qm',name);after=git('show','HEAD:value.go')
  rows.append(dict(case=name,commit=git('rev-parse','HEAD').decode().strip(),result=touches(before,after,helper)))
 before=git('show','HEAD:value.go');git('mv','value.go','moved.go');git('commit','-qm','rename');after=git('show','HEAD:moved.go')
 rows.append(dict(case='rename_content',result=touches(before,after,helper),identity='rename_identity_not_extracted'))
 rows.append(dict(case='new_file',result=touches(None,after,helper)))
 print(json.dumps(rows,indent=2))
 (HERE.parents[1]/'docs/benchmarks/issue-152/iteration-22-results.json').write_text(json.dumps(rows,indent=2)+'\n')
