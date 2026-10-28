# H182-013 kimdel履歴付き推論比較：入力32K・出力4096

## Goal

前回の固定Git差分・照合履歴を再利用し、全フェーズの要求コンテキストを32768固定、出力をThinking＋最終回答合計4096トークンとして、Grouping・Category・Textの品質と性能を測定する。

## Result / Decision

全18条件・50フェーズを測定終了。1155.651秒（約19分16秒）、監視576件すべてvalid、fatalなし。helper/evaluationの終了を別途確認。最終validator受理6/18、Grouping完全正解4/18（前回2/18）。正解GroupingかつCategory全正解はQwen3.5-4B Thinking ONの2条件で、scope=docsが禁止語のため最終拒否。正解Grouping・Category・validatorをすべて満たす条件は0/18。
Qwen3.5-4B Thinking OFF・文法制約ありは今回初めて正解3群とvalidator受理を同時に達成したが、メールの不具合修正をfeatと誤分類したため、内容まで全正解の成功には数えない。モデルの採用確定や追加修正・再推論は行っていない。

## Conditions

同一fixture・モデルrevision・sampling・seed144・KV8/group64・eager・systemプロンプト・評価基準・18条件順序。各条件1回、warmup/retry/repairなし。各phaseでhelperを別起動しload/prefillを含む。前回から変更したのは32K固定入力投影と4096出力上限。Release helperを上限対応のため再ビルドした。従って前回との差を履歴追加だけの因果効果とは断定しない。OS cacheや熱条件は固定できていない。
4096は思考＋最終JSONの合計で、Thinking専用予算や強制回答誘導は追加しない。時間制限なし。上限到達はmax_tokensとして未完了扱い。入力要求32768はホストの投影・事前容量検査の枠であり、KVを32768まで事前確保する指定ではない。今回は全phaseの実測入力＋生成が32768以内であることも確認する。

## Input / ETL verification

全50要求context_tokens=32768、output_tokens=4096を確認。全18 Groupingのmessages+Schemaは同一hash。全てにコマンド14件・会話3件、依頼と最終応答本文を収録。全phaseのagent_history_omission_reasonsは空または未設定で、前回の予算による省略はない。Category/Textは対象群に必要な履歴を選択している。fixtureは前回とbyte一致、実Git差分も全条件で不変。今回履歴の再取得は行わず、前処理は前回凍結結果。取得partial/history_incompleteと9ブロックambiguousという照合自体の制約は残る。

## Cell measurements

正解はA=メール正規化実装＋テスト(fix)、B=期限日tooltip実装＋テスト(feat)、C=README(docs)、breaking=false。正解群はラベルに依存しないファイル集合で評価。Categoryは正解群のみ採点。validator受理は意味の正しさとは別。

| 条件 | calls | 正解群/3 | Category正解/対象 | validator | 秒 | 生成tokens | decode token/s |
| --- | ---: | ---: | ---: | --- | ---: | ---: | ---: |
| qwen25-05-json_schema | 3 | 0/3 | — | 拒否 | 14.36 | 98 | 104.60 |
| qwen25-05-none | 1 | 0/3 | — | 拒否 | 5.29 | 57 | 112.96 |
| qwen25-15-none | 3 | 0/3 | — | 拒否 | 29.59 | 299 | 58.20 |
| qwen25-15-json_schema | 3 | 0/3 | — | 受理 | 26.05 | 90 | 54.87 |
| qwen25-3-json_schema | 3 | 1/3 | 0/1 | 受理 | 57.79 | 196 | 32.45 |
| qwen25-3-none | 3 | 1/3 | 0/1 | 拒否 | 80.42 | 224 | 20.32 |
| coder25-05-none | 3 | 1/3 | 0/1 | 拒否 | 71.88 | 4218 | 74.41 |
| coder25-05-json_schema | 3 | 1/3 | 0/1 | 受理 | 18.84 | 369 | 85.65 |
| coder25-15-json_schema | 2 | 0/3 | — | 拒否 | 25.04 | 93 | 55.64 |
| coder25-15-none | 2 | 0/3 | — | 拒否 | 27.41 | 127 | 52.28 |
| coder25-3-none | 3 | 0/3 | — | 拒否 | 70.67 | 150 | 21.66 |
| coder25-3-json_schema | 3 | 0/3 | — | 受理 | 71.27 | 161 | 24.06 |
| qwen2-off-json_schema | 3 | 0/3 | — | 受理 | 33.82 | 118 | 52.03 |
| qwen2-off-none | 3 | 0/3 | — | 拒否 | 41.66 | 379 | 44.25 |
| qwen4-off-none | 3 | 3/3 | 2/3 | 拒否 | 87.37 | 184 | 17.16 |
| qwen4-off-json_schema | 3 | 3/3 | 2/3 | 受理 | 87.85 | 196 | 16.89 |
| qwen4-on-json_schema | 3 | 3/3 | 3/3 | 拒否 | 202.69 | 2576 | 19.57 |
| qwen4-on-none | 3 | 3/3 | 3/3 | 拒否 | 202.82 | 2609 | 19.31 |

## Comparison

| 指標 | H182-011 自動16K・出力無制限 | H182-013 固定32K・出力4096 |
| --- | ---: | ---: |
| 測定確定 | 18/18 | 18/18 |
| Grouping完全正解 | 2/18 | 4/18 |
| 最終validator受理 | 6/18 | 6/18 |
| 正解GroupingかつCategory全正解 | 2/18 | 2/18 |
| 正解GroupingかつCategory全正解かつvalidator受理 | 0/18 | 0/18 |

前回はQwen4 Thinking ONのTextが英語・scope=feat/fixで拒否。今回は日本語に変わったがREADMEにscope=docsを選び、拒否が残った。Thinking OFFは2条件ともGroupingが正解になった。OFF・文法制約なしはCategory 2/3で長いscopeにより形式拒否、ありはCategory 2/3で最終受理。小型モデルでは全統合、実装とテストの分離、README偏重、汎用例の転記などが残る。

中断H182-012は32K固定・出力無制限で6条件のみ確定。Coder0.5B noneのTextが生成だけ33,856超となり、総コンテキスト終了制御がないため停止した。H182-013の同条件はText4096でmax_tokens終了（セル71.88秒）となり、後続へ進んだ。無制限の最終完了時間との比較値ではない。

## Output / Thinking boundary

json_schema: 26phase、helper成功26、JSON解析成功26、Schema適合26。
none: 24phase、helper成功22、JSON解析成功21、Schema適合17。

出力上限到達: [('coder25-05-none', 'bounded-text')]。上限による終了を正常回答として扱っていない。
qwen4-on-json_schema: bounded-grouping=561 tokens (Thinking 519, forcedClose=False), bounded-category=1370 tokens (Thinking 1298, forcedClose=False), bounded-text=645 tokens (Thinking 576, forcedClose=False)
qwen4-on-none: bounded-grouping=562 tokens (Thinking 519, forcedClose=False), bounded-category=1396 tokens (Thinking 1298, forcedClose=False), bounded-text=651 tokens (Thinking 576, forcedClose=False)

## Performance / resources

helper RSS最大3.070GiB、MLX peak最大3.113GiB。別指標のため加算しない。system CPU busy平均8.13%、GPU利用率範囲[0, 100]%。thermalStateサンプル{'nominal': 46, 'fair': 530}。swap使用量MiB（開始/終了/最大）[2657.31, 2641.31, 2657.31]、system free%範囲[43, 71]。観測helper同時最大1。
フェーズ別load・prepare・prefill・grammar preparation/setup・decode・prefix・Go側serialization/projection/JSON検証・CPU user/system・helper wall・フェーズ間gapはphase-timing.mdとmeasurements.jsonに保存。温度実数・消費電力・プロセス別GPUは取得不可。CPU/GPU/thermal/swapはsystem全体を含み、このrunだけによる因果とはしない。監視は2秒間隔で瞬間peakを保証しない。

## Text review

- qwen25-05-json_schema: Exampleをそのまま出力し、変更目的を説明しない
- qwen25-15-none: Textがgrammar_failureで終了。JSON文法マスクなしでも事後のJSON形式検査は残る
- qwen25-15-json_schema: 全変更を一群に統合し、READMEの品質チェック説明に偏る
- qwen25-3-json_schema: 実装とテストを分離し、UI説明に偏る。メール変更の説明不足
- coder25-05-none: Textが4096上限に達して未完了
- coder25-05-json_schema: 全5群に同じパフォーマンス改善説明を反復。文末が途中で終わる
- coder25-3-json_schema: READMEの説明に偏り、他の変更目的を表現しない
- qwen2-off-json_schema: 全変更を一群にしREADMEに偏る。文末が途中で終わる
- qwen4-off-none: 日本語で3目的を記述するがscope長制約に違反。UIのtypeもrefactorと誤る
- qwen4-off-json_schema: 3目的の日本語説明あり。メール修正のtypeがfeatになりCategory誤り
- qwen4-on-json_schema: 3目的の日本語説明とCategoryは正しいがREADMEのscope=docsが禁止語で拒否
- qwen4-on-none: 3目的の日本語説明とCategoryは正しいがREADMEのscope=docsが禁止語で拒否

## Limits / Next

単一5ファイルfixture、各条件1回。履歴省略の解消による改善の可能性は示すが一般精度・統計的優位性は未評価。次に改善するならQwen4のscope禁止語、OFFのCategoryのfix/feat判断に絞る。追加実装や再試行は今回行わない。生の会話・要求・応答は公開せず原本保持。
