# H182-012 32K固定比較（停止）

## Goal

前回fixtureを再利用し全フェーズを32768固定で比較する。

## Result / Decision

18条件中6条件確定、7条件目Coder 0.5B・文法制約なしのText生成中に、生成だけで33,856トークンを超えた。要求context_tokensは32768だが、研究用ランタイムはmaxTokens=nilで、入力時のTokenBudget検査後に合計コンテキスト長による終了判定がない。入力投影の32K指定は成立しても入力＋生成の上限は成立していない。
異常検出のためランナーへSIGINTを送りfinallyで子プロセスを停止。再試行・条件変更なし。helper/evaluationの終了確認済み。完了結果としてのモデル比較は確定しない。

## Conditions / Evidence

Grouping実要求32768、コマンド14件と会話3件を保持。前回と同じRelease helper・固定fixture・監視。原本と前回結果を保持。completionのKeyboardInterruptは操作上の理由で、根本原因は生成中のコンテキスト上限未適用。

## Measurements

停止まで1611.482秒、監視803件、監視fatalなし。確定セルcalls合計16は中断セルを含まない。時間・CPU・GPU・メモリ・熱ログを保持。

| 条件 | 秒 | calls | validator | 正解群/3 |
| --- | ---: | ---: | --- | --- |
| qwen25-05-json_schema | 13.64 | 3 | False | 0 |
| qwen25-05-none | 5.61 | 1 | False | 未取得 |
| qwen25-15-none | 30.22 | 3 | False | 0 |
| qwen25-15-json_schema | 27.80 | 3 | True | 0 |
| qwen25-3-json_schema | 87.46 | 3 | True | 1 |
| qwen25-3-none | 81.76 | 3 | False | 1 |

## Next

入力＋生成の32K上限を実装・検証してから別runで再試行する。現在は再試行していない。
