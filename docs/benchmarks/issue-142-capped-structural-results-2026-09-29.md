# Issue #142: 構造的加点の上限を設けた scorer の新規合成例での観測

## 固定した条件

- [事前登録](issue-142-capped-structural-preregistered-2026-09-29.md)を `6bf7f04`、scorer と runner を `bc3bd66`、[新規合成 fixture 8件](issue-142-capped-structural-fixtures-2026-09-29.json)を `99dc127` として、順位結果を計算する前に別々にコミットした。
- 既存・二値 lexical・weighted lexical の generator を前回と同じ規則・cap8で実行した。現行 score と、構造的加点を file pair あたり0.4に制限した score を同じ候補へ適用した。lexical token と pair penalty 0.5、同点時の ID 順を維持した。
- 今回はモデル call、新規依存・モデル取得、production planner、Pass 2 の変更は0件。

## 集計

この8件では、既存・二値 lexical・weighted lexical が fixture ごとに同じ候補 ID/grouping を生成し、追加された候補は0件だった。各 generator の候補数は2〜3件。以下は各 generator に共通の値であり、3種類の独立した測定結果を合算したものではない。

| score | recall@1 | recall@2 | recall@3 | total recall | gold 候補ありの条件付き recall@2 |
| --- | ---: | ---: | ---: | ---: | ---: |
| 現行 | 3/8 | 4/8 | 5/8 | 5/8 | 4/5 |
| 構造的加点0.4上限 | 3/8 | 5/8 | 5/8 | 5/8 | 5/5 |

Gold が候補集合に存在しなかったのは `new_test_pair_diverged`、`new_crossdir_collision`、`new_paraphrase` の3件。両 score とも、この3件の gold rank は0。候補生成結果は score によって変わらない。

## fixture 別の順位

全列は各 generator に共通。`—` は gold 候補が生成されなかったことを示す。margin は score 1位と2位の差。非 gold score は最高位の非 gold 候補の score。数値は JSONL 値の小数第3位で丸めた表示。

| fixture | 候補数 | gold ID | gold 順位 現行→上限 | top-2 現行→上限 | margin 現行→上限 | 最高位非 gold score 現行→上限 |
| --- | ---: | --- | --- | --- | ---: | ---: |
| `new_test_pair_shared` | 3 | C001 | 1→1 | C001,C003 → C001,C003 | 1.000→1.000 | 5.600→1.900 |
| `new_test_pair_diverged` | 2 | — | — | C001,C002 → C001,C002 | 6.600→2.300 | 6.600→2.300 |
| `new_stem_doc_shared` | 3 | C001 | 1→1 | C001,C003 → C001,C003 | 1.000→1.000 | 1.500→0.900 |
| `new_stem_doc_diverged` | 3 | C002 | 3→2 | C001,C003 → C001,C002 | 1.000→0.900 | 1.500→0.900 |
| `new_crossdir_shared` | 3 | C001 | 1→1 | C001,C003 → C001,C003 | 1.000→1.000 | 5.000→3.800 |
| `new_crossdir_collision` | 3 | — | — | C003,C001 → C003,C001 | 3.167→3.167 | 4.000→3.400 |
| `new_lexical_bridge` | 3 | C001 | 2→2 | C003,C001 → C003,C001 | 0.333→0.333 | 7.000→5.800 |
| `new_paraphrase` | 2 | — | — | C001,C002 → C001,C002 | 1.500→1.500 | 0.000→0.000 |

`new_stem_doc_diverged` は gold C002 が3位から2位に移った一方、両 score の1位は非 gold C001だった。gold が生成されなかった3件では、順位付けは gold を top-2 に含められなかった。

## 記録と測定範囲

- [結果 JSONL](issue-142-capped-structural-results-2026-09-29.jsonl)は8行。各行に両 score の候補 ID/grouping/score、gold rank、top-2 ID、margin、最高位非 gold ID/score、追加非 gold 数を記録した。生 path/diff は fixture 定義ファイルにのみ含まれる。
- `GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe capped-holdout -fixture all` で実行した。`go test ./...`、`go vet ./...`、`git diff --check` が成功した。
- この8件は score とコードの固定後に作成した少数の合成例で、gold は作成者が定義した変更目的である。実開発者判断、production 入力分布、selector の結果・コストは測定していない。score の採否や重み変更はこの結果から行っていない。
