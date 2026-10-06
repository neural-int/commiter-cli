## 要約

H23を固定し、新独立8fileのwire contract fixtureを推論前に定義・gold固定・before/after検証して1回測定。exact/FM0/FS0/complete、host候補に存在しない正解partitionを生成した。production candidate選定はrange/order/最終planが未達。

## 検証結果

| fixture | files | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 |
|---|---:|---|---:|---:|---|---|---:|---:|---:|---:|
| holdout-event-field-and-duration-unit-8 | 8 | true | 0 | 0 | true | false | 1 | 3218 | 106 | 27.052 |

新fixtureはeventsのJSON trace_id→correlation_id producer/consumer/test各4filesと、独立metrics elapsed_us→elapsed_ms producer/consumer/test各4files。gold2groupsはモデル推論前に定義。source SHAをfixture-pin.jsonへ記録。before/after Go tests各成功（1.76/1.71秒）、LLM0候補診断は3候補・正解包含false。generatorを変更せず推論した。model入力外のgold/nameから候補を追加していない。

最終groupはevents4filesとmetrics4files。hostのall-files/4pair/8singletonのどれにも一致しないことをassignment-check.jsonで確認。全IDちょうど1回、unknown/duplicate/missing0。backend completed、timeout/context overflow/incomplete0。source caller/callee observer probe0（選択source相互呼出なし）。有限observerはJSON APIを実行しない。stdlib JSONプログラムを実行したのは固定synthetic before/after testsのみ。

固定H23 architecture commit b1b703d、Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192/native0/output1536/context16K/call120秒/whole600秒/temp0/top_p1/top_k0/seed144/retry0/repair0。新source/testとfixture dispatchだけ追加、architecture/task/schema/generator変更なし。追加取得/依存/production変更なし。

`go test ./tools/benchmark149`（45.836秒）、benchmark vet/build、diff check成功。推論とtestsは順次。raw prompt/思考を保存せず内部理由を推測しない。

## 考察

資格3caseに加え、異なるbehaviorの独立評価で候補外の正解partitionを観測した。optional proposalは今回のcaseでは不可逆boundaryにならなかった。この1fixtureを未知入力全体の品質保証とせず、候補内容に依存しない性能や非意味的順序安定性は別途確認する。fixtureは今後使用済みと扱い、tuning後のfreshへ戻さない。

production選定の残りは4file baseline/controlled5〜16/weakmissing/impltests/regression、file/candidate提示順、metadata/Validateとcost/責務/dataflow/child Issue。現時点でcheckboxやproduction defaultを変更しない。

## Next Steps

- H23を固定して残る既存contract/controlled/holdout/protocol fixtureのrange/regressionを各1回、順次実行する。資格3caseや今回holdoutを不必要に再実行せず、全主要failureと候補欠落caseを確認するため。
- file反転とhost候補提示順反転のdiagnosticをsource/task/schema/generator集合固定で事前登録する。良い順を選んで採用せず、非意味的依存が残れば棄却するため。
- grouping全評価通過後にmetadata/Validateと同条件4file baseline比較を実施し、責務/dataflow/failure/budget/実装範囲/child Issueを文書化する。有限grouping成功をproduction接続の完成と扱わないため。
- 失敗した評価は保存してH23棄却根拠へ追加。同案のwording/budget/候補規則を調整して独立評価を取り直さない。
