"""C readiness audit after B NO-GO. No gold-driven runtime split."""
import json,pathlib,time,hashlib,base64
from evaluate_a import diagnostic_cases,metrics
from planner import fallback,replay,encode
ROOT=pathlib.Path(__file__).resolve().parents[2];OUT=ROOT/'docs/benchmarks/issue-156'
if __name__=='__main__':
    rows=[]
    for name,files,units,gold in diagnostic_cases():
        # Oracle establishes feasibility only. It must never become a planner result.
        oracle=[]
        for purpose in sorted(set(gold.values())):
            ids=[u['id'] for u in units if gold[u['id']]==purpose]
            oracle.append(dict(file_ids=sorted({u['file'] for u in units if u['id'] in ids}),unit_ids=ids))
        rows.append(dict(name=name,split='used_structural_diagnostic',files=len(files),units=len(units),input_sha256=hashlib.sha256(encode(files)).hexdigest(),baseline=metrics(units,fallback(files,units),gold),oracle_semantic_prediction=False,oracle_only_staging=replay(files,units,oracle),candidate_split=False))
    prior=[json.loads(line) for line in (OUT/'iteration-2-results.jsonl').read_text().splitlines()]
    false_mixed=[]
    for r in prior:
        members={}
        for candidate in r['answer'].values():
            for fid,m in candidate['members'].items():
                if m['status']=='mixed':members[fid]=m
        if members:false_mixed.append(dict(name=r['name'],reverse=r['reverse'],files=sorted(members),gold_all_files_single=True))
    result=dict(contract='frozen prior A/B and #151 structural assets; no semantic threshold changes',rows=rows,false_mixed_triggers=false_mixed,
        complete=sum(r['oracle_only_staging']['final_tree_equality'] for r in rows),cases=len(rows),calls=0,
        structural_capability='GO for known #151 representable subsets only',semantic_adaptive_split='NOT_PROVEN',production_candidate='NO-GO',
        reason='B classifier produces false mixed labels on single-purpose files; finer stageability cannot establish changed intent. No independently validated mixed-file classifier or automatic CU membership.',
        refinement_runtime_cost=None,independent_semantic_holdout=False)
    with (OUT/'iteration-3-results.json').open('x') as f:json.dump(result,f,ensure_ascii=False,indent=2);f.write('\n')
    print(json.dumps({k:v for k,v in result.items() if k not in ('rows','false_mixed_triggers')},ensure_ascii=False))
