# Issue #143: 固定 selector contract でのモデル交換比較の観測結果

予定した240 callを実行した。4モデルは各48/48でcompleted・有効candidate IDを返し、Nemotronは48/48でinternal_errorだった。事前に人間が確定した次段候補条件を全て満たす比較モデルは0件だった。

## 固定条件と記録

- 起点は#142検証ブランチの `aefc2e6`。事前条件/fixture/manifestは `b66d7ea`、実行コードは `d8936a2`、baselineの配布metadata訂正は推論前の `c41de60`。実行source HEADは `c41de60c7934ffda4821b47b851f4e524ed45daa`。
- #142の `issue142GenericForcedInput`、固定2候補forced-choice、同じhelper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。同じsystem/user bytes・candidate ID/grouping・schemaを使い、モデル別のthinking/sampling最適化は行っていない。既存のgrammar拘束/greedy条件のモデル交換比較である。
- 既知4例と未使用合成8例を分けた。各例は正順・逆順・逆順・正順の4call。各方向2反復、5モデル、計240call。2048 output tokens、120秒/call、context設定8192tokens、repair/retry 0。追加warm-upなし。
- holdoutはsplit側4例、正しいjoinを要する4例、うちguardrail2例。2候補はgoldを含むselector対照として事前に手動固定した。候補生成精度の測定ではない。gold/rationale/categoryはモデルrequestに含めていない。
- Apple M3 / unified memory 16GiB / macOS 26.6.2 arm64 / Go 1.27.1。helperは各callで起動し、wallはhelper起動・モデルロードを含むbackend call全体。モデル実行位置をfixture/runごとに固定ローテーションした。
- 全240行でmodel pin・helper hash・入力hash・context・出力予算・call数を検査した。sequenceは1〜240、model/fixture/runは全て一意。既知4例の2方向prompt/schema hashは#142の記録と一致した。
- 結果JSONLにraw prompt/response/生成summaryは保存していない。全240行のoutput token数はunavailable。停止理由、候補ID、pair件数、prompt/output bytes、wall等を記録した。

## モデル別の選択結果

gold成功の分母は全予定call。未完了・不正出力も失敗として含める。有効候補は事前に全file IDのexactly-once割当を検証している。ここでの完全割当は選択partitionの構造であり、Pass 2や最終planの成功率ではない。

| モデル | 既知gold成功 | holdout gold成功 | 全call completed / 有効ID | internal_error | max_tokens / timeout |
| --- | ---: | ---: | ---: | ---: | ---: |
| Ministral 3B baseline | 0/16 | 14/32 | 48/48 | 0 | 0 / 0 |
| Granite 4.2 3B | 8/16 | 16/32 | 48/48 | 0 | 0 / 0 |
| Phi-4-mini 3.8B | 8/16 | 16/32 | 48/48 | 0 | 0 / 0 |
| Nemotron 3 Nano 3.97B | 0/16 | 0/32 | 0/48 | 48 | 0 / 0 |
| Gemma E4B | 10/16 | 12/32 | 48/48 | 0 | 0 / 0 |

## Holdoutのgroupingと費用

false merge/splitは確定したpartitionとgoldのunordered file pair比較。正しいjoinはjoin側4fixture × 4callのgold partition一致。guardrailは2fixture × 4call。

| モデル | 有効partition | false merge pair | false split pair | 正しいjoin | guardrail gold一致 | guardrail false merge pair | wall中央値 | holdout wall合計 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Ministral 3B baseline | 32/32 | 6 | 24 | 4/16 | 6/8 | 2 | 4.312s | 142.164s |
| Granite 4.2 3B | 32/32 | 14 | 16 | 8/16 | 4/8 | 4 | 3.848s | 127.722s |
| Phi-4-mini 3.8B | 32/32 | 14 | 16 | 8/16 | 4/8 | 4 | 3.968s | 131.328s |
| Nemotron 3 Nano 3.97B | 0/32 | 算出不能 | 算出不能 | 0/16 | 0/8 | 算出不能 | 23.45ms（失敗応答） | 0.736s |
| Gemma E4B | 32/32 | 12 | 20 | 8/16 | 0/8 | 8 | 5.518s | 179.158s |

Nemotronは全callがinternal_errorで、選択されたpartitionがない。表のgold成功0/32は予定callに対する有効なgold結果が得られなかった件数であり、完了した候補判断を32回誤ったという意味ではない。false merge/splitは算出不能。短いwallは失敗応答時間であり推論速度の優位を示さない。helperの公開停止理由からエラー内部の原因は特定していない。

既知例ではbaseline 0/16に対しGranite/Phi 8/16、Gemma 10/16だった。holdoutではGranite/Phiはbaseline比+2/32、Gemmaは-2/32。holdoutのfalse merge pairはbaseline 6に対し14、14、12、guardrail false merge pairはbaseline 2に対し4、4、8だった。

## Fixture別gold成功

各セルの分母は4予定call。Nemotronは全fixtureで有効結果0/4・internal_error 4/4のため、選択品質は評価不能。

| 区分 | fixture | Ministral | Granite | Phi | Gemma |
| --- | --- | ---: | ---: | ---: | ---: |
| known | `verify_misleading_relation` | 0/4 | 2/4 | 2/4 | 0/4 |
| known | `lexical_crossdir_pairs` | 0/4 | 2/4 | 2/4 | 4/4 |
| known | `new_stem_doc_diverged` | 0/4 | 2/4 | 0/4 | 2/4 |
| known | `new_paraphrase` | 0/4 | 2/4 | 4/4 | 4/4 |
| holdout | `h143_source_test_independent` | 2/4 | 2/4 | 2/4 | 0/4 |
| holdout | `h143_stem_docs_independent` | 4/4 | 2/4 | 2/4 | 0/4 |
| holdout | `h143_shared_directory_independent` | 0/4 | 2/4 | 2/4 | 0/4 |
| holdout | `h143_two_features_interleaved` | 4/4 | 2/4 | 2/4 | 4/4 |
| holdout | `h143_atomic_validation` | 0/4 | 2/4 | 0/4 | 0/4 |
| holdout | `h143_crossdir_documentation` | 2/4 | 2/4 | 2/4 | 2/4 |
| holdout | `h143_paraphrased_feature` | 0/4 | 2/4 | 2/4 | 4/4 |
| holdout | `h143_crosscomponent_feature` | 2/4 | 2/4 | 4/4 | 2/4 |

## 提示方向と反復

方向差は、元順2回の選択が同じ・逆順2回の選択が同じで、両方向間でcandidate IDが異なるfixture。反復差は同一方向2回で選択/停止理由が異なるfixture。

| モデル | 方向差のあるfixture | 同一方向の反復差 | 先頭提示候補を選択したcall / 有効call |
| --- | ---: | ---: | ---: |
| Ministral 3B baseline | 3/12 | 0/12 | 18/48 |
| Granite 4.2 3B | 12/12 | 0/12 | 48/48 |
| Phi-4-mini 3.8B | 8/12 | 0/12 | 12/48 |
| Nemotron 3 Nano 3.97B | 評価不能 | 評価不能 | 評価不能 |
| Gemma E4B | 3/12 | 0/12 | 30/48 |

Graniteは今回の全48callで先頭に提示した候補を選択した。12fixture全てで方向間のIDが変わり、各方向内の2反復は同じだった。この固定入力・helper・template・grammar条件での観測であり、回答理由や別contractでの一般的な位置バイアスの大きさは測っていない。

## 事前基準との照合

人間が推論前に確定した全条件: holdout gold成功28/32以上、baseline比+4call以上、guardrail false merge 0、正しいjoin/false split/完了率のbaseline比悪化なし、構造的不正0、holdout wall中央値がbaselineの2倍以内。

| 比較モデル | gold 28/32 | baseline比+4 | guardrail false merge 0 | join/false split/完了率 | 構造的不正0 | wall中央値2倍以内 | 全条件 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Granite 4.2 3B | 未達 | 未達 | 未達 | 達成 | 達成 | 達成 | 未達 |
| Phi-4-mini 3.8B | 未達 | 未達 | 未達 | 達成 | 達成 | 達成 | 未達 |
| Nemotron 3 Nano 3.97B | 未達 | 未達 | 判定不能 | 完了率未達・選択品質評価不能 | 完了応答なし | 推論費用比較不能 | 未達 |
| Gemma E4B | 未達 | 未達 | 未達 | 達成 | 達成 | 達成 | 未達 |

次段候補条件を全て満たした比較モデルは0件。モデル系列・容量の単独効果、入力情報不足という単独原因、production精度は測っていない。Nemotronのsemantic性能、モデルごとの公式推奨thinking/sampling条件、Pass 2 + planning.Validate()のend-to-endは未測定。今回の失敗から入力情報不足を確定しない。第二段階は実施していない。

## Callとbyteの記録

| モデル | calls | wall合計 | system+user bytes合計 | 完了output bytes合計 |
| --- | ---: | ---: | ---: | ---: |
| Ministral 3B baseline | 48 | 209.744s | 111,420 | 1,226 |
| Granite 4.2 3B | 48 | 191.211s | 111,420 | 1,344 |
| Phi-4-mini 3.8B | 48 | 192.166s | 111,420 | 1,104 |
| Nemotron 3 Nano 3.97B | 48 | 1.099s | 111,420 | 0 |
| Gemma E4B | 48 | 267.030s | 111,420 | 1,200 |
| 計 | 240 | 861.250s | 557,100 | 4,874 |

完了応答のunknown ID・不正JSON・複数選択は0件。max_tokens・timeout・repair・retryは0件。失敗のNemotronでは生成JSONを取得していない。全240callでoutput token countとモデルへ渡る実token列/input token countは取得していない。

## 取得量と再現

4比較モデルを固定revisionで取得し、既存のMinistralは再取得していない。

| モデル | revision | 実取得計画bytes |
| --- | --- | ---: |
| Ministral 3B baseline | `a962dcb09eee4169c890e544c9eb938f1113fdee` | 2,779,150,244（既存cache） |
| Granite 4.2 3B | `0c6f39b1827afd5eb2c1c3b13751929857434953` | 2,066,169,685 |
| Phi-4-mini 3.8B | `ac1c269cb4222a4e136a3d09edad301056c1f36a` | 2,179,993,199 |
| Nemotron 3 Nano 3.97B | `c4d79ba1901d99806ef757642a552acebb851a35` | 2,254,200,328 |
| Gemma E4B | `475b9088d29754a3379866cf5aeb6b41acd313c2` | 5,179,239,349 |

追加4モデルの取得計画合計は 11,679,602,561 bytes。model cacheはworktree外のローカルcacheに保持し、model weightsはGitへ追加していない。Go/Swift/production依存の追加はない。

```sh
GOCACHE=/private/tmp/commiter-issue143-gocache go run ./tools/benchmark110 -issue143 -helper /private/tmp/issue142-bidirectional-helper/commiter-mlx-helper > docs/benchmarks/issue-143-model-comparison-2026-09-30.jsonl
python3 tools/benchmark110/report_issue143.py .
```

再実行は新たな240callであり、今回の結果ファイルを上書きする。今回の再集計のみなら2行目を実行する。既存helperと固定revisionのmodel cacheが必要。

- `go test ./...`、`go vet ./...` は実行コードの固定前に成功。結果集計scriptのsyntax検査、全240行の一意性・hash・pin・予算検査が成功。
- production planner、既定model/backend、SRS、通常CLIは変更していない。検証ブランチのbenchmark専用実装と合成fixture/数値結果のみ。
