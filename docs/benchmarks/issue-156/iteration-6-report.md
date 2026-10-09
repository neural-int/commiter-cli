## 要約

A2+Cの固定独立評価は**C統合候補NO-GO**。unique goldの7ケース×正逆順14観測で最終exact4/14（28.6%）、A2 fallbackの2/14（14.3%）から改善したが、改善は単一目的cross-file controlに限られた。mixed-file FMは改善せず、同一fileの部分分割8観測は現行planning.Validateがinvalid_assignmentで拒否した。rawの意味正解やstage可能性を最終成功へ算入していない。

## 検証結果

- 事前固定した新規controlled7件と未使用spf13/afero公開mixed履歴1件、それぞれ正逆順でC16観測、B対照4観測。総backend calls27、固定上限36内、再試行0。既存cached Qwen3-8Bとneutral grammar、temperature0/seed144/context16384/output1536を固定した。
- Cのprimary最終FM70/FS36に対してA2 FM70/FS44。exactの改善2観測はimplementation+testのsingle-purpose control。mixed5ケース×2順は全て非exactで、C限定GOに必要なmixed FM改善は成立しなかった。
- Cのraw completed14観測はexact5/14、FM32/FS200。A2の同じprimary FM70/FS44に対し、FM低下と大きなFS増加が併存した。raw exact5のうち1観測は別hunkの同一file分割で、最終validator拒否によりfallbackへ戻った。
- 別hunk正順はraw exact、逆順はrefinement後の6 singletonでFS7。同一symbolのrefinementは両順8groupとなりFS70/85。同一行の2目的は両順singleと扱われ、FM8が残った。
- first-response mixed推定7観測でbounded refinementを要求。refinementはgoldから起動していない。出力CUの全所有権・byte再構築・raw正逆stageを検証した。未知を単にfine singletonへする方式を採用していない。
- C最終計画16/16、B対照4/4がplanning.Validate・正逆temp-index replayを通過した。最終invalid構造や既存indexへのmutationはない。fallbackを自動生成coverageに数えても、実際のgold不一致はexact失敗として残した。
- B対照はunique primary2観測exact0/2、FM8/FS34、A2と同値。公開mixedの2観測も参照境界では改善なし。Iteration2のB NO-GOを撤回していない。
- 公開mixedはUnionFile Close実装/対応testと、独立CacheOnReadFs WriteReader testが同居する履歴。推論前のsingle-intent state監査で、反復する閉じ括弧・空行の所有権に複数のsource witnessがあり、bounded二解DPでは同時完全割当を確定できなかった。unique-gold primaryから事前分離し、source-span参照境界は診断のみとして保存した。2回のpreflight失敗後にこの識別可能性分類を確定し、その後初めてモデルを実行した。後からgoldを正解へ修正していない。
- 公開mixed C両順は60秒timeout。token responseは未観測なので2callのtokenを0と断定しない。C観測済みtoken下限input14788/output1492、B input4648/output480。C総wall437.321秒、mean27.333秒、95p61.184秒（lower empirical order statistic）。B総wall93.757秒。wallはstage監査込み、runtime latencyではない。
- decoder sourceでneutral profileの空白penalty対象が空配列、temperature0/seed144を確認。scoreは非負ordinal、group labelは任意番号であり符号scoreを使用しない。tokenizerの全labelについて偏りがないという統計的証明は行っていない。観測されたorder感度は除外しない。
- composeは現行allowedTypesに含まれずinvalid_type。PR Policyのtype regexもcompose非対応。同一file部分割当はcompleteAssignmentが拒否し、NewConstraintsにもdo not split a fileがある。CUをfile IDへ偽装する迂回は行っていない。
- 研究用7回帰テストPASS。新しい1テストは実際のvalidatorで部分file拒否とunknown完全file収束を確認する。最終repository品質ゲートはVerification完了後に実行する。

## 考察

Cは機械的にstage可能な単位を増やしても、目的帰属の正解を増やせるとは限らなかった。rawでFMが減っても過剰なFSが生じ、さらに現行のfile assignment契約とは互換性がなかった。単一目的controlの改善は残すが、mixed splittingの能力GOやproduction実用性へ一般化しない。

compatibility失敗はファイル数によらない。5〜8/9〜16に拡大しても、同一file IDを複数commitへ割り当てれば現行completeAssignmentに拒否される。大規模C semantic performanceを立証したとは呼ばず、この失敗契約をDへ渡しCをgated outとする。修正にはownership schemaとauthoritative staging/metadata契約の設計が必要で、本Issueのproduction変更禁止境界を越えて修正しない。

## Next Steps

- **既に事前登録したDのA2-only評価を実行する。** Cのmixed意味能力と現行互換性が未成立なので、NO-GO段階を組み込まず到達可能候補の80%達否を確定する。
- DはCに未使用のpublic履歴・新規controlled inputsを用い、1〜4/5〜8/9〜16、negative・shared test・same-line・unknownを測定する。15ケース中12 primary、3識別不能は推論前に固定済みで、除外率を報告する。
- Bの大規模transfer監査を、固定score/coverage/marginのまま別iterationで補う。子Issueの5〜8/9〜16の適用境界がまだ狭いためであり、成功を得るための閾値調整は行わない。
- productionはNO-GOを維持する。CU ownershipとcompose互換性は残余設計課題として記録し、意味能力の不足をvalidator変更だけで解決したことにしない。
