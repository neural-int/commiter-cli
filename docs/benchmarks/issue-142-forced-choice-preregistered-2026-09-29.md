# Issue #142: forced-choice selection の事前条件

## 目的と固定条件

[前回の診断](issue-142-selection-diagnostic-2026-09-28.md)では、正解候補を含む4 fixture の24試行で、完了15件がすべて `none`、未完了9件が `max_tokens` または2分の時間切れだった。[考察コメント](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5871520075)を修正し、次の probe では、合成 gold と一致する候補がある固定入力に対して二択時の選択結果を測る。`none` の除外による因果効果や一般的な semantic selection 能力を、この probe 単独で判定しない。

前回と同じ Louvain weighted graph、seed `142,1`、resolution `[0.25, 0.5, 1, 2, 4]`、候補の正規化・ID、prepared repository input、MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を使用する。モデルの新規取得や依存追加はしない。ここまでの code と本書をモデル実測前にコミットする。

## Forced-choice 条件

- 対象: 既使用の `cross_directory`、`same_directory_independent`、`mixed_24`、`source_test_separate_purposes`。各 fixture の候補数は2、合成 gold と一致する候補が1つあることを実行前に確認する。新しい holdout は使用しない。
- 各 fixture で2反復。1回目は元の候補提示順、2回目は提示順だけを反転し、候補IDとグループ内容の対応は保つ。合計8 calls。順序ごとに1反復しかないため、順序効果は推定しない。
- selection 指示と JSON Schema enum から `none` を除外し、`C001` または `C002` の1件を要求する。repository input と候補集合は前回の同 fixture と同じ。変更された prompt/schema の SHA-256 を各行に記録する。
- output 上限2048 token、wall-time 上限2分、repair 0、transport retry 0、Pass 2なし。前回の2048条件は5/8完了、3/8 `max_tokens`、時間切れ0だった。4096条件は3/8が時間切れだったため、今回は2048に固定する。
- `completed`、`max_tokens`、2分の時間切れ、その他 stop/error、候補IDの有効性、合成 gold に対する正解、完全割当、false merge/split、calls、wall、output token telemetry を行ごとに記録する。`none` が返った場合は契約違反として記録し、候補選択に含めない。未完了も semantic な誤選択として数えない。
- 正解率は全8試行と、完了した有効候補ID応答を分母にした値を別に記録する。前回の2048 `none` 許可条件との対比は記述的な比較にとどめる。二択の偶然水準、小標本、使用済みの合成ラベル、非同時測定による制限を明記する。

各 `(fixture, run)` は一意。数値、failure code、候補ID、prompt/schema hash だけを JSONL に保存し、生の prompt、応答、summary、repository content は保存しない。production planner、SRS、通常 CLI の contract は変更しない。

8試行中4件以上が `max_tokens` または2分の時間切れなら、コメントの次段である candidate-difference-only の入力と評価条件を別途事前登録してから測る。この閾値は次段へ進むための運用条件であり、方式の採否判定ではない。
