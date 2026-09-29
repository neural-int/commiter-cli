# Issue #142: bounded candidate set と pairwise tournament の実測結果

## 条件

[事前条件と実装](issue-142-tournament-preregistered-2026-09-29.md)を `ee7b6a7` で固定し、その後に新しい holdout fixture 4件を `42f7ad0` で追加した。既使用の verification fixture 5件と holdout 4件をそれぞれ2つの bracket 順序で測定した。既存 Louvain 候補を保持し、stem / soft relation の連結成分、all-joined、all-split を順に加え、canonicalize と dedupe を行った。上限は8候補。実際の候補数は全fixtureで2または3件だった。

前回と同じ MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を使用した。各 pair call は forced-choice、2048 output tokens、2分上限、repair/retry 0。2回目は候補IDと grouping の対応を保って bracket 順序を反転した。モデル・依存の新規取得、absolute verifier、Pass 2、通常 CLI は実行していない。

## 候補生成

| 区分 | fixture | 既存候補数 | 拡張候補数 | 既存 gold 包含 | 拡張 gold 包含 |
| --- | --- | ---: | ---: | :---: | :---: |
| 既使用 | `verify_join_present` | 2 | 2 | ○ | ○ |
| 既使用 | `verify_split_present` | 2 | 3 | ○ | ○ |
| 既使用 | `verify_join_absent` | 2 | 3 | × | ○ |
| 既使用 | `verify_split_absent` | 2 | 3 | × | ○ |
| 既使用 | `verify_misleading_relation` | 2 | 2 | ○ | ○ |
| holdout | `holdout_crossdir_semantic` | 2 | 3 | × | × |
| holdout | `holdout_same_directory_pairs` | 2 | 3 | ○ | ○ |
| holdout | `holdout_spurious_test_link` | 2 | 3 | ○ | ○ |
| holdout | `holdout_atomic_feature` | 2 | 2 | ○ | ○ |

合成 gold 候補の包含は既使用5件で既存3/5・拡張5/5、holdout 4件で既存3/4・拡張3/4、全9件で既存6/9・拡張8/9。`holdout_crossdir_semantic` の gold partition は拡張後も候補集合に存在しなかった。

## Pairwise tournament

| 区分 | fixture | gold ID | 選択 ID（昇順 / 降順） | 完了 | 最終 gold 一致 | calls | wall |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: |
| 既使用 | `verify_join_present` | C001 | C001 / C002 | 2/2 | 1/2 | 2 | 11.4s |
| 既使用 | `verify_split_present` | C001 | C001 / C001 | 2/2 | 2/2 | 4 | 31.1s |
| 既使用 | `verify_join_absent` | C003 | C003 / C003 | 2/2 | 2/2 | 4 | 32.8s |
| 既使用 | `verify_split_absent` | C003 | C003 / C003 | 2/2 | 2/2 | 4 | 31.9s |
| 既使用 | `verify_misleading_relation` | C002 | C001 / C002 | 2/2 | 1/2 | 2 | 13.1s |
| holdout | `holdout_crossdir_semantic` | なし | C001 / 未確定 | 1/2 | 0/2 | 3 | 129.8s |
| holdout | `holdout_same_directory_pairs` | C001 | C001 / C001 | 2/2 | 2/2 | 4 | 32.2s |
| holdout | `holdout_spurious_test_link` | C002 | C002 / C003 | 2/2 | 1/2 | 4 | 26.6s |
| holdout | `holdout_atomic_feature` | C001 | C001 / C002 | 2/2 | 1/2 | 2 | 13.2s |

既使用は10/10試行が完了し、最終 gold 一致8/10。holdout は7/8試行が完了し、全8試行での gold 一致4/8、完了7試行では4/7。全体では17/18試行が完了し、全18試行で gold 一致12/18、完了17試行では12/17。gold 候補が存在した16試行に限ると12/16で一致した。gold 候補のない2試行は、1件が候補選択、1件が未確定だった。

29 pair calls のうち28件は `completed` かつ有効な候補ID、1件は `max_tokens`。未完了は `holdout_crossdir_semantic` の降順試行の最初の比較で、選択候補を確定していない。`none` / reject として数えていない。calls は既使用16、holdout13、合計29。wall 合計は既使用120.4秒、holdout201.8秒、合計322.2秒。未完了call単体は113.9秒だった。output token telemetry は29件すべて `unavailable`。

両順序が完了した8 fixture のうち、最終選択IDが一致したのは4件、異なったのは4件。未完了の1 fixture は比較不能。bracket 順序変更は、各比較に入る候補・提示位置も変えるため、差の原因を分解していない。完了17試行の最終 grouping の pair 指標合計は false merge 4、false split 8。未完了試行をこの合計に含めていない。

## 記録と範囲

[モデル結果 JSONL](issue-142-tournament-mlx.jsonl)は18行で、`(fixture, run)` は一意。候補数、包含、bracket ID、各 call の stop reason、選択ID、prompt/schema hash、wall、最終 pair 指標を含む。生の prompt、response、repository content は保存していない。今回の合成 gold と実開発者判断の一致、production での分布・精度、候補追加と選択結果の因果関係は測定していない。

```sh
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe tournament -describe -fixture all
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe tournament -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <上記 SHA-256 と一致する helper> -fixture all
```
