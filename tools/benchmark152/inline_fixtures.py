"""Prospective author-defined source/test requirements; not graph-derived gold."""
import json
import pathlib


def make(name, routing, split, direct=False, common=False, base=1):
    files = []
    source = 'package calc\n'
    if common:
        source += 'import "fixture/common"\n'
    for i in range(3):
        expr = 'common.Clean(n)' if common else 'n'
        source += f'func P{i}(n int) int {{ return {expr} + {base} }}\n'
    changed = source.replace(f'+ {base}', f'+ {base+1}')
    files.append(dict(ID='F001',Path='calc/values.go',Before=source,After=changed))
    repo=[]
    if not direct:
        wrapper='package adapters\nimport "fixture/calc"\n'
        for i,target in enumerate(routing):
            wrapper+=f'func Q{i}(n int) int {{ return calc.P{target}(n) }}\n'
        repo.append(dict(ID='R001',Path='adapters/routes.go',Content=wrapper))
    if common:
        repo.append(dict(ID='R002',Path='common/clean.go',Content='package common\nfunc Clean(n int) int { return n }\n'))
    gold=[]
    for i,target in enumerate(routing):
        namespace,callee=('calc',f'P{target}') if direct else ('adapters',f'Q{i}')
        before=f'package checks\nimport ("testing"; "fixture/{namespace}")\nfunc TestT{i}(t *testing.T) {{ if {namespace}.{callee}(0) != {base} {{ t.Fatal("value") }} }}\n'
        files.append(dict(ID=f'F{i+2:03}',Path=f'checks/case_{i}_test.go',Before=before,After=before.replace(f'!= {base}',f'!= {base+1}')))
        gold.append(dict(requirement=f'P{target}の独立した加算値を{base}から{base+1}へ変更し、対応するTestT{i}を更新する',
                         symbols=[['F001',f'P{target}'],[f'F{i+2:03}',f'TestT{i}']]))
    return dict(Name=name,Split=split,Files=files,Repository=repo,SymbolGold=gold)


def main():
    records=[make('qual-direct-symbols',[0,1,2],'qualification',direct=True),
             make('qual-shared-helper',[0,1,2],'qualification',direct=True,common=True),
             make('independent-routed-a',[2,0,1],'independent',base=1),
             make('independent-routed-b',[1,2,0],'independent',base=5),
             make('independent-routed-c',[2,1,0],'independent',base=7)]
    dest=pathlib.Path(__file__).resolve().parent/'inline-fixtures.json'
    dest.write_text(json.dumps(records,ensure_ascii=False,indent=2)+'\n')


if __name__ == '__main__':
    main()
