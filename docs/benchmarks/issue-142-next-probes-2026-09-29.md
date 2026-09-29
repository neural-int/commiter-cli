# Issue #142: 新規合成 fixture、lexical 候補 ranking、relation context 除去の実測

## 条件

[事前登録](issue-142-next-probes-preregistered-2026-09-29.md)を `1ab8f06`、検証コードを `888994c` でコミットした。ranking の初回 pilot で逆順提示時に schema enum も反転する条件違反を確認し、`2d1d1e1` で修正と回帰テストをコミットしてから本測定を最初から実行した。pilot 8 call は本測定に含めない。

新規4件は前回の結果を見た後に作成した合成 fixture であり、実開発データや盲検の無作為 holdout ではない。lexical tokenization、stopword、edge rule、候補上限8は前回から変更していない。モデルは `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。モデルと helper は既存のものを再利用し、新規依存やモデル取得は行っていない。

モデル call は output 2048 tokens、各 call 2分上限、retry/repair 0。ranking は baseline と lexical の各候補を正順・逆順で1回ずつ、relation context 除去は各 arm を正順・逆順・逆順・正順で実行した。生の prompt、response、path、diff は結果 JSONL に保存していない。

## Lexical candidate generation

| fixture | 候補数 baseline → lexical | 合成 gold を含む baseline → lexical | 新候補 | lexical partition が gold と一致 | lexical edge gold 内 / 間 | cap 到達 |
| --- | ---: | :---: | --- | :---: | ---: | :---: |
| `lexical_crossdir_pairs` | 2 → 3 | × → ○ | C003 = gold | ○ | 2 / 0 | × |
| `lexical_paraphrase_gap` | 2 → 2 | × → × | なし | × | 0 / 0 | × |
| `lexical_collision` | 2 → 3 | ○ → ○ | C003 ≠ gold | × | 0 / 1 | × |
| `lexical_bridge` | 2 → 2 | × → × | なし | × | 2 / 4 | × |
| **4件合計** | — | **1/4 → 2/4** | **追加2件、うち非 gold 1件** | **1/4** | **4 / 5** | **0/4** |

`lexical_crossdir_pairs` の C003 は `{F001,F002}` / `{F003,F004}` で合成 gold と一致した。`lexical_collision` の C003 は `{F001,F002}` / `{F003}` で合成 gold の3 singleton と異なった。`lexical_paraphrase_gap` は edge 0件、`lexical_bridge` は edge 6件で lexical partition が既存候補と重複し、新候補は追加されなかった。

## Lexical 候補を含む ranking

| fixture | baseline 候補数 | lexical 候補数 | baseline 有効回答 / max_tokens | lexical 有効回答 / max_tokens | wall baseline / lexical |
| --- | ---: | ---: | ---: | ---: | ---: |
| `lexical_crossdir_pairs` | 2 | 3 | 0/2 / 2/2 | 0/2 / 2/2 | 140.9s / 155.1s |
| `lexical_paraphrase_gap` | 2 | 2 | 0/2 / 2/2 | 0/2 / 2/2 | 146.1s / 146.0s |
| `lexical_collision` | 2 | 3 | 0/2 / 2/2 | 0/2 / 2/2 | 147.2s / 147.1s |
| `lexical_bridge` | 2 | 2 | 0/2 / 2/2 | 0/2 / 2/2 | 153.8s / 151.4s |
| **合計** | — | — | **0/8 / 8/8** | **0/8 / 8/8** | **588.1s / 599.7s** |

16/16 call が `max_tokens` で停止し、完了・有効な candidate ID または `none` は0件だった。合成 gold の選択率は算出できない。各 fixture・arm 内で正順と逆順の schema hash は一致した。wall の全 call 合計は1187.8秒。output token telemetry は利用できなかった。

## Relation context 除去

候補は relation context を含む prepared input から生成し、各 fixture の C001/C002 pair を両 arm に固定した。除去 arm で省略した repository input のフィールドは `relation_context` のみ。テストで system、schema、候補、他の repository input フィールドが一致することを確認した。各 fixture の8 call で schema hash は一致した。回答順は正順・逆順・逆順・正順。

| fixture | 合成 gold | context ありの4回答 | context なしの4回答 | 有効回答 あり / なし | gold 選択 あり / なし | wall あり / なし |
| --- | --- | --- | --- | ---: | ---: | ---: |
| `verify_misleading_relation` | C002 | C001 / C001 / C001 / C001 | C002 / C001 / C001 / C002 | 4/4 / 4/4 | 0/4 / 2/4 | 13.3s / 11.0s |
| `verify_join_present` | C001 | C001 / C001 / C001 / C001 | max_tokens / C001 / C001 / max_tokens | 4/4 / 2/4 | 4/4 / 2/4 | 19.5s / 168.8s |
| `holdout_spurious_test_link` | C002 | C002 / C002 / C002 / C002 | max_tokens / C001 / C001 / max_tokens | 4/4 / 2/4 | 4/4 / 0/4 | 19.5s / 178.2s |
| **合計** | — | — | — | **12/12 / 8/12** | **8/12 / 4/12** | **52.4s / 358.0s** |

context なし arm の `max_tokens` は4/12件で、`verify_join_present` と `holdout_spurious_test_link` の正順提示に各2件発生した。context あり arm の `max_tokens` は0/12件。`verify_misleading_relation` の context なし arm では正順の2件が C002、逆順の2件が C001 だった。

## 測定記録と範囲

- [generator JSONL](issue-142-next-lexical-2026-09-29.jsonl): 4行。
- [ranking JSONL](issue-142-next-ranking-mlx.jsonl): 4行、16 call。
- [relation context JSONL](issue-142-relation-removal-mlx.jsonl): 3行、24 call。
- [条件違反 pilot JSONL](issue-142-next-ranking-invalid-schema-pilot.jsonl): 2行、8 call。逆順時に schema enum が反転したため本測定から除外。

`GOCACHE=/private/tmp/commiter-issue142-gocache go test ./...`、`go vet ./...`、`git diff --check` は成功。測定対象は少数の合成 fixture であり、実開発者判断との一致、未知の実入力、production 精度・コスト、Pass 2 は測定していない。
