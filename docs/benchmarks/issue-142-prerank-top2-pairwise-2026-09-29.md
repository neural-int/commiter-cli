# Issue #142: 候補事前順位、top-2 選択、条件付き pairwise の観測結果

## 実行条件

- [事前登録](issue-142-prerank-preregistered-2026-09-29.md)と検証コードを結果確認前に、それぞれ `7ccc0be`、`6118de2` としてコミットした。既使用の合成 fixture 13件を使用した。
- 既存、二値 lexical、weighted lexical の候補集合を別々に生成し、事前登録した同一 score で順位付けした。モデル比較には既存の pinned MLX 3B `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48` の helper を使用した。新規依存・モデル取得、production planner、Pass 2、通常 CLI の変更はない。
- モデル call は output 2048 tokens、各 call 2分上限、retry/repair 0。呼び出し時間の合計は各 call の `wall_ms` の合計で、全体経過時間ではない。

## 決定的な順位

分母は全13件。括弧内は gold が候補集合に含まれる fixture の条件付き分母。候補数は既存 2〜3、二値 2〜4、weighted 2〜4。

| 候補集合 | recall@1 | recall@2 | recall@3 | total recall | 追加非 gold 候補 |
| --- | ---: | ---: | ---: | ---: | ---: |
| 既存 | 5/13 (5/9) | 8/13 (8/9) | 9/13 (9/9) | 9/13 | 0 |
| 二値 lexical | 6/13 (6/11) | 10/13 (10/11) | 11/13 (11/11) | 11/13 | 1 |
| weighted lexical | 7/13 (7/12) | 11/13 (11/12) | 12/13 (12/12) | 12/13 | 3 |

weighted 集合で gold が存在して top-2 から落ちた fixture は `holdout_spurious_test_link` の1件で、gold `C002` は3候補中3位だった。`lexical_paraphrase_gap` は3集合とも gold が候補にない。

| fixture | 既存 gold 順位 | 二値 gold 順位 | weighted gold 順位 |
| --- | ---: | ---: | ---: |
| `verify_join_present` | 1/2 | 1/2 | 1/2 |
| `verify_split_present` | 2/3 | 2/3 | 2/3 |
| `verify_join_absent` | 1/3 | 1/3 | 1/3 |
| `verify_split_absent` | 1/3 | 1/3 | 1/3 |
| `verify_misleading_relation` | 2/2 | 2/2 | 2/2 |
| `holdout_crossdir_semantic` | 候補外/3 | 1/4 | 1/4 |
| `holdout_same_directory_pairs` | 2/3 | 2/3 | 2/4 |
| `holdout_spurious_test_link` | 3/3 | 3/3 | 3/3 |
| `holdout_atomic_feature` | 1/2 | 1/2 | 1/2 |
| `lexical_crossdir_pairs` | 候補外/2 | 1/3 | 1/4 |
| `lexical_paraphrase_gap` | 候補外/2 | 候補外/2 | 候補外/2 |
| `lexical_collision` | 1/2 | 2/3 | 2/3 |
| `lexical_bridge` | 候補外/2 | 候補外/2 | 1/3 |

## top-2 forced-choice

事前登録した6 arm は全件で gold が top-2 に含まれた。各 arm は正順、逆順、逆順、正順の4 call。合計24 call 中22件が `completed` かつ有効候補 ID、2件が `max_tokens`。有効22件のうち gold 選択は12件。全 call の `wall_ms` 合計は211,506 ms。

| fixture / 候補集合 | top-2 ID（score 順） | gold ID | 正/逆/逆/正の結果 | 有効 | gold 選択 |
| --- | --- | --- | --- | ---: | ---: |
| `verify_misleading_relation` / weighted | C001, C002 | C002 | C001, C001, C001, C001 | 4/4 | 0/4 |
| `holdout_crossdir_semantic` / 二値 | C004, C002 | C004 | C004, C004, C004, C004 | 4/4 | 4/4 |
| `lexical_crossdir_pairs` / 二値 | C003, C001 | C003 | C003, C003, C003, C003 | 4/4 | 4/4 |
| `lexical_crossdir_pairs` / weighted | C003, C004 | C003 | C004, C004, C004, C004 | 4/4 | 0/4 |
| `lexical_collision` / weighted | C003, C001 | C001 | max_tokens, C003, C003, max_tokens | 2/4 | 0/2 |
| `lexical_bridge` / weighted | C003, C002 | C003 | C003, C003, C003, C003 | 4/4 | 4/4 |

## 条件付き pairwise

事前登録した条件に該当した `holdout_spurious_test_link` の weighted 3候補について、全3組を正順・逆順で1回ずつ比較した。6/6 call が `completed` かつ有効候補 ID。`wall_ms` 合計は25,326 ms。

| 比較 ID | 正順選択 | 逆順選択 | 同一 ID の2回答 |
| --- | --- | --- | --- |
| C001, C002 | C001 | C002 | なし |
| C001, C003 | C003 | C003 | C003 |
| C002, C003 | C002 | C002 | C002 |

3候補すべてに対して同一 ID の2回答で勝った候補は0件。strict winner はなし。gold は C002。

## 生データと検証

- [順位 JSONL](issue-142-prerank-2026-09-29.jsonl): 13行。
- [top-2 JSONL](issue-142-prerank-top2-2026-09-29.jsonl): 6行、24 call。
- [pairwise JSONL](issue-142-prerank-pairwise-2026-09-29.jsonl): 1行、3組、6 call。
- JSONL は fixture、候補 ID/grouping、score、停止理由、選択 ID、時間、prompt/schema hash を記録し、生の path/diff/prompt/response は含まない。
- `GOCACHE=/private/tmp/commiter-issue142-gocache go test ./...`、同 `go vet ./...`、`git diff --check` が成功した。

対象13件は既使用の合成 fixture であり、score は過去の fixture と失敗例を見た後に固定した。未知入力に対する測定はこの実行に含まれない。各 pair の方向は1回ずつで、同方向の反復変動は測定していない。
