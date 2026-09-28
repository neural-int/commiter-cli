# Issue #141: small candidate cluster boundary 検証結果

## 結論

[推論前に固定した条件](issue-141-cluster-boundary-preregistered-2026-09-28.md)で8 fixture・2反復を測った。主比較の exact grouping は6/12で前回の局所 pair と同じ、false split は214から100 pairへ減った。一方、false merge は4 pairのまま、calls は22から48へ増え、guardrail は2件の誤結合と2件の構造エラーで exact 0/4だった。**今回の cluster boundary contract は production 候補にしない。**

特に24-file fixture では、候補 block を最大4ファイルにしたにもかかわらず、block 内判定が24ファイル全てを単独 cluster に分割した。後段は「より多い semantic context を持つ cluster 比較」にならず、実質的に file 間の比較へ戻った。したがって、今回の失敗を cluster-level 判定一般の否定とは読まない。

## 固定条件と実行

事前登録・benchmark code は `8772065`。前回と同じ6主 fixture・2 guardrail、正解 grouping、MLX model revision `a962dcb09eee4169c890e544c9eb938f1113fdee`、Release helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`、1024 output tokens/request、2分/試行、2反復を使った。モデルと helper は既存のローカルキャッシュを利用した。Pass 2 と production planner は実行・変更していない。

## 主比較

既存3条件と局所 pair 条件は以前の[報告書](issue-141-edge-and-keyed-2026-09-28.md)および[局所 pair 報告](issue-141-local-boundary-2026-09-28.md)から引用し、今回再測定していない。入力形式と判断段数が異なるため wall 時間差を単独の速度効果とは扱わない。

| 条件（6 fixture × 2反復） | exact | 完全割当 | false merge pair | false split pair | calls | 合計 wall |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| file-centric | 4/12 | 12/12 | 0 | 16 | 12 | 179.2s |
| soft edge 全強制縮約 | 8/12 | 12/12 | 0 | 6 | 12 | 166.8s |
| edge accept/reject | 8/12 | 12/12 | 0 | 6 | 20 | 212.9s |
| 局所 pair boundary | 6/12 | 12/12 | 4 | 214 | 22 | 104.1s |
| **small candidate cluster boundary** | **6/12** | **12/12** | **4** | **100** | **48** | **218.8s** |

cluster 候補104件の判定は TP 46、TN 8、FP 4、FN 46。全12試行で最終 file ID 割当は完全だった。prompt bytes は82,668、output bytes は5,682。backend が output token 数を返さず、全行 `unavailable`。

| fixture（各2反復） | exact | false merge | false split | calls | stage1 の cluster 数 / run | 候補 cluster pair / run |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| `multi_commit` | 0/2 | 4 | 2 | 6 | 4 | 6 |
| `cross_directory` | 2/2 | 0 | 0 | 4 | 2 | 1 |
| `same_directory_independent` | 2/2 | 0 | 0 | 4 | 2 | 1 |
| `mixed_24` | 0/2 | 0 | 94 | 26 | 24 | 40 |
| `holdout_split` | 2/2 | 0 | 0 | 4 | 2 | 1 |
| `holdout_join` | 0/2 | 0 | 4 | 4 | 3 | 3 |

`mixed_24` の各反復では、6候補 block の内部判定が24個の単独 cluster を作り、stage1 時点で132 false split pairとなった。後段の40候補 pairは正解2 groupを連結可能で、実際に20件を同一目的・20件を別目的と判定した。最終 false split は47 pair/反復。否定した cluster pair が肯定経路で再結合された `negative_closure` は12件/反復で、単独の否定を cannot-link と扱えないことも確認した。候補の真偽判定と合成 gold label の読解可能性を切り分ける試験ではない。

`multi_commit` は stage1 で4単独 cluster となり、後段で別目的の候補を2件/反復誤結合した。`holdout_join` も3単独 clusterとなり、最終的に2 false split pair/反復が残った。この2条件は局所 pair と同じ最終 grouping だった。

## Guardrail と構造的失敗

| 条件（2 fixture × 2反復） | exact | false merge | 完全割当 | calls |
| --- | ---: | ---: | ---: | ---: |
| 局所 pair boundary（前回） | 4/4 | 0 | 4/4 | 4 |
| **small candidate cluster boundary** | **0/4** | **2** | **2/4** | **6** |

`source_test_separate_purposes` は stage1 で正しく2 clusterに分かれたが、後段がそれらを同一目的と判定して false merge 1 pair/反復を生んだ。`import_separate_purposes` はstage1の応答で `invalid_group_id` が2/2発生し、構造検証で止めたため最終 grouping は存在しない。生の応答を保存していないので、具体的な不正ラベルの内容は特定しない。修復呼び出しはこの benchmark contract に含めず、0回。

## 解釈と判断

事前条件の「`multi_commit` と guardrail で false merge 0」「`holdout_join` を結合」「主比較 calls < 22」は満たせなかった。`mixed_24` の false split は減ったが、増えた候補判定・入力・calls の条件全体の効果であり、cluster の意味的文脈を増やした効果とは言えない。stage1 で全て単独になり、狙った treatment 自体が成立していないからである。

次の設計判断では、今回の contract を採用せず、より大きな provisional cluster をモデルが実際に評価する条件を先に定義する必要がある。ただし cluster を hard に固定すると以前の guardrail false merge を再導入するので、内部の分割可能性と最終割当を保つ条件が必要である。これ以上の Pass 1 多段化は calls と失敗経路を増やす可能性が高く、#141 の範囲では別 architecture との比較を優先する判断が妥当である。Pass 2 keyed batch と動的 grouping の接続は未実施。

[全16行の数値・file ID JSONL](issue-141-cluster-boundary-mlx.jsonl)を保存した。各 `(fixture, run, variant)` は一意で、prompt、raw response、生成 summary は保存していない。production planner、SRS、通常 CLI、新しい依存・モデルは変更していない。
