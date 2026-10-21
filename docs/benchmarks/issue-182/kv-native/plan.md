# H182-008 診断とKV精度比較の事前条件

ユーザー承認の方針を実行可能な研究ハーネスとして準備する。今回はprepareのみ。モデル生成・trace採取・速度比較は開始しない。

## 段階1：短い診断（2セル）

Gemma簡潔化C01、Qwen4B現行C01を各1回。従来A/eager、KV8bit/group64、Thinking ON・各phase1024、最終出力予算とcycle時間制限なし。CPU診断のみ有効。初回token取得を独立記録し、それ以降のtoken.item取得待ち、状態更新+進捗、文法mask準備をThinking/response別に集計する。close markerはcommit前のThinking側に分類する。CPU側経過秒はGPU待ちやruntime呼び出しを含み、純粋なGPU演算時間や転送時間とは扱わない。mask準備はhostでのmask作成とMLX演算組立てで、GPU適用完了時間ではない。

最初の生成進捗を確認後、専用helperのPIDが1個だけであることを確認し、そのPIDにMetal System Traceを最大30秒attachする。全プロセス対象にはしない。template存在は準備時に確認するが、実attachの権限と採取成功は未検証。失敗したら診断未成立として記録し、比較へ自動で進まない。traceは端末内のみに保持し公開コピーへ含めない。

モデル生成は2セル・最大6phase、warmup/retry/repairなし。30秒はtrace採取窓であり、モデルの生成制限ではない。診断セルは通常の3phaseを完了まで測るので、セル自体が30秒で終わるとは扱わない。C02のGPU待ちはこの短い診断の対象外。診断には計測負荷があるため、その速度値をKV比較値へ流用しない。

診断終了後に実traceとCPU計測を読み、判明事項・不明点・次の判断をdiagnostics/review.jsonへ記録する。proceed_to_kv_comparison=trueとfindingsを明示するまでrunは拒否する。CPU計測だけでCPU/GPUの実際の重なりを断定しない。diagnoseからrunを自動連結しない。

## 段階2：KV比較（8セル）

Gemma C01/C02、Qwen4B C01/C02それぞれKV8とnativeを新規測定。従来A/eagerに固定し、Bは使用しない。nativeはKV量子化設定を与えず、model.newCacheの実装・dtypeに従う。attention以外のrotating/state-space cacheをFP16へ一律変換しない。実生成後のKV snapshotでdtypeと容量を確認する。準備監査は量子化対象だけがQuantizedKVCacheからKVCacheSimpleへ変わり、他layer構成が不変であることを確認する。

各modelのC01はKV8→native、C02はnative→KV8。各arm1回、GPU生成は直列、warmup/retry/repairなし、最大24phase。KV8側も再測定し過去の速度を比較値として流用しない。同一Release binary、同一metallib、同一source・依存revision、seed144、weight revision、入力、prompt、schema、validator、固定partition、進捗・OS監視。詳細CPUタイマーとtraceは両群とも無効。

Gemmaはtemperature=1.0/top_p=0.95/top_k=64、Qwen4Bは0.6/0.95/20。両方Thinking ON、各phase1024、最終出力予算と時間制限なし。Qwen2Bは今回対象外。

## 停止と評価

既存のRSS12GiB・monitor failure・token進捗300秒なしの安全停止を維持。モデル失敗はそのまま保存し追加生成しない。監視・入力・source/asset hash driftや結果未生成のインフラ障害はsuite停止。pressure・swap・MLX memory・実KV容量を記録し、メモリ圧迫が違う測定を安定した速度差と断定しない。スワップ使用量だけでモデルのswapを断定しない。

生成tokens、decode token/s、cycle秒、Thinking境界/予算、実Grouping、validator、scope/summary品質を分ける。KV精度変更は数値と応答を変え得るため、応答不一致自体をインフラ失敗にしない。原要求・schemaとtoken列は一致が必須。後段固定partitionでend-to-end成功率ではない。各1回で有意差・採用を確定しない。

## 実行口

- kv_compare.py prepare：今回行う生成なし監査
- kv_compare.py diagnose：CPU診断と専用PIDの短いMetal trace。未実行
- kv_compare.py run：診断結果レビューの後に行う8セル比較。未実行

保存先は/private/tmp/issue182-kv-native/。既存原本は上書きしない。追加依存なし。InstrumentsはXcode付属xctraceを利用する。
