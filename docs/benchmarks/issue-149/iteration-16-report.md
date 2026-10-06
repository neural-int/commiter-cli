## 要約

H11として根拠E-ID付きbehavior抽出とglobal assignmentを分離した。weak16は一致したがcross12はFS24で、H9/H10と同じ6pairのpartition。semantic改善を得ず費用が増えたため採用しない。

## 検証結果

観測contract/test/callerを最大4recordずつ意味抽出し、grouping権限を持たないIRを生成した。元の観測recordと全fileをglobal入力へ残した。E-IDの未知/重複/欠落、required status欠落、unresolved、空text/160文字超を拒否。抽出でglobal boundaryを固定しない。

| fixture | exact | FM | FS | complete | unresolved | calls | total input tokens | total output tokens | total wall秒 |
|---|---|---:|---:|---|---|---:|---:|---:|---:|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 5 | 6019 | 1078 | 61.735 |
| contract-cross-boundary-12 | false | 0 | 24 | true | false | 4 | 9754 | 1223 | 71.227 |

weak16は抽出4call+global1call、cross12は抽出3call+global1call。全call completedで、timeout/context overflow/不完全出力を観測しなかった。phase別tokens/wallはJSONLに記録。固定Gemma 475b9088d29754a3379866cf5aeb6b41acd313c2、native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。各fixture1回、同時推論なし。

focused anchor/record conservation tests、欠落・未知・unresolved IRの拒否テスト、benchmark build成功。fresh protocol8、metadata/final Validateの追加試験は未実行。production codeの変更なし。

## 考察

H10の1callと比較し、weak16は14.314→61.735秒、cross12は18.304→71.227秒。partitionとFM/FSは同じで、追加抽出費用を正当化する改善はなかった。hostが根拠ID/形状を確認できても、意味抽出内容の正しさや最終semantic groupingを証明できない。

ここまでのsyntax/call/test観測は、calleeとcallerを個別に変更した場合の出力を含まない。cross12のAuthorized/Restoreは旧callerにdeadline equalityのoverrideがあり、calleeだけ変更してもそれが残る。FormattedCents/PersistedCentsは旧callerに入力補正が残る。一方shared-callee guardrailには同じcalleeを使いながら別のprefix/bracket変更がある。実行による組合せ観測はこれらを区別する新しい情報となる可能性があるが、未測定である。依存やテスト成功だけをfinal commit boundaryの証明とは扱わない。

## Next Steps

- H12の前提検証としてsynthetic executable fixtureだけを対象に、callee/callerのbefore/afterを組み合わせた4通りの観測test呼出し結果を記録する。既存IRにない変更間の実行上の相互作用を観測できるか確認するため。
- cross12とshared-callee independent guardrailで、未知/実行失敗と観測結果を分離する。共通calleeだけを根拠にhard unionする誤りを避け、新情報の限界を確認するため。
- 観測結果に追加情報がある場合のみ、runtime evidenceをsoft入力として使うglobal architectureを事前登録して資格試験へ進む。実行結果をgoldの代理や強制groupingにせず、calls/context/latency budgetと未測定holdoutを維持するため。実コードの任意実行やproduction統合には広げない。
