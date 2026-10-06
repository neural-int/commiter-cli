## 要約

H22としてglobal pair taskを固定Qwen3-8Bで評価。weak16/cross12は120秒timeout、guardrailは未解決pairで停止。全caseでpartitionなし、採用資格不成立。

## 検証結果

| fixture | exact | FM | FS | complete | unresolved | stop / host結果 | input tokens | output tokens | total wall秒 |
|---|---|---|---|---|---|---|---:|---:|---:|
| weak-edges-independent-16 | 未評価 | 未評価 | 未評価 | false | true | timeout / backend_failure | None | None | 120.032 |
| contract-cross-boundary-12 | 未評価 | 未評価 | 未評価 | false | true | timeout / backend_failure | None | None | 120.029 |
| shared-callee-independent-6 | 未評価 | 未評価 | 未評価 | false | true | completed / unresolved_pair | 2326 | 152 | 24.619 |

各case1call/1回。timeout2、context overflow観測0、guardrail backend completed。timeoutのtokens/model response metadataは取得できずnull（0扱いなし）、routeは固定Qwen3-8Bで起動。timeout/未解決とも部分partitionなし。再試行/予算増額なし。semantic exact/FM/FSは全て未評価であり意味誤判定が証明されたとは扱わない。

fixed Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192、native0/output1536/context16K/call120秒/whole600秒/temp0/top_p1/top_k0/seed144/retry0/repair0。input/schema/host gatesはH21と同じ。追加取得/依存/code/production変更なし、推論順次でtests併走なし。新独立評価/metadata未実行。artifact diff check成功。

## 考察

H21 Gemmaはweak16を予算内生成しても誤結合、H22 Qwen8はweak16/cross12が既存call時間に収まらず、guardrailも未解決。global pair責務への変更は現条件で品質・費用・完了の両立を満たさない。timeoutは意味品質を証明しないが、bounded production candidateの資格を失う根拠になる。予算を拡大して再試行しても現時点の採用根拠にはできず、今回同taskのbudget sweepは行わない。

H1〜H22で観測情報、IR抽出、entity/purpose出力、候補生成/比較、global pair境界、generation contractと保存済みrouteを試した。どのfamilyも明示した主要failureの同時解消を立証していない。一方、これを任意architecture/modelの普遍的不可能性へ拡張しない。探索終了判断には残余仮説と現在capabilityを結びつけた監査が必要。

## Next Steps

- H19〜H22とcandidate包含14件診断を既存family監査へ追加し、Issueのno-candidate分岐を満たす根拠を監査する。有限の試験済み候補不成立と、合理的追加探索消尽/現在capability検証不能を区別するため。
- 未試験案は単に名前や組合せを挙げるのではなく、既存failureへ作用する新情報/責務差、実行可能なbounded検証、既存Issueで否定されていない根拠を要求する。根拠ある案が残る場合は事前登録して継続、残らない場合は現在条件でのno-candidate判断と具体的再開prerequisiteを記録する。
- 各完了条件の証拠を対応付けてlive Issueと照合する。全Verificationを実証できる場合だけcheckboxを更新し最終test/lint/buildへ進む。未達をN/Aや進捗件数だけで完了へ変換しない。
