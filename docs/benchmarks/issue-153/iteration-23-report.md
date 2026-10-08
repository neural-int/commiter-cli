## 要約

空白抑制-200対0の固定対照で、同一promptの全4caseのscoreが変化した。抑制ありは過去の全-1を再現しexact1/4/FM0/FS7。抑制なしは2caseが全0/unresolved拒否、残る2caseが誤結合（FM1とFM4）、exact0/4。生成制御の作用は観測したが、品質candidateはNo-Go。空白抑制だけでsemantic品質が成立する仮説は支持されない。

## 検証結果

新規C専用worktree issue-153-neutral-score-decodingで条件/patch/runnerを9bd599d5、setupをd02cbd01へモデル実行前に固定・push。#153事前6057447459、#150事前6057447936。独立helperをAPFS cloneして同一新binary内のbiased/neutralを比較。binary SHA bc61461f860957780394650f8b46fab2b737f9931917d0194a0f736e671b41a4、元helper SHA3b50561e...は保持。Package.resolved/GrammarSamplingStateソース同値、新しいモデル/packageの追加・downloadなし。

neutralでも24合法/8不合法/8prompt replayが通過し、各合法replayの空白penalty0を確認。Metal smoke成功、selected Swift1test成功、Python構造10test/diff check成功。production source/default/manifest/Git mutation経路変更なし。

全8call backend completed、retry/repair0。各caseでpayload/schema/prompt hashとinput tokensが両arm同一。元systemのみを使い、iteration21の追加policyは未使用。全caseは使用済み原因診断であり、新規holdoutに数えない。

| ケース | 抑制あり | 抑制なし |
| --- | --- | --- |
| 既存6file/3intent | accepted、exact false、FM0/FS3、全-1 | 全0・unresolved=true、拒否、品質null |
| Fee切上げとWrap＋対応test | accepted、exact false、FM0/FS2、全-1 | 全0・unresolved=true、拒否、品質null |
| Count負値処理＋無関係test診断文 | accepted、exact true、FM0/FS0、-1 | accepted、exact false、FM1/FS0、+1 |
| 共有TestBothの独立assertion | accepted、exact false、FM0/FS2、全-1 | accepted、exact false、FM4/FS0、+1/+2 |

FM/FSはunit-pair。biased complete4/4、exact1/4、accepted FM0/FS7。neutral complete2/4、exact0/4、accepted2件だけのFM5/FS0、reject2件はnullを保持。neutralのFM5/FS0を全4caseの品質集計へ読み替えない。全pair28個はbiased -1が28、neutral 0が21、+1が5、+2が2。後者21個の0は拒否した2caseのraw score。

shared-testでは強い+2が同一test内の二assertion同士と二source同士、真のsource/対応assertionは+1で、最終は全4unit merge。これはscoreの観測でありモデルの内部理由は推測しない。元controlはiteration21各caseのscore/membershipを全て再現した。

biased4call wall80.109秒/input4723/output436、neutral4call wall140.086秒/input4723/output484。single-call条件なので一般的な時間差・反復変動・順序効果は未推定。各120秒/全960秒/16K context/output1536/8192bytes/8unitを保持。build130.25秒、test build106.90秒はsetupで、計画生成latencyへ合算しない。

## 考察

純空白penaltyの差だけでscore/rejectが変わり、iteration22の符号非対称が今回の出力に作用する根拠を得た。全negativeをsemantic判断だけの結果として解釈できない。positiveが出るようになったことと、正しい対応を選別できることは別で、unknown2件と誤結合2件が残った。

rejectは誤planの生成と区別した安全な未判断で、complete/正解へ換算しない。空白抑制を外すだけの元prompt候補は、全case正解・改善・case回帰なしの事前gateを満たさない。この候補のbias量/条件反復は終了する。productionへの単独採用根拠も得られていない。

iteration21のpolicy比較は抑制ありの契約でしか行っていない。今回生成制御の作用と正のscore出力を確認したため、そこでのpolicy同値結果だけでは、neutralで同じ固定policyの作用を判定できない。未測の組合せというだけでなく、一般的判断基準と、その値を出力する制御の相互作用を切り分ける具体的な根拠がある。ただしbiasが旧policy失敗の原因だったと先に断定しない。

次に比較するpolicyはa78a63b9以前に固定した対応implementation/assertion・独立log・共有testの基準をそのまま使う。今回の結果を見て文言/例示/gold/score尺度を追加しない。全formal gate/過去No-Goを維持し、decoder作用の確認をsemantic改善やD必要性へ置換しない。

## Next Steps

- 新規C専用worktreeで、neutralを固定した元system対iteration21の固定policyの単一比較を事前登録する。biased上でのpolicy比較を有効な生成契約へ一般化できないため。
- 使用済み4case各arm1回で、complete/reject/exact/FM/FS/costを評価。文言・model・bias量を変えず、未知の拒否も保持する。作用と候補品質を分けるため。
- 全case正解と回帰なしを満たした場合だけ未使用データと全段/旧回帰へ進む。不達なら同policy反復を終了し、これ以上のwording/sampling sweepを行わない。
- A bounded GO/B No-Go/C正式未達/D未正当化/production4file/Goal未達を保持する。使用済み原因診断は正式統合/採用の証明にならないため。
