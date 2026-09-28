# Issue #141: edge 採否と key 固定 metadata の結果

## 結論

Pass 1 の edge ごとの accept / reject は、今回の固定モデルでは誤 edge を見分けられなかった。主比較の exact grouping は強制縮約と同じ8/12だったが、guardrail の4試行全てで誤 edge を採用し、false mergeを4 pair生んだ。**この方式を production の grouping contract に採用しない。**

Pass 2 の key 固定 batch は、固定済み正解 grouping と `G1/G2` label の oracle 条件で、`planning.Validate()` まで8/8成功した。従来 batch 配列の4/8より改善し、per-groupと同じ成功数を8 callで得た。**Pass 2 の構造的 contract としては次の候補**だが、Pass 1生成 group ID を使う end-to-end 性能と metadata の意味的品質は未検証であり、production default はまだ変えない。

## 条件

[推論前にコミットした条件と実装](issue-141-edge-and-keyed-preregistered-2026-09-28.md)は `c5f662c`。モデル、helper、fixture、prepared input、1024 output tokens/request、2分/試行、2反復は前回と同じ。Pass 1 と Pass 2 は独立試験で、成功率を合算しない。2巡目は条件順を逆にした。追加依存、model download、production code の変更はない。

## A. Pass 1: edge ごとの採否

主比較6 fixture × 2反復。全条件で完全割当12/12、false merge 0。

| 条件 | exact grouping | false split pair | calls | 合計 wall | 採用 edge |
| --- | ---: | ---: | ---: | ---: | ---: |
| file-centric | 4/12 | 16 | 12 | 179.2s | 0 |
| hybrid 強制縮約 | 8/12 | 6 | 12 | 166.8s | 10/10 |
| edge accept/reject | 8/12 | 6 | 20 | 212.9s | 10/10 |

guardrail 2 fixture × 2反復。2 fixture は source/test 同名だが別目的、direct import があるが別目的という反例。

| 条件 | exact grouping | false merge pair | calls | 誤 edge の採用 |
| --- | ---: | ---: | ---: | ---: |
| file-centric | 4/4 | 0 | 4 | なし |
| hybrid 強制縮約 | 0/4 | 4 | 4 | 4/4 |
| edge accept/reject | 0/4 | 4 | 8 | 4/4 |

edge accept/reject は候補14件を全て採用した。主比較の真 edge 10件では強制縮約の改善を再現したが、guardrail の偽 edge 4件も全て採用した。事前基準の guardrail false merge 0を満たさず、call増もある。`mixed_24` は候補 edge 0で、採否 call を省いて3条件とも2/2 exact だった。今回の model/prompt/fixture の結果であり、構造化された edge 判断全般の不可能性は示さない。unit 表現も file-centric と異なるので、改善を relation edge 情報だけの因果効果とは断定しない。

## B. Pass 2: group ID を JSON key に固定

正解 grouping と固定 `G1/G2` を使用した4 fixture × 2反復。Go は raw duplicate key、期待 ID の完全一致、各 metadata 項目、`planning.Validate()` を確認した。

| 条件 | 最終 validation 成功 | calls | 合計 wall | prompt bytes 合計 | 失敗 |
| --- | ---: | ---: | ---: | ---: | --- |
| batch 配列 | 4/8 | 8 | 142.2s | 48,520 | `invalid_metadata_groups` 4 |
| key 固定 batch | 8/8 | 8 | 131.3s | 50,166 | なし |
| per-group | 8/8 | 14 | 214.9s | 90,798 | なし |

従来 batch 配列の4失敗は `multi_commit` と `holdout_split` の両反復。いずれも2件を返したが、同じ group ID が1件重複し、別の ID が1件欠落した。key 固定 batch はこの4試行を含め全8試行で成功した。prompt bytes は配列 batch より合計1,646 bytes多い。wall の差は小標本の実行揺れも含むため、速度改善とは判定しない。call 数は key 固定 batch 8、per-group 14という構造的差がある。

固定 `G1/G2` の oracle 条件に限定した結果であり、Pass 1 が生成する動的 label と組み合わせた end-to-end の成功率は未確認。`planning.Validate()` は構造・言語などの契約検証であり、summary が各変更内容を的確に表すかの人手評価は今回行っていない。backend が JSON Schema を常に守る前提は置かず、Go 側の検証を維持する。

## 再現と検証

数値のみの生データは [Pass 1 の48試行](issue-141-edge-decision-mlx.jsonl) と [Pass 2 の24試行](issue-141-keyed-metadata-mlx.jsonl)。各 `(fixture, run, variant)` は一意。prompt、raw response、summary、動的 group label は保存していない。

```sh
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -issue141-probe edge-decision -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -issue141-probe metadata -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
```

`GOCACHE=/tmp/commiter-issue141-gocache go test ./...`、`GOCACHE=/tmp/commiter-issue141-gocache go vet ./...`、`git diff --check` を通した。production planner と SRS は変更していない。
