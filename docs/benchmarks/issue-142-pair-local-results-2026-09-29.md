# Issue #142: 2ファイル pair-local judgment の観測結果

## 実行条件

- [事前登録](issue-142-pair-local-preregistered-2026-09-29.md) は `7915015`、検証コードは `f4d4bcf` でモデル実行前にコミットした。
- 対象は既存の合成 fixture `new_paraphrase` F001/F002（gold `same`）と `new_stem_doc_diverged` F001/F002（gold `different`）。各4 call、計8 call。
- モデルは `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。output 上限2048 tokens、各 call 2分上限、retry/repair 0。
- 入力は2ファイルの ID・path・raw diff。complete candidate ID、partition、relation context、第三ファイルは含めていない。4 call はファイル順と schema の enum 順を事前登録した順番で切り替えた。
- 生 prompt・path・diff・response は [JSONL](issue-142-pair-local-2026-09-29.jsonl) に含めていない。prompt/schema の SHA-256、停止理由、選択、有効性、wall、出力 bytes 等を記録した。

## 観測

| Fixture | Gold | Run | ファイル順 | Enum 順 | Stop | 選択 | 有効 | Gold 一致 | Wall ms |
| --- | --- | ---: | --- | --- | --- | --- | --- | --- | ---: |
| `new_paraphrase` | `same` | 1 | F001,F002 | same,different | completed | different | yes | no | 2813.0 |
| `new_paraphrase` | `same` | 2 | F002,F001 | different,same | completed | different | yes | no | 2202.5 |
| `new_paraphrase` | `same` | 3 | F002,F001 | same,different | completed | different | yes | no | 2046.6 |
| `new_paraphrase` | `same` | 4 | F001,F002 | different,same | completed | different | yes | no | 2000.3 |
| `new_stem_doc_diverged` | `different` | 1 | F001,F002 | same,different | completed | different | yes | yes | 2032.0 |
| `new_stem_doc_diverged` | `different` | 2 | F002,F001 | different,same | completed | different | yes | yes | 1942.1 |
| `new_stem_doc_diverged` | `different` | 3 | F002,F001 | same,different | completed | different | yes | yes | 1991.2 |
| `new_stem_doc_diverged` | `different` | 4 | F001,F002 | different,same | completed | different | yes | yes | 1994.5 |

全8 call は各1回の backend call で `completed`、有効ラベル `different` を返した。合成 gold との一致は `new_paraphrase` 0/4、`new_stem_doc_diverged` 4/4。output は各29 bytes、token 数は backend から取得できなかった。`new_paraphrase` の wall は合計9062.4 ms（範囲2000.3–2813.0 ms）、`new_stem_doc_diverged` は合計7959.8 ms（範囲1942.1–2032.0 ms）。各 fixture で prompt hash は4種類、schema hash は enum 順に対応する2種類だった。

## 先行検証との対応範囲

先行の complete-partition 2-way selector は `new_paraphrase` で非 gold C001 を4/4、`new_stem_doc_diverged` で非 gold C001 を4/4選択していた（[two-selectors JSONL](issue-142-two-selectors-2026-09-29.jsonl)、[capped-selector JSONL](issue-142-capped-selector-2026-09-29.jsonl)）。今回の pair-local では選択肢が `same` / `different` で、対象ファイル数・task・system・schema・出力形式が異なる。両検証の正誤差を単一要因の効果として扱わない。

各ファイル順×enum順は1回ずつであり、反復変動や順序の因果効果は測っていない。合成 fixture 2件の観測であり、モデルの一般的能力、判断根拠、production grouping の精度は測っていない。production コードと依存は変更していない。
