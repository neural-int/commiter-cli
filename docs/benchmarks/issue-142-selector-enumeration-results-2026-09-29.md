# Issue #142: 固定 top-2 selector と小規模候補列挙の観測結果

## 固定条件

- [事前登録](issue-142-selector-enumeration-preregistered-2026-09-29.md)を `b7cb94c`、検証コードを `d6fa3c0` として結果確認前にコミットした。fixture、gold、現行 score、構造的0.4上限 score、既存 generator は変更していない。
- Selector は既存 pinned MLX 3B `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee` と helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48` を使用した。output 2048 tokens、各 call 2分上限、retry/repair 0。新規依存・モデル取得、production planner、Pass 2 の変更はない。

## `new_stem_doc_diverged` の固定 top-2 selector

構造的0.4上限 score の weighted 候補の top-2 は C001（非 gold）、C002（gold）で、実行前の契約と一致した。正順・逆順・逆順・正順で4 call 実行した。

| run | 提示順 | stop reason | 有効 ID | 選択 ID | gold 選択 | wall |
| ---: | --- | --- | --- | --- | --- | ---: |
| 1 | C001, C002 | completed | はい | C001 | いいえ | 3,918 ms |
| 2 | C002, C001 | completed | はい | C001 | いいえ | 3,329 ms |
| 3 | C002, C001 | completed | はい | C001 | いいえ | 3,226 ms |
| 4 | C001, C002 | completed | はい | C001 | いいえ | 3,234 ms |

4/4 call が completed・有効 ID、gold C002 の選択は0/4、`max_tokens` は0/4。各 call の `wall_ms` 合計は13,708 ms。結果 JSONL は各 call の prompt/schema hash、出力長、停止理由も記録した。選択理由や一般的な selector 性能はこの4 call では測定していない。

## 3件の候補欠落に対する列挙診断

既存 weighted 候補の後ろに、事前登録した3 partitionずつを ID 順に追加した。9 partition のうち既存候補との重複0件、追加9件、cap による省略0件。各 fixture で追加3件のうちgoldは1件、非 gold は2件だった。追加後の候補数は5、6、5件で、cap8には達しなかった。モデル call は0件。

| fixture | 抽出された relation edge | 初期候補数→追加後 | gold ID | gold 順位 現行 score / 0.4上限 score | 追加後 top-2 現行 / 0.4上限 |
| --- | --- | ---: | --- | --- | --- |
| `new_test_pair_diverged` | F002→F001 `source_test` / `soft` / `matching_test_path` | 2→5 | C004 | 3 / 3 | C001,C003 / C001,C003 |
| `new_crossdir_collision` | 0件 | 3→6 | C004 | 2 / 2 | C003,C004 / C003,C004 |
| `new_paraphrase` | 0件 | 2→5 | C003 | 2 / 2 | C001,C003 / C001,C003 |

`new_test_pair_diverged` は source/test relation edge が実際に抽出されていた。列挙によって gold C004 が追加されたが、両 score で3位だった。0.4上限 score では C003 と C004 がともに0.9で、候補ID順により C003 が先になった。

`new_crossdir_collision` は4ファイルの2 pair 完全マッチング3件を追加し、gold C004 が2位。`new_paraphrase` は3ファイルの1 pair partition3件を追加し、gold C003 が2位だった。後者の追加した3候補の score は両 score で各 -0.5、候補ID順で C003 が先になった。この列挙は lexical / semantic evidence による発見ではない。

| 対象3件・weighted集合 | total recall | recall@1 | recall@2 | recall@3 | 追加非 gold 候補 |
| --- | ---: | ---: | ---: | ---: | ---: |
| 追加前（両 score） | 0/3 | 0/3 | 0/3 | 0/3 | 0 |
| 追加後・現行 score | 3/3 | 0/3 | 2/3 | 3/3 | 6 |
| 追加後・0.4上限 score | 3/3 | 0/3 | 2/3 | 3/3 | 6 |

## 記録と測定範囲

- [selector JSONL](issue-142-capped-selector-2026-09-29.jsonl): 1行、4 call。
- [列挙診断 JSONL](issue-142-small-enumeration-2026-09-29.jsonl): 3行。元候補、列挙 partition、追加候補、dedupe/cap、relation の file ID・kind・class・reason、前後の両 score 順位を記録した。生の path/diff/prompt/response は両結果 JSONL に含まない。
- `GOCACHE=/private/tmp/commiter-issue142-gocache go test ./...`、同 `go vet ./...`、`git diff --check` が成功した。
- Selector は1合成 fixture・4 call、列挙は3〜4ファイルの既使用合成 fixture 3件のみ。実開発者判断、production 入力分布、24ファイルでの列挙量、selector の一般的な精度は測定していない。
