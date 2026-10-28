# H182-019 Spark Thinking本文保存・Low/OFF再測定

## Goal

H182-018 stage2のbrief思考指示を使い、Low/OFFのgrammarありを再測定する。元Category/Textを維持し、実出力token IDsとThinking/境界/最終/raw本文をprivate0600へ保存する。

## Result / Decision

全2セル・3phaseの測定終了、198.412秒（約3分18秒）。監視99件全valid・fatalなし。Lowは思考2048のまま未完了。OFFは全5ファイルを個別群に分離し、Categoryでunresolvedになり停止。最終validator受理0/2。原本保存と各tokenカウント照合に成功。追加推論・条件変更・モデル採用なし。

## New finding from saved thought (summary only)

Lowの序盤はREADME、メール実装+回帰テスト、UI実装+テストという独立した3目的を識別していた。ファイルIDと許容group labelの確認、Groupingルールの再確認を経て、途中から短い同一tokenの連続反復に入った。最後の1631/2048 tokens（約79.64%）が単一tokenの連続。反復前は417 tokens。終了marker0、最終回答0。
従って今回のLow未完了を、単に難しい問題で有用な検討を2048tokens続けたと解釈しない。情報の読み取りと目的候補の識別はできたが、反復へ崩れて回答に至らなかった。これはモデル生成の症状であり、weight/量子化/sampling/実装のどれが原因かは未分離。過去H182017/018は本文未保存なので、過去の上限終了も同じ反復だったとは断定しない。

## Conditions / logs

Spark固定revision、同じfixture/seed144/sampling1/.95/topk無効/penalty1/0/0/native KV/32K入力/2048出力、grammarあり。Low→OFFの順で各1回。元Category/Textを使用しstage1変更なし。モデルへ送るLow requestはH182018 stage2と完全一致。実wireにbrief指示を確認。今回の変更は生成後のprivateログ保存。終了時の保存処理は生成時間の後に走るがhelper wallへ含む。固定fixture/source/weight/Git不変。
全3phaseのprivateログを保存しmode0600を確認。実token数とmetricsのthought/boundary/finalを照合。Thinking OFFはthought空。raw token IDsとdecode本文、停止理由も保存。公開報告には原文・token IDを埋め込まない。

## Comparison

| 指標 | 比較先 | 今回 |
| --- | --- | --- |
| Low Grouping | H182018: Thinking2048・最終0 | 同じ未完了。今回は本文から反復を確認 |
| Low セル秒 | H182018:166.22秒 | 94.86秒 |
| OFF Grouping | H182017:3/3正解 | 1/3、実装とテストも分離 |
| OFF Category | H182017:3群、メールをfeatと誤分類 | 5群へ分離後、3群のbreakingをunresolvedとして停止 |
| OFF Text | H182017:実行 | 今回未実行 |

OFFはH182017に対してbrief共通文面が異なるので、同一条件の再現試験とは扱わない。今回の1/1 Category評価は正解集合のREADMEだけに対する既存採点であり、全Category正解を意味しない。stage1 OFFとの比較もCategory/Text指示が異なり独立対照ではない。

## Performance / resources

Lowはprefill22.54秒/decode68.08秒、約30.08 emitted token/s。OFFはGrouping/Category計2phaseでprefill85.37秒/decode10.28秒、約14.88 emitted token/s。OFF Categoryのprefillが60.73秒と長く、n=1の時間差をprompt効率やハードウェア限界と断定しない。
helper RSS最大2.581GiB、MLX peak3.827GiB（加算不可）。system CPU busy平均6.81%、GPU範囲[0, 100]%、thermal {'nominal': 35, 'fair': 64}、swap MiB開始/終了/最大[2494.94, 2494.94, 2494.94]、free%[31, 70]、観測同時adapter最大1。
load/template/grammar/prefill/decode/host/CPU user-system/KV dtype・bytes等はJSONと時間表。温度実数・消費電力・プロセス別GPUは未取得。

## Next (not executed)

Lowについては、出力上限を増やす前に反復の再発性・サンプリングによる抑制やモデル/runtimeの数値健全性を切り分ける根拠が得られた。反復検知は無駄生成の停止には使えるが、正解最終JSONを保証しない。OFFはbrief思考指示をOFFへ適用する効果が未分離。追加実験は行わず今回の失敗・原本を保持する。
