## 要約

固定検証は **NO-GO** です。既存5条件のCategory Gold一致は現行・候補とも **0/6グループ**、境界例は **11/20**、実際の3段実行で受理された完成planは **0/5** でした。候補の実装・固定条件・失敗を検証ブランチへ保存し、採用・main mergeは行いません。Issueは未完了です。

## 検証結果

- PR #172の `0f9278c33c2f477372bc7dd07c7616a738dd6268` を基点に、既存5 fixtureとGoldを保持してCategory単独のpaired比較10回、推論前にGoldを固定した境界20回、実際の3段実行5回を行いました。生成は上限45回に対して **43回**、retry/repair/追加の品質判定LLM呼び出しは0です。
- 既存6グループ：現行はすべて誤って`feat`、候補は4件を誤って`update`、2件を誤って`feat`。Gold一致は増えていません。
- 境界20件：`update`は選択6・正当2・誤選択4・見逃し1、`change`は選択1・正当1・誤選択0・見逃し1。候補の計26判断に`fix`は0件でした。F11のtest判定は既存test-only Schema制約による強制です。
- Category単独30件はcompletedかつSchema適合。最大入力Eは102.313秒。Category helper RSS最大3.686 GiB、pressure levelは1、1試行前後のswap増分最大630.88 MiBでした。RSSはMetal/GPU総ピークではなく、swapの変化をhelperだけに帰属していません。
- 実際の3段実行ではGroupingは5/5でGold一致。A〜CはTextの`invalid_schema`（scopeまたはsummaryが16/48文字を超過）、D・EはCategory実行中に共通120秒deadlineで中断し、Text未実行。受理planは0/5でした。
- 到達したGrouping/Textリクエストは固定リクエストと一致し、Categoryの実リクエストも固定入力と一致。全試行のHEAD・index登録内容・worktree diff・statusのdigestは前後一致しました。
- 入出力token件数は **未取得** です。既存tokenize-onlyバイナリは推論前の3件を`malformed_request`で拒否し、本番helperも件数を返しません。byte数をtoken数として代用していません。
- `go test ./...`、`go vet ./...`、`go build ./...`、release-note Pythonテスト、差分・構文チェックはPASS。初回のGoテスト失敗は旧benchmark146のCategory packetアダプターを再現確認して修正しました。過去計測・Gold・固定済み測定バイナリは変更していません。CIは未実行です。
- Goldはエージェント作成のsynthetic評価であり、人間による独立評価ではありません。既存fixtureの再利用と新規境界例を区別しています。

詳細・固定条件・生の結果：[検証報告](https://github.com/neural-int/commiter-cli/blob/codex/issue-173-category/docs/benchmarks/issue-173/report.md)、[preregistration](https://github.com/neural-int/commiter-cli/blob/codex/issue-173-category/docs/benchmarks/issue-173/preregistration.json)、[Category結果](https://github.com/neural-int/commiter-cli/blob/codex/issue-173-category/docs/benchmarks/issue-173/category-results.jsonl)、[3段結果](https://github.com/neural-int/commiter-cli/blob/codex/issue-173-category/docs/benchmarks/issue-173/integrated-results.jsonl)。

## 考察

feat/fixの誤断定が6件から2件へ減っても、残り4件は明確なfixをupdateへ置き換えたもので、精度向上ではありません。境界例でも明確な修正、性能改善、UI変更、build/ci、複数目的の区別に失敗し、今回の二段階fallbackを採用する根拠は得られませんでした。

入力・意味判断プロンプト・type enumを合わせて変更した比較なので、各要素の個別効果は断定できません。Raw Diffだけを成功・失敗の原因としません。Categoryと、Textの長さ制約・共通deadlineによる完成plan失敗も分けて扱います。

完了条件のtoken計測・完成plan受理は未達です。Textとdeadlineの変更はIssueで別問題として分離されており、その修正まで現在の検証で進めることはスコープを越えるため、goal.mdの停止条件を適用します。Goal達成とはしません。

## Next Steps

- Textの長さ制約遵守と共通deadline内のphase予算を別Issueで扱う。完成planの失敗をCategory調整だけで解消できないため。
- Categoryを続ける場合は、既存契約を使ったfix判断と複数独立目的のchange判断について、新しい有限契約・未使用fixtureで比較する。今回の誤分類から得た仮説を、現fixtureの反復だけで一般化しないため。
- token計測はhelperの計測契約を確認して別runで固定する。今回の未取得値や過去結果を後から補完しないため。
- 未達条件のチェックは残し、mainへの採用・merge・リリースは行わない。
