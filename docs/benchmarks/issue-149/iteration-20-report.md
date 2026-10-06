## 要約

H13として観測4値のcontrast導出をhostへ移した。weak16/guardrailは一致したがcross12はFS24のままで、Gemmaのglobal判断は改善しなかった。比較情報を増やすだけではproduction条件を満たさない。

## 検証結果

元の4snapshot値を維持し、同じ有限入力におけるno_effect/joint_only_effect/caller_only_effect/callee_only_effect/other_combination_effect/unknownを追加。分類からunionや最終boundaryを作らず、gold/fixture名を入力しない。system prompt/schema/canonicalization/host gate/modelはH12から維持。

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 | observer probes/samples/unknown |
|---|---|---:|---:|---|---|---:|---:|---:|---:|---|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 1 | 2894 | 192 | 15.199 | 0/0/0 |
| contract-cross-boundary-12 | false | 0 | 24 | true | false | 1 | 5440 | 176 | 19.530 | 4/16/0 |
| shared-callee-independent-6 | true | 0 | 0 | true | false | 1 | 2482 | 92 | 10.387 | 2/8/0 |

各1回、全call completed。timeout/context overflow/不完全出力なし。固定Gemma 475b9088d29754a3379866cf5aeb6b41acd313c2、native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。observer上限は前iterationと同じ。任意codeの実行0。fresh protocol8/metadataの追加測定は未実行。

focused testsは原始値oracle一致、contrastの分類、unknown/重複/欠落を効果へ変換しないこと、キャンセル停止を確認して成功。benchmark build成功、production変更なし。

## 考察

cross12 input5383→5440、guardrail2454→2482でcontrastが加わったが、partitionとFM/FSは変わらなかった。Gemmaでは有限比較の負担をhostへ移すだけでsemantic統合を改善する根拠を得なかった。弱いrelationの扱いとcomplete assignmentは成功しても、cross-boundary同一contractの統合を満たしていない。

Iteration14のQwen3-8B試験はH9のcode/call/test入力でcross12 FS30だった。その時点には4snapshot値とhost導出contrastがなかった。新しい観測情報を持つ同architectureでcheckpointを比較する試験は未実施であり、以前の結果だけからこの構成も棄却することはできない。一方、8Bというparameter数自体を改善根拠にはしない。

## Next Steps

- H14としてH13の観測入力とhost gateを固定し、取得済みQwen3-8B-4bitの同revisionへgrouping-onlyでrouteする。raw/codeのH9とは異なる相互作用・contrast情報を消費できるかを、architectureとcheckpointの組合せで検証するため。
- weak16/cross12/shared-callee guardrailを各1回、native0/output1536と既存全budget/seedを維持して順次測定する。追加取得/dependencyやprompt tuningなしで、semantic改善と費用を独立評価するため。
- 全資格通過時だけ未測定protocol8へ進み、失敗なら経路/予算を反復調整せず残る責務・task境界または現capabilityの限界を監査する。有限の失敗を普遍的不可能性や探索完了と混同しないため。
