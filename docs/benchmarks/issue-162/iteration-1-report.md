## 要約

**Gemmaは対応実装・テストと独立変更を8/8観測で区別できた。一方、方向付き依存と根拠不足の保留を含む初回Gateは両モデル未達。** 一律に「小型LLMは関連判断不能」と結論せず、成立した局所関係と未成立の責務を分離する。逐次構築・16file採用評価は事前Gateに従い実施しない。

## 検証結果

事前登録commit 7f8599e52e28a4d1e8547f6d4b66d9a78abeb708をpush後に実行。新規自己作成8ケース、4fileのbefore/after全文、対象A/B固定、2提示順×Gemma/Qwenで32call。入力・prompt・gold・source hashは推論前固定し、全32prompt hashとsource不変を再照合した。512出力上限の初稿は未実行で、helper sourceの固定1536上限へ修正したdraft履歴も保存。

|関係分類|Gemma|Qwen|
|---|---:|---:|
|対応する実装修正＋テスト|4/4|2/4|
|同名symbol/同dirの独立変更|4/4|2/4|
|方向付きAPI依存|0/4|2/4|
|根拠不足のunknown|0/4|0/4|
|事前goldの全分類一致|8/16|6/16|
|正常終了・schema・根拠ID valid|16/16|16/16|
|提示順による回答一致|8/8ケース|5/8ケース|

全call completed、timeout/再試行0、token未観測0。Gemma input7666/output512、Qwen input7140/output368。Gemma計126.803秒/平均7.925秒、Qwen計127.099秒/平均7.944秒。全cycle253.915秒で1200秒以内。モデルload/grammar準備込みのprocess wallであり、常駐serviceの推論速度や一般的な製品latencyとは呼ばない。後半にwall増加があり、熱/負荷等の原因は未特定。

固定した素朴なbaselineはfile-only 2/8、同一symbol/import候補 2/8。後者はpositive2件を回収するが独立変更も結合し、方向依存・unknownを表せない。これは全ての機械的手法よりLLMが優れる証拠ではなく、この2対照に限定した関係分類結果である。

GemmaはAPI依存の4観測を全てtogetherと返した。両変更を同じcommitにするのは本目的で許容されるため、これは依存方向の分類不足であり、4つの不適切partitionを生成したという意味ではない。Qwenはreverse-api-dependencyで両順とも逆方向を返し、positive2件は提示順反転で逆方向の依存回答へ変化。shared-name-independentでも架空の方向依存を返した。

同じtimeout値の変更では両モデルがtogether、二つのflagではGemmaがseparate、Qwenが順序によりtogether/separate。どちらもunknownを出さなかった。根拠IDが存在することと、結論が根拠から一意に導けることは別である。

## 考察

「変更をまとめるか」と「分離可能だが方向依存があるか」と「根拠不足か」を同じ5分類で要求した条件では、局所のpositive/negative能力と依存/保留能力が分かれた。Gemmaの最初の8観測は今回の小さな課題で追加価値を示すが、自己作成2positive/2negative×2提示順であり、母集団の100%精度や16file能力へ外挿しない。4file全体のpartitionも測っていない。

Qwenが同じ依存ラベルを多用しても、モデル内部の理由やdecoder biasをこの観測だけで特定しない。固定profile/grammar/native thought0での結果に限定する。入力が不十分なケースでunknownを返さないため、今回の出力をそのまま安全な関係graphへ変換できない。

初回16/16・unknown・順序のGateは両モデル未達。閾値/gold/入力を緩めずNO-GOを保存する。ただし、これは本診断の5分類構成のNO-GOであり、Gemmaによる局所レビュー対応、LLM一般、将来の責務分離の否定ではない。

## Next Steps

- 今回の分類器を逐次グループ構築へ接続しない。誤依存とunknown未成立を大量の判断へ累積させないため。16file評価もgated-outとし、独立性能は未立証とする。
- 次の候補は、sourceから確定できる新API依存を機械側へ分離し、LLMには少数変更の同時レビュー根拠と独立性だけを判断させる責務分離とする。Gemmaで観測したpositive/negative能力と依存分類不足を区別するため。まだ実装・推論していない仮説である。
- その候補を進める前に、unknownをコミット分割の禁止と同一視せず、「複数許容partition」と「観測できない必要依存」を評価契約へ別記する。現在のflag/timerは複数の分割が妥当であり、誤って一意goldのauthor-intent試験へ戻さないため。
- 新候補のGO判定は今回の使用済み8ケースとは別の未使用実履歴または独立作成ケースで行う。固定入力での診断改善を採用holdoutへ読み替えないため。必要な対応、許容merge/split、依存orderを推論前に監査し、入力から識別できない境界を正解に強制しない。

production・default model・4file上限・SRS・既存validator・依存を変更していない。SQLite追加、cloud推論、model download、外部source/test実行、Skill使用はいずれも0。今回モデル応答から実Git mutationを行っていない。再構築・stage・各中間状態のtest・実revertの成立は次段評価事項であり、今回成立したと主張しない。

ローカル品質チェックは研究3回帰、go test ./...、go vet ./...、go build ./...、既存release-notes tests、差分整合が全PASS。記録はquality-gates.json。CIやproduction動作確認の主張ではない。既存機能の更新/削除がなく、既存test削除は不要。追加3回帰は架空/欠落根拠、重複JSON、構造関連の誤解という実際の契約リスクを確認する。
