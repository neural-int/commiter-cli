# Iteration 1: Stage Aの構造能力とmetadata契約

評価前に固定する。意味品質の独立holdout評価は本iterationでは実施しない。

- base: f275d953f0c15452e9281d1c678cb9bb20adcbbb。production変更なし。
- #151の抽出器: c5acdcb0f9f5fa87d47f070146902333f75e165a の evaluate.py / inline.pyをbyte一致でvendor151へ複製。
- deterministic file fallback。cross-file mergeなし、意味分類器なし。全fileをunknownと記録し、mixed-intentがないという主張をしない。目的未確定・mixed risk unresolvedをmetadataに明記。一律composeは禁止。
- files 1〜16、file 1MiB/20,000line/256CU、inline 4096byte、全体4096CU。既存の上限を結果に応じて変更しない。予算超過は構造・資源停止として記録。
- regular text/binaryの内容編集、作成/削除、100644/100755を対象。rename、symlink、submodule、競合、mode-only変更は本候補の非対応範囲。
- #151使用済み18fixtureは構造回帰・意味診断用。独立holdoutに再分類しない。追加の1/4/8/16fileケースはsynthetic structural-only。評価用goldをplannerへ渡さない。
- Aの意味自動停止baselineは全unknownでcoverage0。file fallbackは合法計画を返すがexactとは数えない。naive CU-singletonとfile-singletonを別比較する。
- 必須構造gate: 完全割当100%、byte source一致・再構築100%、temp-index replayの正逆順で最終tree一致100%、欠落/重複・不正mappingは停止。元repo/indexは変更しない。
- metadata互換性は既存planning.Validateを実呼出しして確認。compose typeを実装へ追加しない。commitlintの有無・release経路をsource監査する。
- 能力GOはunknownでの構造的fallback成立に限定。mixed分類・LLM意味能力・80%・production採用は本gateでは未判定。
- LLM calls/tokens 0。新依存なし。Skill不使用。外部repositoryコードの実行なし。

## 次の評価に必要な入力

B/D用に未使用作者またはrepositoryの公開履歴から、入力・意図根拠・識別可能性を評価前に固定する。既知#149〜#155とsynthetic診断をholdoutに混ぜない。モデル/helper digest、decoder、全体予算、順位/coverage/margin、FM非悪化基準を推論前に保存する。
