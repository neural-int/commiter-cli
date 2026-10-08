## 要約

Cの採点systemに対応実装・変更assertionの判断基準が不足する仮説を、元systemと一つのpolicy追加systemの固定比較で検証する。これは使用済みgoldへのwording反復ではなく、一般的なtask criterionを明示する一候補の要因診断。Bの追加探索、正式C/production gate、過去No-Goを変更しない。親の追加推論停止記録を消さず、今回の再開根拠を別記する。

## 仮説と処置

実装の動作変更と、その変更を直接検査するassertion/期待値の変更は同目的。共有テスト内の異なるassertion、無関係なログ・diagnostic・refactorは別目的。filename/shared call/shared failed testだけでは対応を確定しない。情報不足ならunresolved。この基準を元systemの末尾へ一回だけ追加する。文言はpolicy_ablation.pyのPOLICYで固定し、結果を見て変更しない。

## 評価入力と出所

1. 既存fresh-six-file-behaviors：今は使用済み失敗診断。6file/3intent、対応source/testを3群へ統合する。
2. fee-rounding-and-wrap：Feeの端数切上げとWrapの括弧変更は独立。対応する各期待値変更と合わせ2群。
3. same-call-independent-diagnostic：Countの負値処理変更と、正値のみ検査するTestCountの失敗文言変更は独立。2群。
4. shared-test-independent-assertions：Leftの加算変更とRightの乗算変更は独立。同一TestBothの各assertion変更だけを対応sourceと統合して2群。

2〜4はこのiterationでエージェントが要求とgoldをモデル実行前に定義した合成入力。モデル未使用であるが、別作者/実repositoryの独立評価とは呼ばない。単純な合成変更から一般的なproduction精度を推定しない。gold/要求注釈は評価側のみで、model payloadはbefore/after/path/idを含む従来の2containerに限定。

## 固定条件

上流は全contiguous proposalを機械的にacceptした同一unit列を両armで固定。goldに基づくrefinement/mandatory edgeはなし。既存6fileはiteration17の実上流全acceptと同一。新規例は各proposalが一目的だけを含むことを事前監査した。Aの全再構築・正解intentの全subset20状態のGo test/stagingが成功。各subset成功は境界の意味的唯一性を証明しない。

Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192、既存bounded-grammar helper SHA256 3b50561e987fb166a794bbe7d6a6dd8aca00514a3619a391a523f001f8c64f80。bounded-routed-grammar、context16384/output1536、既存temp0/topP1/topK0/seed144、各120秒、全960秒、8192message bytes、最大8unit、retry0/repair0。モデル/追加情報/schema/optimizer/score尺度を変更せず、system追加だけを変える。追加指示によるinput token増は計測する。

固定順は1・3がoriginal→policy、2・4がpolicy→original、各arm各case1call、最大8callを順次実行。各順序1回なので反復変動や順序効果は推定しない。goldはmodel/optimizerへ渡さず、評価側unit-pair exact/FM/FSを使用。solverは既存partition.py。一意解のないscore、不正/未完了/unknown/unresolvedはrejectし、品質値nullを保持する。

## 判定条件と停止条件

限定的な継続候補には、policyの既存6file exact成功、新規3caseすべてexact/FM0/FS0、originalより新規exact件数増加、各caseでFM/FSとrejectの悪化なしをすべて要求する。使用済みだけ改善、新規で両arm同値、反例回帰、reject、その他gate不達は「この実験で一般化改善未立証」として同policy調整を終了する。新規3caseは少数合成であり、全通過でも正式C GOやB GO/production推薦にしない。

継続条件成立時は、新規専用worktreeで別作者または実repositoryの未使用データと上流を含む全段評価の準備へ進む。失敗時は追加wording/model/閾値sweepをせず、情報不足・評価前提・scorer判断を区別して報告する。D必要性を自動判定しない。

## 事前証拠

fixture SHA256 9d35816274c56cd5c2cbc0931678e801547562f93afe17e6989759ab2505b95e。iteration-21-preflight.jsonにpayload hash、unit順、gold、各message size、Go test/staging20状態成功を保存。モデル呼出0。構造10testとgit diff --check成功。条件・code・fixtureをcommit/pushし、Issueへ条件を登録してから推論する。
