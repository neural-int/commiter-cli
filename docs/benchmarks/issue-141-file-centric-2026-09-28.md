# Issue #141: file-centric Pass 1 の比較結果（2026-09-28）

## 結論

今回の file-centric 契約は、12/12で全 file ID の構造的に有効な割当を返し、分離4 fixture の false merge を0 pairにした。ただし、12試行中8試行で結合すべき file まで分割し、false split は16 pairだった。正解 grouping の4試行も Pass 2 の `invalid_metadata_groups` で止まり、最終 plan の成功は0/12である。[事前に固定した採用基準](issue-141-file-centric-preregistered-2026-09-28.md)の「結合例の false split を増やさず、完全割当を維持する」を満たさないため、現行の prompt/schema とモデルによる production 採用は見送る。

## 条件と採点

[事前登録](issue-141-file-centric-preregistered-2026-09-28.md)と benchmark code を推論前のコミット `7a428df` で固定した。6 fixture × full / grouping-first / file-centric × 2反復の36試行。2巡目は条件の順序を逆にした。モデルは `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。各 request の出力上限は1024 token、plan 全体の timeout は2分。入力 fixture、正解 group、repository input と relation context、backend、Pass 2 と採点は前回と同じである。追加のモデル取得や依存追加は行っていない。

完全割当は Pass 1 の構造的に有効な全 ID 割当、exact は正解 group との一致を示す。false merge / false split は正解が異なる / 同じ file pair を誤って結合 / 分離した件数である。group label の文字列自体は採点しない。file-centric では生の JSON 応答で重複 key を拒否したうえで、Go が未知・欠落 ID と label を検証した。最終成功には、正解 Pass 1 の後の Pass 2 が元の group に対応し、`planning.Validate()` を通ることも必要とした。構造的失敗には1回だけ修復を許し、意味的な誤りは修復しない。

## 主比較

表の pair 件数は各 fixture 2試行の合計。grouping-first の構造的失敗6試行には pair 指標がなく、0件には含めない。`最終成功` は full では有効な従来 plan、two-pass では正解 grouping と有効 metadata の両方を指すため、別途 exact と Pass 2 の結果も示す。

| 条件 | 完全割当 | exact grouping | false merge | false split | per-file accuracy | 最終成功 | calls | wall 合計 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| full | 12/12 | 4/12 | 306 | 0 | 44/78 | 12/12 | 12 | 149.8s |
| grouping-first | 6/12 | 4/12、6件欠測 | 2、6件欠測 | 0、6件欠測 | 12/14、6件欠測 | 4/12 | 22 | 306.2s |
| file-centric | 12/12 | 4/12 | 0 | 16 | 64/78 | 0/12 | 16 | 201.6s |

| fixture | 正解の用途 | full exact | grouping-first exact | file-centric exact | file-centric false merge / split | file-centric Pass 2 / 失敗 |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| multi_commit | 2 group に分離 | 0/2 | 0/2、2件構造的失敗 | 0/2 | 0 / 4 | 未実行 / `semantic_grouping` 2件 |
| cross_directory | 1 group に結合 | 2/2 | 2/2 | 0/2 | 0 / 2 | 未実行 / `semantic_grouping` 2件 |
| same_directory_independent | 2 group に分離 | 0/2 | 0/2 | 2/2 | 0 / 0 | 2件実行 / `invalid_metadata_groups` 2件 |
| mixed_24 | 24 file を2 group に分離 | 0/2 | 0/2、2件構造的失敗 | 2/2 | 0 / 0 | 2件実行 / `invalid_metadata_groups` 2件 |
| holdout_split | 2 group に分離 | 0/2 | 0/2、2件構造的失敗 | 0/2 | 0 / 4 | 未実行 / `semantic_grouping` 2件 |
| holdout_join | 1 group に結合 | 2/2 | 2/2 | 0/2 | 0 / 6 | 未実行 / `semantic_grouping` 2件 |

file-centric の `multi_commit`、`cross_directory`、`holdout_split`、`holdout_join` は、各 file を単独 group に置いた。これにより false merge は消えたが、結合すべき pair も分割した。一方、`mixed_24` は24 ID の完全割当と正解の2 group を両試行で得た。したがって、file-centric の結果を単に「常に singleton」とは説明できない。前回の grouping-first で採点不能だった `mixed_24` の grouping を評価できた点は確認できる。ただし、これだけで一般的な分離精度や最終 plan の改善とは言えない。

file-centric の Pass 1 は12 calls、構造修復0、Pass 2 は正解4試行に対して4 callsで、4件とも `invalid_metadata_groups`。この failure code は metadata 件数、group ID の重複・欠落、`breaking` 欠落のいずれでも返る。生の metadata 応答は保存していないため、どの条件に該当したかや原因は特定していない。transport retry は0で、この benchmark には retry 実装がない。16 request の stop reason はすべて `completed`。出力 token count は backend から得られず、`unavailable` である。

| 条件 | 初回入力 | repair 入力 | Pass 2 入力 | 出力 bytes | 最大 plan wall |
| --- | ---: | ---: | ---: | ---: | ---: |
| full | 83,326 B | 0 B | 0 B | 4,252 B | 35.3s |
| grouping-first | 64,162 B | 54,504 B | 11,964 B | 7,228 B | 90.3s |
| file-centric | 66,738 B | 0 B | 32,672 B | 4,472 B | 67.2s |

入力 bytes は prompt の記録値であり、実 token 数ではない。file-centric の累積入力は99,410 B。group label を含める file-centric Pass 2 は、前回の grouping-first Pass 2 と prompt サイズも変わる。prompt/schema も表現とともに変わるため、差を構造表現だけの因果効果とは断定しない。

## 前回との照合と次の判断

今回の full と grouping-first は、[前回の #141 主比較](issue-141-grouping-first-2026-09-28.md)と対応する24試行すべてで初回 prompt bytes、完全割当、grouping、failure、calls が一致した。wall time の差はあり、今回の full は149.8秒、前回は166.1秒である。#140 の full は12/12完全割当・4/12 exact・153.2秒で、今回も同じ正解率の範囲にある。したがって同じ fixture と判定で比較できるが、少数反復の wall 差を性能改善とは扱わない。

今回の file-centric は「全 ID を一度ずつ返す」という構造的問題を解いた一方、結合例で8 pairの false split（2試行ずつの合計）が出た。分離4 fixture 側にも8 pairの false split がある。完全割当と false merge の改善だけで採用せず、結合の recall を維持することを次の候補の必須条件に残す。metadata の `invalid_metadata_groups` は Pass 1 の grouping 品質とは別の失敗として扱い、後続の検証を行うなら group label の受け渡しを固定して Pass 2 単独を診断する。ただし今回の生データでは失敗原因をこれ以上特定できず、この報告から production の call/retry budget や別方式の採用は決めない。

6つの合成 fixture、各2反復、1つのローカルモデル、今回の prompt/schema/output budget の結果である。production planner、通常 CLI、SRS は変更していない。

## 再実行と検証

生データは [36試行の JSONL](issue-141-file-centric-mlx.jsonl)。再実行 command は次のとおり。

```sh
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -issue141-probe file-centric -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
```

`GOCACHE=/tmp/commiter-issue141-gocache go test ./...`、`GOCACHE=/tmp/commiter-issue141-gocache go vet ./...`、`git diff --check` を通した。
