## 要約

H5は独立評価16fileと共通callee独立guardrailを含む7/9 observationsで一致したが、weak16でFM120、逆順12fileでFS24を観測した。非意味的な提示順への依存と、test契約がない入力の誤統合が残り、H5をproduction candidateにしない。

## 検証結果

| fixture / 条件 | files | exact | FM / FS | calls | input / output tokens | wall s |
| --- | ---: | --- | --- | ---: | --- | ---: |
| baseline4 | 4 | true | 0 / 0 | 1 | 1354 / 561 | 28.194 |
| controlled8 | 8 | true | 0 / 0 | 1 | 1439 / 594 | 30.428 |
| controlled9 | 9 | true | 0 / 0 | 1 | 1717 / 604 | 42.222 |
| guardrail6 | 6 | true | 0 / 0 | 1 | 2008 / 574 | 30.498 |
| holdout16 | 16 | true | 0 / 0 | 1 | 4923 / 674 | 40.291 |
| holdout6 | 6 | true | 0 / 0 | 1 | 1979 / 587 | 30.322 |
| missing16 | 16 | true | 0 / 0 | 1 | 2791 / 707 | 54.955 |
| reversed12 | 12 | false | 0 / 24 | 1 | 4590 / 634 | 50.151 |
| weak16 | 16 | false | 120 / 0 | 1 | 2599 / 674 | 48.105 |

全9calls completed、全件一意割当9/9、unresolved0、context overflow/timeout/incomplete0、unknown/duplicate/missing selected ID0。metadata未実行。逆順12fileは同じsource/selected IDs/gold/profile/modelでpresentationだけを反転し、input4590/output634 tokensは順方向と同じだった。groupingは6つのsource/test組へ変化した。

holdout16は8つの独立source/test契約で全て一致したが、既存5目的と新規3目的の合成であり、全内容が未使用だったとは主張しない。shared-callee guardrailはH4評価前にsource/goldを固定し、H5選定にはその結果を使用していない。controlled8/9/missing16/weak16は#146の既使用・理想化graphを含む定数変更probeで、実開発者の意図やproduction精度推定には使わない。weak16のFM120は固定参照との不一致として保持する。

## 考察

test contract IRは既使用cross-boundary12と実行可能な独立評価で改善を示した一方、入力のpresentationを変えるだけで12fileの一致が崩れた。全件割当の厳密性や一方向の成功だけでは品質条件を満たさない。

weak16はsupported test assertionがない定数変更で、16目的の固定参照を同じ「全定数を増加」のgroupへ統合した。これを速度や完全割当と引き換えに採用しない。現在のIR coverageの限界とglobal semantic taskのgeneration budgetを分けて検討する。

## Next Steps

- 現行production adapterと同じsource-first/path基準でmodel inputをcanonical化し、opaque selected IDをpath順で安定したmodel IDへ写像する。外部入力のpresentationをmodel判断の交絡にしないため。
- Go declarationのkind/name/before-after observed valueをIRへ追加する。testがない変更もproseによる推測ではなく観測entityとして提示するため。
- 同じ12/weak16を最小回数で測り、改善がなければglobal grouping専用のbounded generation contractまたはcached grouping-only model routingを比較する。4file向け固定思考枠を多ファイルsemantic taskへそのまま拡大する設計が十分かを確認するため。
- metadata adapterは現行日本語4fileのmessage/schema/profile/budget同一性と不正group拒否をhost testで確認済みだが、H5の採用資格不成立のため実モデルの最終plan成功をまだ主張しない。
