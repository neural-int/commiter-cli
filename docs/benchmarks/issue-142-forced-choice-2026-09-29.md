# Issue #142: forced-choice selection の検証結果

## 条件

[事前条件](issue-142-forced-choice-preregistered-2026-09-29.md)と実装をコミット `b05f512` で固定した後に測定した。既使用の4 fixture で、前回と同じ Louvain 候補生成、prepared repository input、MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を使用した。新たなモデル取得や依存追加はしていない。

各 fixture の候補は2つで、そのうち1つが合成 gold と一致した。selection 指示と JSON Schema enum から `none` を除き、出力上限2048 token、各 call の時間上限2分、repair/retry 0、Pass 2なしで各2反復した。2回目は候補の提示順だけを反転し、候補IDとグループの対応は保った。

## 実モデルの結果

| fixture | 1回目（元順序） | 2回目（逆順） | 正解/2 | 有効な候補ID/2 | wall 合計 |
| --- | --- | --- | ---: | ---: | ---: |
| `cross_directory` | `C002`、不一致 | `C002`、不一致 | 0 | 2 | 7.0s |
| `same_directory_independent` | `C001`、不一致 | `C002`、一致 | 1 | 2 | 6.4s |
| `mixed_24` | `C001`、一致 | `C001`、一致 | 2 | 2 | 33.8s |
| `source_test_separate_purposes` | `C002`、一致 | `C002`、一致 | 2 | 2 | 6.5s |

8試行すべてが `completed` し、有効な候補IDを返した。合成 gold との一致は5/8、完了した有効候補ID応答を分母としても5/8。`max_tokens`、2分の時間切れ、`none`、不正 JSON/schema、未知候補IDは各0件。候補IDを得た8試行はすべて完全割当で、pair 指標の合計は false merge 1、false split 2。backend calls は8、wall 合計53.6秒。output token telemetry は8件とも `unavailable`。8行の `(fixture, run)` は一意で、各 fixture の2反復は異なる prompt/schema hash を持つ。

前回の同じ4 fixture・2048 token・`none` 許可条件は、8試行中5件が完了してすべて `none`、3件が `max_tokens`、有効な候補IDは0件、wall 合計287.5秒だった。今回の二択条件は8件完了・8件候補ID・5件正解だった。この対比は異なる日時の非同時測定であり、指示とschemaを同時に変更している。`none` の除外だけの因果効果、汎化性能、実際の開発者判断との一致は判定できない。2候補からの5/8正解は小標本であり、二択の偶然水準を明確に超える証拠とは扱わない。提示順ごとは1反復ずつなので順序効果も推定しない。

## 実行環境と記録

最初の通常sandbox実行では8件すべてが0.2〜0.3秒程度で `backend_error` となり、選択応答は得られなかった。これは[別の JSONL](issue-142-forced-choice-sandbox-errors.jsonl)に保存し、上の実モデル8試行には含めていない。同じ事前条件をMLX helperへアクセスできる実行権限で再実行した[結果 JSONL](issue-142-forced-choice-mlx.jsonl)では8件すべてが完了した。最初の `backend_error` の内部原因は記録できていない。

事前に決めた candidate-difference-only へ進む条件は、8試行中4件以上の `max_tokens` または時間切れだった。今回の該当件数は0件なので、その後続 probe は実行していない。実モデルの Pass 2、production planner、SRS、通常 CLI は測定・変更していない。

```sh
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe forced -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <上記 SHA-256 と一致する helper> -fixture all
```

数値、stop reason、候補ID、prompt/schema hash だけを JSONL に保存した。生の prompt、応答、repository content は保存していない。
