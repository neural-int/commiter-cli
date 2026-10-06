## 要約

H17としてpurpose-records/native0を取得済みQwen3-8Bで評価。weak16と独立callee6は一致したが、cross12は全fileを別groupへ分割しFS30。production candidateには採用しない。

## 検証結果

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 |
|---|---|---:|---:|---|---|---:|---:|---:|---:|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 1 | 2852 | 210 | 30.463 |
| contract-cross-boundary-12 | false | 0 | 30 | true | false | 1 | 4938 | 158 | 43.364 |
| shared-callee-independent-6 | true | 0 | 0 | true | false | 1 | 2230 | 80 | 18.774 |

各case1回、全call completed、timeout/context overflow/incomplete outputなし。cross12は12 singleton、guardrailは3組の実装/test pair。observerはcross12 4probes/16samples、guardrail2/8、unknown0。weak16は0。固定Qwen3-8B-4bit revision545dc4251c05440727734bcd94334791f6ab0192、bounded-routed-grouping、native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。追加取得/依存/production/code変更なし。fresh protocol8/metadataは未実行。

## 考察

Gemma native0の同taskは3case全て一括groupだった。Qwen8は独立変更を分離したが、cross12では同じ変更目的に属する実装とtestの関係も失った。出力identity/task変更とcheckpoint変更だけで資格を満たさず、これらを予算や言い回しで調整し続ける根拠はない。

残余を棚卸しすると、raw/batched/per-file semantic IR、host declarations/calls/tests、canonical evidence、契約record、per-record before/after意味抽出、有限snapshot/contrast、E-root/G-ID、bounded generation、grouping-only routeを試した。H11は各entityのbefore/after抽出であり、複数entityを跨ぐ共通変更目的のprovisional discoveryは行っていない。

#140の[分解検証](https://github.com/neural-int/commiter-cli/issues/140#issuecomment-5853482243)はPhase A 6件中5件が1024/2048でも未完了、完了1件も重複。既存raw intent discoveryを再実装する根拠はない。ただし現在のhost保持contract record、上限付き短いprovisional purpose、元観測を全件保持した後段assignmentの組合せは未検証。これは成功の保証ではなく、生成可能性と責務分離を独立測定できる残余仮説。有限の失敗から探索余地消尽を結論しない。

## Next Steps

- H18として、host contract record全体から複数entityを跨ぐprovisional purposeを短いbounded schemaで抽出し、その後元観測全件と候補を用いてglobal assignmentする責務分離を事前登録する。直接assignmentが意味統合とfile割当を同時に負う点を切り分けるため。
- #140の未完了/重複を再発させない契約を先に定義する。候補はfinal boundaryにせず、後段で追加/split/merge可能なG-IDを維持。観測E-ID参照のunknown/duplicate、候補数/文字長/出力budget、missing outputをhostで拒否し、抽出失敗を回復推測しない。goldで候補を固定しないため。
- 小さいprototypeで安全gateを確認後、weak16/cross12/guardrailを各1回測定。全資格通過時のみfresh protocol8へ進む。失敗時はdiscoveryとassignmentの失敗を分離して、既存結果を踏まえ次のarchitecture根拠または再開prerequisiteを監査する。
