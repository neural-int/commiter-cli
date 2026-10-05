## 要約

H9のhost保持観測anchorへのassignmentはweak16でexact/FM0/FS0となり、自由文contractより出力tokensとwallが減った。一方cross12では6つのsource/test pairに分断しFS24だった。構造処理を軽くしてもglobal shared contractをまとめるsemantic failureが残るため採用しない。

## 検証結果

| fixture | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | ---: | --- | ---: |
| weak16 | true | 0 / 0 | true | false | 1 | 2640 / 192 | 13.702 |
| cross12 | false | 0 / 24 | true | false | 1 | 4657 / 176 | 17.154 |

両calls completed、unknown/duplicate/missing selected ID0、timeout/context overflow0。cross12は6pairs、weak16は16singleton。metadata未実行。contradictory anchor/cycle、unknown root/file、missing/unresolvedを拒否するhost testsがpass。AST宣言が解決できないpartial snippetはdiff-literalとして原before/afterを保持し、全fileを観測対象へ残すtestがpass。packageや完全programを推測していない。fresh protocol8は資格不成立のため未推論。

## 考察

hostが観測rootの実体を保持する役割分担により、引用coverageを再生成する構造負荷は除けた。しかしrootの実在と正しいglobal groupingは別であり、1つの期限契約を構成するsource・consumer・testsが別rootへ分かれた。source/test local boundaryを採用する理由にはできない。最終groupはglobal出力から作り、soft relationをhard constraint化していない。

H8 cross12では根拠coverage gateで止まり、draftのsemantic groupingは未計測だった。ここを確認せずにH8のhost/model分担を更に変更すると、単なる構造失敗の修理なのかsemantic failureの反復なのかを識別できない。

## Next Steps

- H8 cross12を同条件で1回だけdiagnostic計測し、draft partitionの全件性とexact/FM/FSを、final gateの結果とは別に記録する。以前のraw responseはRAMから破棄済みで回復不能のため。この診断以外に同条件反復を増やさない。
- draftがsemantic条件を満たす場合のみ、hostが全memberの観測を直接付与し、modelへ引用bookkeepingを要求しない責務分割を検証する。観測の実在・coverageを保ち、品質gateを緩和する変更と混同しないため。
- draft自体が失敗する場合は、残る根本的なsemantic能力の仮説と、その検証に必要な未導入capabilityを明記する。installed checkpointの既知失敗をpromptや順序調整で繰り返さず、採用条件の緩和もしないため。
