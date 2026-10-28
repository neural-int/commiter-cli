# H182-017 Spark-X2.6-4B履歴付きE2E比較

## Goal

改訂Gemma compactを使い、Spark OFF/Low/High×JSON制約あり/なし6条件を同じkimdel固定fixtureで比較する。Lowが指定baseline。

## Result / Decision

全6セル・10phaseが測定終了。632.842秒（約10分33秒）、監視316件全てvalid、fatalなし。Grouping完全正解はOFF両条件の2/6。有効Groupingを得た条件だけでは2/2だが、残る4条件はThinking上限で採点不能であり成功扱いにしない。validator受理1/6（OFF grammarあり）、その条件もCategory2/3と文章不備があり、全工程の意味まで全正解は0/6。モデル採用・追加推論・条件変更なし。
Low/Highの全4条件はGroupingで2048 emitted tokensをすべてThinkingに使い、終了markerも最終JSONも出ないままmax_tokensとなった。小さなsmokeでは自然遷移したが、本fixtureでの2048予算には足りなかった。予算を増やせば成功するという保証はない。

## Conditions

Affine4/group64、model revision9aa68e67d64f5728b42f15661524fb7dca8d984d。専用venv mlx-lm0.32.2/llguidance1.9.1（ユーザー承認済み）。Three-phase PlannerとSchemaは維持し、Sparkアダプターが同一IPCへ接続。temperature1/top_p.95/top_k0、repetition1/presence0/frequency0、seed144、標準native cache。32K入力/Thinking込み2048出力、時間制限なし、各条件1回、warmup/retry/repairなし。実Groupingを後段へ渡す。
改訂Gemmaプロンプトを実要求で確認。フィールド/型/必須/完全Schemaの指示を共通generatorで生成し、偏った自動構造例を除外。LLGuidanceはThinking中にmaskを適用せず、終了token4後だけJSON mask。OFFは最初からJSON mask。文法制約なしでも事後のJSON/host validatorは維持。
chat template実引数はreasoning_effort=low/high（thinking_effortではない）。実templateとOFF/Low/Highの末尾を非推論監査。開始3/終了4/EOS1を別扱い。全要求は32K/2048、Groupingにコマンド14件・会話3件あり。fixture byte一致、Git・source・weight不変。履歴自体は既存partial/history_incomplete・ブロック曖昧を含む。
初回smokeのKV scalar計測例外とok保持バグを修正し、原本保持。修正後3smokeの自然終了とcache/peak測定を確認し、境界/JSONmask/EOS/Schemaの4unitと既存7テストに成功してから本登録。

## Cell measurements

速度はemitted tokens（EOS除外）/runtimeGenerationSeconds。runtimeGenerationSecondsは公式APIのgeneration_tokens/generation_tpsから換算。API token countは正常EOS時に1多いのでlibraryGenerationTokensとlibraryDecodeTokensPerSecondも別保存。Swiftとの完全同定義とはしない。prefill速度はinput合計/prefill時間合計。

| Thinking | grammar | Grouping | Category | validator | 秒 | output | prefill tok/s | decode tok/s |
| --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: |
| off | あり | 3/3 | 2/3 | 受理 | 77.82 | 248 | 287.97 | 29.99 |
| off | なし | 3/3 | 1/3 | 拒否 | 78.60 | 257 | 275.49 | 30.65 |
| low | なし | 採点不能 | — | 拒否 | 116.85 | 2048 | 255.79 | 23.16 |
| low | あり | 採点不能 | — | 拒否 | 115.66 | 2048 | 273.70 | 23.12 |
| high | あり | 採点不能 | — | 拒否 | 117.88 | 2048 | 268.42 | 22.69 |
| high | なし | 採点不能 | — | 拒否 | 119.25 | 2048 | 264.15 | 22.37 |

## Quality / stop reasons

OFF grammarあり: README=docs、UI=featは正しいがメール不具合修正をfeatと誤る。Textは日本語だがUI summaryに「固定日数とは-days=2日超過」という不明瞭な記述があり、形式受理を文章品質保証としない。
OFF grammarなし: Grouping正解。Categoryが全群featで1/3のみ正解。TextでJSONDecodeError（line1 column82）となり停止。adapterはこの形式不備をruntime_errorとして返しているが、モデルload失敗ではない。raw不正Textは返却されず内容の再採点不可。
Low/High: 全4条件、Thinking2048/終了marker0/最終回答0。EOS自然終了やJSON遷移ではなく予算終了。既存採点器のevaluation_note=JSONDecodeErrorは空generated_jsonによる採点不能で、上限終了の本体と分ける。

## KV and resources

27層RotatingKVCache（window512）+9層KVCache、実配列dtypeは全てbfloat16。OFFのKV容量はGrouping297MiB/Category261MiB/Text252MiB、Low/Highの上限時は369MiB。OFF none TextはJSON parse例外で終端snapshot/peak採取前に抜けたため未取得。0bytesとは扱わない。
helper RSS最大2.554GiB、MLX peak最大3.827GiB（別指標・加算不可）。system CPU busy平均10.43%、GPU範囲[0, 100]%。thermal {'nominal': 58, 'fair': 258}。swap MiB開始/終了/最大[2542.94, 2542.94, 2542.94]。free%範囲[28, 67]。観測同時adapter最大1。
load/template+共通contract IPC/grammar/prefill/decode/helper wall/user-system CPU/host projection等はphase-timing.mdとmeasurements.json。Python起動とmodule importはhelper wallに含みloadSecondsには含まない。温度実数・消費電力・プロセス別GPUは未取得。2秒監視で背景処理を含む。

## Comparison / Limits

H182-015 GemmaはGrouping4/4、Grouping/Category/validator全正解3/4。H182-016 QwenはGrouping4/18、全正解0/18。Spark OFFはGrouping2/2だがCategoryを全正解にできていない。Low/Highは有効Groupingなし。
Gemma OFF native grammarありは約62.86秒/17.13token/s、Spark OFF grammarありは約77.82秒/約30token/s。Sparkはdecodeが速くても、prefillが約61秒を占めて全体時間は短くなっていない。Python/Swift、llguidance/既存grammar、2048/4096の差があり、モデル単体の速度差や優劣には換算しない。
単一fixture各1回、一般精度は未評価。Lowで自然回答へ移れるかは今回未達。次の候補はThinking予算と最終回答用予算の分離だが、今回は追加せず全条件を原本保持して終了する。
