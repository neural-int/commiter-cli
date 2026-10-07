## 要約

Bの最初の実行を、6/18行時点で中断した。same-directory5とcross-directory9の各3条件で完全assignmentを受理できず、全行exact/FM/FSは未取得。現状のvalidation分類では形式不正とunresolvedを区別できないため、semantic evidenceのGo/No-Goは判定しない。

## 検証結果

- 2fixture × A-only/repository/history = 6 completed measurement rows、全complete=false。
- helperのinput/output token telemetryとwallは取得したが、host分類はinvalid_assignment_or_jsonの合成値で、元のbackend stopを上書きする実装だった。
- 7行目の推論中に自己起動したbenchmark process groupを停止し、未完行は評価に含めていない。
- fresh2fixtureへ到達しておらず、fresh結果は未閲覧。
- model/helper/source pin、事前登録、初回結果を保存。結果を削除・良い結果への置換はしない。
- A-only payloadでは事前登録のAST symbol annotationが欠けていた。実装と登録条件の不一致があり、evidence source効果の資格評価には使わない。

## 考察

共通のvalidation停止が続くため、source効果以前に実行・評価contractを診断する必要がある。結果からsemantic model能力やrepository/history signalの無効性を結論できない。有限回のprotocol診断を先に行い、単なるwording改善の反復として扱わない。

## Next Steps

- 同じ最小fixtureのA-onlyを1callだけ診断し、backend stopとhost validation reasonを別々に記録する。形式不正とunresolvedを混同しないため。
- 事前登録したAST annotationをA-only payloadに復元し、入力/出力schemaと受理contractをtestsで固定する。比較条件の不一致を修正するため。
- 修復ができた場合は、失敗行を保存したまま別iterationとして固定3条件を評価する。fresh2caseは結果未閲覧のまま保持し、最初の妥当な比較に使用する。
