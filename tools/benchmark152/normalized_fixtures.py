"""Prospective five-file source/test correspondence requirements."""
import json,pathlib

records=[]
sets=[(['AccountQuota','UploadLimit','CacheSize','RetryWindow'],[2,0,3,1],2),(['ReadLimit','WriteLimit','BatchSize','PoolSize'],[1,3,0,2],5),(['QueueDepth','PageSize','TimeoutLimit','BufferSize'],[3,2,1,0],7)]
for case,(names,routing,base) in enumerate(sets):
 before='package calc\n'+''.join(f'func {name}(n int) int {{ return n * {base} }}\n' for name in names)
 files=[dict(ID='F001',Path='calc/limits.go',Before=before,After=before.replace(f'* {base}',f'* {base+1}'))]
 wrapper='package adapters\nimport "fixture/calc"\n'+''.join(f'func C{i}(n int) int {{ return calc.{names[target]}(n) }}\n' for i,target in enumerate(routing))
 gold=[]
 for i,target in enumerate(routing):
  test=f'package checks\nimport("testing";"fixture/adapters")\nfunc TestCheck{i}(t *testing.T) {{ if adapters.C{i}(1) != {base} {{ t.Fatal("limit") }} }}\n'
  fid=f'F{i+2:03}';files.append(dict(ID=fid,Path=f'checks/case_{i}_test.go',Before=test,After=test.replace(f'!= {base}',f'!= {base+1}')))
  gold.append(dict(requirement=f'{names[target]}の独立した係数変更と対応test更新',symbols=[['F001',names[target]],[fid,f'TestCheck{i}']]))
 records.append(dict(Name=f'normalized-fresh-{case+1}',Split='independent',Files=files,Repository=[dict(ID='R001',Path='adapters/routes.go',Content=wrapper)],SymbolGold=gold))
pathlib.Path(__file__).with_name('normalized-fixtures.json').write_text(json.dumps(records,ensure_ascii=False,indent=2)+'\n')
