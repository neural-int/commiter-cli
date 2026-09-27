# Issue #141: grouping-first two-pass の比較結果（2026-09-28）

## 結論

今回の grouping-first 案を production implementation に進める根拠は得られなかった。正解分離例の false merge は改善せず、Pass 1 の全 file ID 完全割当は 6/12 に低下した。Pass 2 は正解 grouping が得られた4試行すべてで既存 validation を通ったが、その4試行はいずれも正しい結合例であり、問題となっている分離判断の改善を示さない。

## 前提の確認

[事前条件](issue-141-preregistered-2026-09-28.md)を推論前のコミット 85001d9 で固定した。#140 の最終比較と同じ6 fixture・正解ラベル、同じ MLX model revision と helper SHA-256、1024 output token/request、2分/plan、2反復を使った。2巡目は arm 順を反転した。repository input は同じ contextinput.Prepare 済み Document から作り、Pass 1 は metadata を要求しない。Pass 1 は全 ID の exactly-once assignment、Pass 2 は固定 group の metadata のみを要求する。Pass 2 の出力には file_ids がなく、Go 側で元の group に結合して planning.Validate() を通す。

#141 の前提には次の限定が必要だった。#140 は試した single-pass 入力・指示での改善不在を示すが、single-pass のあらゆる調整が無効だとは示さない。provisional intent は完全な自由形式ではなく、title と evidence_file_ids を持つ JSON schema だった。#140 の atomicity 付き mixed_24 は invalid_assignment で grouping 欠測だったため、誤結合件数に換算していない。いずれも [#140 の詳細](issue-140-2026-09-27.md)と[atomicity 結果](issue-140-atomicity-2026-09-27.md)に照らして確認した。これらは grouping-first を比較する理由を失わせる誤りではない。

## 主比較

表の false merge は**構造的に有効な割当のみ**の pair 合計。grouping-first で無効な6試行の値は欠測であり、0件の改善とは解釈しない。exact は無効な試行も分母に残す。各 fixture 2試行。

| fixture | full 完全割当 → Pass 1 | full exact → Pass 1 | per-file accuracy | full false merge → Pass 1 | Pass 1 失敗 | full calls → two-pass calls | full wall → two-pass wall |
| --- | ---: | ---: | ---: | ---: | --- | ---: | ---: |
| multi_commit | 2/2 → 0/2 | 0/2 → 0/2 | 4/8 → 欠測 | 8 → 欠測 | duplicate_assignment 2件 | 2 → 4 | 17.1s → 29.6s |
| cross_directory | 2/2 → 2/2 | 2/2 → 2/2 | 4/4 → 4/4 | 0 → 0 | なし | 2 → 4 | 12.7s → 20.1s |
| same_directory_independent | 2/2 → 2/2 | 0/2 → 0/2 | 2/4 → 2/4 | 2 → 2 | semantic_grouping 2件 | 2 → 2 | 13.5s → 8.9s |
| mixed_24 | 2/2 → 0/2 | 0/2 → 0/2 | 24/48 → 欠測 | 288 → 欠測 | duplicate_assignment 2件 | 2 → 4 | 76.5s → 196.6s |
| holdout_split | 2/2 → 0/2 | 0/2 → 0/2 | 4/8 → 欠測 | 8 → 欠測 | duplicate_assignment 2件 | 2 → 4 | 24.5s → 46.5s |
| holdout_join | 2/2 → 2/2 | 2/2 → 2/2 | 6/6 → 6/6 | 0 → 0 | なし | 2 → 4 | 21.7s → 31.9s |

両方式の false split は、採点できた割当で0件。per-file accuracy は full 44/78、Pass 1 は有効な6試行で12/14であり、Pass 1 の無効6試行はこの分母に入っていない。Pass 1 は6/12で完全割当、うち4/6で正解 grouping、2/6で誤結合。正解4試行のみ Pass 2 を実行し、4/4で metadata と grouping boundary を維持して planning.Validate() に合格した。Pass 1 初回12 calls、repair 6 calls、Pass 2 が4 callsで、two-pass 総計22 calls・repair 6件・transport retry 0件・wall 333.7秒。full は12 calls・repair 0件・wall 166.1秒。最大 plan wall は100.4秒（full は41.7秒）。今回の benchmark には transport retry を実装していないため、production の retry budget は未確定である。

Pass 1 の初回 prompt 合計は64,162 bytes、repair input は54,504 bytes、Pass 2 input は11,964 bytesで、two-pass の累積入力は130,630 bytes。full の初回 prompt 合計は83,326 bytes。Pass 1 の単一 request は小さくなっても、repair と Pass 2 を足した総入力は増えた。出力は full 4,252 bytes、two-pass 7,228 bytes。モデルが出力 token 数を返さないため全試行で unavailable と記録した。推定 input tokens は harness の byte ベース見積もりであり、実 token 数ではない。

## #140 との照合

| 条件 | 完全割当 | exact grouping | backend calls | repair | wall 合計 |
| --- | ---: | ---: | ---: | ---: | ---: |
| #140 full（前回） | 12/12 | 4/12 | 12 | 0 | 153.2s |
| #140 full+atomicity（前回） | 10/12 | 4/10、2件欠測 | 14 | 2 | 309.4s |
| #141 full（今回再測定） | 12/12 | 4/12 | 12 | 0 | 166.1s |
| #141 grouping-first Pass 1 | 6/12 | 4/12、6件構造的失敗 | 22（Pass 2 込み） | 6 | 333.7s（Pass 2 込み） |

今回の full と前回の full は、対応する12試行すべてで prompt bytes、完全割当の成否、grouping が一致した。wall time は変動したが、比較の基準は再現した。grouping-first の false merge 合計は有効割当に限ると2件だが、無効だった6試行の誤結合は採点不能であり、前回 full の306件と直接比較して改善とは言えない。

## 正解 group を固定した Pass 2 補助試験

主比較で正しい2 group の Pass 1 結果が得られなかったため、[別途事前固定した条件](issue-141-metadata-preregistered-2026-09-28.md)で2 group の正解ラベルを直接 Pass 2 に与えた。これは end-to-end の成功数には加算しない。

| fixture | batch 1 request | per-group requests | batch wall | per-group wall |
| --- | --- | --- | ---: | ---: |
| multi_commit | invalid_metadata_groups | 成功、2 calls | 8.3s | 11.7s |
| mixed_24 | 成功、1 call | 成功、2 calls | 25.9s | 42.1s |
| holdout_split | invalid_metadata_groups | 成功、2 calls | 7.7s | 12.8s |
| japanese | 成功、1 call | 成功、1 call | 4.7s | 4.9s |

合計では batch 2/4成功・4 calls・46.6秒・入力24,260 bytes、per-group 4/4成功・7 calls・71.5秒・入力45,399 bytes。日本語 summary は両方式で planning.Validate() の言語検証を通った。batch は2 group のうち2例で group_id の完全対応を満たさず、1 request で済む利点と引き換えに構造的失敗が出た。per-group の成功だけを根拠に production で group 数比例の calls を許す判断はできない。

## 判断と限界

#141 の採用基準である「分離例で false merge を減らし、結合例を維持し、完全割当を保つ」を満たさない。現行 Pass 1 contract とこの model/budget の組合せによる grouping-first は見送る。Go 側で missing / duplicate / unknown / empty group / schema を deterministic に検出でき、誤った割当を Pass 2 や Git mutation に渡さないことは確認した。semantic grouping の正しさは Go で保証できず、今回の same_directory_independent では構造的に有効でも誤結合した。

この結果は、固定したモデル、6つの合成 fixture、2反復、今回の partition prompt/schema に限る。他のモデル、別の出力表現、token budget や evidence の変更まで否定しない。production call/retry budget や Pass 2 の batch/per-group 採用は未決定のままとし、#139 に実装見送りの根拠として反映する。今回の実装は benchmark-only であり、production planner、CLI の通常動作、SRS は変更していない。

## 再実行と検証

生データは [主比較](issue-141-grouping-first-mlx.jsonl) と [Pass 2 補助試験](issue-141-metadata-mlx.jsonl)。固定した helper と model を使い、以下の2 command で再実行できる。

```sh
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -issue141-probe metadata -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 1 -timeout 2m -output-tokens 1024
```

go test ./...、go vet ./...、git diff --check を実行して確認した。
