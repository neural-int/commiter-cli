## 要約

Dの到達可能なA-only候補は**80%目標未達・現候補NO-GO**。固定公開履歴6workloadのend-to-end exactは2/6（33.3%）、automation coverageは5/6（83.3%）。16 selected filesはCU予算超過で停止し、exact0/1。H23は比較6観測exact0/6、現行4file baselineは同条件比較4観測exact4/4だった。production導入はNO-GOを維持する。

**Goal完了とはしない。** Cの独立mixed-file意味能力と、Dのnegative/mixed/shared-test/weak-edge等の独立分布が未評価で、#156の全Verificationはまだ成立していない。現候補のNO-GOとIssue全作業の完了を混同しない。

## 検証結果

|条件・層|exact|自動計画coverage|known CU FM / FS|備考|
|---|---:|---:|---:|---|
|A 1〜4file|2/4|4/4|0 / 27348|同一目的のcross-fileを分割|
|A 5〜8file|0/1|1/1|0 / 89500|8 singleton commits|
|A 9〜16file|0/1|0/1|未計測 / 未計測|unit_budgetで停止|
|現行baseline 1〜4file|4/4|4/4|0 / 0|2workload×正逆順|
|H23 1〜4file|0/2|1/2|0 / 19700（完成1観測）|一方metadata timeout|
|H23 5〜8file|0/2|0/2|未計測 / 未計測|両順invalid_metadata_value|
|H23 9〜16file|0/2|0/2|未計測 / 未計測|入力20070token、context上限超過|

- Aの完成5/5でauthoritative planning.Validate受理、byte再構築・正逆順temp-index replay・最終tree一致。欠落/重複・source mapping破損はない。停止ケースはGit計画を返していない。Aは全完成計画でfallback、単独file commit率100%、compose0、LLM calls/tokens0。
- 16file移行は未使用google/go-cmp b133f1f1の19 changed files中、辞書順16 selected files。全部interface{}→anyの同じalias移行で、対応例・testsにも同変更がある。元commit全体や16独立作者の評価ではない。1/4/8/16は相関するprojectionと明記している。
- 残りのD入力は未使用jmoiron/sqlxのSelect destination length reset（実装＋対応assertion、2file）とazuresql bind追加（1file）。Bのtoml入力はD primaryへ再利用していない。
- Aの16file拒否はcmp/compare_test.goのUTF8 edit atomsが256CU/file上限を超えたため。他の15fileは個別上限内。上限・gold・file選択を結果に応じて変更していない。
- 4file any移行と2file Selectの同条件4観測は、baseline exact4/4に対しA exact0/4相当。Aのcomplete file ownershipは意味的なsame-intent groupingの代替にならなかった。
- H23はe88f61bbのproposal-records、observer、metadataを変更せず薄いpublic-history adapterで再現。既定Gemma metadataとQwen3-8B global grouping、同じfixed helper、既存decoder/予算を保持した。
- H23 4file正順はmetadata timeout、逆順は合法4commitへFS。8fileはglobal assignment後のmetadata invalid、全体計画は不成立。draftのfile-pair FS6/28を完成計画のCU metricsへ水増ししない。
- H23 16fileは両順ともprompt20070tokenを記録しinternal_error。20070+1536は固定context16384を超えるため、記録とhelperのTokenBudget契約からcontext拒否と推定する。error_class自体は旧measuredBackendが保存していないため、内部errorの詳細を断定しない。
- D比較のbackend callは28（固定上限30内）。metadata packetが4group単位のため、group数によりcall数が増える。実行結果のcallsを単純に3phase×caseと推定しない。再試行0。
- H23の非completed応答は一部token telemetryがnull。JSONLのoutput_tokens集計0は観測された数値の和であり、未観測tokenが0という意味ではない。summaryで不完全件数と下限であることを明示する。
- A wallにはtemp-index正逆監査を含む。baseline比較総wall292.249秒、H23比較総wall692.288秒。runtime planner latencyへ読み替えない。RSS記録は子process集合の累積最大約4.91GBで、各call独立peakではない。
- mixed-file false absorb、同率bridgeの推移統合、架空evidenceを拒否する2回帰テストを追加し、研究用5テストPASS、差分整合PASS。全Issue Verification未達のため、goal.md指定の最終repository test/lint/build gateは未実施。
- 初回history adapter buildはrun関数signatureの引数数不一致で失敗。adapter側を固定sourceのsignatureへ修正してbuild成功した後、Dを事前登録している。失敗を測定成功へ算入していない。

## 考察

Aのfallbackは完成計画を増やしたが、複数fileにまたがる同一目的を回収できないためFSが残る。B/Cは意味GOを得ていないため、Dで無条件に組み込んでいない。80%の未達はA-onlyの固定小規模setについて測定された事実であり、adaptive architecture全般が不可能という結論ではない。

Dのprimaryはpositive中心で、negative/mixed/shared-test/unknown等を十分に独立評価していない。小規模で相関するcase集合の割合を一般ユーザー分布の推定値にしない。Cは既知oracleのみで、自動目的分割の改善を示していない。以上の不足を残したまま親Issueのチェックボックスをすべて埋めない。

資源失敗の改善仮説は残る。現在はfallback前に全fileをinline atomまで細分化するため、同一目的の単純な表記移行でもleaf budgetに到達する。要求されたFile Change Block→必要時だけCU splitという責務に合わせ、file-levelのlossless ownershipを先に確定し、refinement時だけbounded inline atomsを生成する案は、意味品質の閾値緩和とは異なる構造変更である。

## Next Steps

- **A2のcoarse-first ownershipを事前登録して検証する。** 16fileの単純な変更が不要なinline atom生成で停止したというsource監査を根拠にする。元の256leaf budgetを事後拡大せず、refinementの責務とfallback ownershipを分ける。
- A2は使用済みDを構造診断として扱い、再実行結果を新しい独立holdoutへ再分類しない。gold不定のupstream同期を都合のよいsingle goldにしないため。
- Cの独立mixed-file分類・CU帰属を評価する入力と、Dのnegative/shared-test/weak-edge/orderingケースを推論前に別途固定する。現在のpositive中心のDだけでは全条件を完了できないため。
- production実装・4file上限変更は引き続きNO-GO。現時点ではimplementation Issue/PRを作らず、残余能力を研究trackで解決する。#145の解決として扱わない。
