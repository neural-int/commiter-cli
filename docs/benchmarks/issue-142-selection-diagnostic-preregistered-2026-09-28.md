# Issue #142: selection completion 診断の事前条件

## 前回から固定する条件

#142 の `de88318` に保存した Louvain weighted graph、seed `142,1`、resolution `[0.25, 0.5, 1, 2, 4]`、候補の正規化・ID、prepared repository input、selection system/prompt、JSON Schema、MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を維持する。追加モデルを取得しない。測定前に本書と code をコミットする。

前回の `output_bytes=196` は完了した7件の `none` 応答から来る。未完了時の generated text は helper/backend が返さず、内部 token の内訳も得られない。`max_tokens` の原因を reasoning 長や選択能力と断定せず、stop reason と完了応答の選択結果を別に測る。

## A. Output budget sweep

対象は正解候補を含む既存4 fixture：`cross_directory`（結合）、`same_directory_independent`（分離）、`mixed_24`（24 files）、`source_test_separate_purposes`（false relation guardrail）。gold は prompt に渡さない。各 fixture で候補集合と prepared Document を一度生成し、同一反復内では output token 上限以外の system、prompt、schema をバイト単位で同じにする。各行に system+prompt と schema の SHA-256 を保存して照合する。

- output tokens: `1024`, `2048`, `4096`。fixture ごとに2反復。第1反復は候補の元順序で budget 昇順、第2反復は候補提示順を反転し budget 降順。計24 calls。これらの fixture と正解ラベルは既に使用済みで、generalization 評価ではない。
- 各 call の wall-time 上限2分、repair 0、transport retry 0。Pass 2 は呼ばない。時間切れは `timeout` として `max_tokens` と区別する。モデル helper は各 call で起動する既存経路のまま。
- `completed`、`max_tokens`、`timeout`、その他 stop/error、valid candidate ID、`none`、invalid JSON/schema、correct selection を記録する。正解選択率は全試行と、完了した有効応答を分母にした値を別に示す。実際の candidate ID が得られた場合のみ grouping、complete assignment、false merge/split を計算する。output token telemetry が無ければ `unavailable`。
- budget 変更と完了率の相関を記録する。改善があっても、decoder、hidden reasoning、semantic 判断のどれが原因かは断定しない。小標本・順序反転・時間上限の影響を併記する。

## B. Gold 追加の補助 probe

A の測定後、正解候補が欠けた既使用 fixture `multi_commit` と `new_same_directory_split` を対象にする。各 fixture の元の2 Louvain 候補を使う `original` arm と、gold partition + 元の非正解候補1つを使う `gold-injected` arm を同じ repository input で比較する。gold label や arm 名は prompt に渡さない。候補 ID と表示順は反復間で入れ替え、gold は一方で `C001`、もう一方で `C002`。両 arm の budget は4096、timeout 2分、各2反復、計8 calls。Pass 2、repair、retry は実行しない。

この補助 probe は、候補漏れを補ったときの `completed`、候補 ID、`none`、未完了を観測するためのもの。gold-injected arm は候補集合を変えるため、A や original arm との比較から Louvain と semantic selection の独立した因果効果は推定しない。新しい holdout も使用しない。

## 記録と判断

各 `(probe, fixture, run, budget, arm)` は一意。数値、failure code、candidate ID、prompt/schema hash だけを JSONL に保存し、生の prompt、応答、summary、repository content は保存しない。実モデルで有効な候補 ID が得られれば、後続の end-to-end 条件を別に事前登録する。0件なら今回の固定 model・prompt・budget・wall 条件で Pass 2 へ進めない事実を記録する。production planner、SRS、通常 CLI の contract は変更しない。
