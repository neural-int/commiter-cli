## 要約

test assertionのinputとbefore/after期待条件を観測IRへ追加したH5は、既使用6/12fileの両方で固定参照に一致した。12fileで初めてFS0を観測した。採用はまだ行わず、独立評価・共通callee guardrail・大規模・順序・metadata/Validateへ進む。

## 検証結果

| H5 fixture | files | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | ---: | --- | --- | --- | --- | ---: | --- | ---: |
| contract-independent-6 | 6 | true | 0 / 0 | true | false | 1 | 1994 / 574 | 29.675 |
| contract-cross-boundary-12 | 12 | true | 0 / 0 | true | false | 1 | 4590 / 634 | 37.754 |

全2calls completed、context overflow/timeout/incomplete0、unknown/duplicate/missing selected ID0。12fileはauth/gateway/storageのF001〜F006、billing/receipts/ordersのF007〜F012へ確定した。metadata未実行、plan成功ではない。

抽出はmodel outputやgold labelを使わず、Go testの直接Fatal分岐からsourceのcall/input/expected conditionを抽出した。roundingの3つのtestは同じ引数1.999、before期待199、after期待200を保持するhost testを通過した。これらはsource assertionの観測で、runtimeの追加観測やtype checkingの成立証明ではない。

## 考察

caller/calleeだけでは届かなかった既使用12fileで、振舞い変更を明示したrepresentationに一致例が得られた。prompt/profile/modelはH4と同じだが、source literal、call facts、test assertionsの複合入力の結果であり、一般的なtest IRの優位性や未知入力の品質は未確認。Goの対応assertion shapeがない入力では同じ成功を保証できない。

## Next Steps

- 共通callee独立guardrailと未調整holdout6を各1回測る。依存の存在から独立目的を誤統合していないかを確認するため。
- 資格が維持されれば独立16、current4比較、controlled8/9/16、順序逆転を測る。既使用6/12だけの成功から規模や非意味的要因の安定性を推定しないため。
- 選定条件を満たす場合のみfinal global groupingの後に現行metadata generationを接続しplanning.Validateを確認する。割当の成功を最終plan成功と混同しないため。
- guardrail/holdoutが失敗する場合はgoldやIR抽出をそのfixtureへ調整せず、観測coverageまたはgrouping専用routingの責務を検討する。独立評価を調整用へ使った後に未使用holdoutとして再使用しないため。
