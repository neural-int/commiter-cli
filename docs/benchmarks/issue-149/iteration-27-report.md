## 要約

H19のgold非参照candidate生成は3case全て正解partitionを含んだ。固定8Bによるglobal selectionではcross12/guardrailが一致したが、weak16はunresolvedで停止。採用資格は未成立。

## 検証結果

LLM0包含診断でweak16 2候補/cross12 4候補/guardrail3候補、全包含。generatorはsingleton、観測source/test component、同対応+source calls component、all-filesをcanonical化・重複除去。gold/name/order変更で候補不変のtest成功。元goldに合わせた手動追加なし。

| fixture | exact | FM | FS | complete | unresolved | input tokens | output tokens | total wall秒 |
|---|---|---:|---:|---|---|---:|---:|---:|
| weak-edges-independent-16 | None | None | None | False | True | 1457 | 10 | 10.427 |
| contract-cross-boundary-12 | True | 0 | 0 | True | False | 3675 | 8 | 23.200 |
| shared-callee-independent-6 | True | 0 | 0 | True | False | 1859 | 12 | 12.821 |

各case1call/1回、全backend completed、timeout/context overflow/incomplete outputなし。weak16のexact/FM/FSはnull・未評価であり0成功扱いしない。cross12は2group、guardrailは3implementation/test pair。observer4/16/0、2/8/0、weak0。candidate text/raw prompt/思考は保存なし。

fixed Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192、native0/output1536/context16K/call120秒/whole600秒/temp0/top_p1/top_k0/seed144/retry0/repair0。unknown/unresolved selectorはgroupsなしで停止。structural gateとsemantic測定を分離。追加取得/依存/production変更なし。fresh protocol8/metadata未実行。

focused candidate/gold-order/unknown-unresolved/既存strict gate tests、benchmark vet/build、diff check成功。`go test ./tools/benchmark149`も43.764秒で成功。

## 考察

H17/H18のQwen8 cross12 FS30に対して今回cross12 exactとなった。入力/schema/taskが同時に違うため候補selection単独の因果効果ではないが、このarchitecture組合せで品質改善の観測を得た。guardrailも一致し、call数1でoutput8〜12tokens。ただしweak16の未解決を単独fileに自動fallbackするとscope外の推測recoveryになるため行わない。候補包含成功は選択成功や未知入力包含の保証ではない。

Gemmaは同じhost観測のE-root taskでweak16一致/cross12 FS24だった。今回のbounded candidate comparisonはGemmaで未測定。今回cross12改善があるため、現行checkpointで同じ比較taskを固定評価する根拠があり、任意task/model総当たりとは分ける。fixtureごとのmodel選択は行わない。

## Next Steps

- H20として今回candidate生成/input/task/schemaを固定し、grouping-only routeを既存Gemma native0へ変更して3case各1回測定する。cross12改善が得られたcomparison taskと、weak16を区別できた既存checkpointの未測定組合せを独立評価するため。
- 全case同じrouteと既存budget、retry/repair0を維持。weak16だけfallback/別routeにせず、全資格通過時のみfresh protocol8以降へ進む。未解決を成功へ変換しないため。
- 失敗ならgenerator包含とselector失敗を分離して監査一覧へ追加し、合理的残余/具体的再開prerequisiteを再判定する。観測したcross12改善を採用条件全達成と混同しないため。
