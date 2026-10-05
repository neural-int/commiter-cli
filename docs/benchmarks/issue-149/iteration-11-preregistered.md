# Iteration11事前登録: H8 observed-contracts

ユーザーの回答に従いdiff/codeの観測事実のみ使用する。H6と同じcanonicalizationのもと、hostがsource/test各fileのbefore/after宣言をE-ID付き観測として供給する。global modelはflat membershipではなく、具体的subject（観測宣言名）、before/after contract、members、観測evidenceを持つcontract集合を1callで発見する。hostはanchorの実在、根拠IDの実在・member coverage、全file一意assignmentを検証し、余計な根拠/未知subject/部分割当/unresolvedで停止する。soft edgeによるunionを作らない。構造検証はsemantic proofではなく、semantic correctnessはgoldで別採点する。

canonical entity naming/source-path order、observed calls/assertions/soft graphを継承。Gemmaをgrouping-only bounded-routed-grouping native0/output1536、context16K/call120秒、全600秒、temperature0/top_p1/top_k0/seed144、repair/retry0で実行する。metadataは従来Gemma profile。schema/semantic responsibility/profileを同時に変えるので単独効果を主張しない。

まずweak16/cross12を各1回。構造失敗またはFM/FS非0なら資格不成立、profile/subject wordingを結果に合わせてtuningせず棄却。両方exactならfresh holdoutと4/6/8/9/16・guardrail・逆順・最終planへ進む。候補を採用するには全既存production条件が必要。

fresh holdout（weak16通過後、cross12結果確認前に固定）: holdout-protocol-and-health-8。protocol header、envelope、accepted versionの3source/test pairは観測可能なwire protocol v1→v2互換変更として同じgroup、health文字列変更のpairは別group。calleeのshared rootなしでも共通contractをまとめる能力と別purposeの分離を同時評価する。author goldはpayloadへ渡さない。全before/afterのgo testで合成programを検証してから推論する。過去holdout6/16はregression、freshとして再利用しない。
