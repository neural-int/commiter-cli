"""Fixed multifile research: dynamic requests and label-independent scoring. No acquisition."""
import copy,json,itertools
SCHEMA_MARKER='The required JSON Schema is: '
def validate(value,schema):
    kind=schema['type']
    if kind=='object':
        if not isinstance(value,dict):raise ValueError('not object')
        if not set(schema.get('required',[]))<=set(value):raise ValueError('missing field')
        props=schema['properties']
        if schema.get('additionalProperties') is False and not set(value)<=set(props):raise ValueError('unknown field')
        for k,v in value.items():validate(v,props[k])
    elif kind=='string':
        if not isinstance(value,str):raise ValueError('not string')
        if 'enum' in schema and value not in schema['enum']:raise ValueError('enum')
        if len(value)<schema.get('minLength',0) or len(value)>schema.get('maxLength',2**30):raise ValueError('length')
    else:raise ValueError('unsupported schema')
def partition(membership):
    groups={}
    for fid,label in membership.items():groups.setdefault(label,[]).append(fid)
    return sorted([sorted(ids,key=lambda f:int(f[1:])) for ids in groups.values()],key=lambda fs:int(fs[0][1:]))
def score_groups(membership,gold):
    actual={frozenset(g) for g in partition(membership)};expected={frozenset(g['file_ids']) for g in gold['groups']}
    owner={f:i for i,g in enumerate(expected) for f in g};pred={f:i for i,g in enumerate(actual) for f in g}
    tp=fp=fn=0
    for a,b in itertools.combinations(sorted(owner),2):
        same=owner[a]==owner[b];yes=pred.get(a)==pred.get(b) and a in pred and b in pred
        tp+=same and yes;fp+=not same and yes;fn+=same and not yes
    return dict(exact_partition=actual==expected,exact_groups=len(actual&expected),gold_groups=len(expected),predicted_groups=len(actual),pairwise_precision=tp/(tp+fp) if tp+fp else 1.0,pairwise_recall=tp/(tp+fn) if tp+fn else 1.0,false_merged_pairs=fp,false_split_pairs=fn)
def groups_from_membership(membership,files):
    byid={f['file_id']:f for f in files}
    if set(membership)!=set(byid):raise ValueError('incomplete assignment')
    return [dict(group_id=f'G{i:03d}',files=[copy.deepcopy(byid[f]) for f in ids]) for i,ids in enumerate(partition(membership),1)]
def payload_for(template,groups,categories=None):
    out=copy.deepcopy(template);out['groups']=copy.deepcopy(groups)
    if categories is not None:
        if set(categories)!={g['group_id'] for g in groups}:raise ValueError('category keys')
        for g in out['groups']:
            c=categories[g['group_id']]
            if c['breaking_evidence_ref']=='unresolved':raise ValueError('unresolved_breaking_evidence')
            g['type']=c['type'];g['breaking']=c['breaking_evidence_ref']!='none'
    return out
def request_for(template,payload,group_ids=None):
    r=copy.deepcopy(template)
    if group_ids is not None:
        proto=next(iter(r['schema']['properties'].values()));r['schema']['properties']={k:copy.deepcopy(proto) for k in group_ids};r['schema']['required']=list(group_ids)
    base,sep,_=r['messages'][0]['content'].rpartition(SCHEMA_MARKER)
    if not sep:raise ValueError('schema marker absent')
    r['messages'][0]['content']=base+sep+json.dumps(r['schema'],ensure_ascii=False,sort_keys=True,separators=(',',':'))
    r['messages'][1]['content']=json.dumps(payload,ensure_ascii=False,separators=(',',':'))
    return r
