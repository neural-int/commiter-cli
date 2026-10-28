# 実Git差分・保存Codex履歴の初回照合テスト

## Goal

Go製PoCが実データで内容対応を返せるかを確認する。モデル生成や本番接続なし。既存worktreeを書き換えず、過去の実コミット間diffと保存App Server履歴を固定した。現在worktreeのcaptureも別に確認。

## Result / Decision

合計17ケース（主比較13件、観測後に選んだ確認用positive control3件、現在worktree1件）。compareの実行回数は18回で、R10の入力拒否だけ初回driverと停止後の記録継続時に計2回確認した。主比較は12件ambiguous、1件入力拒否。positive control3件はmatched。現在worktreeは履歴受信上限でunavailable、Git入力保持・unmatched/fallbackを確認。精度改善・一般的正確率・本番採用を立証したものではない。

## Main cases

Git差分は保存履歴にある安全なpathを辞書順に選び、そのpathを変更した最新の実Git commitと親の差分を使用。結果を見る前に13件の入力hash・source/binary hash・Git revisionを私有planへ固定した。R01–R10は途中取得/除外を反映したpartial履歴、R11–R13は完全読取済み2Turnに限定した閉じた候補集合であり、全リポジトリ履歴の網羅性を主張しない。

| ケース | 条件 | ファイル | 判定 | 完全一致候補のあるblock数 | パス候補数 → 内容候補数 |
| --- | --- | --- | --- | ---: | ---: |
| R01 | strict_partial | docs/benchmarks/issue-182/follow-up-plan.md | {'ambiguous': 1} | 0 | 1 → 1 |
| R02 | strict_partial | docs/benchmarks/issue-182/ministral-official/goal.md | {'ambiguous': 1} | 0 | 6 → 6 |
| R03 | strict_partial | docs/benchmarks/issue-182/ministral-official/raw/README.md | {'ambiguous': 1} | 1 | 1 → 1 |
| R04 | strict_partial | docs/benchmarks/issue-182/prompt-candidate.md | {'ambiguous': 1} | 1 | 1 → 1 |
| R05 | strict_partial | docs/benchmarks/issue-182/prompt-comparison-goal.md | {'ambiguous': 1} | 0 | 1 → 1 |
| R06 | strict_partial | docs/benchmarks/issue-182/prompt-comparison-preflight.md | {'ambiguous': 1} | 1 | 1 → 1 |
| R07 | strict_partial | docs/benchmarks/issue-182/suite-v2/README.md | {'ambiguous': 1} | 0 | 3 → 3 |
| R08 | strict_partial | docs/benchmarks/issue-182/suite-v2/raw/README.md | {'ambiguous': 1} | 0 | 1 → 1 |
| R09 | strict_partial | docs/benchmarks/issue-182/suite-v2/results-report.md | {'ambiguous': 1} | 0 | 6 → 4 |
| R10 | strict_partial | tools/benchmark182/mistral_audit.py | input.schema:unsafe_or_duplicate_file | — | — → — |
| R11 | complete_turn_scope | docs/benchmarks/issue-182/suite-v2/README.md | {'ambiguous': 1} | 0 | 2 → 2 |
| R12 | complete_turn_scope | docs/benchmarks/issue-182/suite-v2/raw/README.md | {'ambiguous': 1} | 0 | 1 → 1 |
| R13 | complete_turn_scope | docs/benchmarks/issue-182/suite-v2/results-report.md | {'ambiguous': 1} | 0 | 4 → 2 |

R10ではGit由来のソース差分を既存機密値検出が拒否した。検出された文字列や値は公開しない。真の秘密か検出のfalse positiveかは今回未判定。初回driverはこのエラーで停止したため、R01–R09の結果を再実行せず再利用し、固定済みR10の拒否を同じ入力で再確認してエラーを保存し、残りR11–R13を継続した。先の9件の処理時間はdriver終了で失われたため掲載しない。入力や判定ルールは成功させるために変更していない。

## Positive diagnostic controls

R03/R04/R06で完全一致候補を観測した後、その対象ファイルと元の2Turnだけを明示し、同じApp Serverの保存履歴を改めて取得。各Turnの全itemページが5秒/4MiB枠内に収まり、対象pathのdiffと依頼/最終応答が安全フィルタを通ることを確認した。対象外pathは収集しない。本体PoCの一致条件は変更せず、同じGit差分でcompareを実行した。選定が結果観測後なので独立した精度評価に混ぜない。全threadに競合編集がないことの保証ではない。

| ケース | Git差分のファイル | 結果 | compare subprocess経過ms |
| --- | --- | --- | ---: |
| C01 | docs/benchmarks/issue-182/ministral-official/raw/README.md | {'matched': 1} | 29.59 |
| C02 | docs/benchmarks/issue-182/prompt-candidate.md | {'matched': 1} | 19.52 |
| C03 | docs/benchmarks/issue-182/prompt-comparison-preflight.md | {'matched': 1} | 19.31 |

この3件では、実Git差分の追加blockと保存された実add内容が一致し、操作ID→Turn→依頼/応答の参照を保持できた。ただし候補pathごとに元の履歴操作は1件で、パス一致より高精度になったことを示す難例ではない。

## Current worktree capture

主checkoutの既存.gitignore変更を読み取り専用captureで確認。履歴は4MiB上限、history.items/receive_limitでunavailable（1thread/70turn/1941items、受信4,194,305bytes）。Git差分1ファイルは失われず保存、compareはunmatchedとgit_diff_only_fallback=trueを返した。変更内容や生履歴は本報告へ掲載しない。capture内のHEAD/index/選択worktree不変確認を通過。

## Interpretation

実データで内容一致を返す処理自体は確認できた。しかし複数回編集後に初めてcommitされたファイルは、Gitの完成版一括addと履歴の途中add/update列が異なり、ブロック完全一致だけでは一意化できない例が残った。R09は候補6→4、R13は4→2と絞れたが依然ambiguous。他の多くは候補数不変。候補数の削減は正しい帰属の保証ではない。

現在の取得枠では大きなthreadで履歴が途切れることも確認。partialをmatchedへ昇格させない保守的な契約が実データでも働いた。現在worktree全体の帰属精度、独立Goldによる誤帰属率、取消し/rename/手動変更の実例網羅は未評価。正解を独立確定していないので正確率は算出しない。

## Safety / Artifacts

履歴収集は1回の実行を無制限化せず、Turnごとに5秒/4MiB枠。探索時は83Turnを別々に調べ、フィルタ通過した完全読取Turnを保存、除外/失敗のattemptも保持。後から対象を絞った確認用取得は別実験として区別。生diff・ユーザー発話・最終応答・参照IDは私有ディレクトリにのみ保存。公開用summaryには集計・repo内path・Git revisionだけを含む。既知秘密の可能性があるデータは拒否。外部へ履歴を送信せず、LLM呼出しなし。

過去diffの比較前後にHEAD/index/追跡済みworktreeのhash一致を確認。checkout/reset/stashや利用中ファイルへの書込みなし。既存fixture/Gold/過去測定を変更していない。履歴取得のためのCodex内部state/cache書込みまで皆無とは主張しない。peak memory未測定、処理時間も取得全体と比較だけの時間を混同しない。

私有原本: <PRIVATE_TMP>/issue182-attribution-real-evaluation/ と <PRIVATE_TMP>/issue182-attribution-real-history-v2/、確認用履歴は <PRIVATE_TMP>/issue182-attribution-real-history-controls/。実入力はそのまま公開しない。source/binary/input hashと各結果は私有plan/resultsへ保存。

## Next gate

今回確認できたのは、閉じた候補集合における3つの実データ完全一致と、曖昧/拒否/fallbackの挙動。一般精度の採用Gateは未達。LLMへの投影・本番接続へ進める前に、取得coverageと逐次編集での取り逃がしを検討する。Patch Replayや取得制約の緩和を自動採用しない。
