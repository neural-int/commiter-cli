## 固定仮説

iteration22で観測した符号非対称に基づき、signed scorerの純空白penalty -200を0へ変えると、全negativeへの偏りとsource/test誤分割が改善するか検証する。新規C専用worktree issue-153-neutral-score-decoding、独立した/tmp helperコピーを使用。前回のmodel capability不足/grammar bias原因を断定せず、単一機序の対照とする。wording/model/閾値sweepはしない。

## 処置と保持条件

同一新helper binary内のbiased profile（既存bounded-routed-grammar）とneutral profile（bounded-routed-grammar-neutral）を比較。差はgrammar stateの純空白token setを通常値か空集合にすることだけ。JSON schema/GrammarSamplingState/EOS整合/unknown/context/output/停止理由/coverage/solverのhard gateは維持。native thought0、Qwen3-8B revision545dc4251c05440727734bcd94334791f6ab0192、temp0/topP1/topK0/seed144、context16384/output1536、120秒/call、960秒全体、8192message bytes、8unit cap、retry0/repair0。

元systemのみを使用し、iteration21の追加policyは両armで使わない。各caseでmessages/user/schema/source mapping/上流の機械的acceptanceを完全一致させる。すべて使用済み4caseの原因診断で、fresh holdoutとして数えない。fixture SHA256 9d35816274c56cd5c2cbc0931678e801547562f93afe17e6989759ab2505b95eを保持。

順序はcase1/3 biased→neutral、case2/4 neutral→biased、各1call、計8call。input/output tokens、wall、stop/reject、score、unit-pair exact/FM/FSを記録。未完了/不正/unresolved/一意解なしはreject、品質null。score変化とsemantic品質改善を区別する。新binaryのcontrolが過去と異なっても結果を隠さず、過去結果と直接単独比較せず同binary内の対照を優先する。

## 判定条件

空白biasの作用は同caseのscore/reject差で観測する。限定的な次semantic候補にはneutralが4case全complete/exact/FM0/FS0、biasedよりexact件数増、caseごとのFM/FS/reject悪化なしを要求する。score変化のみ、誤結合増加、unknown増加、全通過しない場合は「機序の作用」と「候補品質」を分離し、この一候補をproduction向け成立とは扱わない。使用済み結果を見た後の条件/score尺度/gold変更なし。

## setupと安全

既存一時helperをAPFS cloneで独立コピー。新規package/version/model downloadなし。patchはneutral-score-helper.patch。原helper binaryのSHA 3b50561e987fb166a794bbe7d6a6dd8aca00514a3619a391a523f001f8c64f80を保持。新binaryのSHAはbuild後・モデル呼出前にiteration-23-setup.jsonへ固定する。

既存sourceとproduction manifest変更なし。neutral状態でもiteration22の24合法/8不合法/8prompt replayが通過し、合法応答の有限空白penalty0を確認してからモデルを呼ぶ。test wiring/Metal資源の既存setupを再利用する。patch・runner・条件を結果前にcommit/pushし、setup証拠も固定してから計測。失敗はsetupとして記録し、validator無効化やJSONのrepairはしない。

## Next Steps

- 全条件通過なら、新規未使用入力を別worktreeで構成し、全段/旧回帰/同条件production比較へ進む。使用済み診断だけで採用しないため。
- 不達なら当該bias変更候補の追加推論を終了し、機序の作用・残存semantic failure・正式未達を記録する。有限bias量の総当たりや既知goldへの調整を行わないため。
- B No-Go、D未正当化、production4file上限とGoal未達を保持する。C局所診断を正式統合やGoal完了と同一視しないため。
