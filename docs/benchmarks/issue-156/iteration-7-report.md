## 要約

独立Dの到達可能候補A2は**exact4/12（33.3%）、9〜16fileは1/3（33.3%）で80%未達、production導入NO-GO**。安全な完全自動計画のcoverageは15/15（100%）だった。比較4観測の現行baselineはexact2/4、H23は比較4観測exact0/4。A2のfallback成立を意味正解や4file品質回帰なしへ読み替えない。

## 検証結果

|A2 primary層|exact|計画coverage|主なfailure|
|---|---:|---:|---|
|1〜4file|2/7|7/7|source+testのFS、same-line/shared-testのFM|
|5〜8file|1/2|2/2|8file mechanical normalizationのFS|
|9〜16file|1/3|3/3|public16複数目的のFS、16file/8purposeのFS|
|全primary|4/12|12/12|80%未達|

- Cに未使用のstretchr/testify公開履歴と、新規controlled入力を先に固定した。15件中3件（20%）はopaque unknown、nonUTF8 unknown、interleaved hypothetical purposeで、sourceからgold境界を観測できず事前別集計。exact/FM/FSをnullとし、3/3のunknown継続と構造安全だけを確認した。
- 公開履歴由来の7 primaryはexact2/7（28.6%）、authored control5 primaryは2/5（40%）。controlledの目的ラベルはfixture作者の契約であり、同じ値の一括変更等にはsourceだけで目的を一意推定できない場合がある。公式12件の分母は変更せず、この分離結果も記録する。いずれも80%未達。
- public16は9fileのsource正規化、mock2file、assertion2file、README1file、CI1file、require godoc selected1fileのdisjoint-history superposition。自然な同時差分や16独立作者とは呼ばない。8file projection・小入力とcomponentが相関するため、母集団confidenceは主張しない。
- A2全15/15でcomplete byte ownership・zero missing/duplicate・reconstruction・authoritative planning.Validate・正逆temp-index replay・最終tree一致。全15件でinput file orderの逆転後もfinal plan hashが同じだった。元index/worktreeのGit mutationはない。
- 16fileはpublic workload254行operationと、authored各16operationを完全保持。各16commitの正逆監査は約49〜50秒。stage監査を除いたruntime latencyではない。1MiB/20kline/256operation per file、4096total、16file上限を変えていない。
- A2 line-operation粒度10 primaryでFM0/FS12775、mixed2件のUTF8 atom粒度でFM9/FS2。異なる粒度のpairを同じ数値へ合算しない。FMケース2/12、FSケース7/12。fallback率・file-singleton commit率は100%、compose0、LLM calls/tokens0。
- Exact purpose clusterの回収はA2で27/47（57.4%）、precision27/76（35.5%）。「goldの目的全体とpredicted cluster全体が同じ」という定義であり、exact case率の代わりではない。source/test fixtureの対応値はcontrolled契約であり、公開projectのtestを実行した結果ではない。
- 同じmock2fileとmock+assert4fileの正逆4観測では、A2 exact0/4に対し現行baseline2/4。完成2件はFM0/FS0、残る2件は未完成のためpair metrics null。正順negativeは3backend全てcompletedでも最終baseline_failed、逆順はtext timeoutを記録。正順の詳細invalid原因は旧adapterが保存していないため断定しない。raw groupingのexactを最終exactへ昇格しない。
- H23は4file negative両順がmetadata_timeout、8fileがhost context_overflow、16fileがglobal grouping timeout。全4件未完成なのでFM/FSは未計測。H23をFM0/FS0とは報告しない。旧source/observer/metadata/model/profileを変更せず比較した。
- D比較の観測backend callsは20（baseline12/H23 8）、固定上限64内、再試行0。baseline総wall388.700秒、H23 479.195秒、比較予算1200秒内。A2監査総wall179.934秒、mean11.996秒、95p49.816秒（lower empirical order statistic）。process peak RSSは累積最大約4.89GBで、call単位peakではない。
- token telemetryが不完全なのはbaseline1/H23 3観測。観測token下限はbaseline input38108/output4074、H23 input26938/output1084。H23 16fileのtokenは未観測で、lower bound0を実際の0tokenと扱わない。
- 初回summaryの完成0件のH23 pair sumは空集合の和0だった。rawと初回summaryを保持し、summary-clarifiedでFM/FSをnullとして明示した。測定・gold・停止結果は変更していない。報告はclarifiedを使用する。
- 補助source formatter監査では9file中7fileで標準gofmtのbefore出力が公開afterとbyte-exact。残る2fileはbuild directive直前の空コメント行1行の差だけ。現在のformatterで全9一致とは主張しない。外部Goコードやtestは実行せず、gold再分類・case除外も行っていない。

## 考察

A2は不要なinline atom生成による拒否を解消し、unknownを合法な計画へ収束させた。しかし同一fileの混合目的を保持したFMと、cross-file同一目的を回収できないFSが残った。自動化coverage100%は80%exactという目標の代替ではない。

Cのraw partial splitは現行validatorと互換性がなく、Bは小規模独立評価で順序依存があったためDへ統合していない。best reachable subarchitectureはA2の構造fallbackだが、意味的production候補としては採用しない。現行baselineにもnegative completion失敗があり、過去や単発positiveの成功だけで全入力の実用性を保証できない。

production実装、既定モデル、4file上限の変更はNO-GO。現時点でimplementation Issue/PRを作らないと決定する。将来の導入には、安定した目的帰属と、CUをfile IDへ偽装しないownership schema・authoritative validation・partial-stage safety・compose/type/releaseの一貫した契約を別の設計/実装Issueで確立する必要がある。#145の拡張課題が解決したとは扱わない。

## Next Steps

- **事前固定したB大規模transfer監査を実行する。** 子Issueの5〜8/9〜16境界を補うため、固定score3/margin1/full-member coverage/neutral decoderを維持し、6/8/16fileを正逆で評価する。Dの採否は変更しない。
- その結果を含めA〜Dの成果物/失敗契約を最終監査し、成立した完了条件のチェックを更新する。NO-GOを成功GOへ書き換えず、検証作業の完了とproduction採否を区別する。
- 全Issue Verificationとcheckboxが成立した後、goal.md指定の最終test/vet/buildと既存release-notes test、研究回帰・差分整合を実行する。現在は最終ゲート未実施。
