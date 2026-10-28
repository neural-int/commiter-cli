# H182-018 Sparkプロンプト段階比較結果

## Goal

H182-017保存結果を対照に、OFF grammarありのCategory/Text指示改善、続いてLow grammarありの簡潔なThinking指示を独立に比較する。対照は再実行しない。

## Result / Decision

指定順で両段階が測定完了。2セル・4phase、全体265.383秒（約4分25秒）。両段階monitor正常、adapter例外なし、fixture/source/weight/Git不変。段階1はGrouping3/3・Category2/3・validator受理で、分類誤りは解消しない。Textは前回の崩れた表現が消えた一方、UIの意味が不正確になり文章品質の改善を確定できない。段階2は再びThinking2048・終了marker0・最終回答0で上限。追加指示のみでの改善は今回未確認。採用・追加試行・条件変更はしない。

## Conditions / isolation

同一weight revision・改訂Gemma compact・固定fixture・32K/2048（Thinking含む）・native KV・sampling1/.95/topk無効・penalty1/0/0・seed144。各1回、時間制限なし、対照再測定なし、正解group注入なし。stage1のGrouping実要求はH182017と完全一致。CategoryとTextだけが指示変更。stage2のモデルへ送る元requestはH182017と一致し、runtime共通のthink freely文だけbrief instructionへ置換。stage2にstage1の分類/文章変更を混ぜていないことをwire保存とsource監査で確認。構造化出力の型・Schema・grammar・停止制御は維持。
生wire messagesをprivate保存。今回Thinking本文自体は引き続き未保存で、何を反復/検討したかは断定しない。例外時はstage2を開始しないゲートを設けたが、両段階とも測定・監視上の異常なく終了した。

## Quality comparison

| 対象 | 保存対照H182-017 | 今回H182-018 |
| --- | --- | --- |
| OFF Grouping | 3/3 | 3/3 |
| OFF Category | 2/3、メール修正をfeat | 2/3、同じ誤分類 |
| OFF validator | 受理 | 受理 |
| OFF Text | テスト日数らしい不明瞭な断片 | 不明瞭断片は消えたがUIを「日数だけの表示」と記述 |
| Low Grouping | 思考2048で未完了 | 思考2048で未完了 |
| Low Thinking終了marker/最終回答 | 0/0 | 0/0 |

段階1のUI summary「期限日ツールチップを追加して日数だけの表示を可能にする」は、ISO日付のラベル本文を維持しtooltipに相対日数を追加する変更と正確に対応しない。README・メールのsummaryは主目的を記述する。文章が短く/読みやすくなったことを内容の正解とは扱わない。Category出力は保存対照と同じdocs/feat/feat。

## Performance

| 比較 | 対照のセル秒 | 今回のセル秒 | 対照decode emitted tok/s | 今回decode emitted tok/s |
| --- | ---: | ---: | ---: | ---: |
| stage1 spark-off-json_schema | 77.82 | 91.68 | 29.99 | 13.83 |
| stage2 spark-low-json_schema | 115.66 | 166.22 | 23.12 | 29.94 |

段階2のprefillは90.84秒（対照24.38秒）、decodeは68.39秒（対照88.59秒）。文面差だけで約66秒のprefill増加の原因を断定しない。監視の最大取得時間も約4.7–5.0秒となり、予定2秒間隔を常に維持した観測ではない。背景負荷・OS状態など未分離。n=1・異なる時刻の保存対照との比較であり性能改善率を確定しない。

## Resources

stage1: RSS最大2.594GiB、MLX peak3.827GiB（加算不可）。CPU busy平均15.51%、GPU[4, 100]%、thermal {'nominal': 44}、swap MiB開始/終了/最大[2542.94, 2542.94, 2542.94]、free%[28, 68]、同時adapter最大1。
stage2: RSS最大2.077GiB、MLX peak3.827GiB（加算不可）。CPU busy平均11.85%、GPU[0, 100]%、thermal {'nominal': 74, 'fair': 9}、swap MiB開始/終了/最大[2542.94, 2534.94, 2542.94]、free%[32, 58]、同時adapter最大1。

KV各層dtype/容量、load/template/grammar/prefill/decode/host/CPU user-system等をmeasurementsへ保存。温度実数・消費電力・プロセス別GPUは未取得。

## Limits / Next

この2つの指示変更だけで結果が改善するという仮説は、今回の範囲では支持できない。既存修正をfeatとする分類の不安定さと、Lowの回答用予算確保が未解決。次に検討する場合は分類入力の関連付け・判断根拠と、Thinking/最終回答の予算分離を別課題にする。どちらも未実装・未実行。原本・失敗と保存対照を保持する。
