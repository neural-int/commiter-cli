"""Coverage of natural string-table tests, not developer intent classification."""
import json,re
from attribution import ROOT,prepare
from local_facts import observe,sha
OUT=ROOT/'docs/benchmarks/issue-155'

def table_rows(text):
    rows=[]
    for f in re.finditer(r'^func (toSnake|toSnakeWithIgnore)\([^\n]*\n.*?^}',text,re.M|re.S):
        body=f.group();table=body.split('for _, i := range cases')[0]
        for m in re.finditer(r'\{("(?:[^"\\]|\\.)*"(?:,\s*"(?:[^"\\]|\\.)*"){1,2})\}',table):
            values=json.loads('['+m.group(1)+']');start=f.start()+m.start();end=f.start()+m.end()
            rows.append(dict(function='ToSnakeWithIgnore' if f.group(1).endswith('WithIgnore') else 'ToSnake',values=values,span=[len(text[:start].encode()),len(text[:end].encode())],text=m.group()))
    return rows

def query(row,implementation):
    args=[row['values'][0]]
    if row['function']=='ToSnakeWithIgnore':
        ignore=row['values'][2] if len(row['values'])==3 else ''
        # Retain version-specific natural API arguments; no string->byte guess.
        if re.search(r'func ToSnakeWithIgnore\([^\n]*ignore uint8',implementation):args.append(ord(ignore[0]) if ignore else 0)
        else:args.append(ignore)
    return dict(package='.',function=row['function'],arguments=args)

if __name__=='__main__':
    snapshots=json.loads((OUT/'iteration-7-source-snapshots.json').read_text());results=[]
    for record in snapshots:
        test=next(f for f in record['files'] if f['path']=='snake_test.go');implementation=record['files'][0]
        old=table_rows(test['before']);after=table_rows(test['after']);keys={(r['function'],tuple(r['values'])) for r in old};selected=[r for r in after if (r['function'],tuple(r['values'])) not in keys];observations=[]
        for row in selected:
            predecessor=[r for r in old if (r['function'],r['values'][0])==(row['function'],row['values'][0])]
            versions={}
            for version,natural in [('after',row),('before',predecessor[0] if len(predecessor)==1 else None)]:
                if natural is None:versions[version]=dict(status='no_matching_natural_row',fact=None);continue
                a,b=natural['span'];assert test[version].encode()[a:b].decode()==natural['text']
                q=query(natural,implementation[version])
                # Each observed call uses exactly the source of its natural version.
                files=[dict(f,before=f[version],after=f[version]) for f in record['files']]
                fact=observe(files,q)['facts']['before']
                versions[version]=dict(status='observed',query=q,test_source=natural,fact=fact)
            observations.append(dict(versions=versions,semantic_attribution=None))
        try:
            p,m=prepare(record['files']);units=len(m);anchors=len(p['observations']);reject=None
        except ValueError as exc:units=None;anchors=None;reject=str(exc)
        results.append(dict(commit=record['commit'],parent=record['parent'],changed_files=record['changed_files'],source_hashes=[dict(path=f['path'],before=sha(f['before'].encode()),after=sha(f['after'].encode())) for f in record['files']],units=units,anchors=anchors,input_reject=reject,observations=observations,semantic_gold=None,holdout='not_certified'))
    with (OUT/'iteration-7-natural-audit.json').open('x') as f:json.dump(dict(repository='iancoleman/strcase',model_calls=0,rows=results),f,ensure_ascii=False,indent=2);f.write('\n')
    for r in results:
        states=[v for o in r['observations'] for v in o['versions'].values()];print(r['commit'],'rows',len(r['observations']),'natural states',sum(v['fact']is not None for v in states),'known',sum(v['fact']['known'] for v in states if v['fact']is not None),'reject',r['input_reject'])
