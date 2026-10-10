# Phase 2 実測の事前条件

推論結果を取得する前にこの条件と fixture ソースを固定する。既知の #151/#157/#160 fixture を再評価して独立意味品質と呼ぶことはしない。

## 比較対象と操作条件

- 現在の同一 Git snapshot から `gitstate.Collect` → `syntax.AnalyzeChange` → `contextinput.Build` の同一 document を作る。1〜4 は同じ Prepared を Three-phase / File-first の双方へ渡す。
- File-first は1 file = 1 group、ID順。既存 category/text の4-group packet、各512/768 output tokens、16K context、全 cycle 120秒、最大8 callsを維持する。再試行・metadata fallback・並列推論は使わない。
- 既存の固定 Gemma `mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2`、4bit、temperature 0、top_p 1、top_k 0、seed 144。保存済みローカルモデルを使い、モデル download はしない。
- 各 workload / route は3反復。1〜4の順序は反復0で Three-phase → File-first、反復1で逆順、反復2で正順。1→2→3→4→5→8→9→16の workload 順を固定する。
- 本番 helper は call ごとに新プロセスで model をロードする。model 常駐・KV cache 再利用はない。反復0を first-observed、反復1/2を filesystem-warm と区別する。OS cache を破棄しないため true storage-cold の主張はしない。各 call のロード費用は wall time に含める。
- 他アプリを停止・設定変更しない。同じホストの共存負荷、memory pressure、swap を前後記録する。比較の時間差をファイル数だけの因果効果とは解釈しない。

## fixture と評価参照

fixture は今回の実測専用に作成する通常ファイルの変更であり、モデル調整には使わない。目的の参照 partition は fixture ソースで推論前に固定し、モデル入力に渡さない。

| files | workload | 参照目的 |
|---:|---|---|
| 1 | one-independent | 単一の境界条件修正 |
| 2 | source-test-pair | 実装と対応テストを同じ修正として扱う |
| 3 | cross-directory-purpose | 3ディレクトリの同目的の境界変更 |
| 4 | boundary-base | 同一ディレクトリの4つの独立修正 |
| 5 | boundary-plus-consumer | 基底4変更を保持し、1番目の修正の対応テストを追加 |
| 8 | four-source-test-pairs | 4目的、各実装と対応テスト |
| 9 | cross-directory-scale | 9ディレクトリの同目的変更 |
| 16 | independent-scale | 独立した16変更、4-group packet の費用上限確認 |

4→5は最初の4ファイルの before/after bytes を同一にする。File-first 4対5の増分と、Three-phase 4対File-first 5の方式差を別計算する。入力選択で index/working tree を変更しない。mixed intent、曖昧 gold、rename/delete/untracked 等の独立採否は Phase 3 が許可された場合の別 holdout として残す。

## 記録と採否

- 各 run に snapshot/Prepared の digest、wall time、構造 replay 時間、call別 profile/time、input/output tokens（native thought を含む）、runtime TTFT、ロード時間、helper peak RSS / MLX peak allocation、pressure/swap、diff bytes/lines、group数、validator 成否、FM/FS/exact を記録する。生成内容・プロンプト・helper stderr は保存しない。
- TTFT は固定 MLX runtime の `GenerateCompletionInfo.promptTime` を利用する。同 runtime は最初の token を iterator から受け取った際に時間を確定し、prompt prefill を加算する。helper の load/schema/input準備時間も別計測する。text chunk の到着を token arrival と混同しない。外側のプロセス起動時間は call wall time にだけ含む。
- 非完了・timeout は分母から除外しない。完成した plan の品質と失敗数を分ける。値が取得できなければ null と不足理由を保存し、Gate 達成とみなさない。
- p50/p95 は nearest-rank で全3反復の値を記録し、成功のみも別集計する。n=3 の p95 は最大値であり、production tail latency の推定根拠にしない。
- Gate 2 は受入 workload の plan 成功100%、authoritative validator・全file境界・元Git状態保持100%、call/token/context/120秒上限、helper RSS/MLX allocation各8 GiB以下、run中swap増分256 MiB以下、memory pressure normal、同一1〜4の速度と意味品質記録、既存経路回帰なしを満たすこと。未達なら Phase 3 を gated-out とする。
- production 採否は別途の独立 holdout、semantic exact80%以上、既存方式を超えるFSを許容しない、中間依存破損0、構造/安全100%、p95 cycle120秒以内、上記資源条件を最低基準とする。この少数の合成実測だけでは production GO にしない。#160 の80%未達・NO-GOは変更しない。
