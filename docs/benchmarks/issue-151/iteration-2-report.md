## 要約

Aは **GO（限定したline/whole-operation representationの次段階評価）** と判定する。同一fileの離れたline変更と隣接line変更をgoldなしに分割し、個別stageと全変更byte復元を確認した。既存#149の意味判断の曖昧さは解消したとは判定しない。

## 検証結果

| 対象 | cases | files | units | Git stage / reconstruction |
|---|---:|---:|---:|---|
| #149回帰 | 14 | 128 | 128 | 全成功 |
| synthetic境界 | 10 | 10 | 13 | 全成功 |
| 合計 | 24 | 138 | 141 | 全成功 |

- adjacent-editは2unitsへ変化、forward/reverse順で各中間index blobと最終after byte一致を確認した。
- 複数line insert/deleteの個別stage、binary NUL、Unicode/repeated lines、未知/重複IDs、stale snapshot、overlap、byte/unit budget停止を6 testsで確認。
- create/rename/mode/symlink/deleteをtemporary Git indexで確認するwhole-operation testも成功。partial contentとmetadataの統合plannerは未実装。
- 純extraction secondsとserialized representation bytesはiteration-2-results.jsonに保存。Go AST parser起動費用とGit操作費用はwall側に含み、純extraction時間から分離した。
- 既存回帰では各file 1unit、syntheticでは最大2units。実大規模repositoryでのfragmentation実測はなく、per file 256units/1MiB/20,000linesの停止契約を採用。
- same-lineは1unitのまま。Gitのline単位selectionを超える同一line内複数intentはこのrepresentationでは表現できない。
- LLM calls/tokensは0、semantic predictionなし、exact/FM/FS/unresolvedはN/A。coverage成功やgold表現可能性をsemantic inference成功と混同しない。

## 考察

Aのgateにある「構造的制約の解消」は、同一fileを独立line editsへ割り当てる反例で確認できた。抽出器はgold、fixture path/name、semantic scoreを使わず、source spansとbytesで再構築する。file単位policyに適合するproductionへの導入判断ではなく、次の研究段階の基盤としてGOとする。

達成条件の対応: representation/source mapping/staging contractはdesign.md、changed-byte保存とdeterminism/reconstructionは24ケース、1file複数intentは離れた/隣接line synthetic、cross-file intentと#149回帰は14fixtures、gold非依存は抽出器引数と固定algorithmで確認。同一line限界、operation order、bounds、実production未統合を残す。

## Next Steps

- #152専用の新規worktreeでbounded evidence layerを評価する。Aだけでは意味判断の改善を示せないため。
- #149回帰に加えて新independent fixtureを事前固定し、local-only / symbol-call / historyの各source効果を別々に測定する。gold leakageとsource混在による過大評価を防ぐため。
- Aの達成条件を確認したうえで最終品質ゲートを実行し、結果とdeliveryを記録する。prototype品質とsemantic adoptionを区別するため。
