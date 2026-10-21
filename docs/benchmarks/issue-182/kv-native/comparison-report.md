# H182-008 KV8/native比較結果

## Goal

既存の採用済みprompt・従来A/eagerを固定し、KV8とモデル標準native cacheの速度、生成量、Thinking境界、構造と応答、メモリを比較する。各arm1回。

## Result / Decision

全8セル・24phaseが正常終了し、plan validatorは8/8受理。実Groupingの参照partition一致は7/8で、Qwen C02 nativeのみ不一致。nativeのdecode速度差は-15.0%から+23.3%と混在し、全4組で応答と生成量が変化した。全モデル一律nativeへの採用はNO-GO。既存KV8設定は変更しない。単発観測でモデル別の採用も確定しない。

## Conditions

同一Release binary・metallib・weights4bit・seed144・入力・schema・validator。Gemma簡潔化、Qwen現行prompt。Thinking ON・各phase1024、最終出力とcycle時間制限なし。Gemma temperature/top_p/top_k=1.0/0.95/64、Qwen=0.6/0.95/20。KV以外は固定。詳細CPU診断とtraceはOFF。直列、warmup/retry/repairなし。実行順はユーザー指定どおりC01 KV8→native、C02 native→KV8。

## Measurements

decode token/s = 3phaseのgeneratedTokens合計 / runtimeGenerationSeconds合計。単純なphase速度の平均ではない。cycle秒はprefill、モデルload、文法準備、Go呼出し等を含む。生成量が異なるためcycle差を純粋なKV演算の速度差としない。

| モデル / fixture | KV8 token/s | native token/s | native速度差 | KV8 / native生成tokens | KV8 / native cycle秒 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Gemma E4B / C01 | 24.24 | 20.61 | -15.0% | 1817 / 1887 | 89.74 / 109.37 |
| Gemma E4B / C02 | 12.75 | 14.37 | +12.7% | 2511 / 2409 | 313.44 / 280.57 |
| Qwen3.5-4B / C01 | 20.50 | 21.01 | +2.5% | 2449 / 2296 | 140.16 / 130.52 |
| Qwen3.5-4B / C02 | 14.25 | 17.57 | +23.3% | 2706 / 2659 | 366.95 / 326.98 |

## Thinking / Grouping

G/C/TはGrouping/Category/Text。強制終了は予算到達後の終了誘導を指し、生成異常やJSON失敗ではない。全24phaseのnative応答はok・stop_reason=completed。

| セル | Thinking G/C/T | 予算終了誘導 | 実Grouping参照一致 |
| --- | --- | --- | --- |
| gemma-on-C01-compact-thinking1024-kv8 | 360 / 651 / 676 | なし | 一致 |
| gemma-on-C01-compact-thinking1024-kvnative | 408 / 603 / 748 | なし | 一致 |
| gemma-on-C02-compact-thinking1024-kvnative | 599 / 1024 / 625 | Category | 一致 |
| gemma-on-C02-compact-thinking1024-kv8 | 563 / 1022 / 769 | なし | 一致 |
| qwen4-on-C01-current-thinking1024-kv8 | 515 / 1024 / 781 | Category | 一致 |
| qwen4-on-C01-current-thinking1024-kvnative | 513 / 1024 / 637 | Category | 一致 |
| qwen4-on-C02-current-thinking1024-kvnative | 1024 / 1024 / 485 | Grouping, Category | 不一致 |
| qwen4-on-C02-current-thinking1024-kv8 | 1024 / 1024 / 513 | Grouping, Category | 一致 |

Qwen C02 nativeはdocs=G003、test=G002、実装2ファイル=G001の3グループに分割した。参照は4ファイル1グループ。後段Category/Textへは全群で参照partitionを注入するため、plan受理は実Grouping成功を意味しない。参照は探索用で独立した人手Goldではなく、不一致だけで意味的な誤りを断定しない。

## Response quality

全4組で応答が変わった。以下はvalidatorとは別の定性的観察であり、外部judgeや人手採点は行っていない。

| セル | type / scope | summary |
| --- | --- | --- |
| gemma-on-C01-compact-thinking1024-kv8 | fix / ImageAssetFix | 静的アセット参照を正規化し、[object Object]エラーを解消。 |
| gemma-on-C01-compact-thinking1024-kvnative | fix / StaticAssetFix | オブジェクト型パスを正規化し、404エラーを解消しました。 |
| gemma-on-C02-compact-thinking1024-kvnative | feat / FileFirst | Issue #167 の Phase1 の検証が完了し、ファイル優先の構造経路が確認できました。 |
| gemma-on-C02-compact-thinking1024-kv8 | feat / FileFirst | Issue #167 Phase.1「ファイル優先構造」を実装・検証完了。 |
| qwen4-on-C01-current-thinking1024-kv8 | fix / 画像コンポーネント | 静的アセットの正規化で画像描画の404エラーを修正 |
| qwen4-on-C01-current-thinking1024-kvnative | fix / StaticAsset | 静的アセットのオブジェクト化による404を解消 |
| qwen4-on-C02-current-thinking1024-kvnative | feat / internal/git | Issue #167 ファイル先フェーズ実装 |
| qwen4-on-C02-current-thinking1024-kv8 | feat / internal/gitstat | Issue#167のPhase1で、一時indexとobjectを保持した構造経路の検証 |

C01は両群とも静的アセット正規化と404等の修正を記述したがscopeは一致しない。C02はGemma両群で検証完了の文言を含む。Qwen C02のscopeはnative=internal/git、KV8=internal/gitstatで、入力のinternal/gitstateとの表記一致はない。これらは構造受理だけでは判定しない文章品質上の観察として残す。

## KV and memory

実生成後のsnapshotでGemmaはattention4層、Qwenはattention8層がQuantizedKVCacheからKVCacheSimpleへ変わった。nativeのattention配列はbfloat16。GemmaのRotatingKVCache20層はbfloat16のまま、QwenのMambaCache24層はbfloat16/float32のままで、全層FP16への一律変換ではない。KV8のquantized配列はuint32とbfloat16 metadata。

| セル | phase終了KV MiB G/C/T | MLX peak GiB（3phase最大） |
| --- | --- | ---: |
| gemma-on-C01-compact-thinking1024-kv8 | 40.4 / 47.3 / 47.3 | 4.796 |
| gemma-on-C01-compact-thinking1024-kvnative | 58.4 / 71.4 / 71.4 | 4.796 |
| gemma-on-C02-compact-thinking1024-kvnative | 313.4 / 325.5 / 317.5 | 5.012 |
| gemma-on-C02-compact-thinking1024-kv8 | 175.9 / 182.3 / 178.0 | 4.796 |
| qwen4-on-C01-current-thinking1024-kv8 | 93.9 / 108.8 / 104.6 | 3.214 |
| qwen4-on-C01-current-thinking1024-kvnative | 133.5 / 161.5 / 145.5 | 3.285 |
| qwen4-on-C02-current-thinking1024-kvnative | 594.3 / 598.0 / 582.0 | 4.064 |
| qwen4-on-C02-current-thinking1024-kv8 | 338.8 / 340.7 / 332.2 | 3.784 |

OS監視365サンプルすべて有効、monitor_failed=false。helper RSS最大1.923GiB。system-wide memory free percentageは30–77%。system swap使用は開始3239.94MiB→終了4557.56MiB。thermal警告の記録はなし。RSS、MLX peak、KV snapshotは異なる測定対象で足し合わせない。swapはシステム全体の値でhelper由来とは特定できず、background負荷やメモリ圧迫による速度への影響を排除していない。KV容量は異なる生成長と確保blockを含む実測snapshotで、同一長での容量比ではない。

## Verification / Artifacts

8セル正常終了、24phase完了、24要求の不変条件、12組の同一要求、source/asset hash一致を再確認。詳細CPUタイマーなし、実cache modeとdtype/構成を全phaseで確認。追加生成・条件変更なし。生成結果とOSログをraw/performanceに公開用コピーし、メール形式文字列とホーム絶対pathを置換。原本はローカル保持し、trace/XML/binary/weightsは含めない。SHA256SUMSは公開コピーのhash。

## Limits / Next

n=1で統計的優位性は不明。全組で生成量が変わるため、速度向上をCPU/GPU同期や量子化kernel単独の効果と特定しない。Gemma C01の低下とQwen C02のpartition差があるため一律採用しない。次の実験や本番設定変更は未実行。
