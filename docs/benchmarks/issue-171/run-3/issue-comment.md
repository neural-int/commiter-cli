## 要約

4条件×12fixture×通常/逆順の96callを完了しました。固定品質Gateは**NO-GO**です。D（Raw Diff＋暫定B）はExact11/22で、FSと順序依存も悪化しました。構造・プロセスメモリ・元Git/byte/tree安全はPASSです。有限研究として終了し、Productionを変更しません。

## 検証結果

| Arm | 入力 / prompt | Exact | FMペア | FSペア | 順序一致 /11 |
|---|---|---:|---:|---:|---:|
| A | 既存 / 既存 | 10/22 (45.5%) | 26 | 0 | 11 |
| B | Raw / 既存 | 10/22 (45.5%) | 26 | 0 | 11 |
| C | 既存 / 暫定B | 12/22 (54.5%) | 17 | 2 | 7 |
| D | Raw / 暫定B | 11/22 (50.0%) | 20 | 1 | 9 |

- 分母22cell/armはconfirmed11件×2順序。順序反転はpairedであり独立標本とはしません。F12は推論前からundeterminedで主評価除外（1/12、8.33%）。multiple_valid実fixture0件。構造不正/unscorable confirmed0件。
- F01は旧2runでA通常順を観測した既知診断ケースとして明記。未生成confirmed F02〜F11だけではExact A8/20・B8/20・C10/20・D9/20。旧固定Gate/分母を事後に変更していません。
- 通常/逆順のExactはA5/11・5/11、B5/11・5/11、C5/11・7/11、D5/11・6/11。FM発生unique caseはA/B6/11、C/D5/11、FS発生caseはA/B0、C2/11、D1/11。ペア件数とcase/cell件数を区別。
- A→B: Exact0/FM0/FS0/順序一致0。A→C: +2/−9/+2/−4。B→D: +1/−6/+1/−2。C→D: −1/+3/−1/+2。D−B−C+A: −1/+3/−1/+2。D対Aだけで要因効果を断定しません。
- 全96callはcompleted、Schema・exactly-once割当24/24/arm。A/Bのconfirmed22分割も全一致。全統合cellはA18/B18/C12/D15、全singletonはA0/B0/C5/D1（各22）。Goldの正しい全統合/単独コミットも含むため、偏り件数だけを誤りとしません。
- Gold R6/R7 Fallback4ファイル、目的分断7件。正しいFallback隔離はA0/8、B0/8、C1/8、D0/8（2順序のfile試行）。F05/F07は全条件Exact0、F06はC逆順のみExact。Goldで許容する目的分断はFSに加えていません。
- F01/F03/F09/F11は全条件・両順序Exact。F02はD逆順のみExact、C逆順/D通常はFS1。F04はC通常にFS1。F08はCだけ独立目的を分離。F10の独立coverage目標は全条件で統合されFM1。F11が正解でも、入力に見えないhost仕様を識別できた証明とはしません。
- A/CのGo ASTは名前・位置中心、Raw B/Dは差分本体を保持。生成前のGold/入力識別可能性監査を維持し、結果を見て再分類・除外・prompt調整していません。採点は事前に定義したfixture目標への一致で、作者意図の唯一性や実repository品質を立証しません。
- Input中央値 A3172.5/B1401/C3250.5/D1479tokens。wall p50/p95秒: A47.547/62.115、B41.384/54.663、C49.881/83.431、D46.896/57.551。cold loadを含むhelper第1段階の費用で、背景負荷もあり一般的因果効果とはしません。
- helper最大RSS A3.785/B3.802/C3.771/D3.923GiB、MLX peak全arm4.796GiB。RSSとMLX allocationを加算しません。ユーザー指示に従いシステム全体swapを停止/受理条件から除外し、raw contextとしてのみ保存しました。旧run1/run2の各1call停止は改変せず保持。全体98call、各run内retry0、run4なし。
- 旧と新の全request/Gold/prompt/evaluator/helper/model pin、12fixture元Git digest、temp index/object directoryでのbyte/tree一致、main HEAD/index/既存.gitignore diffの事後一致を確認。Production default・4file上限・#169/#170 NO-GOは維持。
- DのExact11/22（必要18）、A比+1（必要+4）、FS1（A0以下）、順序一致9/11（必要10）、必須境界F02/F08が未達。FM20≤A26、全実行・構造・プロセス資源・Git安全はPASS。
- 詳細: `docs/benchmarks/issue-171/run-3/report.md`、raw `measurements.jsonl`、固定採点 `evaluation.json`、cohort/case `cohort-and-case-metrics.json`、安全 `final-audit.json`。root report.mdも最新結果へ更新し、旧報告をrun-1-report.mdに保存。

## 考察

Raw Diffだけでは分割品質は改善しませんでした。暫定Bは既存入力で誤統合を減らしましたが、誤分割と順序依存が増えました。Dも誤統合26→20は改善した一方、FS0→1と順序一致11→9が悪化し、同時受理Gateに達していません。今回のNO-GOは意味品質による判断であり、swap停止やSchema失敗ではありません。

同じ契約を実現する実装/テスト/docsは一部で正しく統合できましたが、複数目的file・共有testのFallbackを安定して扱えていません。小規模合成・既知F01・multiple_valid実評価0件という制約を残し、16file約80%・別作者/実repository・Productionへ一般化しません。

## Next Steps

- このNO-GOで有限研究を終了し、全完了条件と報告目次を反映して最終repository test/vet/buildを実施する。GOを得るまで無期限調整せず、結果と品質証跡を保存するため。
- Production採用は進めない。Exact/FS/順序の同時Gate未達で、Cにも誤分割と順序不安定が残るため。
- 次の研究が必要なら、人間の新しい有限計画で同一目的統合とR6/R7 task契約・入力の根拠を検討する。現runでprompt/Goldを修正せず、観測した境界失敗を再現可能な条件にするため。
