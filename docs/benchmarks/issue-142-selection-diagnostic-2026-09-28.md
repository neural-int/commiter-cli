# Issue #142: 候補選択の出力予算と gold 追加の診断結果

## 条件

[事前条件](issue-142-selection-diagnostic-preregistered-2026-09-28.md)と実装をコミット `20b5c36` で固定してから実測した。前回と同じ Louvain 候補生成、prepared input、selection prompt/schema、MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を使用した。新たなモデル取得は行っていない。各 call は2分で打ち切り、repair/retry は0、Pass 2 は実行していない。

## A. 正解候補がある4 fixture の出力予算比較

既使用 fixture `cross_directory`、`same_directory_independent`、`mixed_24`、`source_test_separate_purposes` に対し、1024/2048/4096 token 上限を各2反復で比較した。各 `(fixture, run)` 内の system+prompt、schema、prompt bytes は3予算で一致した。2回目の反復では候補提示順を反転した。

| 出力上限 | 試行 | `completed` | `max_tokens` | 2分で時間切れ | 完了応答の `none` | 有効な候補ID | 正解選択 | wall 合計 |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1024 | 8 | 5 | 3 | 0 | 5 | 0 | 0 | 176.7s |
| 2048 | 8 | 5 | 3 | 0 | 5 | 0 | 0 | 287.5s |
| 4096 | 8 | 5 | 0 | 3 | 5 | 0 | 0 | 426.8s |

| fixture | 6試行の stop / 応答 | 有効な候補ID |
| --- | --- | ---: |
| `cross_directory` | `max_tokens` 4、時間切れ 2 | 0 |
| `same_directory_independent` | `none` 6 | 0 |
| `mixed_24` | `none` 6 | 0 |
| `source_test_separate_purposes` | `none` 3、`max_tokens` 2、時間切れ 1 | 0 |

24試行のキーは重複せず、出力 token telemetry はすべて `unavailable`。完了した有効応答15件の正解選択は0/15、全24試行では0/24。選択された候補IDがないため、実モデルの grouping、完全割当、false merge/split の pair 指標は得られなかった。4096 token で `max_tokens` が時間切れに置き換わった3条件について、2分以内に完了するかは未確認。出力予算や stop reason から、内部 token の内訳や semantic 判断の原因は特定できない。

## B. 候補漏れ fixture への gold 追加

既使用の `multi_commit` と `new_same_directory_split` で、元の2候補と、合成 gold + 元の非正解候補1つをそれぞれ2反復で比較した。両 arm とも出力上限4096 token、2分の時間上限を使用した。gold 追加側の gold ID は各 fixture の1回目が `C001`、2回目が `C002` で、提示位置も入れ替えた。arm 名と gold label は prompt に渡していない。

| arm | 試行 | 正解候補を含む | `completed` | 完了応答の `none` | 2分で時間切れ | 有効な候補ID | wall 合計 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 元の Louvain 2候補 | 4 | 0 | 1 | 1 | 3 | 0 | 366.3s |
| gold + 元の非正解候補 | 4 | 4 | 0 | 0 | 4 | 0 | 482.7s |

`multi_commit` の元候補1回目だけが正しい `none` を返した。残りの元候補3試行と gold 追加4試行は時間切れだった。8試行のキーは重複せず、schema hash はすべて一致した。出力 token telemetry は8試行とも `unavailable`。gold を追加した条件で正解候補の選択は0/4、完了した有効応答を分母とする率は分母0で算出できない。候補IDが得られず、この補助 probe でも実モデルの Pass 2、完全割当、pair 指標は測れなかった。

## 検証範囲と再実行

この結果は既使用の合成 fixture、固定モデル、固定 prompt/schema、2分の時間上限に限定される。gold は合成 fixture に付けた正解であり、実際の開発者判断との一致を測っていない。追加候補による変化から generator と selector の独立した因果効果は推定しない。

```sh
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe budget -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <上記 SHA-256 と一致する helper> -fixture all
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe gold -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <上記 SHA-256 と一致する helper> -fixture all
```

数値・stop reason・候補ID・hash を [予算比較 JSONL](issue-142-selection-budget-mlx.jsonl)と [gold 追加 JSONL](issue-142-selection-gold-mlx.jsonl)に保存した。生の prompt、response、repository content は保存していない。production planner、SRS、通常 CLI の contract は変更していない。
