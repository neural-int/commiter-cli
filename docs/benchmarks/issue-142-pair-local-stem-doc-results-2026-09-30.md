# Issue #142: stem-doc pair-local probe の観測結果

## 実行条件

- [事前登録](issue-142-pair-local-stem-doc-preregistered-2026-09-30.md) を `c4110ce`、検証コードを `5b1f872` でモデル実行前にコミットした。
- 既存の合成 fixture `new_stem_doc_shared` F001/F002（gold `same`）だけを使用し、既存の pair-local 入力生成・system・task・schema・response 判定を再利用した。4 call はファイル順×enum順の事前登録済み4条件を各1回。
- モデルは `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。output上限2048 tokens、各 call 2分上限、retry/repair 0。
- [結果 JSONL](issue-142-pair-local-stem-doc-2026-09-30.jsonl)には生 path/diff/prompt/response を保存していない。schema hash 2種類は先行の pair-local control と一致した。

## 観測

| Run | ファイル順 | Enum 順 | Stop | 選択 | 有効 | Gold 一致 | Wall ms |
| ---: | --- | --- | --- | --- | --- | --- | ---: |
| 1 | F001,F002 | same,different | completed | different | yes | no | 2468.6 |
| 2 | F002,F001 | different,same | completed | different | yes | no | 1948.7 |
| 3 | F002,F001 | same,different | completed | different | yes | no | 1845.4 |
| 4 | F001,F002 | different,same | completed | different | yes | no | 1809.0 |

全4 callは各1回の backend callで `completed`、有効ラベル `different` を返した。合成 gold `same` との一致は0/4。outputは各29 bytes、token数は取得できなかった。wall合計は8071.7 ms、範囲は1809.0–2468.6 ms。prompt hashは4種類、schema hashは2種類だった。

## #142 の関連測定値の整理

| 対象・固定条件 | 観測 |
| --- | --- |
| [構造的加点0.4上限・新規合成8件](issue-142-capped-structural-results-2026-09-29.md) | 既存・二値 lexical・weighted lexical generator は各 fixture で同じ候補を生成し、追加候補0件。現行 score の recall@2 は4/8、0.4上限 score は5/8。3件は gold 候補なし。 |
| [3件の小規模 partition 列挙](issue-142-selector-enumeration-results-2026-09-29.md) | 各3件・計9件を追加し、対象3件の gold は全件候補集合に入った。gold の順位は3位、2位、2位。追加非 gold は計6件。 |
| [固定 top-2 complete-partition selector](issue-142-selector-enumeration-results-2026-09-29.md)、[別2件の selector](issue-142-two-selector-breakdown-results-2026-09-29.md) | `new_stem_doc_diverged` は非 gold C001を4/4、`new_crossdir_collision` は gold C004を4/4、`new_paraphrase` は非 gold C001を4/4選択。計12/12が completed・有効ID。 |

先行と今回の pair-local 測定を同じ表に置く。各 fixture は同じ4種類のファイル順×enum順を各1回使ったが、fixture 内容は異なる。

| Pair-local fixture | 2ファイルの形 | Gold | `same` | `different` | Gold 一致 | 出典 |
| --- | --- | --- | ---: | ---: | ---: | --- |
| `new_paraphrase` | code/docs | same | 0/4 | 4/4 | 0/4 | [先行結果](issue-142-pair-local-results-2026-09-29.md) |
| `new_stem_doc_diverged` | code/docs | different | 0/4 | 4/4 | 4/4 | [先行結果](issue-142-pair-local-results-2026-09-29.md) |
| `new_test_pair_shared` | source/test | same | 3/4 | 1/4 | 3/4 | [対照結果](issue-142-pair-local-controls-results-2026-09-30.md) |
| `new_crossdir_shared` | code/docs | same | 0/4 | 4/4 | 0/4 | [対照結果](issue-142-pair-local-controls-results-2026-09-30.md) |
| `new_stem_doc_shared` | code/docs | same | 0/4 | 4/4 | 0/4 | 今回 |

上記は少数の合成 fixture での記述的集計。complete-partition と pair-local は task、入力、schema、出力形式が異なる。pair-local の fixture 間でも path、内容、語彙、役割が同時に異なり、各条件の反復はない。回答理由、差の単一原因、反復変動、一般的能力、production精度は測っていない。新規依存と production コードは変更していない。
