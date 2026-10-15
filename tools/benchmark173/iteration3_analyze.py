"""Summarize short-prompt results without rewriting historical evidence."""
import json, collections, statistics
from pathlib import Path
SRC=Path(__file__).resolve().parents[2];BASE=SRC/'docs/benchmarks/issue-173';DATA=BASE/'iteration-3-framed'
def metric(rows):
 gs=[g for r in rows for g in r['groups']]
 return {'groups':len(gs),'gold_exact':sum(g['actual']==g['expected'] for g in gs),'false_feat_fix':sum(g['actual'] in ['feat','fix'] and g['actual']!=g['expected'] for g in gs),'type_counts':dict(collections.Counter(g['actual'] for g in gs)),'fallback':{t:{'selected':sum(g['actual']==t for g in gs),'correct':sum(g['actual']==t and g['expected']==t for g in gs),'false':sum(g['actual']==t and g['expected']!=t for g in gs),'missed':sum(g['actual']!=t and g['expected']==t for g in gs)} for t in ['update','change']}}
def main():
 rows=[json.loads(s) for s in (DATA/'results.jsonl').read_text().splitlines()];old=[json.loads(s) for s in (BASE/'category-results.jsonl').read_text().splitlines() if json.loads(s)['arm']=='candidate'];assert len(rows)==25 and (DATA/'completion.json').exists()
 out={'existing':{'old':metric([r for r in old if len(r['case'])==1]),'short':metric([r for r in rows if len(r['case'])==1])},'boundary':{'old':metric([r for r in old if len(r['case'])>1]),'short':metric([r for r in rows if len(r['case'])>1])},'schema_valid':sum(r['schema_valid'] for r in rows),'classification_calls':25,'rejected_transport_calls':25,'tokens':'unavailable','historical_comparison':True,'product_changed':False}
 out['supported']=out['existing']['short']['gold_exact']>out['existing']['old']['gold_exact'] and out['boundary']['short']['gold_exact']>=out['boundary']['old']['gold_exact'] and sum(out[k]['short']['false_feat_fix'] for k in ['existing','boundary'])<=sum(out[k]['old']['false_feat_fix'] for k in ['existing','boundary'])
 (DATA/'analysis.json').write_text(json.dumps(out,indent=2)+'\n')
 lines=['## 要約','',f'指定されたCategory短縮Promptを既存25条件で各1回検証した。改善仮説は{"支持" if out["supported"] else "不支持"}。本番採用は行わない。','', '## 検証結果','','| 対象 | 前回候補 | 短縮版 |','| --- | ---: | ---: |']
 for name,key in [('既存Gold一致','existing'),('境界Gold一致','boundary')]:
  x=out[key];lines.append(f'| {name} | {x["old"]["gold_exact"]}/{x["old"]["groups"]} | {x["short"]["gold_exact"]}/{x["short"]["groups"]} |')
 lines += ['',f'- Category Schema適合 {out["schema_valid"]}/25。入出力token件数は未取得。', '- 最初の25送信はIPCの末尾改行欠落により推論前にmalformed_requestで拒否された。原記録はiteration-3に保存。送信形式だけを修正し、同一Prompt・入力・Goldの別記録iteration-3-framedで各条件1回推論した。helper送信計50件、分類用推論25件。失敗した送信を分類失敗として集計していない。','- Raw Diff、関連テスト、Schema（update/changeを含むenum）、breaking evidence契約、安全指示、モデル・samplingは固定。Grouping/Text再推論0、本番コード変更0。','- モデル・helper hashを推論前に照合し、入力・Gold・履歴・実行スクリプトのhashを前後照合。過去結果とGoldは未変更。','- 前回候補の履歴との比較であり、今回同時に実行した対照群ではない。既知のエージェント作成synthetic Goldであり、独立した人間評価や未使用holdoutではない。','','| 条件 | Gold | 前回候補 | 短縮版 |','| --- | --- | --- | --- |']
 om={r['case']:r for r in old}
 for r in rows:
  for g in r['groups']:
   previous=next(x['actual'] for x in om[r['case']]['groups'] if x['group_id']==g['group_id']);lines.append(f'| {r["case"]}/{g["group_id"]} | {g["expected"]} | {previous} | {g["actual"]} |')
 lines += ['', 'fallback詳細：','', '```json',json.dumps({k:v['short']['fallback'] for k,v in out.items() if k in ['existing','boundary']},ensure_ascii=False,indent=2),'```','',f'Category wall中央値 {statistics.median(r["wall_seconds"] for r in rows):.3f}秒、最大 {max(r["wall_seconds"] for r in rows):.3f}秒。helper peak RSS最大 {max(r["helper_peak_rss_bytes"] or 0 for r in rows)/1024**3:.3f} GiB。RSSはMetal/GPU総ピークではない。','', '## 考察','','Promptの意味判断部分だけを変更した歴史的比較である。条件ごとの変化と誤断定・fallbackの正誤を別に評価し、断定回避を正答増加と数えない。今回の結果から未使用入力への一般化や完成planの改善は主張しない。Textと共通deadlineは今回検証していない。','', '## Next Steps','','- 今回の結果を候補の検証結果として保持し、mainへ反映しない。','- 次の検証を行う場合は別の有限契約を固定する。既存Goldや過去測定を修正せず、既知fixtureの反復による改善を一般化しない。','']
 (DATA/'report.md').write_text('\n'.join(lines));print(json.dumps(out,indent=2))
if __name__=='__main__':main()
