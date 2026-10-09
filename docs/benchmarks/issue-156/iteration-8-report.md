## 要約

Bの6/8/16file transferは**改善なし・NO-GO維持**。モデル6callは全て60秒timeout、公開16file両順は事前固定presentation上限で推論なし。8/8で安全なA2 fallbackへ戻り、partition一致4/8はA2と同じだった。監査込み総wall639.897秒は固定600秒を超え、resource gateも失敗した。Dの候補・gold・80%閾値を変更しない。

## 検証結果

- score3、margin1、full-member coverage、最大8proposal、32768message bytes、neutral grammar、cached Qwen3-8B、16384context/1536output、temperature0/seed144を保持。再試行0、モデル変更0、閾値変更0。
- 8file source正規化は両順timeout。6file独立設定と16file独立設定も両順timeout。これら6callのinput/output tokenは未観測で、0tokenだったとは報告しない。completed answerが無いためnaive highest対照も未計測。
- 公開16fileのmessage bytes47549は事前上限32768を超え、両順0callでA2 fallback。別の16file入力へ差し替えたりschema上限を増やしたりしていない。
- final partition match4/8、5〜8層2/4・9〜16層2/4、A2 baselineと同じ。FM0/FS15738、fallback100%、file singleton100%、compose0。全8観測でownership/reconstruction/正逆stage/authoritative Validateが成立した。これらは構造安全・partition一致の観測でありBの意味能力やbounded end-to-end成功のGOではない。
- model call数6は上限8内。総wall639.897秒、mean79.987秒、95p113.727秒（lower empirical order statistic）。固定600秒に39.897秒超過。推論dispatch/timeoutのdeadline後にもmandatory temp-index監査が続く実装のため、全cycleの厳格な時間上限は成立していなかった。予算を事後変更せずresource failureとして保存した。
- C補助監査は推論無しで事前固定した6/16fileのoracle partial ownershipを検証。raw subsetは両規模で正逆stage・最終tree一致を通過したが、同じfile IDを複数commitへ出すためplanning.Validateがinvalid_assignment。file fallbackは両方valid。これは構造feasibilityの監査であり大規模Cの自動意味正解率ではない。
- metadata互換性の実検証はcomposeのみがinvalid_type、chore+bullet bodyとcompose+bullet bodyがinvalid_schema。現行wireCommitにbodyが無く、未知fieldを拒否する。混合目的bulletを黙って落として互換と呼ばない。
- 元mainの既存.gitignore変更は維持。追加/変更はdocs/benchmarks/issue-156とtools/benchmark156内に限定し、production・既定モデル・4file上限・依存を変更していない。

## 考察

大規模Bはscoreの意味品質を評価できるcompleted responseに達していないため、scoreが正しかった/間違っていたという推論は行わない。観測できたのは固定decoder/presentation/resource条件でBが改善を提供できず、安全なfile fallbackだけが残ったことである。既存小規模独立Bの順序感度・naive FMリスクと合わせ、production候補には採用しない。

総時間上限はinferenceだけでなくstaging監査まで予約・管理する必要がある。しかし今回の候補は意味品質も未成立なので、測定後にタイマーや予算だけを修正して再採用しない。B2のraw partition一致は保持し、時間条件失敗を明示する。

6/16fileのC監査は、byte ownershipと部分stageが成立しても、現行file assignment/metadata contractを通過できないことを再確認した。この制約は意味模型の変更だけでは解決しない。CU ownership schema、type/body/release互換性と目的分類の安定性を別の検証・設計で確立する必要がある。

## Next Steps

- A〜Dの成果物とfailure contractをcompletion-audit.mdで照合し、親子Issueの完了条件を更新する。検証作業の完了と80%達成・production採用を区別するため。
- **全Verificationとcheckbox成立後**に最終repository test/vet/build、既存release-notes tests、研究回帰、format/差分整合を実行する。必要な品質gateだけを行い、自明なtestを追加しない。
- 最終gateの結果をIssueへ報告しcommit/pushする。PASSを確認してからGoalを完了とする。production導入・#145解決の主張は行わない。
