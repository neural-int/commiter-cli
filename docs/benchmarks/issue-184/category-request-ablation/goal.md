# H184-003 Categoryの依頼本文有無の比較

## Goal / Conditions

補完済みH184-KIMDEL-REPAIRED-002の日本語Category入力を使用し、user_requests本文ありとfinal_responsesのみを各1回、新規比較。後者は必須配列を削除せずuser_requests: []とした。依頼3本文、計1933文字だけを除去し、逆置換で他のuser入力bytes不変を確認。意図した入力削減であり予算不足の省略ではないため、省略理由は追加しない。

C184-02、Qwen3.5-4B 4bit固定revision、Swift Release、Thinking OFF、json_schema、native KV、32768/4096、temperature0.6/top_p0.95/top_k20、penalty1.0、seed144、eager、時間制限なし、warmup/retry0。最終応答・補完候補・raw_diff・system・Schemaは固定。Groupingは既存実応答で固定、生成はCategory計2回のみ。Textおよび最終plan validatorは未評価。

## Results / Measurements

| 条件 | README 正解docs | UI 正解feat | Mail 正解fix | Category | 入力tokens | 出力tokens | decode tok/s | native helper wall秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| with-request | docs | feat | feat | 2/3 | 4996 | 71 | 27.10 | 20.997 |
| final-only | docs | feat | fix | 3/3 | 4052 | 71 | 27.27 | 17.335 |

## Validation / Limits

実wireのsystem/Schema一致、Thinking OFF、json_schema、ソースハッシュ維持、正常終了を確認。資源監視正常、同時helper最大1。詳細なruntime時間・MLXメモリ・RSS・熱状態はmeasurements.jsonと非公開原本に保存。保持済み履歴のpartial/ambiguous/history_incompleteは維持。単一fixture/seed、各1回、順序は本文あり→なし。OSキャッシュや背景負荷を固定していないため速度差の安定性や一般的精度を断定しない。原本保持。
