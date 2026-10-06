## 要約

固定H19/H20 generatorを14fixtureでLLM0診断。正解包含10/14、baseline4/implementation8/boundary12/protocol8で欠落。selectorが完全でもこのgeneratorは採用条件を満たせず、提示順推論は行わない。

## 検証結果

| fixture | files | 候補数 | gold包含 |
|---|---:|---:|---|
| contract-baseline-4 | 4 | 3 | true |
| contract-independent-6 | 6 | 3 | true |
| contract-cross-boundary-12 | 12 | 4 | true |
| baseline-4 | 4 | 2 | false |
| same-directory-independent-5 | 5 | 2 | true |
| implementation-tests-8 | 8 | 2 | false |
| cross-directory-single-intent-9 | 9 | 2 | true |
| multiple-intents-boundary-12 | 12 | 2 | false |
| missing-edges-single-intent-16 | 16 | 2 | true |
| weak-edges-independent-16 | 16 | 2 | true |
| holdout-independent-6 | 6 | 3 | true |
| holdout-independent-16 | 16 | 3 | true |
| shared-callee-independent-6 | 6 | 3 | true |
| holdout-protocol-and-health-8 | 8 | 3 | false |

全14件backend calls0、generator/code規則/gold固定、候補手動追加なし。`go test ./tools/benchmark149 -run TestH19FullCoverageDiagnostic -v`成功（0.675秒）。このtestのpassは測定処理完了であり、包含14/14成功ではない。`git diff --check`成功。追加取得/依存/production変更なし。

protocol8はmodel未推論のままだが、今回generator選定の診断に使用したため、今後candidate generatorの未使用adoption holdoutとして扱わない。新しいcandidateが成立した場合は別の独立評価を事前固定する必要がある。

## 考察

3資格caseの包含成功から一般的な候補生成成立へ拡張できない。候補欠落を順位変更/selector改善で修復できないため、今回提示順診断はskip。Gemmaの先頭all-files選択を位置バイアスと断定しない。graph weight/lexical規則を欠落fixtureへ合わせる反復は#142の既存探索と重なるため行わない。

今回のcandidate selectionはcross12に改善観測があるが、候補制限とselector未解決/誤結合の両方が残る。別の責務境界として、有限候補を先に固定せず、全fileの観測を同時に読んだglobal relation judgmentからhostがpartitionを構成する方式は未測定。#146のlocal pair/window/all-pairsは異なるcontextでの矛盾と整合した誤りを示した。次案も誤判断を整合性検証で正解に変えられない点は同じで、品質を独立測定する必要がある。

## Next Steps

- H21として、元contract/test/contrast全件を同一global contextに保持し、全file pairへsame/different/unresolvedを一度に判断させる責務分離の実行可能性を事前監査する。候補集合の欠落を避け、global目的統合と個別境界判断のtaskを分けるため。#146のlocal bounded-window reconciliationを再実装しない。
- hostは全pairちょうど1回/unknown/duplicate/missing拒否、sameの推移closureとdifferent矛盾を拒否し、unresolved/timeout/invalidではpartitionなし。soft syntaxをmust-linkへ変えず、majority/recoveryしない。構造整合をsemantic正解と扱わないため。
- 最大16files=120pairのoutput schema/token/120秒call budgetと利用可能helperを先に確認する。実行可能なら小さいprototypeでGemma native0のweak16/cross12/guardrail各1回、失敗なら予算sweepせずcapability/再開prerequisiteを監査する。単一global contextによるarchitecture差の検証とboundednessを両立するため。
