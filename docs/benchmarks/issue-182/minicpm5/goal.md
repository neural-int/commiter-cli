# H182-023 MiniCPM5-2B int4/int8比較

## Goal / Conditions

同じ5ファイル・3目的のkimdel入力を再取得・固定して、MiniCPM5-2B int4/int8×JSON文法あり/なし4条件を各1回直列測定。Thinking ONのみ、自然終了。32K入力/Thinkingを含む4096出力、native KV、seed144、temperature1/top_p.95/min_p0/top_k0/repetition1、時間制限なし。warmup/retry/repairなし。Groupingの実結果をCategory/Textへ渡し、正解partitionを注入しない。

int4はopenbmb/MiniCPM5-2B-MLX revision8a9ad7539ac86281d0ac2b017ba04a5de53fe9a3、int8はmlx-community/MiniCPM5-2B-8bit revision2d20e8e672ce892d50f7265bfd3fc9b59b718f2a。affine/group64。モデルの公開元が異なり、同一BF16原本から自前変換した量子化単独実験ではない。configは量子化欄以外一致、tokenizer/templateのSHA一致、ロード済量子化層bits4/8を確認。

改訂Gemma compactの保存仕様からsystemを再構成し、履歴の現行フィールド名、docs定義、scope禁止語と文字数を適用。既存Swiftの非推論contract generatorからフィールド/型/必須/Schema説明を付与し、自動構造例を除外。旧private wire原本が消失しているため旧Gemma systemとのbyte同一性は未検証。Sparkのbrief/label追記や反復penaltyは混ぜない。

## Input / Environment

旧private tmpのfixture・venvが見つからず、ユーザー了承の上で現在のfixtureから再取得。5ファイル/3履歴context/3操作/12RPC、固定化PASS（2.67秒）。過去H182-015等と履歴本文完全同一とはしない。原本は一時領域ではない専用privateフォルダへ0600で保存。新専用venv mlx-lm0.32.2/llguidance1.9.1。推移依存はrequirements.lockへ固定し、旧環境不変。Python/Swiftランタイム差があるため過去速度とのモデル単独差は断定しない。

## Preflight

両モデルload成功、tokenizer/template同一、公式templateにenable_thinking=Trueで末尾<think>を確認。終了列は実tokenizerで[9]、EOSは[1,130073]。Thinking中は自由生成し、終了列の後だけJSON mask。複数token終了列も扱う。早期EOSを最終JSON成功としない。多token境界/早期EOS/思考中非maskの3unit成功、実tokenizer+LLGuidanceによる境界後のJSON消費検証成功。remote code実行なし。Go検証用overlayビルドとgit diff --check成功。

## Result / Decision

全4セル・11phaseが測定完了。973.767秒（約16分14秒）。completion state=complete、監視486件すべてvalid、fatalなし、全セルGit不変、専用runner/adapterの終了を確認。追加推論・再試行・条件変更なし。

int8は両条件Grouping3/3・最終validator受理だがCategory1/3。UI機能追加をfix、メール不具合修正をfeatへ逆分類している。int4は両条件5ファイルを5群へ分離し、期待3群のうちREADMEのみ一致。文法ありはText上限、なしはCategoryの構造違反で停止。Grouping/Category/validator全正解は0/4で、モデル採用は保留。

## Cell measurements

Groupingはラベルではなくファイル集合の一致。Categoryは完全正解集合の群のみ採点し、全体分類の正解率とはしない。decodeはemitted tokens合計/runtimeGenerationSeconds合計（Thinking込み、load/prefill除外）。正常EOS時にlibraryGenerationTokensはemittedより1多いため、library計測値はJSONへ別保存。

| 量子化 | grammar | calls | 正解群/3 | Category | validator | 秒 | 生成tokens | Thinking | decode token/s | prefill token/s |
| --- | --- | ---: | ---: | --- | --- | ---: | ---: | ---: | ---: | ---: |
| int4 | あり | 3 | 1/3 | 1/1（READMEのみ） | 拒否 | 239.114 | 8705 | 8511 | 43.50 | 501.04 |
| int4 | なし | 2 | 1/3 | Schema違反・採点不能 | 拒否 | 157.794 | 4589 | 4458 | 35.83 | 446.67 |
| int8 | なし | 3 | 3/3 | 1/3 | 受理 | 282.118 | 5172 | 4980 | 22.12 | 405.83 |
| int8 | あり | 3 | 3/3 | 1/3 | 受理 | 294.342 | 5164 | 4980 | 21.10 | 395.38 |

## Phase timing / generation

prefill=runtimePromptSeconds、decode=runtimeGenerationSeconds。包含関係のあるload/prepare/grammar/host/generationWallは単純加算しない。Go側の投影/serialization/応答JSON検査、helper wallとuser/system CPU、runtime詳細をmeasurements.jsonへ保存。

| case | phase | stop | output | Thinking | final | prefill秒 | decode秒 | emitted token/s |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| minicpm5-int4-json_schema | bounded-grouping | completed | 1809 | 1761 | 47 | 12.894 | 37.213 | 48.61 |
| minicpm5-int4-json_schema | bounded-category | completed | 2800 | 2697 | 102 | 9.103 | 57.996 | 48.28 |
| minicpm5-int4-json_schema | bounded-text | max_tokens | 4096 | 4053 | 42 | 10.092 | 104.925 | 39.04 |
| minicpm5-int4-none | bounded-grouping | completed | 1794 | 1761 | 32 | 12.141 | 53.656 | 33.44 |
| minicpm5-int4-none | bounded-category | completed | 2795 | 2697 | 97 | 12.549 | 74.430 | 37.55 |
| minicpm5-int8-none | bounded-grouping | completed | 1558 | 1509 | 48 | 13.866 | 66.893 | 23.29 |
| minicpm5-int8-none | bounded-category | completed | 1203 | 1139 | 63 | 12.413 | 54.484 | 22.08 |
| minicpm5-int8-none | bounded-text | completed | 2411 | 2332 | 78 | 12.245 | 112.460 | 21.44 |
| minicpm5-int8-json_schema | bounded-grouping | completed | 1557 | 1509 | 47 | 14.658 | 75.822 | 20.53 |
| minicpm5-int8-json_schema | bounded-category | completed | 1197 | 1139 | 57 | 12.399 | 56.299 | 21.26 |
| minicpm5-int8-json_schema | bounded-text | completed | 2410 | 2332 | 77 | 12.485 | 112.650 | 21.39 |

## Quality / Thinking / stop

- int4 grammarあり：Groupingは全5ファイル別群。READMEのdocs、単独実装のfeat/fix、単独テストのtestという分類を返したが、期待する実装とテストの統合を満たさない。TextはThinking4053＋境界1＋最終回答42=4096で上限。Thinking自体は自然終了したが、回答を完成する残予算が足りなかった。思考終了前に上限となったケースと混同しない。
- int4 grammarなし：同じ5群。Categoryは各groupの値をオブジェクトではなくPython辞書表記を含む文字列として返した。外側はJSON解析可能でもSchema不適合となり、hostがinvalid_schemaで停止。Text未実行。既存採点器のTypeErrorは文字列を辞書として読めなかった二次的な採点不能で、モデルload/adapter障害ではない。文字列を再解析して成功へ修復していない。
- int8両条件：正解3群、CategoryはREADME=docs/UI=fix/メール=feat。期待docs/feat/fixと後2つが逆。breaking refは全てnone。Textは3目的に対応する日本語でscope/長さ等のvalidatorを通った。メール説明の「norm化」は省略的な表現で、文章品質の完成を意味しない。確定した誤typeをTextで修正する仕様ではない。

全11phaseで自然なThinking終了境界を観測し、forced closeなし。最終回答まで完了は10phase、うち1phaseは外側JSON成立でもSchema不適合。int4 Textは上限未完了。Thinkingの同一token最大連続長は全phaseで1または2で、Sparkで見た長い単一token反復は観測なし。思考が長いことと単一token反復を分ける。生Thinking/token IDs/wire本文は公開せずprivate原本のみ0600保存。

## Input / accounting audit

全4条件のGrouping effective messagesは同一SHA。int8両条件のCategory/Text effective messagesも一致。int4とint8の後段は実Groupingが異なるため別入力であり、後段の思考量差を量子化単独効果としない。全11phaseでemitted数=Thinking＋境界＋最終回答数=metrics generatedTokensを確認。runtimeの先読みtokensは本文へ混ぜていない。EOSの具体的な最終IDは出力原本へ保存していないため1と130073のどちらで終わったかは未確定。ランタイムの両EOS設定と正常stopの確認に留める。

凍結対象のsource/helper/fixture/依存lock hashは実行後も一致。再取得した履歴のcoverage自体は固定fixtureの制約を持つ。取得網羅性とLLM精度を同一視しない。

## Performance / resources

| case | CPU busy平均% | GPU平均% | adapter RSS peak MiB | MLX peak MiB | free最小% | thermal |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| minicpm5-int4-json_schema | 17.05 | 94.56 | 1784.75 | 2240.24 | 45 | {'nominal': 40, 'fair': 65} |
| minicpm5-int4-none | 18.88 | 93.62 | 1626.84 | 2240.24 | 42 | {'fair': 77} |
| minicpm5-int8-none | 40.96 | 89.41 | 2826.33 | 3363.02 | 39 | {'fair': 136} |
| minicpm5-int8-json_schema | 51.19 | 88.75 | 2792.02 | 3363.02 | 39 | {'fair': 142} |

全期間swapは0MiB。監視で同時adapter最大1、追加監視の安全停止0。RSSとMLX peakは別指標であり加算不可。CPU/GPUはsystem全体。thermalStateはnominal/fairのみでserious/criticalなし。状態から実throttleの有無・程度は確定しない。温度実数・消費電力・プロセス別GPUは未取得。

初期のRSS欠測は53.565秒、thermalState欠測は27.125秒。既存Swift向けRSSフィルタが実体Python pathを拾わず、専用のprocess-monitorで以後を補完。MLX peakは各phaseから取得したが、RSSの瞬間peakや初期peakは保証できない。追加monitorの負荷はmain monitor.collection_secondsには含まれない。OS監視約2秒間隔、他アプリ/実行順/cache/熱を統制しておらず、単発速度差を安定性能やgrammarの純粋負荷としない。

native KVは42層、実配列dtypeはbfloat16。終端確保容量はint4各phase315〜378MiB、int8各phase252〜315MiB。生成長・input・確保ブロック境界が異なるのでweight bitsの効果でKVが減ったとはしない。各層shape/dtype/bytesはJSONに記録。mlx cacheMemoryBytesはアロケータcacheでありKV容量とは別指標。

## Limits / Next

単一fixture各条件1回、4bit/8bitの公開元違い、再取得履歴と再構成prompt、Pythonと過去Swift環境の差を含む探索比較。今回の標本ではint8にGrouping上の差が見られたが、量子化のみが原因と確定しない。int8は約21〜22 token/s、int4は約36〜43 token/s（全phase加重）で、異なる仕事量・成否の速度を等価な成功時間として順位付けしない。全工程意味正解0/4のため採用保留。追加反復抑制・予算増加・強制遷移・プロンプト変更は本runでは行わない。原本保持。
