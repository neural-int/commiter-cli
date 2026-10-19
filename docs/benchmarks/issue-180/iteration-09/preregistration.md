## 要約
H180-003を採用済みH180-002（iteration08保存結果）と比較。chunk512と同期的evalを維持し、明示同期・解放を4chunkごと＋末尾へ変更。通常prefill callbackは変更しない。

## 検証結果
推論前。C01→C02各1回、最大6要求、retry/repair/judgeなし。baseline再推論なし。C01同期解放5→2回、C02 36→9回。model/prompt/Gold/予算/KVコピー/grammar保持を固定。

## 考察
MLX active/cacheとOSメモリ・swap・helper RSS・CPU/GPU全体・thermal pressureを記録。摂氏温度・プロセス別GPUは前回未取得で今回も取得可否を確認。snapshot最大は連続peakではない。追加計測負荷と過去比較、baselineのcache未記録を制約とする。

## Next Steps
helper/validator/出力入力一致/reuse/count不一致で残り停止。監視10秒途絶またはOS計測失敗、OS memory_pressure free<5%、helper RSS>=12GiBで自分のworkerを停止し再実行しない。自動採用・追加推論なし。
