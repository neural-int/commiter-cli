# Issue #142: 固定2 fixture selector と score 内訳の事前登録

## 対象・固定条件

- [修正済み考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5892051369)に従う。fixture/gold は `99dc127`、現行 score と構造的0.4上限 score は `6bf7f04` / `bc3bd66`、小規模列挙規則は `b7cb94c` / `d6fa3c0` のまま。generator、score、列挙順、candidate ID/grouping、repository input、production planner、Pass 2 は変更しない。
- 検証コードを結果確認前にコミットする。モデル call を先に実行し、次に決定的な score 内訳を計算する。両結果は別 JSONL に記録し、結果を見て条件を変えない。

## 2-way forced-choice

- `new_crossdir_collision`: 列挙後の weighted 候補を構造的0.4上限 score で並べた top-2、C003（非 gold）/ C004（gold）。
- `new_paraphrase`: 同条件の top-2、C001（非 gold）/ C003（gold）。候補 ID/gold/top-2 が上記と異なればモデル call 前に中止する。
- 両 fixture とも既存の generic forced-choice、`none` なし、schema enum ID順。提示順は正順・逆順・逆順・正順の4 call、計8 call。pinned MLX 3B `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。output 2048 tokens、各 call 2分上限、retry/repair 0。
- 停止理由、有効候補 ID、選択、gold 選択、方向、wall、prompt/schema hashを記録する。各 fixture 4 call の結果は他 fixture や一般的な能力・原因に外挿しない。

## `new_test_pair_diverged` の score 内訳

- 列挙後の weighted 候補 C003（F001+F002）と C004（F001+F003）を固定する。prepared document の soft source-test / `matching_test_path` +4、soft direct-import / `observed_import_path` +3、stem一致 +1、同じ directory +0.1を pair ごとに別記する。raw structural合計と `min(0.4, raw structural)` を両方記録する。pair penalty は0.5。
- lexical token は既存の `[A-Z][a-z]+|[A-Z]+|[a-z]+`、lowercase、4文字以上、既存 stop list を使用し、path と raw diff の union を file token とする。各共有 token の重みは全 file の document frequency `df` に対する `1/(df-1)`。共有 token を相互排他的に、**双方の diff にある場合は diff-diff、そうでなく双方の path にある場合は path-path、それ以外は mixed** と分類する。path/diff 双方に存在する token を二重計上しない。この分類は計算上の帰属規則であり、因果的な証拠源の独立性を示さない。
- pair 別に structural各種、capped structural、lexical総量と上記3分類、共有token数、penalty、現行/上限付きpair scoreを記録する。候補のpair score合計が前回の候補score（現行 C003=4.6、C004=1.5、上限付き C003=C004=0.9）と一致することを検証する。生 path/diff/token文字列は結果 JSONL に保存しない。

## 測定範囲

- 新規依存・モデル取得はしない。モデル条件2件・8 call、内訳1 fixture・2候補のみ。production採用、score調整、generator追加、Pass 2は行わない。
