# Issue #142: candidate partition selection の事前条件

## 固定条件

この文書と benchmark code を候補採点・モデル推論前に固定する。#141 の6主 fixture と2 guardrail に加え、未使用の2 fixture（`new_cross_purpose_docs`、`new_same_directory_split`）を使用する。gold grouping は候補生成にも prompt にも渡さない。従来の holdout は既に使用済みであり、新規 fixture だけを未使用評価とする。

- Gonum `v0.16.0` の `community.Modularize`。Go module の設定 `go 1.23.0` を維持し、依存は benchmark 専用とする。
- node は全 file ID。無向 edge の重みは `source_test` 4、`direct_import` 3、同じ小文字 basename stem（拡張子と末尾数字を除く）1、同一親 directory 0.1。重複根拠は加算する。self、未知 ID、負重みは拒否する。すべての同一 directory pair は24 file 上限内だけ展開する。gold label は重みに使わない。
- resolution は `[0.25, 0.5, 1, 2, 4]`。各 run の乱数源は `rand.NewPCG(142, 1)`。全 group と group 内 ID を昇順に正規化し、同一 partition を除去する。最大5候補で、生成順を保持する。候補の全 ID exactly once、空 group なしを検証する。入力は2〜24 file。
- 候補のみの結果は全10 fixture に対し1回。候補 raw 数、dedupe 後数、gold 包含、生成時間、selection 用 system+prompt bytes を記録する。候補包含率が低くても条件を変更しない。
- MLX selection は前回と同じ local `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。各 fixture 2反復、2回目は候補提示順を反転する。選択 JSON は単一の `candidate_id` に候補 ID または `none` のみを許す。raw duplicate key、unknown ID、複数選択、不正 JSON、未完了を拒否する。
- selection と metadata は各 request の出力上限1024 token、plan 2分、合計最大2 backend calls、repair 0、transport retry 0。`none` は追加生成しない。選択が valid candidate の場合だけ、Go が順序を固定した `G001` 等の group ID を付けて既存 keyed batch metadata を呼び、`planning.Validate()` まで接続する。Pass 1 不正解は end-to-end 成功に数えない。
- candidate recall と、その条件付き selection accuracy を分母を分けて集計する。全試行の exact grouping、false merge/split、complete assignment、none、calls、wall、prompt/output bytes、構造的失敗、semantic failure を記録する。取得できない output token 数は `unavailable` とする。

## 採否基準と解釈

主比較では6 fixture の exact grouping が既存最良の8/12を上回り、guardrail false merge は0、完全割当は維持し、平均 calls は2以下を目指す。未使用 fixture の欠落や順序依存は別に示す。基準を満たさなければこの固定 generator と selection contract の production 採用を見送る。候補漏れは selection failure と区別し、Louvain 一条件の結果から candidate selection architecture 全般の可否を断定しない。既存比較値は #141 の保存結果を引用し、今回再計測したように扱わない。

生の prompt、応答、生成 summary、repository content は保存せず、数値・file ID・failure code のみ残す。production planner、SRS、Git mutation は変更しない。
