"""One fixed source-context selection, independent of evaluation labels."""
import hashlib
import json
import pathlib

MAX_INPUT_BYTES = 16384

def encode(value):
    return json.dumps(value,sort_keys=True,ensure_ascii=False,separators=(',', ':')).encode()

def digest(value):
    return hashlib.sha256(encode(value)).hexdigest()

def select(files, observations):
    paths={f['id']:f['path'] for f in files}
    source={f['id']:f for f in files}
    facts=observations['evidence']
    by_id={e['id']:e for e in facts}
    if len(by_id)!=len(facts):
        raise ValueError('duplicate_evidence_id')
    for e in facts:
        raw=source[e['file']][e['version']].encode()
        lo,hi=e['span']
        if not 0<=lo<=hi<=len(raw) or raw[lo:hi].decode()!=e['text'] or hashlib.sha256(raw).hexdigest()!=e['source_sha256']:
            raise ValueError('source_mapping')
    declarations={'function_source','type_source','binding_source'}
    seeds=[e for e in facts if e['changed'] and e['kind'] in declarations]
    selected={e['id'] for e in seeds}
    # A single syntactic call step, no type inference and no commit relation.
    unresolved=[]
    for seed in seeds:
        if seed['kind']!='function_source':
            continue
        package=str(pathlib.PurePosixPath(paths[seed['file']]).parent)
        for name in seed.get('calls',[]):
            options=[e for e in facts if e['kind']=='function_source' and e['function']==name and e['version']==seed['version'] and str(pathlib.PurePosixPath(paths[e['file']]).parent)==package]
            if len(options)==1:
                selected.add(options[0]['id'])
            elif options:
                unresolved.append(dict(source=seed['id'],name=name,status='ambiguous_declaration'))
            # Builtins/external functions have no selectable declaration here.
            else:
                unresolved.append(dict(source=seed['id'],name=name,status='not_in_selected_snapshots'))
    selected_files={(by_id[k]['file'],by_id[k]['version']) for k in selected}
    for e in facts:
        if e['kind']=='import_source' and (e['file'],e['version']) in selected_files:
            selected.add(e['id'])
    # Exact-text dedup only. Distinct snapshots retain their own evidence refs.
    grouped={}
    for eid in sorted(selected):
        e=by_id[eid]
        key=(e['file'],e['kind'],e['function'],e['text'])
        grouped.setdefault(key,[]).append(e)
    context=[]
    for (fid,kind,name,text),versions in sorted(grouped.items()):
        refs=[{k:e[k] for k in ('id','file','version','span','source_sha256')} for e in versions]
        context.append(dict(id=digest(refs)[:24],kind=kind,symbol=name,text=text,source_refs=refs))
    anchors=[]
    for e in facts:
        if e['kind']=='failure_predicate' and e['changed']:
            parents=[c for c in context if c['kind']=='function_source' and any(r['file']==e['file'] and r['version']==e['version'] and r['span'][0]<=e['span'][0]<=e['span'][1]<=r['span'][1] for r in c['source_refs'])]
            if len(parents)!=1:
                raise ValueError('anchor_context_mapping')
            anchors.append({k:e[k] for k in ('id','file','version','span','text','source_sha256','condition_span')} | dict(context_id=parents[0]['id']))
    return dict(context=context,anchors=sorted(anchors,key=lambda e:e['id']),unresolved_calls=sorted(unresolved,key=lambda e:(e['source'],e['name'])))
