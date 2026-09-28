# Issue #141: 解除可能な relation 候補と Pass 2 診断の結果（2026-09-28）

## 結論

解除可能な soft candidate は、今回の guardrail 2 fixture では誤結合を起こさなかった。一方、主比較6 fixture の exact grouping は file-centric と同じ4/12で、false split は16→14 pairの小幅な減少にとどまった。[事前登録した採用基準](issue-141-soft-and-metadata-preregistered-2026-09-28.md)の false split 6以下、exact 8/12以上には届かない。**今回の候補提示 prompt を production の grouping contract として採用しない。** 前回の hard 縮約の改善と、解除可能な候補の安全性をこのまま同時に得ることはできなかった。

独立した固定 group label の Pass 2 では、batch metadata が4/8、per-group が8/8成功した。batch の4失敗はすべて、metadata 件数は期待どおり2件だが group ID が1件重複し、別の ID が1件欠落する `invalid_metadata_groups` だった。per-group は14 calls、batch は8 callsで、wall 合計も214.8秒対151.8秒。per-group の成功だけを根拠に group 数比例の production call budget を認めない。

## 固定した条件

推論前のコミット `aa8df77` で[条件と benchmark code](issue-141-soft-and-metadata-preregistered-2026-09-28.md)を固定した。Pass 1 は同じ6 fixtureに、source/test 関係が別目的の例と直接 import が別目的の例を加え、8 fixture × 3条件 × 2反復の48試行。Pass 2 は先行 oracle 比較と同じ `multi_commit`、`mixed_24`、`holdout_split`、`japanese` に正解 group と固定 `G1` / `G2` label を与え、4 fixture × batch/per-group × 2反復の16試行。各2巡目は条件順を逆にした。両試験の成功数は合算しない。

ローカル MLX model は `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。各 request 1024 output token、plan cycle 2分 timeout、2反復。追加モデル・依存・クラウド API は使わなかった。出力 token 数は backend から取得できず、全行で `unavailable`。

## A. Pass 1: 全 file ID を分離可能なまま割り当てる

soft 条件は前回と同じ file ID → group label schema を使い、既存 `source_test` / `direct_import` の soft edge を「同じ目的かもしれないが分離可能」な候補として追加した。edge は正解 group を見ずに選ぶ。Pass 2 は呼んでいないため、下表は Pass 1 grouping だけの比較である。全試行が構造的に有効で、pair 指標に欠測はない。

| 主比較6 fixture | 完全割当 | exact grouping | false merge | false split | per-file accuracy | calls | wall 合計 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| file-centric | 12/12 | 4/12 | 0 | 16 | 64/78 | 12 | 176.3s |
| soft-source-test | 12/12 | 4/12 | 0 | 14 | 66/78 | 12 | 162.4s |
| soft-source-test-import | 12/12 | 4/12 | 0 | 14 | 66/78 | 12 | 179.2s |

| fixture | file-centric exact / split | soft-source-test exact / split | soft-source-test-import exact / split | 観測 |
| --- | ---: | ---: | ---: | --- |
| multi_commit | 0/2・4 | 0/2・2 | 0/2・2 | source/test pair の分割だけ改善。README / docs は分割されたまま |
| cross_directory | 0/2・2 | 0/2・2 | 0/2・2 | direct import 候補を提示しても結合できない |
| same_directory_independent | 2/2・0 | 2/2・0 | 2/2・0 | 候補 edge 0 |
| mixed_24 | 2/2・0 | 2/2・0 | 2/2・0 | 候補 edge 0。全24 ID を両回で正しく割当 |
| holdout_split | 0/2・4 | 0/2・4 | 0/2・4 | hard 縮約時の改善を維持できない |
| holdout_join | 0/2・6 | 0/2・6 | 0/2・6 | implementation / test / docs を結合できない |

| guardrail 2 fixture | 完全割当 | exact grouping | false merge | 候補の false pair |
| --- | ---: | ---: | ---: | ---: |
| file-centric | 4/4 | 4/4 | 0 | 0 |
| soft-source-test | 4/4 | 4/4 | 0 | 2（2反復合計） |
| soft-source-test-import | 4/4 | 4/4 | 0 | 4（2反復合計） |

guardrail では候補 edge の false pair が実在しても、モデルは別 group を選べた。ただし2つの小さな合成例であり、一般的な誤結合率の保証ではない。主比較6 fixture の候補 edge は source-test 8件、import 追加後10件（2反復合計）で false pair は0。元の file-centric の12試行は、前回の対応する試行と初回 prompt bytes・完全割当・grouping が一致した。今回は Pass 2 を呼んでいないため、failure code・calls・wall を前回の end-to-end 行と直接比較しない。

主比較6 fixture の Pass 1 初回 calls は各条件12、guardrail は各4。repair と transport retry は0で、全 request の stop reason は `completed`。Pass 1 入力は主比較で66,738 / 68,780 / 68,900 bytes、出力は2,568 / 2,432 / 2,400 bytes。prompt と候補リスト・指示が同時に変わるため、効果を edge 情報だけへ帰属しない。wall の小差も性能差とは断定しない。

## B. Pass 2: 固定 group label の batch / per-group

正解 group と固定 group ID を与え、Pass 1 の生成は行わなかった。各応答の本文はメモリ上でのみ検証し、JSON/schema failure と件数だけ記録した。生成された group ID、summary、prompt、raw response は保存していない。

| 方式 | validation 成功 | calls | wall 合計 | 入力 | 出力 |
| --- | ---: | ---: | ---: | ---: | ---: |
| batch | 4/8 | 8 | 151.8s | 48,520 B | 3,240 B |
| per-group | 8/8 | 14 | 214.8s | 90,798 B | 3,438 B |

| fixture | batch | per-group | batch 失敗の構造的内訳 |
| --- | ---: | ---: | --- |
| multi_commit | 0/2 | 2/2 | metadata 2件、group ID 重複1、期待 ID 欠落1 |
| mixed_24 | 2/2 | 2/2 | なし |
| holdout_split | 0/2 | 2/2 | metadata 2件、group ID 重複1、期待 ID 欠落1 |
| japanese | 2/2 | 2/2 | なし。日本語 summary は既存 validation を通過 |

失敗4件に未知 group ID や `breaking` 欠落はなく、返却 metadata 件数は期待値と一致した。この条件では `invalid_metadata_groups` の直接の拒否理由を group ID の重複・欠落へ絞れた。[先行する oracle 比較](issue-141-grouping-first-2026-09-28.md)の1反復目と prompt bytes、成功／失敗、calls が一致し、2反復目でも同じ fixture の結果が続いた。一方、前回 hybrid の Pass 1 生成 label を使った `mixed_24` batch 失敗は、今回の固定 label 試験では再現しなかった。label 形式、group 順序、入力の細部などが異なるため、その原因をこの試験だけで特定できない。

## 判断と次の境界

今回の soft candidate prompt は guardrail の false merge を避けたが、事前基準の false split 6以下と exact 8/12以上を満たさない。hard 縮約は[反例](issue-141-hybrid-2026-09-28.md)で不適格であり、soft 候補もこの prompt/schema/model では代替策にならなかった。現時点では production planner / SRS / 通常 CLI を変更しない。

Pass 2 は batch で複数 group ID の対応が安定しない例があり、per-group は今回8/8成功したが calls と wall が増えた。次の設計候補を比較するなら、group ID の対応を Go が固定できる別の metadata contract と plan 全体の call budget を事前に定義する必要がある。この結果だけで per-group を production の既定方式にしない。pairwise / local boundary classification への移行も今回の結果だけでは確定しない。

## 再実行と品質確認

生データは [Pass 1 の48試行](issue-141-soft-mlx.jsonl) と [Pass 2 の16試行](issue-141-metadata-diagnostic-mlx.jsonl)。再実行には同じ model/helper を指定する。

```sh
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -issue141-probe soft -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -issue141-probe metadata -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
```

`GOCACHE=/tmp/commiter-issue141-gocache go test ./...`、`GOCACHE=/tmp/commiter-issue141-gocache go vet ./...`、`git diff --check` を通した。両 JSONL は48件・16件の一意な試行行を持ち、prompt / response 本文を含まないことも確認した。
