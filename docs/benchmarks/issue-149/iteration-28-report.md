## 要約

H20としてcandidate-recordsを既存Gemma/native0で固定評価。全3caseでall-files候補を選択しweak16 FM120/cross12 FM36/guardrail FM12。採用しない。

## 検証結果

| fixture | exact | FM | FS | complete | unresolved | input tokens | output tokens | total wall秒 |
|---|---|---:|---:|---|---|---:|---:|---:|
| weak-edges-independent-16 | false | 120 | 0 | true | false | 1494 | 19 | 6.065 |
| contract-cross-boundary-12 | false | 36 | 0 | true | false | 4071 | 19 | 10.732 |
| shared-callee-independent-6 | false | 12 | 0 | true | false | 2072 | 19 | 6.695 |

各case1call/1回、全backend completed、timeout/context overflow/incomplete outputなし。unknown/duplicate/missing file IDなし。candidate generator/input/task/schema/orderはH19と同じ。canonical候補のall-filesは各case先頭で、全3caseそれを選択した。これは観測であり、位置バイアスや内部選択理由の証明ではない。

fixed Gemma revision475b9088d29754a3379866cf5aeb6b41acd313c2、bounded-routed-grouping/native0/output1536/context16K/call120秒/whole600秒/temp0/top_p1/top_k0/seed144/retry0/repair0。helper/source変更なし。observer cross12 4probes/16samples/unknown0、guardrail2/8/0、weak0。追加取得/依存/code/production変更なし。fresh protocol8/metadata未実行。コード変更がないため前回成功したbenchmark tests/vet/buildを繰り返さず、artifact diff checkを確認。

## 考察

Qwen8ではcross12/guardrail一致、weak16未解決。Gemmaでは3case全て誤結合。正解候補の包含とselector品質は独立であり、候補生成に責務を移しても両品質の成立は保証できない。速い19token出力は採用根拠にならない。case別route選択、未解決fallback、all-files候補排除で成功へ変えない。

今回の3caseはcandidate包含が成功したが、generatorはsyntax関係の少数familyであり、意味上のcross-directory目的を任意入力で網羅できると確認していない。selectorの改善だけを追って候補欠落を忘れることはできない。また全3caseが先頭候補選択でも、内容が同じall-filesだから位置効果を特定できない。

## Next Steps

- 次iterationは追加推論前に、固定candidate generatorの既存range/regression/holdout fixture全体でLLM0の包含診断を行う。今回3caseの包含成功を汎用generator成立と読み替えないため。ルールやgoldは固定し、欠落へ候補追加しない。
- 候補順の影響は、同partition集合/元観測で候補の提示順だけを反転した診断として事前固定する。測定する場合も各case/route1回までで、良い順を選んで採用する用途にしない。先頭候補選択という観測から因果を推測しないため。
- generator/selector/提示安定性の限界を監査へ追加し、既存capabilityで根拠ある未試験architectureが残るか、再開に何が必要かを具体化する。有限の失敗を普遍的不可能性とせず、追加試験を微調整ループにしないため。
