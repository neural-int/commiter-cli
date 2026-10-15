"""Read captured final JSON only; never calls the model or rewrites receipts."""
import argparse, json
from pathlib import Path
from prepare import DATA, save

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--root',type=Path,required=True);a=ap.parse_args();rows=[];gold=json.loads((DATA/'prior-gold.json').read_text())
    for directory in sorted((a.root/'runs/integrated').glob('*')):
        result=directory/'result.json'
        if not result.exists():continue
        row=json.loads(result.read_text());case=row['case'];trace=directory/'trace';out={'case':case,'plan_accepted':row['plan_accepted'],'cli_error':row['cli_error'],'phase_request_matches_frozen':{},'text_lengths':None,'category_output':None}
        for phase in ['01','03']:
            request=trace/f'{phase}.request.json'
            if request.exists():out['phase_request_matches_frozen'][phase]=json.loads(request.read_text())==json.loads((DATA/f'integrated-inputs/{case}-{phase}.json').read_text())
        request=trace/'02.request.json'
        if request.exists():
            actual=json.loads(request.read_text());groups=json.loads(actual['messages'][1]['content'])['groups'];paths=[sorted([*g['contents'],*(g['related_tests'] or {})]) for g in groups]
            expected=[sorted(g['paths']) for g in gold[case]['groups']]
            out['grouping_partition_exact']=sorted(paths)==sorted(expected)
            out['category_request_matches_frozen']=actual==json.loads((DATA/f'inputs/{case}-candidate.json').read_text())
        response=trace/'02.response.json'
        if response.exists():
            r=json.loads(response.read_text())
            if r.get('ok'):out['category_output']=json.loads(r['generated_json'])
        response=trace/'03.response.json'
        if response.exists():
            r=json.loads(response.read_text())
            if r.get('ok'):
                text=json.loads(r['generated_json']);out['text_lengths']={k:{'scope_characters':len(v.get('scope','')),'summary_characters':len(v.get('summary','')),'scope_over_16':len(v.get('scope',''))>16,'summary_over_48':len(v.get('summary',''))>48} for k,v in text.items()}
        rows.append(out)
    save(DATA/'integrated-diagnostics.json',rows)
    print(json.dumps(rows,indent=2))
if __name__=='__main__':main()
