# Iteration30事前登録: H21 global pair judgment

Iteration29 Next Stepsに従い、元contract/test/calls/4snapshot/contrast全件を同一global contextで提示し、全pair P-IDへS/D/Uを一度に判断。最大16file=120pair、compact ASCII出力1321bytes。これはtoken数/完了保証ではなく、有限出力形の実行可能性確認。actual tokens/stopはhelperで測定し、1536token/120秒を超えた場合停止。予算sweepしない。

#146のlocal windows/bridge/複数context all-pairs reconciliationとは異なり、局所call/監査/correctionなし。hostは全pair一意coverage/unknown/missing/invalidを拒否、S推移closureとD矛盾、Uはpartitionなし。soft syntaxをmust-linkにしない。整合した誤りはgoldで独立評価。

fixed Gemma revision475b9088d29754a3379866cf5aeb6b41acd313c2、bounded-routed-grouping/native0/output1536/context16K/call120秒/whole600秒/temp0/top_p1/top_k0/seed144/retry0/repair0、weak16/cross12/guardrail各1回順次。全資格通過時のみ新しい未使用独立評価を事前固定。追加取得/依存/production変更なし。
