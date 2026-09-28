# Issue #141: edge 採否と key 固定 metadata の事前条件

この文書と benchmark code を推論前にコミットする。[前回の実測](issue-141-soft-and-metadata-2026-09-28.md)と同じローカル MLX model、helper、fixture、prepared input、1024 output tokens/request、2分/試行、2反復を使う。model は `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。追加依存・model download・production code の変更はない。

## A. Pass 1: relation edge ごとの採否

前回と同じ主比較6 fixture と guardrail 2 fixture で、次の3条件を同一実行内で比較する。8 fixture × 3条件 × 2反復の48試行。2巡目は条件順を逆にする。Pass 2 は呼ばない。

| 条件 | 手順 |
| --- | --- |
| file-centric | 前回と同じ全 file ID → group label |
| hybrid-source-test-import | 対象の soft source/test・direct import edge を全て unit へ強制縮約し、unit → group label |
| decision-source-test-import | 対象 edge を `E001` 形式の固定 key で列挙し、モデルが各 edge を true/false で採否判断。Go が採用 edge のみ unit へ縮約し、unit → group label |

候補 edge は `source_test` + `matching_test_path` と `direct_import` + `observed_import_path` の `Soft` edge に限定する。正解 group は入力・採否・unit 構築に使わず、採否後に true/false pair を監査する。edge が0件なら採否 call を省き、singleton unit で grouping する。decision 出力は Go が raw 重複 key、期待 edge ID の完全一致、boolean を検証し、不正なら grouping へ進まない。grouping は完全割当を確認し、構造不正だけ最大1回修復する。意味的誤結合・誤分割は修復しない。

主指標は主比較6 fixture の完全割当、exact grouping、false merge/split pair、guardrail 2 fixture の false merge。補助指標は候補・採用 edge、採用 unit の true/false pair、calls、wall、prompt/output bytes、failure code。主比較で完全割当12/12、false merge 0、false split 6以下、exact 8/12以上、guardrail false merge 0を探索的な採用候補基準とする。decision stage 追加分の call と wall も比較する。unit 表現・prompt が file-centric と異なるので、差を edge の情報量だけの因果効果とは解釈しない。小規模合成 guardrail 合格だけで安全性は証明しない。

## B. Pass 2: group ID を key に固定

前回の oracle 試験と同じ `multi_commit`、`mixed_24`、`holdout_split`、`japanese` に正解 grouping と固定 `G1/G2` ID を与える。Pass 1 は実行しない。batch 配列、今回の keyed batch、per-group の3条件 × 4 fixture × 2反復、計24試行。2巡目は条件順を逆にする。

keyed batch は Go が期待 group ID を JSON object key と JSON Schema の `required` / `additionalProperties:false` に設定する。Go は raw JSON の重複 key をデコード前に拒否し、欠落・未知 key、各 metadata の項目、最終 plan を `planning.Validate()` まで確認する。backend の schema 遵守だけに依存しない。成功率、failure code、calls、wall、prompt/output bytes を比較する。8/8 成功を満たさない場合は採用候補としない。成功しても固定 ID の oracle 診断であり、Pass 1 が生成した label の end-to-end 結果や production の信頼性には外挿しない。

両試験の成功率は合算しない。保存する JSONL は数値・fixture 名・failure code のみとし、prompt、raw response、生成した summary、動的 group label を保存しない。production planner、通常 CLI、SRS は変更しない。
