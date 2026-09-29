# Issue #142: forced 選択・weighted lexical・relation 注記の探索的実測

## 条件

[事前登録](issue-142-forced-weighted-annotation-preregistered-2026-09-29.md)を `f0c7cc7` でコミットし、3候補用の forced-choice 文言を `d9ae5b8` でモデル実行前に追記した。検証専用コードとテストを `af646fe` でコミットしてから実行した。対象13件はすべて既使用の合成 fixture（最初の9件と前回新たに作成した4件）。weighted 規則・閾値は以前の失敗例を見た後に固定しており、独立した未知入力による一般化検証ではない。

モデルは既存の `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`。helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。新規依存・モデル取得、production planner、Document schema、Pass 2、通常 CLI の変更はない。モデル call は output 2048 tokens、各 call 2分上限、retry/repair 0。

## Weighted lexical evidence

既存の二値 lexical partition 1件と、共有 token ごとの重み `1/(df-1)` の合計 score を使う閾値 `>=0.75`、`>=1.25` の2 partition を、それぞれ独立した arm で既存候補の末尾へ追加した。df は同一 fixture 内でその token を持つ file 数。canonicalize/dedupe と候補 cap8を適用した。

| fixture | 候補数 既存 / 二値 / weighted | gold 包含 既存 / 二値 / weighted | 新候補 二値 / weighted | 新規非 gold 二値 / weighted |
| --- | ---: | :---: | ---: | ---: |
| `verify_join_present` | 2 / 2 / 2 | ○ / ○ / ○ | 0 / 0 | 0 / 0 |
| `verify_split_present` | 3 / 3 / 3 | ○ / ○ / ○ | 0 / 0 | 0 / 0 |
| `verify_join_absent` | 3 / 3 / 3 | ○ / ○ / ○ | 0 / 0 | 0 / 0 |
| `verify_split_absent` | 3 / 3 / 3 | ○ / ○ / ○ | 0 / 0 | 0 / 0 |
| `verify_misleading_relation` | 2 / 2 / 2 | ○ / ○ / ○ | 0 / 0 | 0 / 0 |
| `holdout_crossdir_semantic` | 3 / 4 / 4 | × / ○ / ○ | 1 / 1 | 0 / 0 |
| `holdout_same_directory_pairs` | 3 / 3 / 4 | ○ / ○ / ○ | 0 / 1 | 0 / 1 |
| `holdout_spurious_test_link` | 3 / 3 / 3 | ○ / ○ / ○ | 0 / 0 | 0 / 0 |
| `holdout_atomic_feature` | 2 / 2 / 2 | ○ / ○ / ○ | 0 / 0 | 0 / 0 |
| `lexical_crossdir_pairs` | 2 / 3 / 4 | × / ○ / ○ | 1 / 2 | 0 / 1 |
| `lexical_paraphrase_gap` | 2 / 2 / 2 | × / × / × | 0 / 0 | 0 / 0 |
| `lexical_collision` | 2 / 3 / 3 | ○ / ○ / ○ | 1 / 1 | 1 / 1 |
| `lexical_bridge` | 2 / 2 / 3 | × / × / ○ | 0 / 1 | 0 / 0 |
| **13件合計** | — | **9/13 / 11/13 / 12/13** | **3 / 6** | **1 / 3** |

候補 cap8 到達は二値・weighted とも0/13件。weighted の閾値0.75で partition が gold と一致したのは7/13件、edge は gold group 内18件・間5件。閾値1.25では exact 5/13件、edge は内7件・間2件。`lexical_bridge` では weighted で gold C003 が追加され、二値では追加なし。`lexical_crossdir_pairs` では weighted に gold C003 と非 gold C004 が追加された。`lexical_paraphrase_gap` は両 arm とも gold 不在だった。

## Gold を含む候補集合の forced-choice ranking

合成 gold 不在の arm では selector を呼ばず、candidate recall failure として記録した。実行したのは `lexical_crossdir_pairs` の lexical 3候補、`lexical_collision` の baseline 2候補と lexical 3候補の計3 arm。各 arm は正順・逆順・逆順・正順の4 call。すべて `none` を schema と指示から除いた。2候補と3候補には同じ候補数非依存の選択文言を使用した。

| fixture / arm | gold ID | 4回答 | 完了・有効 ID | gold 選択 | wall |
| --- | --- | --- | ---: | ---: | ---: |
| `lexical_crossdir_pairs` / lexical | C003 | C003 / C001 / C001 / C003 | 4/4 | 2/4 | 16.0s |
| `lexical_collision` / baseline | C001 | C001 / C001 / C001 / C001 | 4/4 | 4/4 | 12.8s |
| `lexical_collision` / lexical | C001 | C003 / C001 / C001 / C003 | 4/4 | 2/4 | 13.7s |
| **合計** | — | — | **12/12** | **8/12** | **42.5s** |

`lexical_crossdir_pairs` baseline、`lexical_paraphrase_gap` 両 arm、`lexical_bridge` 両 arm は合成 gold 不在で selector call 0件。実行した各 arm 内で4 call の schema hash は同一。12/12 call が `completed` し、`max_tokens`、backend error、無効 ID は各0件。output token telemetry は利用できなかった。

## Relation context 注記

同じ C001/C002 pair、repository input、system/task、schema を固定した。annotated arm は `repository_input.relation_context` object に `relation_interpretation: "structural_hint_not_shared_purpose_proof"` を1フィールド追加した。テストでは、この挿入を取り除くと prompt bytes が current arm と完全一致する。production の Document schema は変更していない。回答順は正順・逆順・逆順・正順。

| fixture | gold ID | current の4回答 | annotated の4回答 | 有効 ID current / annotated | gold 選択 current / annotated | wall current / annotated |
| --- | --- | --- | --- | ---: | ---: | ---: |
| `verify_misleading_relation` | C002 | C001 / C001 / C001 / C001 | C001 / C001 / C001 / C001 | 4/4 / 4/4 | 0/4 / 0/4 | 13.3s / 13.3s |
| `verify_join_present` | C001 | C001 / C001 / C001 / C001 | C001 / C001 / C001 / C001 | 4/4 / 4/4 | 4/4 / 4/4 | 15.4s / 15.6s |
| `holdout_spurious_test_link` | C002 | C002 / C002 / C002 / C002 | C001 / C002 / C002 / C001 | 4/4 / 4/4 | 4/4 / 2/4 | 15.8s / 16.2s |
| **合計** | — | — | — | **12/12 / 12/12** | **8/12 / 6/12** | **44.5s / 45.1s** |

24/24 call が `completed` し、`max_tokens` は0件。fixture 内の8 call で schema hash は同一。`verify_misleading_relation` の C001 4/4 は両 arm で同じだった。`holdout_spurious_test_link` の annotated arm は正順2件で C001、逆順2件で C002 を選択した。

## 記録と測定範囲

- [weighted JSONL](issue-142-weighted-lexical-2026-09-29.jsonl): 13行。
- [forced ranking JSONL](issue-142-forced-lexical-mlx.jsonl): 4行、12 call。
- [relation annotation JSONL](issue-142-relation-annotation-mlx.jsonl): 3行、24 call。

`GOCACHE=/private/tmp/commiter-issue142-gocache go test ./...`、`go vet ./...`、`git diff --check` は成功。JSONL に生の path/diff/prompt/response は保存していない。この測定は既使用の少数の合成 fixture に限られ、実開発者判断、未知入力、production 精度・コスト、候補数と提示順の単独因果効果、Pass 2 は測定していない。
