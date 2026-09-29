# Issue #142: candidate-difference-only と lexical candidate の探索的実測

## 条件

[事前登録条件](issue-142-difference-lexical-preregistered-2026-09-29.md)を `3ed528f` でコミットし、検証専用コードを `1522022` でコミットしてから実行した。対象は既使用5件と前回 holdout 4件の合成 fixture。selector probe はうち3件の既存 C001/C002 pair を固定し、complete-partition と difference の2 armを4回ずつ比較した。generator probe は9件全てに同一の lexical edge 規則を適用し、ranking は行っていない。

selector は MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`、output 2048 tokens、call ごとに2分上限、retry/repair 0。候補 ID と grouping、repository input、pair 内の JSON Schema enum 順を固定した。各 arm の提示順は `A,B`、`B,A`、`B,A`、`A,B`。difference arm では両候補で co-membership が異なる file pair だけを候補表示とし、repository input は維持した。両 arm の system/task 文言も異なる。

## Selector probe

| fixture | 合成 gold | 差分 file pair | complete-partition の4回答 | difference の4回答 | 完了・有効 ID（complete / difference） | gold 選択（complete / difference） | wall（complete / difference） |
| --- | --- | ---: | --- | --- | ---: | ---: | ---: |
| `verify_misleading_relation` | C002 | 1 | C001 / C001 / C001 / C001 | C001 / max_tokens / max_tokens / C001 | 4/4 / 2/4 | 0/4 / 0/4 | 13.5s / 132.4s |
| `verify_join_present` | C001 | 3 | C001 / C001 / C001 / C001 | C001 / C001 / C001 / C001 | 4/4 / 4/4 | 4/4 / 4/4 | 15.4s / 16.3s |
| `holdout_atomic_feature` | C001 | 3 | C001 / C002 / C002 / C001 | C001 / C001 / C001 / C001 | 4/4 / 4/4 | 2/4 / 4/4 | 15.6s / 17.2s |
| **合計** | — | — | — | — | **12/12 / 10/12** | **6/12 / 8/12** | **44.4s / 165.9s** |

合計24 call、wall 210.3秒。22件が `completed` かつ有効 ID、2件が `max_tokens`。未完了2件はいずれも `verify_misleading_relation` difference arm の逆順提示だった。complete-partition は同 fixture で C001 を4/4回選び、difference は完了した2件で C001 を選んだ。`holdout_atomic_feature` の complete-partition は各提示順の先頭候補を2回ずつ選び、difference は4回とも C001 を選んだ。

同一 fixture・pair の両 arm は同じ Schema hash を持ち、同一 arm・同一提示順の prompt hash は一致した。output token telemetry は24件全て `unavailable`。先頭提示順の prompt bytes は `verify_misleading_relation` で complete 2304 / difference 2487、`verify_join_present` で2928 / 3353、`holdout_atomic_feature` で2917 / 3342だった。

## Generator probe

lexical 規則は path と diff の英字 token を CamelCase/snake_case/区切り文字で分割し、固定 stopword と4文字未満を除外した。共有 token が1つ以上ある file pair を結び、連結成分の complete partition を候補末尾に追加した。重複候補は追加せず、上限8を維持した。

| fixture | 既存候補数 → lexical 後 | 既存 gold → lexical 後 | lexical partition の gold 一致 | lexical edge（gold 内 / gold 間） | 新候補 |
| --- | ---: | :---: | :---: | ---: | --- |
| `verify_join_present` | 2 → 2 | ○ → ○ | ○ | 3 / 0 | なし |
| `verify_split_present` | 3 → 3 | ○ → ○ | × | 2 / 4 | なし |
| `verify_join_absent` | 3 → 3 | ○ → ○ | ○ | 1 / 0 | なし |
| `verify_split_absent` | 3 → 3 | ○ → ○ | ○ | 1 / 0 | なし |
| `verify_misleading_relation` | 2 → 2 | ○ → ○ | × | 0 / 1 | なし |
| `holdout_crossdir_semantic` | 3 → 4 | × → ○ | ○ | 2 / 0 | C004 |
| `holdout_same_directory_pairs` | 3 → 3 | ○ → ○ | × | 2 / 1 | なし |
| `holdout_spurious_test_link` | 3 → 3 | ○ → ○ | × | 0 / 1 | なし |
| `holdout_atomic_feature` | 2 → 2 | ○ → ○ | ○ | 3 / 0 | なし |

9件中1件で新候補を追加し、既存候補に対する合成 gold 包含は8/9から9/9となった。新候補 C004 は `holdout_crossdir_semantic` の gold partition と一致し、F001/F003 は共有 token `session`、F002/F004 は `latency` で結ばれた。全21 lexical edge のうち合成 gold 同一 group 内は14、異なる group 間は7。後者には `verify_split_present` の `shared`、`verify_misleading_relation` と `holdout_spurious_test_link` の `export`、`holdout_same_directory_pairs` の `fatal` / `testing` が含まれる。上限8に到達した fixture は0件。gold と一致しない lexical partition は4/9件で、いずれも既存候補と重複した。

## 記録と範囲

[selector JSONL](issue-142-difference-mlx.jsonl)は3行、[generator JSONL](issue-142-lexical-2026-09-29.jsonl)は9行。両方とも fixture 名は一意。selector の各 fixture は8 call で、schema hash は8 call で同一、同方向の prompt hash は arm 内で一致した。JSONL には生の prompt、response、ファイル diff・path を保存していない。generator JSONL には抽出した共有 token とファイル ID を保存した。新規依存、モデル取得、production planner、Pass 2、通常 CLI は実行していない。

測定対象は既使用の少数の合成 fixture に限る。実開発者判断との一致、未知 fixture の candidate recall、production 精度・コスト、提示形式の単独因果効果、候補追加後の ranking 結果は測定していない。

```sh
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe lexical -fixture all
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe difference -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <上記 SHA-256 と一致する helper> -fixture all
```
