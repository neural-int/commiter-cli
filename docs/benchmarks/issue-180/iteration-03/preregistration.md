## KV準備処理の改善と追加検証（iteration 03）

iteration 02では共通prefixのKV準備がC01約13秒、C02約116〜120秒を占めた。通常Categoryの入力処理はC02約55〜58秒であり、追加の改善対象とする。C02で40〜55秒の全体短縮は通常prefillと同等の効率に達する場合の試算で、達成保証ではない。

依存runtimeの標準prefillはcacheのみを評価する一方、今回の独自処理は各chunkのlogitsも明示評価していた。KV保存に不要な出力側計算を評価対象から外す最小変更を行う。chunk=512、同期・メモリ解放、model/prompt/Gold/token予算は変更しない。

過去結果は保持し、同一fixture C01/C02・履歴あり・D-T/D-Nの4条件を各1回、計12生成要求で比較する。順序C01-D-T、C01-D-N、C02-D-T、C02-D-N。timeoutなし、retry/repair/追加調整なし。prefix準備、Category+Text、総時間、入力bytes、生成JSON、メモリとfixture不変性を確認する。Grouping変動をKV効果に含めない。改善しない・出力が変化する場合も記録し、採用は別途判断する。
