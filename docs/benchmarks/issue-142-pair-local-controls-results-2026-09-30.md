# Issue #142: pair-local same 対照の観測結果

## 実行条件

- [事前登録](issue-142-pair-local-controls-preregistered-2026-09-30.md) は `4dfd034`、検証コードは `0d90fc9` でモデル実行前にコミットした。
- 既存の合成 fixture `new_test_pair_shared` F001/F002と `new_crossdir_shared` F001/F002を使い、両方の合成 gold は `same`。各4 call、計8 call。
- 前回の `pair-local` と同じ入力生成・system・task・schema・出力検証を使った。モデルは `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。output上限2048 tokens、各 call 2分上限、retry/repair 0。
- 結果は [JSONL](issue-142-pair-local-controls-2026-09-30.jsonl) に記録した。生 path/diff/prompt/response は保存していない。schema hash 2種類は前回の pair-local probe と一致した。

## 観測

| Fixture | Gold | Run | ファイル順 | Enum 順 | Stop | 選択 | 有効 | Gold 一致 | Wall ms |
| --- | --- | ---: | --- | --- | --- | --- | --- | --- | ---: |
| `new_test_pair_shared` | `same` | 1 | F001,F002 | same,different | completed | same | yes | yes | 2775.9 |
| `new_test_pair_shared` | `same` | 2 | F002,F001 | different,same | completed | same | yes | yes | 1973.6 |
| `new_test_pair_shared` | `same` | 3 | F002,F001 | same,different | completed | different | yes | no | 1969.4 |
| `new_test_pair_shared` | `same` | 4 | F001,F002 | different,same | completed | same | yes | yes | 1956.5 |
| `new_crossdir_shared` | `same` | 1 | F001,F002 | same,different | completed | different | yes | no | 1835.8 |
| `new_crossdir_shared` | `same` | 2 | F002,F001 | different,same | completed | different | yes | no | 1870.5 |
| `new_crossdir_shared` | `same` | 3 | F002,F001 | same,different | completed | different | yes | no | 1844.3 |
| `new_crossdir_shared` | `same` | 4 | F001,F002 | different,same | completed | different | yes | no | 1892.1 |

全8 callは各1回の backend callで `completed`、有効ラベルを返した。合成 goldとの一致は `new_test_pair_shared` 3/4、`new_crossdir_shared` 0/4。`new_test_pair_shared` の回答は `same` 3回と `different` 1回、`new_crossdir_shared` は `different` 4回だった。outputは `same` が24 bytes、`different` が29 bytesで、token数は取得できなかった。

## 前回との記述的比較と範囲

| Pair-local fixture | Gold | `same` | `different` | Gold 一致 |
| --- | --- | ---: | ---: | ---: |
| 前回 `new_paraphrase` | `same` | 0/4 | 4/4 | 0/4 |
| 前回 `new_stem_doc_diverged` | `different` | 0/4 | 4/4 | 4/4 |
| 今回 `new_test_pair_shared` | `same` | 3/4 | 1/4 | 3/4 |
| 今回 `new_crossdir_shared` | `same` | 0/4 | 4/4 | 0/4 |

前回の結果は [pair-local JSONL](issue-142-pair-local-2026-09-29.jsonl) に記録されている。今回の `same` 3回は、この固定 contract が `new_test_pair_shared` の入力では `same` を返した事実を示す。両 positive control は fixture 内容、relation type、語・path の手掛かりが異なり、各順序条件は1回ずつである。選択理由、fixture間の差の単一原因、反復変動、順序の因果効果、一般的な semantic capability、production精度は測っていない。新規依存と production コードは変更していない。
