## 要約

**機械側へ依存方向を分離したGemma構成は、対応実装/testをまとめる能力は確認できたが、独立変更の誤統合と提示順感度が残り、今回のGateはNO-GO。** 許容分割13/16、順序一致7/8。機械的なtest-assisted方式は8/8 workloadで本試験の規範・ordered tests・再構築を満たした。LLMを必須とせず、追加価値を比較した結果として保持する。

## 検証結果

事前登録commit548e838bf55e988c569a48c65422a422f11aace6をpush後に実行。初回とは別の自己作成Go8 workload（4/6/8 selected files）で、対象EA/EBの境界だけを判断。その他の変更は固定singleton。2提示順×Gemma=16call。元の5分類/gold/NO-GOを変更していない。source hashes、全16prompt hashes、準備監査との全8input hashesは再照合で一致。

許容境界は対応実装/assertionのjoin、独立変更のsplit、新APIのmergeまたは依存順付きsplit、多義的timeout/flagsのmerge/split。複数妥当な選択を一意goldへ押し込めず、unknownを必須成功条件にしない新しい評価契約を推論前に固定した。初回正解率や#156の80% exactと同じ指標ではない。

|方式|許容分割＋ordered tests成立|観測単位|
|---|---:|---|
|file-only|5/8|8 workload、固定入力順|
|static-new-api-order|6/8|同じ8 workload|
|test-assisted-pair|8/8|同じ8 workload、追加test実測あり|
|Gemma＋static host|13/16|8 workload×提示順2|

|Gemmaの対象関係|許容分割|
|---|---:|
|対応実装＋変更assertion|4/4|
|独立変更|1/4|
|新APIの依存付き変更|4/4|
|複数妥当なpolicy/flag|4/4|

全16応答completed/schema/根拠ID valid、timeout/retry0、input10064/output272tokens、token未観測0。推論process計73.277秒/平均4.580秒。baselineの初回監査を含む全cycle248.050秒で1800秒以内。モデルload/grammar込み、同じordered partitionの検証cache再利用あり。全体にはbaselineの全検証を含むので単一製品経路のlatencyではなく、方式別の純粋な時間差も測っていない。

全Gemma計画でplanning.Validate・完全割当・各Git treeのbyte整合・forward適用・ordered reverse revert・fixture tests・各group単独revertが成立（16/16）。それでもレビュー規範では3観測不成立。shared-helper-independentは両順でmerge、same-package-independentは正順defer→保守split、逆順merge。不要な統合と提示順差は、schemaやtests PASSだけで検出できなかった。原因を共有helper/packageやdecoderへ因果帰属する試験ではない。

Gemmaは新API2ケースを両順ともmergeした。これを合法な選択として受け入れた。静的hostはprovider/consumerを正方向に抽出し、split時も適用順を修正した。test-assistedのAPI2ケースは依存順付きsplitで有効だが、providerだけを独立revertするとcallerが残りtestsがfailする。したがってtest-assistedはordered妥当性8/8、全group単独revert成立6/8。この2性質を混同しない。

準備中のGo before-state未使用importは推論0call時に再現・修正済み。失敗と元入力を保存し、修正後before/after全PASSと片側stateの期待されたfail/passを監査してから事前登録した。推論後は入力・gate・sourceを変更していない。

ローカル品質ゲートは研究6回帰、go test ./...、go vet ./...、go build ./...、CLI build、既存release-notes tests、追加Goのgofmt、全差分整合の8項目PASS。既存Go testsは標準cacheを使用。CI/production検証の主張ではない。

## 考察

方向ラベルをLLMから外すことでAPIケースの合法mergeを失敗扱いにしなくなったが、新規Go入力の独立性判断は安定しなかった。前回のPython単純独立ケースの成功を任意のshared-helper/同packageケースへ外挿できない。この比較は入力・schema・責務・評価基準が異なり、Gemma能力の退化や単一変更の因果効果を示すものではない。

test-assisted baselineは今回の対応実装/assertionの片側state不整合を機械的に観測できた。source/static-onlyでは残った2caseを回収し、独立の4caseでは双方PASSとしてsplitを保持した。ただし実測testsという追加情報があり、LLMと同情報の比較ではない。8/8は自己作成の限定pair診断で、自然履歴・testsなし・不十分なtests・未変更assertion・複雑な依存への一般化や、全8file groupingの能力を証明しない。

レビュー規範は本試験で固定した「不要な独立変更を混ぜない」を含む。shared-helperの2変更をまとめてもコンパイル/テスト/revert可能であり、あらゆるworkflowでそのcommitが不適切だと証明したわけではない。動作上の可逆性と、レビュー単位の最小性を別条件として測った。

固定Gateは16/16妥当・順序一致・独立誤統合0を要求し、13/16と7/8では未達。13/16を#156の80%達成へ転用しない。production・16file候補のGOは出さない。

## Next Steps

- 本Gemma提案を逐次グループ構築へ接続しない。独立変更の誤統合を複数判断へ累積させないため。今回のNO-GOと対応ケースの限定成功を両方保持する。
- 次の検証候補はtest-assisted機械方式の成立範囲と費用を先に確認する。自己作成ケースで動作した方式があるため、LLM必須の複雑な仕組みを先に作らない。
- testsなし/未変更tests/片側の変更をtestsが観測しないケースを未使用入力で用意し、sourceから識別可能な依存と観測不足を分離する。tests PASSを独立性・目的同一性の万能証拠にしないため。必要な未知能力と採用holdoutの条件を事前固定する。
- 機械方式で解けない境界だけをLLMへ渡す候補を評価する場合は、source参照・棄却可能性・監査込み費用とnegative guardrailを固定する。同じ使用済み8ケースのprompt tuningで採用を決めないため。

今回の作業はdocs/benchmarks/issue-162とtools/benchmark162のみ。production/default model/4file上限/SRS/既存validator/依存を変更していない。SQLite追加、外部推論、モデル取得、Skill使用0。実行したfixtureは自己作成コードのみ、外部repo code/test実行0。Git操作は使い捨てrepo内、元worktree/indexを変更しない。初回の記録を保持し、次段2項目は未実施としてIssue openを維持する。
