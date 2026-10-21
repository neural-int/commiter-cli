# H182-004 プロンプト比較（全セルの結果確定）

## 要約

事前監査を通過し、12ケースを計画し、ユーザー指定により全体を中断。Gemmaの4ケースは完了し、全てplan受理・実Groupingの探索Gold一致。Qwen4B C01現行もplan受理。Qwen4B C01簡潔化はユーザー指定でText生成を中断し、Qwen4B C02簡潔化は完了し、現行群はユーザー指定でGrouping生成を中断。C02簡潔化の実Groupingは基準不一致。その後、ユーザー指定により未実行のQwen2B OFFの4セルだけを再開・完了した。全12セルはplan受理8、言語検証不受理2、ユーザー指定中断2。Gemma簡潔化には速度改善の観測があるが、Qwenでは言語不受理やGrouping不一致があり、全モデル共通の簡潔化採用は行わない。

## 条件と監査

prompt-comparison-preflight.mdを参照。Gemma ON、Qwen4B ON、Qwen2B OFF。各現行/簡潔化×C01/C02、1回ずつ。Thinking/出力予算無制限、120秒制限なし。sampling、seed144、weight、template、KV8bit/group64、schema、validator、Git factsは両群で固定。全12セルのtoken/prefix監査とSwift39テスト成功。過去の結果は比較値として流用せず現行群も再測定する。

## 完了した結果

| モデル・fixture | 現行秒 | 簡潔化秒 | 状態 |
| --- | ---: | ---: | --- |
| Gemma ON C01 | 228.08 | 144.63 | 両群plan受理・実Grouping一致 |
| Gemma ON C02 | 453.60 | 400.72 | 両群plan受理・実Grouping一致 |
| Qwen4B ON C01 | 267.12 | 中断 | 現行plan受理、簡潔化はユーザー指定でTextを中断 |
| Qwen4B ON C02 | 中断 | 865.69 | 簡潔化は固定Groupingで後段plan受理、実Groupingは基準不一致 |
| Qwen2B OFF C01 | 27.65 | 26.13 | 両群invalid_summary_language、実Grouping一致 |
| Qwen2B OFF C02 | 81.27 | 69.40 | 両群後段plan受理。実Groupingは現行一致・簡潔化不一致 |

Qwen4B C01簡潔化のGroupingは457 tokens（Thinking397）、Categoryは340 tokens（Thinking322）で正常終了。Textは最終回答が得られないまま、ユーザー指定により27,776 Thinking tokens・2,583.63秒の進捗時点で中断した（セル全体の終了処理を含むwall timeは2,694.22秒）。user-interruption.jsonに明示しており、成功にも自然終了によるモデル失敗にも数えない。部分出力から最終回答のtoken/sは算出しない。短縮した前段だけで改善と評価しない。

## 追加観測（2026-10-21 11:57 JST頃）

Qwen4B C02簡潔化の実Groupingは、基準の全4ファイル同一グループに対して3グループへ分割した。後段は固定partitionを注入するためplan受理と実Grouping品質を区別する。完了した3フェーズ合計は5,913 tokens（Thinking 5,778）、生成区間の合算token/sは8.69。

2026-10-21 12:41 JST、ユーザー指定により検証全体を停止した。Qwen4B C02現行はGroupingのThinkingが57,984 tokens・9,037.53秒（約150分38秒、prefill込み）の時点で最終回答なし。suite-interruption.jsonと当該セルのuser-interruption.jsonに記録した。実行制御・評価・プロキシ・生成プロセスの終了を確認。Qwen2Bの4セルは未実行のまま保持。全12セル中、plan受理6、中断2、未実行4。

GemmaはC01で228.08→144.63秒（36.6%短縮）、C02で453.60→400.72秒（11.7%短縮）。3フェーズ合計生成トークンはC01が2,913→1,817、C02が2,724→2,511。生成区間の合算token/sはC01が14.65→15.36、C02が9.44→9.84。各1回の観測であり、安定した改善や文章品質の同等性は未検証。

## Qwen2B再開

ユーザー指示「後続のQwen 2Bの検証をすすめて」により、元の4セルだけを同順で実行・完了した。元の全ハッシュ一致と、復元fixtureから再captureしたC01/C02の全3フェーズ要求の完全一致を確認した。Thinking OFF、temperature=1.0、top_p=1.0、top_k=20、seed144、出力予算・時間制限なしを維持。元の凍結済みハーネスのrun関数を再利用し、再開契約・ログ・結果はqwen2-resume/に分離して保存する。Gemma/Qwen4Bの過去結果・中断記録は上書きせず、再実行もしない。

## Qwen2B結果

| ケース・prompt | cycle秒 | 生成tokens | decode tokens/s | validator | 実Grouping |
| --- | ---: | ---: | ---: | --- | --- |
| C01現行 | 27.65 | 116 | 16.05 | invalid_summary_language | 基準一致 |
| C01簡潔化 | 26.13 | 114 | 17.38 | invalid_summary_language | 基準一致 |
| C02現行 | 81.27 | 148 | 14.49 | 固定partition後のplan受理 | 基準一致 |
| C02簡潔化 | 69.40 | 136 | 16.76 | 固定partition後のplan受理 | 基準不一致（2群） |

生成tokensは3フェーズ合計、decode tokens/sは生成tokens合計/runtimeGenerationSeconds合計。prefill/load/文法準備等を含むcycle秒とは分ける。Thinkingは設定OFFで、進捗ログも0。OFF時は完了メトリクスのnativeThoughtTokensフィールドが存在しないため、その欠測を実測0の完了カウンタとして補完しない。

C01現行のsummaryは「Static asset URL normalization for AVIF/WebP/JPG」、簡潔化は「Static asset URL normalization for progressive,4」で、両方とも日本語要件を満たさなかった。簡潔化の末尾にも不自然な内容がある。retry/repairは行っていない。

C02現行のsummaryは「1〜16 ファイルの所有権を保持し、バイナリ復元、一時 index を用いた正逆 replay、」で、文末が途切れている。簡潔化は「1〜16 ファイルの所有権保持とバイナリ再構築の構造経路の検証」。両方type=testを出力した。文章・分類の意味的な正しさをvalidator受理だけでは保証しない。簡潔化は約14.6%短時間だが、Groupingが基準の1群から2群へ分割され、成功条件を満たす速度改善とは評価しない。

## 最終監査

全34実送信要求について、元要求から予定どおりpromptだけを変換し、他の入力と設定が不変であることを再確認した。元実行のOS監視2,829サンプル、Qwen2B再開の48サンプルは全件有効。Qwen2B再開の4セルは237.25秒（hash照合等を含むdriver区間）、monitor_failed=falseで終了。各セルの時間にはdriverのhash照合を含めない。

元のresults.jsonでは、外部中断によるEOFがbackendエラーに見え、interruptedフィールドもnullのままである。user-interruption.jsonとsuite-interruption.jsonを正として解釈し、final-summary.jsonで2セルをuser_interruptedに分類した。生ログは書き換えていない。元suiteのcompletion.jsonがないのはユーザー指定の全体停止によるもので、Qwen2B再開のcompletion.jsonと区別する。

## 成果物と限界

原本は/private/tmp/issue182-prompt-comparison/、Qwen2B再開は同qwen2-resume/に保存。公開コピーはprompt-comparison/raw/、実行時source snapshot・Swiftテストログも含む。公開コピーではメール形式文字列と絶対ホームパスを置換し、raw/SHA256SUMS.txtに公開ファイルのchecksumを記録する。実行契約内hashは置換前の原本の値であり、公開コピーhashとは区別する。

各1回の小規模探索比較であり、安定した改善率や16ファイル性能は未検証。後段は固定partitionを使用し、end-to-end成功と区別する。探索Goldは独立人手承認なし。今回は全登録セルを実行または明示的中断として記録し、追加生成・再試行・モデル採用は行っていない。Issue #182全体のGoalは未完了。
