# H182-022 Qwen3.5-4B repetition penalty比較

## Goal

Thinking ON/OFFでrepetition_penalty=1.0と1.1だけを変更し、収束・生成量・精度を対比較する。小型モデルへの展開判断の材料とする。

## Result / Decision

全4セル・12phaseが正常測定完了。全体745.979秒（約12分26秒）、monitor372件valid、fatalなし。全phase自然終了し、4096上限到達なし。Groupingは全条件3/3、最終validator受理は0/4。ONはCategory3/3だがTextが英語、OFFはCategory1/3かつscope違反が残った。

1.1のONはThinking合計5167→4450 tokens（13.9%減）、セル時間293.353→277.709秒（5.3%減）。ただしGroupingの思考は増えており、全phaseを一様に短縮したわけではない。decode自体の高速化や精度改善は示さない。OFFはCategory/Textの出力が同一で改善なし。今回だけを根拠に小型モデルへ自動展開しない。追加推論は実行しない。

## Conditions / verification

Qwen3.5-4B 4bit、revision 0e7ffd5c629ef7719d4cbc04069232580bfa9d9c。固定kimdel fixture、H182-016改訂Qwen prompt、32K/4096、native KV、temperature=.6/top_p=.95/top_k=20/seed144、JSON Schema grammarあり。専用Swift Releaseの同一binaryで4条件各1回。各phase別helper。ONはThinking限定、OFFはJSON回答に適用。生成済直近20tokensのみ、prompt除外、EOS/runtime stops/Thinking境界は免除。ONの最終JSONには適用しない。初回load/prefillも計測に含める。warmup/retry/repairなし。

生成を伴わないwire監査でON内・OFF内の全3phaseそれぞれのeffective messagesとSchemaが一致した。生成履歴を使うsign-aware penalty、1.0無変更、window20、境界保護、ON/OFF適用区間の非推論self-test成功。既存fixture/Gitは全セル不変。履歴取得不足など固定fixture由来の制約は前回から維持する。

## Cell measurements

decode=生成tokens合計/runtimeGenerationSeconds合計。Thinking込み、load/prefill除外。Categoryは完全正解groupに対する分類。

| Thinking | penalty | Grouping | Category | validator | セル秒 | 生成tokens | Thinking tokens | decode token/s |
| --- | ---: | --- | --- | --- | ---: | ---: | ---: | ---: |
| ON | 1.0 | 3/3 | 3/3 | invalid_summary_language | 293.353 | 5348 | 5167 | 23.38 |
| ON | 1.1 | 3/3 | 3/3 | invalid_summary_language | 277.709 | 4634 | 4450 | 22.03 |
| OFF | 1.0 | 3/3 | 1/3 | invalid_scope | 87.774 | 204 | 0 | 17.52 |
| OFF | 1.1 | 3/3 | 1/3 | invalid_scope | 86.923 | 204 | 0 | 17.62 |

## Phase measurements

prefillはruntimePromptSeconds、decodeはruntimeGenerationSeconds。grammarはgrammarSetupSeconds。これらの計測は包含関係があるため全列を単純加算しない。詳細のprepare/grammar準備/生成wall/CPU診断/host user-system時間とMLXメモリはmeasurements.jsonに保持。

| case | phase | 生成tokens | Thinking | prefill秒 | decode秒 | token/s | load秒 | grammar秒 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| qwen4-on-rp10-json_schema | grouping | 527 | 474 | 20.923 | 21.705 | 24.28 | 1.559 | 0.692 |
| qwen4-on-rp10-json_schema | category | 3985 | 3913 | 16.974 | 166.267 | 23.97 | 1.244 | 0.721 |
| qwen4-on-rp10-json_schema | text | 836 | 780 | 17.416 | 40.774 | 20.50 | 1.549 | 1.193 |
| qwen4-on-rp11-json_schema | grouping | 598 | 545 | 23.189 | 34.239 | 17.47 | 1.234 | 0.642 |
| qwen4-on-rp11-json_schema | category | 3331 | 3259 | 18.838 | 141.534 | 23.54 | 1.411 | 0.700 |
| qwen4-on-rp11-json_schema | text | 705 | 646 | 17.352 | 34.621 | 20.36 | 1.430 | 1.192 |
| qwen4-off-rp10-json_schema | grouping | 52 | 0 | 23.811 | 2.968 | 17.52 | 1.205 | 0.644 |
| qwen4-off-rp10-json_schema | category | 71 | 0 | 21.292 | 4.208 | 16.87 | 1.245 | 0.703 |
| qwen4-off-rp10-json_schema | text | 81 | 0 | 22.721 | 4.467 | 18.13 | 1.549 | 1.544 |
| qwen4-off-rp11-json_schema | grouping | 52 | 0 | 25.202 | 3.203 | 16.23 | 1.307 | 0.646 |
| qwen4-off-rp11-json_schema | category | 71 | 0 | 20.864 | 3.946 | 17.99 | 1.282 | 0.645 |
| qwen4-off-rp11-json_schema | text | 81 | 0 | 20.951 | 4.426 | 18.30 | 1.500 | 1.514 |

## Output / Thinking boundary

全12phaseがcompleted。ON全6phaseでThinking終了境界を経て最終JSONへ自然遷移、強制終了なし。保存したsampled IDsには先読み2tokensが含まれたため、runtime emitted countのprefixで最終本文とThinking本文を確定。全12phaseでruntime countと保存emitted countが一致し、本文ファイル0600を確認。生のThinking・token IDs・会話・wire本文はprivate原本に保持し公開しない。

ONのThinkingはGrouping474→545、Category3913→3259、Text780→646。全phaseの同一token最大連続長は2で、Sparkで見られた長い単一token反復は観測されなかった。Categoryでは分類と破壊的変更の証拠参照を繰り返し確認し、最終的には正解へ到達した。1.1でこの検討量は減ったが、Textの日本語指定違反は残った。これは内容要約であり思考原文は転載しない。

OFFのGroupingラベルは変わったが実際の3分割は同一。CategoryはUIをfix、メール不具合修正をfeatと逆分類。Textのscopeにdocs/fix/featを使用し、禁止語の制約を満たさなかった。ONはscopeを満たしたがsummaryが英語。構造化出力が成立しても意味と業務制約の成功を保証しない。

## Performance / resources

約2秒間隔。CPU/GPUはシステム全体、RSSとMLX peakは別指標で加算しない。セル順序と熱・OS cache・背景負荷を固定していないため、単回の速度差をpenalty固有の効果と断定しない。

| case | CPU平均% | GPU平均% | helper RSS peak MiB | swap開始/終了MiB | free最小% | thermal |
| --- | ---: | ---: | ---: | --- | ---: | --- |
| qwen4-on-rp10-json_schema | 31.05 | 89.48 | 884.77 | 2472.81/2472.81 | 31 | {'nominal': 37, 'fair': 110} |
| qwen4-on-rp11-json_schema | 11.17 | 91.46 | 2096.86 | 2472.81/2472.81 | 35 | {'fair': 138} |
| qwen4-off-rp10-json_schema | 9.97 | 92.18 | 3136.64 | 2472.81/2472.81 | 35 | {'fair': 44} |
| qwen4-off-rp11-json_schema | 7.90 | 93.88 | 2969.94 | 2472.81/2472.81 | 40 | {'fair': 43} |

同時helper最大1。全セルswapは2472.81MiBで増加なし。nominalからfairへ移行、serious/criticalは観測なし。RSSは共有GPUメモリやMLX全量を表す指標ではない。native KVはworkingCache=nilからランタイム内部に委譲する既存経路で、実配列dtype/bytesは未export。温度実数・消費電力・プロセス別GPUは未取得。監視負荷は平均.173秒、最大1.341秒/回。

## Limits / Next

単一fixture・各条件1回。1.0が既に全phase自然終了しており、今回の比較では収束失敗を解消したとは言えない。1.1の採用や小型モデルへの展開は未確定。現在明確に残る失敗はON Textの言語指定とOFF Category/scope。必要ならこれらを別条件として検証するが、本測定では変更しない。原本保持。公開用成果物は本文を除く集計と報告のみ。
