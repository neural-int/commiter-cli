## 要約

Bは **NO-GO（今回固定したrepository/history evidence設計）**。修復後の全18比較はcomplete assignmentを受理したが、fresh独立2ケースでA-onlyを改善したsourceはない。repository追加は独立変更でFM12、history追加はFS3のregressionを起こした。#150のgateに従いCへ進まない。

## 検証結果

同一cached Qwen3-8B/model/helper/semantic task/schema/budgetで、A-only、repository、historyを各case各1回比較した。goldはmodel payloadに含めていない。

| fixture | source | exact | FM | FS | complete | input tokens | output tokens | wall秒 |
|---|---|---|---:|---:|---|---:|---:|---:|
| same-directory-independent-5 | A-only | True | 0 | 0 | True | 1307 | 81 | 13.999 |
| same-directory-independent-5 | repository | True | 0 | 0 | True | 1364 | 81 | 14.042 |
| same-directory-independent-5 | history | True | 0 | 0 | True | 1644 | 81 | 15.183 |
| cross-directory-single-intent-9 | A-only | False | 0 | 36 | True | 2718 | 133 | 26.148 |
| cross-directory-single-intent-9 | repository | False | 0 | 36 | True | 2775 | 133 | 26.919 |
| cross-directory-single-intent-9 | history | False | 0 | 20 | True | 3861 | 133 | 36.132 |
| multiple-intents-boundary-12 | A-only | False | 0 | 30 | True | 3861 | 172 | 39.898 |
| multiple-intents-boundary-12 | repository | False | 0 | 9 | True | 3919 | 172 | 44.640 |
| multiple-intents-boundary-12 | history | True | 0 | 0 | True | 5934 | 172 | 74.342 |
| shared-callee-independent-6 | A-only | False | 6 | 3 | True | 2247 | 94 | 25.657 |
| shared-callee-independent-6 | repository | False | 3 | 3 | True | 2461 | 94 | 28.112 |
| shared-callee-independent-6 | history | False | 6 | 3 | True | 2739 | 94 | 30.590 |
| fresh152-unchanged-adapters-6 | A-only | False | 0 | 3 | True | 2097 | 94 | 25.932 |
| fresh152-unchanged-adapters-6 | repository | False | 0 | 3 | True | 2425 | 94 | 28.085 |
| fresh152-unchanged-adapters-6 | history | False | 0 | 3 | True | 2589 | 94 | 29.264 |
| fresh152-common-helper-independent-6 | A-only | True | 0 | 0 | True | 2178 | 94 | 26.336 |
| fresh152-common-helper-independent-6 | repository | False | 12 | 0 | True | 2454 | 94 | 28.040 |
| fresh152-common-helper-independent-6 | history | False | 0 | 3 | True | 2670 | 94 | 29.900 |

| source | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| A-only | 2/6 | 6 | 72 | 6/6 | 0 | 6 | 14408 | 668 | 157.969 |
| repository | 1/6 | 15 | 51 | 6/6 | 0 | 6 | 15398 | 668 | 169.839 |
| history | 2/6 | 6 | 29 | 6/6 | 0 | 6 | 19437 | 668 | 215.412 |

- 全18行backend completed / host accepted / complete=true / unresolved=false。timeout/context overflowなし、repair/retryなし。actual token telemetryはhelperのinput.text.tokens.size / generationTokenCount。
- schema未伝達だったIteration1の6失敗行、7行目の中断attempt、最小診断1callは別記録。修復後18行とのaggregate混在なし。研究全体は26開始attempt（25completed measurement/diagnostic + 1中断）。中断callのtokens/wallは取得不能。
- qualificationのsame-directory5 / cross-directory9 / multiple-boundary12のrepository graphは抽出edge0、parse unavailable0/4/6。boundary12のFS30→9を新semantic relationの効果とは断定できない。
- shared-callee6では直接call5edges、そのうちgoldで同一intent3/異intent2。structural callのfalse-positive relation2/5。関連性はshared purposeを保証しない。
- fresh adaptersではunchanged sourceの3adapterが別々のchanged functionへ接続するsource factsを追加したが、A-only/repository/historyはいずれもFS3。fresh common-helperはA-only exact、repository FM12、history FS3。
- historyは固定mixed housekeepingを実Git commitから抽出したsynthetic data。root/restore commitにより全selected pairがco-changeを持つ。頻度をsemantic ground truthにしない。実repository history分布への外挿はできない。
- extraction/context費用はiteration-2-summary.json。model inferenceのwallにmodel loadを含む。context_bytesはpayload bytesで、token budget/input tokensにはschema/systemも含む。history repository生成費用は計測外、既存Git historyのscan費用をextraction_secondsに計上。

## 考察

qualification集計ではrepository/historyのFS改善が見られる。しかしindependentケースではrepositoryが12pairのfalse mergeを追加し、historyが3pairのfalse splitを追加した。合計の改善だけでは一般化gateを通せない。特にcommon-helperの構造的関連を、別purposeの変更を統合する根拠へ読み替えた出力を観測した。モデルの内的理由を推測して断定しない。

この結果は固定source表現・固定generic model・6synthetic fixture・各1回の判定であり、repository/history evidence一般が無効という証明ではない。選択sourceの有効性が十分でなく、専用scorerだけが残存bottleneckという条件も立証していない。Aのsource mapping/staging能力は維持されるが、AのGOをsemantic品質の証拠には転用しない。

Bの達成条件は、independent improvementとより大きなregression回避が未達。rename/new-file/sparse-history/real-size memory/cache評価の未実施項目も達成扱いしない。B全checkboxや親#150全条件をチェックしない。production baseline比較/最終C holdout/optimizerは依存gate未通過のため未実行。Goal最終品質ゲートの開始条件を満たしていない。

## Next Steps

- #150 pipelineをB No-Goで停止し、親Issueへ測定と未達条件を記録する。依存gateを迂回してCを開始しないため。
- C未実施とdedicated scorer必要性未立証を明示し、Dを作成しない。残存bottleneckを過大解釈しないため。
- 固定fixtureへのrouting、wording、gold/threshold変更で採用を成立させない。再開には新しいbounded evidence仮説と、その仮説に情報追加がある根拠、未使用independent評価を別途必要とする。
- artifact/codeのfocused checksとcommit/push/remote head確認を完了し、配送記録を残す。停止判断と未配送状態を区別するため。
