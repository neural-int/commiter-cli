# H182-008 診断・KV比較（準備完了、未実行）

## 要約

従来A/eagerに固定した診断2セルとKV比較8セルを用意した。モデル生成とtrace採取は未実行で、性能・品質の結果は未確定。詳細条件はplan.md。

## ケース一覧

| 段階 | モデル・fixture | prompt | KV | 詳細CPU診断 | Metal trace |
| --- | --- | --- | --- | --- | --- |
| 診断D1 | Gemma E4B C01 | 簡潔化 | 8bit/group64 | ON | 生成開始後最大30秒 |
| 診断D2 | Qwen3.5-4B C01 | 現行 | 8bit/group64 | ON | 生成開始後最大30秒 |
| 比較K1/K2 | Gemma E4B C01 | 簡潔化 | 8bit→native | OFF | OFF |
| 比較K3/K4 | Gemma E4B C02 | 簡潔化 | native→8bit | OFF | OFF |
| 比較K5/K6 | Qwen3.5-4B C01 | 現行 | 8bit→native | OFF | OFF |
| 比較K7/K8 | Qwen3.5-4B C02 | 現行 | native→8bit | OFF | OFF |

診断2セルと比較8セルは別の実行契約・保存先。診断の速度を通常比較へ流用しない。診断終了・trace成立・結果レビューを通過するまで比較runは開始できない。診断の結果によって比較開始を見送る余地を残す。

## 固定条件

同一Release binary・metallib・weight・依存revision・入力・JSON schema/validator・固定partitionを使用。両モデルThinking ON・各phase1024、最終出力予算/時間制限なし、seed144。Gemma sampling 1.0/0.95/64、Qwen4B 0.6/0.95/20。通常の進捗とOS監視は全セルで維持し、B方式とQwen2Bは対象外。各セル1回、warmup/retry/repairなし、直列実行。診断最大6phase、比較最大24phase。

nativeはモデル標準のcacheとdtypeを使い、全層をFP16/BF16へ固定しない。重み4bitは両群とも同じ。変更するのはKV量子化で、メモリ増加・転送量増加・数値差による出力変化の可能性がある。

## 準備結果

Release45テスト成功。Thinking→responseの診断ラベルが終了markerのcommitで切り替わる回帰テストを追加。全8比較セルで再capture要求と入力token列が保存済み基準と一致。KV8側のKV構成は基準と一致、native側は対象attention KVだけがQuantizedKVCacheからKVCacheSimpleへ変わり、rotating/state-space層は不変。空cache時のdtype表示を実生成後のdtypeの証拠とは扱わず、実行時snapshotを確認する。

xctraceの最初のtemplate一覧取得はsandboxからInstrumentsキャッシュへ書けず失敗した。通常の許可された実行環境で再確認し、Metal System Traceの存在を確認した。実attachの権限・trace内容・GPU/CPU重なりは未確認。新しい依存の追加はない。

## 評価と安全条件

生成tokens・decode token/s・cycle秒・Thinking/終了境界・実Grouping・validator・文章品質を別々に評価。RSS12GiB、monitor failure、進捗300秒なしの既存停止条件を維持。pressure・swap・実KV容量も記録する。メモリ圧迫の違いがある場合は安定した速度差と断定しない。

CPUタイマーは初回token取得を別記し、以後の取得待ち・状態更新/進捗・mask準備をThinking/response別に記録する。取得待ちはGPU待ちも含み、GPU演算時間そのものではない。実際のCPU/GPUの重なりは30秒traceを解析して判断する。traceは端末内に保持し公開コピーへ含めない。GPUを毎token同期する追加計測はしない。

## 保存と次の手順

原本は/private/tmp/issue182-kv-native/。raw/に生成なし監査・契約・capture・source/checksを保存する。公開コピーはメール形式文字列とホーム絶対pathを置換し、SHA256SUMS.txtは公開コピーhash。実行契約のhashは原本の値。binary/weight/metallibはhashで識別する。

次はdiagnoseで診断2セルだけを実行し、traceとCPU内訳をレビューする。その後に比較8セルを実行するか決める。runにはdiagnostics/review.jsonの判明事項とproceed_to_kv_comparison=trueが必要。今回の作業は準備までであり、診断・比較の結果や速度改善はまだない。

## 診断の適用範囲

使用revisionの標準generateはTokenIteratorへfactory生成のprocessorを直接渡し、今回の非speculative/eager経路ではprocessor.copyを呼ばないことを確認した。診断集計はそのprocessorと同じprogressオブジェクトを参照する。SpeculativeTokenIteratorのcopy分岐には対応を拡張せず、将来利用する場合は試行tokenの誤加算を避ける別設計が必要。実際の診断結果でCPU計測値・event countが保存されていることもレビューで確認する。
