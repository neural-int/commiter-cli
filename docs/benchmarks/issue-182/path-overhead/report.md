# H182-009 生成経路比較結果

## Goal

Go経由の実行経路とSwift内の独自processorが生成速度に与える影響を切り分ける。固定条件はgoal.md。

## Result / Decision

段階1・2の全16セル・48phaseが正常終了。段階1の12要求・応答・生成tokens一致を確認。段階1はGoを省いても全ケースが速くなる結果ではなく、Go側が速度低下の主因という根拠は得られなかった。段階2の標準構成は3組でdecode速度が上昇、Qwen C02では低下。大幅な高速化を一律に示さず、独自processorの影響と生成内容・環境の影響を分離し切れていない。各arm1回のため因果と安定性は未確定。採用設定は変更しない。

## stage1

| セル | driver秒 | tokens | decode token/s | 終了 G/C/T |
| --- | ---: | ---: | ---: | --- |
| stage1-gemma-on-C01-compact-thinking1024-go | 88.58 | 1817 | 24.67 | completed / completed / completed |
| stage1-gemma-on-C01-compact-thinking1024-direct | 96.37 | 1817 | 22.13 | completed / completed / completed |
| stage1-gemma-on-C02-compact-thinking1024-direct | 305.93 | 2511 | 12.68 | completed / completed / completed |
| stage1-gemma-on-C02-compact-thinking1024-go | 307.40 | 2511 | 13.21 | completed / completed / completed |
| stage1-qwen4-on-C01-current-thinking1024-go | 137.14 | 2449 | 20.80 | completed / completed / completed |
| stage1-qwen4-on-C01-current-thinking1024-direct | 133.59 | 2449 | 21.84 | completed / completed / completed |
| stage1-qwen4-on-C02-current-thinking1024-direct | 355.14 | 2706 | 14.72 | completed / completed / completed |
| stage1-qwen4-on-C02-current-thinking1024-go | 361.92 | 2706 | 14.51 | completed / completed / completed |

| セル | load合計秒 | input prepare合計秒 | grammar setup合計秒 | prefix prefill合計秒 | runtime prompt合計秒 | runtime decode合計秒 | generation wall合計秒 | MLX peak GiB | phase末KV最大MiB |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| stage1-gemma-on-C01-compact-thinking1024-go | 2.331 | 0.070 | 0.986 | 4.106 | 6.226 | 73.661 | 84.000 | 4.796 | 47.3 |
| stage1-gemma-on-C01-compact-thinking1024-direct | 2.045 | 0.069 | 1.524 | 4.164 | 6.172 | 82.092 | 92.432 | 4.796 | 47.3 |
| stage1-gemma-on-C02-compact-thinking1024-direct | 2.020 | 0.190 | 1.286 | 48.731 | 55.426 | 197.953 | 302.242 | 4.796 | 182.3 |
| stage1-gemma-on-C02-compact-thinking1024-go | 2.173 | 0.184 | 1.114 | 50.631 | 62.336 | 190.139 | 303.117 | 4.796 | 182.3 |
| stage1-qwen4-on-C01-current-thinking1024-go | 1.168 | 0.105 | 1.359 | 6.641 | 9.387 | 117.747 | 133.784 | 3.214 | 108.8 |
| stage1-qwen4-on-C01-current-thinking1024-direct | 1.706 | 0.168 | 1.358 | 8.261 | 9.593 | 112.148 | 130.012 | 3.214 | 108.8 |
| stage1-qwen4-on-C02-current-thinking1024-direct | 1.058 | 0.429 | 1.239 | 77.466 | 90.864 | 183.791 | 352.129 | 3.784 | 340.7 |
| stage1-qwen4-on-C02-current-thinking1024-go | 1.859 | 0.582 | 1.516 | 79.861 | 90.630 | 186.510 | 357.014 | 3.784 | 340.7 |

監視370件すべて有効。helper RSS最大1.845GiB、system-wide free percentage [32, 72]。swap開始/終了: ['vm.swapusage: total = 5120.00M  used = 3873.19M  free = 1246.81M  (encrypted)\n', 'vm.swapusage: total = 5120.00M  used = 4503.12M  free = 616.88M  (encrypted)\n']。swapはシステム全体でありhelperへの帰属はできない。

## stage2

| セル | driver秒 | tokens | decode token/s | 終了 G/C/T |
| --- | ---: | ---: | ---: | --- |
| stage2-gemma-on-C01-compact-thinking1024-bounded-cap | 88.57 | 1817 | 24.37 | stop / stop / stop |
| stage2-gemma-on-C01-compact-thinking1024-standard-cap | 82.03 | 1864 | 27.03 | stop / stop / stop |
| stage2-gemma-on-C02-compact-thinking1024-standard-cap | 291.10 | 2514 | 13.73 | stop / length / stop |
| stage2-gemma-on-C02-compact-thinking1024-bounded-cap | 297.52 | 2486 | 13.02 | stop / length / stop |
| stage2-qwen4-on-C01-current-thinking1024-bounded-cap | 137.39 | 2423 | 20.88 | stop / length / stop |
| stage2-qwen4-on-C01-current-thinking1024-standard-cap | 132.16 | 2453 | 21.84 | stop / length / stop |
| stage2-qwen4-on-C02-current-thinking1024-standard-cap | 352.62 | 2615 | 14.19 | length / length / stop |
| stage2-qwen4-on-C02-current-thinking1024-bounded-cap | 353.97 | 2607 | 14.37 | length / length / stop |

| セル | load合計秒 | input prepare合計秒 | grammar setup合計秒 | prefix prefill合計秒 | runtime prompt合計秒 | runtime decode合計秒 | generation wall合計秒 | MLX peak GiB | phase末KV最大MiB |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| stage2-gemma-on-C01-compact-thinking1024-bounded-cap | 2.551 | 0.076 | 0.968 | 4.186 | 6.053 | 74.544 | 84.774 | 4.796 | 47.3 |
| stage2-gemma-on-C01-compact-thinking1024-standard-cap | 2.029 | 0.068 | 0.000 | 4.401 | 6.287 | 68.967 | 79.708 | 4.796 | 47.3 |
| stage2-gemma-on-C02-compact-thinking1024-standard-cap | 2.435 | 0.187 | 0.000 | 47.566 | 57.512 | 183.060 | 288.215 | 4.796 | 182.3 |
| stage2-gemma-on-C02-compact-thinking1024-bounded-cap | 1.960 | 0.172 | 1.641 | 47.939 | 54.708 | 190.893 | 293.546 | 4.796 | 182.3 |
| stage2-qwen4-on-C01-current-thinking1024-bounded-cap | 1.397 | 0.121 | 1.327 | 8.552 | 9.703 | 116.030 | 134.296 | 3.214 | 108.8 |
| stage2-qwen4-on-C01-current-thinking1024-standard-cap | 1.080 | 0.110 | 0.000 | 8.565 | 9.696 | 112.334 | 130.669 | 3.214 | 108.8 |
| stage2-qwen4-on-C02-current-thinking1024-standard-cap | 1.174 | 0.423 | 0.000 | 78.006 | 88.559 | 184.238 | 350.802 | 3.784 | 340.7 |
| stage2-qwen4-on-C02-current-thinking1024-bounded-cap | 1.097 | 0.438 | 1.409 | 77.875 | 91.388 | 181.479 | 350.745 | 3.784 | 340.7 |

監視359件すべて有効。helper RSS最大2.699GiB、system-wide free percentage [34, 77]。swap開始/終了: ['vm.swapusage: total = 5120.00M  used = 4407.12M  free = 712.88M  (encrypted)\n', 'vm.swapusage: total = 6144.00M  used = 4600.94M  free = 1543.06M  (encrypted)\n']。swapはシステム全体でありhelperへの帰属はできない。

## Interpretation / Limits

段階1のGo/direct全体時間はGemma C01 88.58/96.37秒、C02 307.40/305.93秒、Qwen C01 137.14/133.59秒、C02 361.92/355.14秒。Goが速い観測もあり、この差をそのまま固定オーバーヘッドとしない。fixture・proxy・planner/validator・起動・通信と時間変動を含む経路比較。

段階2は共通maxTokens=1024。lengthは診断上の正常終了でtask完了ではない。標準側はJSON制約・独自Thinking制御・token単位progressなし。ThinkingやJSONの出力・生成量は変わり、同一token列の処理時間比較ではない。全体時間に占めるprefix/prefillをdecodeと分ける。load/grammar/prefix/generation計測は一部包含関係があり、表の列をすべて足して全体時間を再構成しない。KVとMLX peakとRSSも異なる対象。

本比較の標準構成は同一helper内の公式generate/TokenIteratorと空GenerationComponentsであり、独立した公式CLIや最高速度ではない。共通の独自prefix処理、KV8、AsyncStream consumerは残る。C02のprefix/prefillとメモリ状況も候補に残り、特定の同期・grammar mask・MLX kernel単独が主因とは未特定。追加生成・再試行・条件変更は行っていない（初回の推論前mkdir競合修正は別記）。

## Preparation corrections / Verification

生成前にcapture stub fixと実C02 featの差を検出し、実要求を再生するよう修正した。さらに初回起動でGoとPythonのmkdir競合を検出、推論前に停止し修正した。初回契約・停止証跡を保持。Release45テスト成功、Python構文成功、最終source/asset hash一致、段階1の12要求/応答/token一致、全48phase ok、全監視有効を確認。rawのホームpath・メール等を除去し公開、原本はローカル保持。次段階や本番変更は未実行。
