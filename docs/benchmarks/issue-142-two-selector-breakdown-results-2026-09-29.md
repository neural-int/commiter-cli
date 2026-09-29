# Issue #142: 固定2条件の selector と C003/C004 score 内訳の観測結果

## 固定条件

- [事前登録](issue-142-two-selector-breakdown-preregistered-2026-09-29.md)を `c72d4e5`、検証コードを `e53a2b4` として結果確認前に別コミットにした。fixture、gold、generator、列挙候補、現行/構造的0.4上限 score は変更していない。
- Selector は既存 pinned MLX 3B `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48` を使用した。output 2048 tokens、各 call 2分上限、retry/repair 0。新規依存・モデル取得、production planner、Pass 2 の変更はない。

## 固定2候補の forced-choice

列挙後の weighted 候補を構造的0.4上限 score で順位付けし、事前登録した2 fixture の top-2 ID/gold と一致することをモデル call 前に確認した。各 fixture を正順・逆順・逆順・正順で4回実行した。

| fixture | top-2 ID（score順） | gold ID | 正/逆/逆/正の選択 | completed・有効ID | gold選択 | call wall 合計 |
| --- | --- | --- | --- | ---: | ---: | ---: |
| `new_crossdir_collision` | C003, C004 | C004 | C004, C004, C004, C004 | 4/4 | 4/4 | 16,584 ms |
| `new_paraphrase` | C001, C003 | C003 | C001, C001, C001, C001 | 4/4 | 0/4 | 14,020 ms |
| 合計 | — | — | — | 8/8 | 4/8 | 30,604 ms |

8 call 中 `max_tokens`、無効ID、backend errorは各0件。各 fixture の4 call は同じ schema hash と、提示順に対応する2種類の prompt hashを持った。この比較では、モデルが選択に使った根拠は記録していない。

## `new_test_pair_diverged` の score 内訳

列挙後の C003（F001+F002、非 gold）と C004（F001+F003、gold）は、どちらも同一 group 内 file pair が1組。lexical共有tokenは各2件で、事前登録した相互排他的な帰属規則で集計した。数値は file pair に対する加点で、候補scoreは「構造的加点 + lexical - pair penalty」。

| 候補 / file pair | source-test | direct-import | stem | 同じdirectory | raw structural | 0.4上限後 | lexical合計 | diff-diff | path-path | mixed | penalty | 現行score | 上限付きscore |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| C003 / F001+F002 | 4.0 | 0 | 0 | 0.1 | 4.1 | 0.4 | 1.0 | 0.5 | 0.5 | 0 | 0.5 | 4.6 | 0.9 |
| C004 / F001+F003 | 0 | 0 | 1.0 | 0 | 1.0 | 0.4 | 1.0 | 0.5 | 0.5 | 0 | 0.5 | 1.5 | 0.9 |

両候補とも上限付きの structural 0.4、lexical 1.0、penalty 0.5で、上限付きscoreは同値0.9。候補ID順により C003 が C004 より先になった。内訳のpair score合計は前回の候補scoreと一致した。lexicalの diff-diff/path-path は共有tokenの帰属規則による分類であり、独立した因果的証拠を示す値ではない。

## 記録と測定範囲

- [selector JSONL](issue-142-two-selectors-2026-09-29.jsonl): 2行、8 call。候補 ID/grouping、停止理由、選択ID、gold選択、wall、prompt/schema hashを記録。
- [score内訳 JSONL](issue-142-score-breakdown-2026-09-29.jsonl): 1行、2候補。file pairごとの加点とlexical帰属を記録。両JSONLに生の path/diff/prompt/response/token文字列は含まない。
- `GOCACHE=/private/tmp/commiter-issue142-gocache go test ./...`、同 `go vet ./...`、`git diff --check` が成功した。
- Selectorは既使用の合成 fixture 2件・各4 call、内訳は1 fixture・2候補。実開発者判断、production入力分布、一般的な selector能力・選択理由は測定していない。
