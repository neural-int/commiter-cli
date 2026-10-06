## 要約

H23固定の残る11range/regressionを各1回測定し7一致/4不一致。same-directory5 FM10、cross-directory9 FS32、boundary12 FS30、protocol8 FS12。資格/新独立評価の成功は全評価へ一般化できず、H23をproduction candidateへ採用しない。

## 検証結果

| fixture | files | exact | FM | FS | complete | unresolved | input tokens | output tokens | total wall秒 |
|---|---:|---|---:|---:|---|---|---:|---:|---:|
| contract-baseline-4 | 4 | true | 0 | 0 | true | false | 1533 | 54 | 13.356 |
| contract-independent-6 | 6 | true | 0 | 0 | true | false | 2200 | 80 | 18.621 |
| baseline-4 | 4 | true | 0 | 0 | true | false | 970 | 54 | 9.997 |
| same-directory-independent-5 | 5 | false | 10 | 0 | true | false | 903 | 67 | 10.195 |
| implementation-tests-8 | 8 | true | 0 | 0 | true | false | 1878 | 106 | 18.894 |
| cross-directory-single-intent-9 | 9 | false | 0 | 32 | true | false | 2172 | 119 | 22.497 |
| multiple-intents-boundary-12 | 12 | false | 0 | 30 | true | false | 2982 | 158 | 37.635 |
| missing-edges-single-intent-16 | 16 | true | 0 | 0 | true | false | 3594 | 210 | 59.390 |
| holdout-independent-6 | 6 | true | 0 | 0 | true | false | 2164 | 80 | 25.850 |
| holdout-independent-16 | 16 | true | 0 | 0 | true | false | 5338 | 210 | 79.524 |
| holdout-protocol-and-health-8 | 8 | false | 0 | 12 | true | false | 2977 | 106 | 35.466 |

全11case各1call/1回、順次。全complete=true/unresolved=false、全backend completed、timeout/context overflow/incomplete output0。unknown/duplicate/missing ID0。FM合計10/FS合計74、exact7/11。集計はsummary.json。選定使用済みholdoutはfreshとして数えず、新wire8fileの初回独立結果はIteration34に別記した。再試行/gold変更/rule tuningなし。

fixed H23/Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192/native0/output1536/context16K/call120秒/whole600秒/temp0/top_p1/top_k0/seed144/retry0/repair0。architecture/schema/generator/code変更なし、tests併走なし。事前登録のgrouping全range通過条件を満たさず、候補/file順診断とmetadata/Validateは実行していない。追加取得/依存/production変更なし。artifact diff check成功。

## 考察

H23は資格3case+独立wire8fileで改善を観測したが、small same-directory分離とcross-directory統合の両方で未達が残る。16fileの独立/欠落relationで成功しても、小さいfile setを安全に扱えるとは限らない。schema完成/予算内/全件割当が意味品質を保証しないことを再確認した。

提示順を変えて良い結果を選ぶ、5fileだけ別route、goldの一括目的への変更、候補に正解を手動注入するなどで採用を成立させない。global proposalsは今回のnew holdoutで候補外partitionを生成できたが、任意目的の識別能力不足は解消していない。quality条件を緩めたfile上限拡大は行わない。

## Next Steps

- H23の資格/独立成功と4regression不一致をcompletion auditへ追加し、試験済みarchitecture familyと現在capabilityのno-candidate判断を監査する。改善例を隠さず、成功例だけで採用を決めないため。
- 残余案はfailureへ作用する新情報/責務差/実行可能なbounded試験の根拠と、既存Issueで否定済み方式との違いを要求する。根拠ある案が残れば継続、残らなければ現在条件での不採用と再開prerequisiteを明文化する。単なるwording/budget/組合せ反復で探索を延長しないため。
- Issueのcandidate-found/no-candidate各条件に証拠を対応付け、Verification達成を実証できる場合だけ最終品質ゲートへ進む。未達をN/Aや測定件数で達成へ変換しない。
