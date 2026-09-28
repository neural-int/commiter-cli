# Issue #142: Louvain 候補選択の固定条件による検証結果

## 条件

[事前登録した条件と benchmark 実装](issue-142-preregistered-2026-09-28.md)をコミット `1c6c934` で固定後に測定した。Gonum `v0.16.0`、固定5 resolution、seed `142,1`、既存6主 fixture・2 guardrail・新規2 fixtureを使用した。MLX model は `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。model の新規取得は行っていない。selection は各 fixture 2反復で2回目に候補表示順を反転した。各 request の出力上限1024 token、plan timeout 2分、最大2 calls、repair/retry 0を維持した。

## 候補生成のみ

5 resolution の出力は全 fixture で重複排除後2候補になった。candidate recall は主比較5/6、guardrail 2/2、新規 fixture 0/2、計7/10。新規 fixture は従来の `holdout_*` と異なり、今回初めて検証に使った。これらの gold grouping は合成入力に付けたラベルであり、実際の開発者判断との一致を別に検証していない。全10 fixture の生成時間合計は1.31ms、selection 用 system+prompt bytes の合計は38,384B（各 fixture 1回）。これはこの環境での小さい合成入力に限定した値。

| fixture | 候補数 | 正解候補 | `none` / `max_tokens`（2反復） | 有効な候補選択 | calls | wall 合計 |
| --- | ---: | :---: | ---: | ---: | ---: | ---: |
| `multi_commit` | 2 | なし | 1 / 1 | 0 | 2 | 41.0s |
| `cross_directory` | 2 | あり | 0 / 2 | 0 | 2 | 66.8s |
| `same_directory_independent` | 2 | あり | 2 / 0 | 0 | 2 | 7.0s |
| `mixed_24` | 2 | あり | 2 / 0 | 0 | 2 | 62.7s |
| `holdout_split` | 2 | あり | 1 / 1 | 0 | 2 | 60.2s |
| `holdout_join` | 2 | あり | 0 / 2 | 0 | 2 | 82.1s |
| `source_test_separate_purposes` | 2 | あり | 1 / 1 | 0 | 2 | 45.4s |
| `import_separate_purposes` | 2 | あり | 0 / 2 | 0 | 2 | 84.6s |
| `new_cross_purpose_docs` | 2 | なし | 0 / 2 | 0 | 2 | 92.0s |
| `new_same_directory_split` | 2 | なし | 0 / 2 | 0 | 2 | 95.9s |

## Selection と最終 plan

正解候補が存在する14試行の正解候補選択は0/14。そのうち6試行は `none`、8試行は `max_tokens`。正解候補が存在しない6試行では正しい `none` が1/6、残り5試行は `max_tokens`。全20試行で `none` は7件、`max_tokens` は13件。unknown ID、不正 JSON、複数選択の完了応答は0件。有効な候補 ID の選択は0件で、完全割当された実際の grouping、false merge/split の pair 指標、Pass 2 の実モデル評価は得られなかった。`none` や未完了を完全割当に含めない。

候補生成失敗は3 fixture、選択側の失敗は正解候補を含む14試行で別に記録した。正解を含まない3 fixture × 2反復のうち、`multi_commit` 1回だけが正しい `none` を返した。順序反転時の `none` は2/10、元順序では5/10だったが、20試行の小標本と失敗が多い条件であり、順序効果の有無は確定できない。

| 条件（6主 fixture × 2反復） | candidate recall | 有効な grouping / exact | 完全割当 | calls | wall 合計 |
| --- | ---: | ---: | ---: | ---: | ---: |
| production `full`（#141 の保存結果） | 対象外 | 4/12 exact | 12/12 | 12 | 149.8s |
| file-centric（#141 の保存結果） | 対象外 | 4/12 exact | 12/12 | 16（Pass 2含む） | 201.6s |
| local boundary（#141 の保存結果） | 対象外 | 6/12 exact | 12/12 | 22 | 104.1s |
| small cluster boundary（#141 の保存結果） | 対象外 | 6/12 exact | 12/12 | 48 | 218.8s |
| 今回の固定 Louvain + selection | 5/6 | 0/12 有効な grouping | 0/12 | 12 | 319.9s |

前回値は再測定していない。file-centric の表は Pass 2 を含む run の calls/wall で、local/cluster boundary は Pass 1 のみ。今回の `false merge / false split` は grouping が0件なので比較可能な pair 値がない。guardrail でも同様で、誤結合0件という安全性の証明ではない。候補生成は正解を含む7/10であったが、今回の model/prompt/schema/output budget でその候補を選ぶ実測経路は成立しなかった。

selection の prompt bytes 合計76,768B、output bytes 合計196B、backend calls 20、repair/retry 0、wall 合計637.9秒。output token 数は20行とも `unavailable`。20行の `(fixture, run, variant)` は一意。候補のみ10行と selection 20行の数値・fixture 名・候補 ID JSONL を保存し、生の prompt、raw response、生成 summary は保存していない。

## Keyed metadata 接続の検証範囲

実モデルが候補 ID を選ばなかったため、実モデルの Pass 2 call と end-to-end success は0件で、metadata の実測成功率は算出できない。backend stub が正しい候補 ID と keyed metadata を順に返す integration test では、実際の benchmark 関数から動的 `G001` 等の group ID を付け、`issue141ValidateKeyedMetadata()` 経由で `planning.Validate()` に到達して2 callsで成功した。これはコード経路の確認であり、モデルの end-to-end 成功ではない。

この事前条件では主比較の exact grouping が既存の8/12を上回らず、完全割当も維持できなかった。従って、今回の固定 generator・selection prompt・model・1024-token 上限を production に採用する根拠は得られていない。候補生成の漏れと選択の失敗が両方あるため、候補選択方式一般の可否や Gonum の production 採用を、この結果だけでは判断しない。

## 再実行

```sh
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -describe -fixture all -repeats 1 -output-tokens 1024
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <SHA-256 が上記と一致する helper> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
```

production planner、SRS、通常 CLI、Git mutation は変更していない。
