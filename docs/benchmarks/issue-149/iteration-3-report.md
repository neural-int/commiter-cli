## 要約

bounded batch抽出は12fileのFSを24から8へ減らしcallsも13から4へ減らしたが、期待groupingに一致しなかった。6fileは必須IR項目が空で停止した。raw-global対照でも12fileの同じFS8が残り、IR圧縮だけを原因と断定できない。H2とraw-globalをproduction candidateにしない。

## 検証結果

| architecture / fixture | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | ---: | --- | ---: |
| batch IR / independent6 | null | null / null | false | true | 2 | 1888 / 933 | 42.291 |
| batch IR / cross-boundary12 | false | 0 / 8 | true | false | 4 | 6471 / 2394 | 151.115 |
| batch IR診断 / independent6 | null | null / null | false | true | 2 | 1888 / 933 | 42.357 |
| raw-global / independent6 | true | 0 / 0 | true | false | 1 | 1609 / 587 | 30.831 |
| raw-global / cross-boundary12 | false | 0 / 8 | true | false | 1 | 3542 / 634 | 44.708 |

全10callsはcompleted。context overflow/timeout/incomplete0。batch6は2回目の抽出後にinvalid_ir、生成条件を変えない追加診断はinvalid_ir_emptyを返した。どのIR項目か、未知項目の置換か空文字そのものかはこの分類だけでは確定しない。global/metadataは実行していない。生成本文は保存していない。構造失敗をFM0として集計しない。

batch12とraw12はF001〜F006をjoin、F007/F008とF009〜F012を別groupにした。root billing/centsの修正とorders/receiptsのconsumer修正の間にFS8が残った。診断はpostprocessing分類だけを変更し、prompt/schema/profile/payloadを変更していない。batch6の両runはinput/output tokensも一致したが、全architectureの反復安定性の証明ではない。

H2抽出はbounded-text/native0、globalはbounded-grouping/native512。per-file方式とはbatchとgeneration contractの両方が異なり、改善をbatch単独へ帰属しない。

## 考察

raw diffを全件提示しても同じ12fileの分割になったため、抽出による情報欠落だけでは残る失敗を説明できない。shared rounding contractと変更されたconsumerの依存が、入力表現からgrouping判断へ十分に伝わらない可能性はあるが、modelの内部理由は測定していない。

追加callによるpair監査は#146で整合した誤判断を修正できなかったため再試行しない。必須IR項目の欠落を推測で補完したり、dependency edgeをhard unionへ変えたりしない。H2の費用改善は観測できたが、独立6fileの構造資格とcross-boundary12fileの意味資格を同時に満たしていない。

## Next Steps

- H3としてhostが差分のbefore/afterをliteral factsへ分解し、既存parser/extractorのsoft relationとともに全件global判断へ渡す。LLMのprose抽出を必須経路から外し、事実の欠落と追加抽出callsを避けつつ依存の提示境界を再検討するため。
- 同じ6/12fileを各1回測る。原始差分・typed facts・learned IRのいずれが既知失敗を改善するか、architecture全体の成立を比較するため。
- H3が有望なら4file比較、controlled8/9/16、独立6/16と順序逆転を測り、metadata/Validate接続を検証する。既知fixtureだけの改善からcandidateへ進めないため。
- H3でも共有契約の判断が残る場合、変更契約のsemantic dependencyを観測する別責務またはgrouping専用model契約が必要かを整理する。現在の入力やモデルの天井を少数の失敗だけで断定しないため。
