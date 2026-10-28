# H182-017 Spark-X2.6-4B事前検証

## Conditions

同じkimdel固定fixture、32K入力、改訂Gemma compact、出力2048（Thinking込み）、seed144、sampling1/.95/topk無効、penalty1/0/0、標準native cache。OFF/Low/High×grammarあり/なし6条件・各1回・最大18phase。Lowをbaselineとし、OFF/Highも同条件で比較する。正解partitionは注入しない。時間上限なし、監視RSS12GiB等の安全停止は維持。

## Adapter

現行Swiftのspark2_5非対応のため、ユーザー承認の専用venvへmlx-lm0.32.2/llguidance1.9.1と推移依存を固定。Spark固有アダプターが既存helperと同じ要求/応答を変換し、Three-phase Planner/Schemaは変更しない。Gemmaの旧研究overlayを使用する。公式chat templateの実引数reasoning_effortへLow/Highを渡す。tokenizerでthink開始3/終了4/EOS1を別に確認。モデル同梱Pythonはレビュー・revision固定後にローカル読み込み。モデルrevision=9aa68e67d64f5728b42f15661524fb7dca8d984d、Affine4/group64。

## Gates

初回3smokeでは生成JSONは正常だったが、KVのscalar metadataをarrayと誤認する計測例外と例外時ok保持を検出し、不採用とした。原本保持。native cache stateをMLX配列だけに限定し、例外はruntime_errorでfail closedへ修正。再度OFF/Low/Highの各1smokeがすべてcompleted、answer=ok、Thinking自然終了、36cache層、peak記録を確認。Thought数0/45/90、final5tokens。切替template監査と境界/マスク/不正EOS/Schema値の4unitテスト成功。既存7Pythonテスト成功。

## Metrics and comparison limits

実出力のThinking/境界/最終回答を分けて数える。EOSは除外しlibraryGenerationTokensを別保存。prefill/decode token/sはPython公式API値と時間換算値を保存し、Swiftとカウンタ差があり得る。native cache dtype/容量も保存。load/template/grammar/prefill/decode/wall、CPU/GPU/memory/swap/thermalを計測する。PythonプロセスをOS監視対象へ追加。temperature・電力実数は従来と同じ取得不可。

Python mlx-lm/llguidanceとSwift runtimeでは実装が異なり、最大出力も2048対4096なので、既存Gemma/Qwenとの時間差をモデル単体差と断定しない。初期smokeはruntime診断で本測定に混ぜない。原本と依存lockをprivate保存。
