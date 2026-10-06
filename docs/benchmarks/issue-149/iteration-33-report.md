## 要約

H23はhost complete partitionsをsoft proposalとして保持し、最終partitionを自由global assignmentへ戻した。固定Qwen3-8Bのweak16/cross12/guardrail全3caseでexact/FM0/FS0/complete。資格通過だがproduction candidate選定は未完了。

## 検証結果

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 |
|---|---|---:|---:|---|---|---:|---:|---:|---:|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 1 | 3080 | 210 | 32.944 |
| contract-cross-boundary-12 | true | 0 | 0 | true | false | 1 | 5268 | 158 | 46.393 |
| shared-callee-independent-6 | true | 0 | 0 | true | false | 1 | 2399 | 80 | 20.094 |

各case1call/1回、全backend completed、timeout/context overflow/incomplete outputなし。全file一意、unknown/duplicate/missing ID0。cross12は2group、guardrailは3implementation/test pair。observer cross12 4/16/unknown0、guardrail2/8/0、weak0。候補規則/最終schema/元観測は固定し、candidate ID選択後のfallbackなし。

fixed Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192、native0/output1536/context16K/call120秒/whole600秒/temp0/top_p1/top_k0/seed144/retry0/repair0。コード変更は既存H17へのoptional host proposals追加とarchitecture dispatch、production変更なし。候補にないpartitionも既存G-ID schema/partition gateで許し、候補を正解と扱わない。追加取得/依存なし。新独立評価/range/order/metadata未実行。

focused partition/ID/canonical/selection tests、benchmark vet/build、diff check成功。`go test ./tools/benchmark149`も42.621秒で成功。

## 考察

H17はweak16/guardrail一致だがcross12 FS30、H19はcross12/guardrail一致だがweak16未解決。H23は今回3caseの両立を観測した。これは有限の既使用qualificationであり、任意入力品質やproposalが単独原因であること、内部推論理由を証明しない。candidate generatorは10/14包含だったがH23では最終boundaryではない。ただし候補欠落caseで新partitionを実際に生成できるかは未確認。

非意味的なcandidate提示順、file順、既存4file最終plan、他failure pattern、metadata/Validate、cost実用性は未達。ここでproductionを実装/default変更したり、Issue完了チェックを埋めたりしない。方式を固定して独立評価へ進む根拠が得られた段階。

## Next Steps

- 現H23 source/task/schema/generator/route/budgetを固定し、新しい独立fixtureを推論前に定義・gold固定・before/after Go検証して1回測定する。protocol8等は選定に使用済みのためfreshとせず、既使用qualificationへの適合だけで採用しないため。
- 独立fixtureは既存expiry/cents/constant-limitとは異なるbehaviorと、candidate familyに正解がなくても自由boundaryを生成する必要のある構成を含める。goldはdiff/code観測に根拠付け、モデル結果から変更しない。自由partition契約の実効性を確認するため。
- 独立評価通過時は既存4file/5〜8/9〜16/controlled弱欠落relation/impltests/regressionへ各必要最小回数、file反転とcandidate提示順反転を事前登録して測る。scopeの全主要failureと非意味的安定性を確認するため。
- 全grouping資格通過後のみmetadata/authoritative Validateとbaseline同条件比較、責務/dataflow/failure/budget/child Issue判断へ進む。新評価の失敗は保存して棄却し、wording/budget/候補ルールで取り直さない。
