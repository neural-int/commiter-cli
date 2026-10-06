# Issue149 architecture探索の判断

## 判断と適用範囲

現在の試験済みlocal checkpoint/helper/generation budgetと、diff/code観測だけを根拠にする契約では、production candidateを選定しない。現行productionの4file上限を維持する。これは検証したarchitecture familyの測定と、下記の合理的残余監査に基づくDesign/Verification上の見送り判断であり、全model/任意architecture/将来の不可能性の主張ではない。

#143/#139〜#142/#146、H1〜H23の個別改善と棄却根拠はIteration26/32監査およびIteration1〜35 report/JSONLに保持。gold/品質条件/安全gateを変更して採用しない。特にH23の改善を消去しない。

## 最後の有望案

H23: 全観測contract/test/calls/finite contrast + host complete partition proposals → one global free G-ID assignment → exactly-once/invalid/unresolved gate → finalized grouping後metadata → authoritative planning.Validate。

候補はsoft proposalsで、新partitionを許す。qualification3caseと初回独立wire8fileは全一致。新wire評価では候補集合にない正解partitionを生成した。一方、残るrange11caseは7一致/4不一致。H23合計15observationsは11exact、15complete、FM10/FS74。same-directory5でFM10、cross-directory9でFS32、boundary12でFS30、protocol8でFS12。全call完成してもsemantic条件を満たさない。range全通過の前提を満たさず、order/metadata追加測定は採用資格を回復できないため実行しない。

## 共通して残るbottleneck

- 異なるentity/behaviorを独立に分けることと、複数entityの一つのchanged contractをまとめることが同時に安定しない。
- 完全割当/JSON/根拠参照/推移整合が成立しても誤ったpartitionは成立する。hostの構造gateは意味正しさのauthorityにならない。
- 固定候補は欠落を持ち、候補外生成を許すと欠落は避けられてもsemantic品質は未達。global pair化は誤り/未解決/timeoutを解消しない。
- test-pass/有限snapshotは観測事実で、commit目的や全inputの証明ではない。soft call/source-test/proximityをhard boundaryへ変えられない。

## 合理的追加hypothesisの監査

これは未試験案が物理的にゼロという証明ではなく、現在のevidence/capability/scopeで追加実験を正当化できる作用根拠が残るかの判断。

| 残余案 | 現在のevidenceとの関係 | 現時点の判断 |
|---|---|---|
| raw/batch/per-file/contract/purpose IRの別encoding | H1〜H18で元観測保持と意味抽出/出力identityを分離しても同failure。新しい独立観測を追加する根拠なし | wording/formatの探索へ戻らない |
| local verification/correction/多数決 | #141/#146のcontext矛盾と整合した誤り。信頼できるsemantic authorityを追加しない再callではcorrectnessを保証せず、推測recoveryは対象外 | 同一手法の再実装をしない |
| graph weights/lexical/embedding/signed clustering | #142とH19の候補欠落/selector誤り。H21/H22は信頼できるglobal signed判断を供給せず、未校正similarityを独立目的境界へ変える根拠なし | 新しい検証済みsemantic relationが前提。既使用goldへのtuningをしない |
| optional proposals + free output | H23で独立評価成功を確認した上で全rangeに4counterexamples | この案を棄却。新規失敗へcandidate規則を追加しない |
| checkpoint/task/予算総当たり | H7/H9/H14/H16/H17/H19〜H22でroute/task/generation比較。未測定組合せというだけでは作用根拠にならない。native拡大も独立した改善証拠なし | arbitrary Cartesian sweepはしない |
| feature/count別routing | 同じconstant-only観測classでweak16成功/same-dir5失敗。file数やfixture名で成功routeを選ぶと観測された失敗へ適合する | 新独立根拠なしのrouterは作らない |
| full semantic analysis/runtime拡大 | controlled constant5のover-mergeと同named contract9のsplitはscalar観測が既に利用可能。loop/外部API実行追加だけでpurpose識別改善する根拠なし。任意repo実行は新安全契約が必要 | 限定observerの対応域を無根拠に拡げない |
| new model/finetuning | 現在未準備のsemantic capabilityを追加する必要がある。単なるmodel名/size増加は独立改善根拠ではない | 下記prerequisiteを満たす将来の再開条件 |
| 外部author intent/正解変更/hard relation | userのdiff/code-only、gold維持、Issue対象外に抵触 | 採用条件を緩和しない |

初案だけで終了せず、上位の観測representation、責務分解、自由assignment、候補比較、候補外生成、同一global pair boundary、generation/model routingを、個別failureに作用する根拠のある範囲で検証した。最後に残したH23もfull rangeで反証された。以上から現在の準備済みcapabilityで、既存否定案/局所tuning/条件緩和へ戻らず追加検証する根拠を持つhypothesisは選定できない。この工学的判断を合理的探索の現在の停止点とする。将来の可能性を排除する証明とは区別する。

## 再開prerequisite

1. 新しいlocal semantic engine/checkpoint/学習済みrelation extractorについて、独立したsmall independent-vs-shared-contract capabilityの改善根拠を先に得る。sizeだけで選ばず、取得revision/runtime/容量/依存/許可とbounded出力を確認する。
2. relation/embedding方式なら、異なるentityの独立性と同contractのcross-directory統合を同時に識別できるpositive/negative判断を独立データで評価し、unknown/contradictionのfail-closedを維持する。
3. 新しい情報または責務差で上記counterexamplesへ作用する因果仮説を事前登録。既使用15case/旧holdoutでtuningした後の結果をfresh評価と扱わず、新未使用観測goldを固定する。
4. 5〜8/9〜16、same-dir/weakmissing/impltests/crossdir/guardrail/順序/最終metadata/Validate/費用の全条件を再評価する。production defaultやGit mutation契約の変更判断は分離する。

## production責務と実装範囲

選定candidateなし。production implementation/child Issueは作成しない。candidate-found条件は条件未発動として記録し、実装したかのように扱わない。benchmark/観測/棄却根拠だけを保存。current4fileとH5の同条件最終plan比較はIteration10に保持（双方exact/Validate pass）。本判断でcurrent baselineの意味品質を保証したとは拡張しない。

## 証拠と安全

- Iteration26: H1〜H18/既存Issue、Iteration32: H19〜H22/完了条件、Iteration33〜35: H23qualification/新独立/range、Iteration36-h23-audit.json:再計算15observations。
- unknown/duplicate/missing ID、invalid JSON、unresolved、timeout、context overflow、根拠不備、直接/推移矛盾はpartial planを返さないbenchmark testsと実測停止を保持。
- metadataはfinalized grouping後のみ、authoritative Validateを使用。未達candidateをmetadata成功で採用しない。
- local inferenceのみ、Skill不使用、production/default/backend/Git safety変更なし、main既存.gitignore変更を保持。
