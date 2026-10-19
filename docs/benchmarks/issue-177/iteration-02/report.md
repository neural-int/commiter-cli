## 要約

評価用buildだけで共有期限を120秒から360秒にすると、元のC02を両条件ともTextまで評価できた。しかしC02-Aはscope=docs、Bはscopeの文字数超過で拒否され、Groupingも参照と不一致だった。期限延長は本番の既定値へ採用しない。ユーザーの選択に従い、出力契約と意味品質は別Issue #178へ分離した。#177の「C02を両条件で最終Validatorまで実行」は未達として残す。

## 検証結果

| ケース | 条件 | Grouping / Category / Text（秒） | cycle（秒） | 最終結果 |
| --- | --- | --- | ---: | --- |
| C01 | A ログなし | 28.80 / 22.82 / 8.17 | 59.79 | Validator受理。type=refactor、要約は404修正を説明せず意味不一致 |
| C01 | B ログあり | 29.28 / 30.69 / 17.03 | 77.02 | type=fix、Textの意味は参照一致。summary51文字>48でinvalid_schema |
| C02 | A | 112.26 / 96.15 / 70.83 | 279.24 | 3群で参照1群と不一致。scope=docsでinvalid_scope |
| C02 | B | 112.43 / 99.68 / 84.34 | 296.46 | Aと同じ3群。scope38/28/36文字>16でinvalid_schema |

2ケース×A/B×1、生成要求12、retry/repair/追加採点モデル0。全cycleが360秒内に終了した。実験全体は約714秒だった。元の120秒条件は約364秒でC02のCategoryがtimeoutしており、今回を速度改善として扱わない。計測専用Go build overlayが変更したのはCandidateCycleTimeoutだけで、本番ソースとSRSの120秒は維持した。

120秒条件で既に到達した全10要求と今回の同じ要求は、保存したJSONのbytesが一致した。新たに到達したC02 Textでは、それぞれのCategory出力をそのまま渡している。A/Bのtypeは一致させていない。既存capture、Gold、ソース、overlayのハッシュ、fixtureのHEAD/index/全作業ファイルbytesを照合し、不変を確認した。

事後の生成なしtokenize-only計測では、実際のC02入力はA=17,362/17,707/17,611、B=18,113/18,458/18,363 token。いずれも出力予約を加えて実生成の64k枠内だった。旧tokenizer helperは自身の16k上限で拒否した状態からcountを得ており、成功した本番受理として捏造しない。本番helperは各実行で実際のcontextを独立に検証している。出力token数やload/prefill/reasoning/generationの時間内訳は本番protocolにないため未取得であり、0として解釈しない。

完了した12helperの最大RSSは3,857,776,640 bytes（約3.59GiB）。GPU/Metal peakは未計測。16GiB機のsystem snapshotを記録したが、他のworkloadと既存swap使用を制御しておらず、resourceの十分な余裕や処理単独のswap増分は推定しない。終了後のnative helper残存processは0だった。全実行でGit mutationを行っていない。

## 考察

入力受理改善でC01のTextまでの評価が可能になり、C02も圧縮拒否から実推論へ進んだ。共有360秒ではC02の全phaseの出力が得られたため、120秒の残時間不足が後続phaseを妨げていたことを確認した。一方、約4.7–4.9分を要し、最終planは受理されなかった。長い期限をそのまま既定にする根拠は不足している。

C01-Bは意味一致と出力契約不合格が同時に成立する。C02は所属不一致のため、個別type/Textを1群の参照に対する正解として採点しない。AのG001は文書と実装を含むが、要約は文書化を述べている。Bのscopeは全文パスとなっており16文字制約を満たさない。いずれも事後切り詰め・再生成を行っていない。

採否は分ける。全文保持の入力表現選択と非圧縮受理は限定した修正候補としてbranch上に保持する。360秒の期限変更は実験overlayだけに留める。元の2ケースだけによるproduction品質の採用判断はしない。Goldも親から固定したagent作成の探索的参照であり、独立した人のGold評価ではない。

開発用のgo test ./...、go vet ./...、go build ./...は入力改善ソースで成功済み。同じソースのSHAは今回の実測後も一致し、overlayのThreePhase関連テストも成功。Python compileとdiff checkも成功した。既存single-pass圧縮テストは現行動作に必要なため残し、追加テストは実際の入力膨張・保存境界・非圧縮予算拒否に限定した。IssueのVerificationは未完了なのでGoalの最終品質ゲート完了やIssue完了を宣言しない。

## Next Steps

- #178でsummary上限、scope上限、scope=docsの拒否の発生箇所・契約を調べる。入力・時間制限から独立した不合格が確認でき、ユーザーが別Issueへ分離する方針を選んだため。
- #178で元fixtureと参照を保持したままGrouping/Category/Textの意味品質を評価する。今回のC02所属不一致を期限延長で解決したと扱わないため。
- #177のC02最終Validator到達条件は未チェックで維持し、#178での出力契約修正後に再確認する。元の成功条件を事後に緩めないため。
- #175の2ケースの探索的意味評価を今回の結果へ接続する。広いケース網羅・独立Gold・production採用は未実施として残すため。
