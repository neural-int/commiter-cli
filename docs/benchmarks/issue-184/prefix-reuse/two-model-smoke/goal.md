# H184-005 / typed-prefix-two-model-smoke

## 条件と評価範囲

ハーネス003の最終fixture（5files/3目的）、同じプロンプト本文を両モデルで使用。Thinking OFF、32K/4096、native KV、grammar/scope ON、seed144、eager、legacy prefill保護。モデル別samplingはQwen .6/.95/20、Gemma 1/.95/64。両モデルのruntime revisionはc6446cf7bfb7cea76408013b614d4b2c530eaa03。Gemmaは4-E4B-it-4bit revision475b9088d29754a3379866cf5aeb6b41acd313c2、Qwenは3.5-4B-4bit revision0e7ffd5c629ef7719d4cbc04069232580bfa9d9c。追加モデル取得なし。新しい同一Release helperで各off/onを1回ずつ、計12生成。

固定独立phase入力で、生成Grouping/Categoryを次phaseへ反映しない。全3phaseのプロトコル疎通・生成完走を確認する試験であり、結果伝播型E2E成功とは呼ばない。runtimeモデル保持とは別にGroupingでprefixを保存し、Categoryは独立prefill、Textでforkを再利用。候補検証用planを使用し、production検証済み登録は未作成。

初回の非推論監査スクリプトでnativeTypeをtypeと誤記しKeyErrorで停止。生成前であり、失敗原本を保持して修正後の別原本で実行した。実装条件/入力条件の変更や生成再試行なし。

## 結果

| モデル | 再利用 | Grouping | Category | 合成validator | 全体秒 | Text再利用tokens |
| --- | --- | --- | --- | --- | --- | --- |
| qwen | off | 3/3 | 3/3 | 通過 | 49.470 | 0 |
| qwen | on | 3/3 | 3/3 | 通過 | 40.977 | 2873 |
| gemma | off | 3/3 | 3/3 | 通過 | 41.093 | 0 |
| gemma | on | 3/3 | 3/3 | 通過 | 36.943 | 3273 |

Qwenの型はKVCacheSimple/MambaCache、GemmaはKVCacheSimple/RotatingKVCache。監査後の共通prefixは2873/3273tokens。両群の要求は同一で、envによる再利用切替のみ。Gemmaは全出力一致、QwenはGrouping/Category一致、Textのメールsummaryのみ「空 Message-ID の正規化」→「空の山括弧正規化」。Qwenの本文一致/数値同等性は未成立で、丸め・chunk境界の変更による生成分岐かcache不具合かは今回未特定。全モデルの正しさを保証せず、自動有効化しない。

## フェーズ時間

| モデル | arm | phase | 全体秒 | runtime prefill秒 | 別prefix計算秒 | snapshot秒 | decode tok/s | MLX process peak MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen | off | 01 | 18.863 | 15.018 | 0.000 | 0.000 | 27.42 | 3242.7 |
| qwen | off | 02 | 14.348 | 11.704 | 0.000 | 0.000 | 27.42 | 3242.7 |
| qwen | off | 03 | 16.256 | 11.641 | 0.000 | 0.000 | 22.98 | 3242.7 |
| qwen | on | 01 | 18.766 | 6.015 | 8.814 | 0.001 | 27.19 | 3358.1 |
| qwen | on | 02 | 14.831 | 12.222 | 0.000 | 0.000 | 27.73 | 3358.7 |
| qwen | on | 03 | 7.377 | 2.500 | 0.000 | 0.000 | 22.46 | 3358.7 |
| gemma | off | 01 | 15.114 | 11.093 | 0.000 | 0.000 | 24.39 | 4910.7 |
| gemma | off | 02 | 10.849 | 8.686 | 0.000 | 0.000 | 25.23 | 4910.7 |
| gemma | off | 03 | 15.126 | 8.448 | 0.000 | 0.000 | 19.09 | 4910.7 |
| gemma | on | 01 | 15.517 | 4.630 | 6.696 | 0.001 | 25.14 | 4910.7 |
| gemma | on | 02 | 10.995 | 8.780 | 0.000 | 0.000 | 24.58 | 4910.7 |
| gemma | on | 03 | 10.429 | 1.619 | 0.000 | 0.000 | 15.61 | 4910.7 |

Groupingの再利用ありでは共通prefix計算時間をruntime prefillとは別に計上し、全体時間には含める。TextのprefillはQwen11.641→2.500秒、Gemma8.448→1.619秒。合計はQwen約8.49秒、Gemma約4.15秒短縮。各1回の順序固定比較であり一般的な性能向上保証ではない。MLX peakはプロセス累積値でphase単独ピークではない。

Categoryは意図的に対象profilesに含めずcacheReusedTokens=0。今回の診断文字列disabled_unverified_planは対象外phaseにも共通して出るため、Categoryの検証失敗と解釈しない。

## 保存と判断

全4条件で3セル確定、完了state complete、monitor fatalなし。CPU/GPU/VM/swap/thermal時系列とraw応答/token列はprivate保存、公開は集計のみ。実装と本報告は未コミット。003manifestの再利用なしという採用済み条件を変更しない。次はQwenのText差をlogits等で検証し、復元同等性の成立範囲を確認する。

## 採用追記

ユーザー指示によりH184-HARNESS-004 / revision 1へ採用。QwenのText差分と厳密同等性の未確定は維持する。003は再利用なしの比較基準として保存する。
