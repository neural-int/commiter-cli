## 要約

生成時の空白抑制を外した同一条件でも、固定済みの対応実装/assertionの判断基準は品質を改善しなかった。元systemと追加policyはいずれも完全正解0/4。追加policyは1件のunresolvedを解消したが、その出力は独立変更の全結合で誤りだった。このpolicyと現score契約の候補はNo-Go。同じ文言・bias量・model名の反復比較を終了する。

プロンプトが無関係とは言えない。今回も3件でscoreが変わった。一方で、指示の追加だけで正しい境界が得られるという仮説は支持されなかった。過去の入力情報不足、粒度制約、生成契約の偏り、意味判断の誤りを分けて扱い、一つの原因へまとめない。

## 検証結果

新規C専用worktree issue-153-neutral-policy-interactionで条件とコードをec1ae6bbに固定・pushし、#153事前コメント6057826584、#150事前コメント6057827056へ登録後に8callを実行した。両armで同一neutral helper SHA bc61461f860957780394650f8b46fab2b737f9931917d0194a0f736e671b41a4、Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192を使用。新しいモデル/package/依存の追加・downloadなし。native0/temp0/topP1/topK0/seed144、16K context/output1536、120秒/call・960秒全体、8192bytes・8unit、retry0/repair0を保持した。

処置はiteration21以前に固定したPOLICYの一追加のみ。case内payload/schema hashは全4件で一致し、policyのinputは各92token増。goldと要求注釈は評価側だけにあり、上流の機械的parent acceptanceとsolverも共通。全ケースは使用済みの合成原因診断で、fresh holdoutに数えない。

| ケース | 元system | 固定policy追加 |
| --- | --- | --- |
| 6file/3intent・対応source/test | 全0・unresolved拒否、品質null | 全0・unresolved拒否、品質null |
| Fee切上げとWrap＋対応test | 全0・unresolved拒否、品質null | 全4unit結合、exact false、FM4/FS0 |
| Count負値処理＋無関係test診断文 | +1で結合、exact false、FM1/FS0 | +2で結合、exact false、FM1/FS0 |
| 共有TestBothの独立assertion | 全4unit結合、exact false、FM4/FS0 | score変更後も全結合、exact false、FM4/FS0 |

全8call backend completed、timeout/invalid0。元はcomplete2/4・reject2・exact0/4、policyはcomplete3/4・reject1・exact0/4。acceptedだけのFM/FSは元5/0（2件）、policy9/0（3件）。母集団が違うためこの合計を全4件の品質の大小比較へ置換しない。拒否はFM/FS/exactをnullのまま保持した。共通accepted2件のpartitionと品質は同じ、Feeは拒否から誤planへ変わった。

元controlはiteration23のneutralのscore/unresolved/membership/品質/理由を4件全て再現した。同binary/同profile内の比較を主証拠とし、旧biased armの結果と混ぜない。policyでは6fileのみscore同値、他3件はscoreが変化した。共有testでも最終partitionの改善はなかった。

元4call wall79.102秒/input4723/output484、policy4call wall106.285秒/input5091/output484。単回比較のため一般的なlatency差・分散・順序効果は未推定。各callの事前上限内で完了した。構造Python10testとgit diff --checkは成功。iteration23で検証済みの同helperを再使用し、追加GPU test/buildは行っていない。production source/default/Git mutation経路は変更なし。

正式条件19件の以前の未達/partialをiteration-24-completion-audit.jsonへ保持した。A bounded GO/B No-Go/C正式未達/D未正当化/production4file/Goal未達。最終全project gate実行条件は未成立。

## 考察

iteration21ではbiased下のpolicyの作用が見えず、iteration22で整数scoreのtoken化と純空白penaltyの符号非対称を発見し、iteration23でpenalty除去だけの出力変化を観測した。今回はneutral下でも同policyの品質改善がないことを確認した。従って、全negativeの過去結果を意味能力だけに帰属する解釈は訂正すべきだが、偏りの除去と一般的判断基準の追加だけで品質が成立するという解釈も支持されない。

Feeのunresolved解消は成功率の改善とは扱わない。拒否から誤結合へ進んだだけで、完全自動plannerの安全品質を満たさない。Countの無関係診断文に対して強い同目的scoreへ変わったことも、正しい対応関係を選別できた証拠ではない。

今回の比較は、現scorer契約における一つの判断基準の不合格を示す。全promptが無効、モデルが意味を理解できない、情報が常に十分、専用学習が必須、などは導かない。使用済み少数合成、単回のlocal scorer比較であり、全段・#149回帰・production baseline・別作者/実repositoryの未使用入力は未検証。Dの前提であるevidenceと全partitionの妥当性も未達のため、Dへ進む根拠にはならない。

過去#149 H1〜H23のproduction candidateなし、Bの追加evidence一般化未達、Cの入力識別性・生成契約・意味scoreの各失敗を保存する。これらを一律に「prompt問題」または「model能力限界」とまとめず、今回のdecoder/policy結果を追加する。optimizerを変更するだけでは、誤った全positive scoreの正しい最適化から正解を回復できない。

## Next Steps

- 現整数score契約での固定policy/decoder単独・相互作用の比較を終了し、追加wording/sampling/model sweepを行わない。単独と相互作用のどちらでも事前品質gateが不達だったため。
- 次のモデル計測は、現在と区別できる入力情報または判断責務について独立した根拠があり、対応変更の結合と無関係変更の分離を同時に検証できる場合に限る。未測という理由だけで新しい文言やモデルを試さないため。
- 再開候補が具体化したら、新規専用worktreeで条件を固定し、別作者/実repository未使用入力から評価する。今回の使用済み合成例への適合を一般化の証明にしないため。その後に全段/旧回帰/同条件production比較/metadata/Validate/costの正式gateへ進む。
- まず既存の停止監査と今回の結果を照合し、未達条件と再開に必要な入力を確定する。Goalを完了扱いせず、Dやproduction変更へ進めないため。
