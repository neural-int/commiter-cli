"""Audit immutable paired inputs, content safety, cohort overlap and syntax bounds."""
import hashlib,json,re
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2];DATA=ROOT/'docs/benchmarks/issue-169'
COMMON=['id','status','old_path','new_path','language','binary','vendor','opaque']
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def audit():
    rows=json.loads((DATA/'frozen-inputs/manifest.json').read_text());sm=json.loads((DATA/'source-mapping.json').read_text());fixtures=json.loads((ROOT/'tools/benchmark169/fixtures.json').read_text())
    pairs={};summary=[]
    for row in rows:
        path=DATA/'frozen-inputs'/row['request_file'];req=json.loads(path.read_text());assert sha(path)==row['request_sha256']
        user=json.loads(req['messages'][1]['content']);assert set(user)=={'task','files'}
        assert req['output_tokens']==768 and req['context_tokens']==16384 and req['generation_profile']=='bounded-grouping'
        files=user['files'];assert len(files)==row['selected_files'];assert len({f['id'] for f in files})==len(files)
        for f in files:
            assert all(k in f for k in COMMON)
            assert not any(f[k] for k in ['binary','vendor','opaque'])
            assert not set(f)&{'old_mode','new_mode','head_identity','worktree_identity','worktree_kind','change_hash','size','staged','unstaged'}
            assert not re.search(r'(^|/)(\.env[^/]*|credentials[^/]*|[^/]*\.(pem|key))$',f['new_path'] or f['old_path'],re.I)
            if row['arm']=='A':
                assert f['mode']=='structural' and 'raw_diff' not in f
                assert all(set(e)<={'kind','name','enclosing_declaration','role'} for e in f['evidence'])
            else:
                assert f['mode']=='raw_diff' and 'evidence' not in f
                assert hashlib.sha256(f['raw_diff'].encode()).hexdigest()==sm[row['fixture']]['files'][f['id']]['raw_diff_sha256']
                hc=len(re.findall(r'^@@ ',f['raw_diff'],re.M));assert hc==sm[row['fixture']]['files'][f['id']]['hunks'],'hunk count metadata mismatch'
                if row['arm']=='C':
                    sc=f['syntax_context'];assert len(sc)<=32
                    assert sc==sorted(sc,key=lambda x:(x['hunk_index'],x['enclosing_declaration']))
                    assert len({(x['hunk_index'],x['enclosing_declaration']) for x in sc})==len(sc)
                    for c in sc:assert set(c)=={'hunk_index','enclosing_declaration'} and type(c['hunk_index']) is int and 0<=c['hunk_index']<hc and 0<len(c['enclosing_declaration'].encode())<=128
                else:assert 'syntax_context' not in f
        key=(row['fixture'],row['order'],row['phase']);pairs.setdefault(key,{})[row['arm']]=files
    for key,arms in pairs.items():
        assert set(arms)=={'A','B','C'}
        assert [{k:f[k] for k in COMMON} for f in arms['A']]==[{k:f[k] for k in COMMON} for f in arms['B']]==[{k:f[k] for k in COMMON} for f in arms['C']]
        assert arms['B']==[{k:v for k,v in f.items() if k!='syntax_context'} for f in arms['C']]
        if key[1]=='normal':assert arms['A']==list(reversed(pairs[(key[0],'reverse',key[2])]['A']))
    for f in fixtures:
        rec=[r for r in rows if r['fixture']==f['name'] and r['phase']=='discovery']
        summary.append({'fixture':f['name'],'cohort':f['cohort'],'admitted_cells':sum(r['admitted'] for r in rec),'planned_cells':len(rec),'tokens_by_arm':{a:[r['token_count_before_inference'] for r in rec if r['arm']==a] for a in ['A','B','C']}})
    result={'paired_cells':len(rows),'paired_file_metadata_and_BC_raw_bytes_match':True,'gold_and_provenance_absent_from_requests':True,'raw_diff_truncation_count':0,'summary':summary}
    (DATA/'input-audit.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
if __name__=='__main__':audit()
