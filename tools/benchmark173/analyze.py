"""Derive evidence summaries; never edits Gold, requests, or receipts."""
import collections, json, statistics
from pathlib import Path
from prepare import DATA, save

def load(name):
    p=DATA/name
    return [json.loads(s) for s in p.read_text().splitlines()] if p.exists() else []
def metrics(rows):
    groups=[g for r in rows for g in r.get('groups',[])]
    counts=collections.Counter(g['actual'] for g in groups)
    fallback={typ:{'selected':sum(g['actual']==typ for g in groups),'correct':sum(g['actual']==typ and g['expected']==typ for g in groups),'false_selection':sum(g['actual']==typ and g['expected']!=typ for g in groups),'missed':sum(g['actual']!=typ and g['expected']==typ for g in groups)} for typ in ['update','change']}
    return {'requests':len(rows),'groups':len(groups),'schema_valid':sum(r['schema_valid'] for r in rows),'gold_exact':sum(g['expected']==g['actual'] for g in groups),'type_counts':dict(counts),'confusion':dict(collections.Counter(f"{g['expected']}->{g['actual']}" for g in groups)),'false_definite_feat_fix':sum(g['actual'] in ['feat','fix'] and g['expected']!=g['actual'] for g in groups),'feat_fix_confusion':sum({g['expected'],g['actual']}=={'feat','fix'} for g in groups),'fallback':fallback,'wall_seconds':{'median':statistics.median([r['wall_seconds'] for r in rows]) if rows else None,'max':max([r['wall_seconds'] for r in rows],default=None)},'helper_peak_rss_max_bytes':max([r['helper_peak_rss_bytes'] or 0 for r in rows],default=None),'input_bytes_total':sum(r['input_bytes'] for r in rows),'native_input_tokens':None,'native_output_tokens':None,'git_unchanged':all(r['git_unchanged'] for r in rows)}
def main():
    category=load('category-results.jsonl');integrated=load('integrated-results.jsonl')
    summary={'paired':{arm:metrics([r for r in category if r['case'] in 'ABCDE' and r['arm']==arm]) for arm in ['baseline','candidate']},'boundary':metrics([r for r in category if r['case'].startswith('F')]),'integrated':{'cycles':len(integrated),'accepted_plans':sum(r['plan_accepted'] for r in integrated),'generation_calls':sum(r['generation_calls'] for r in integrated),'git_unchanged':all(r['git_unchanged'] for r in integrated)},'completed':len(category)==30 and len(integrated)==5}
    baseline=summary['paired']['baseline'];candidate=summary['paired']['candidate'];boundary=summary['boundary']
    summary['gate']={'paired':candidate['gold_exact']>=5 and candidate['gold_exact']>baseline['gold_exact'],'boundary':boundary['groups']==20 and boundary['gold_exact']==20,'integrated':len(integrated)==5 and summary['integrated']['accepted_plans']==5,'schema':len(category)==30 and all(r['schema_valid'] for r in category),'git':all(r['git_unchanged'] for r in category+integrated),'token_measurement':'unavailable; not a zero measurement','software_quality':'see quality.json; never inferred from model schema'}
    summary['adoption']='NO-GO' if summary['completed'] and not all(summary['gate'][k] for k in ['paired','boundary','integrated','schema','git']) else 'not_decided'
    save(DATA/'analysis.json',summary)
    print(json.dumps(summary,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
