# #143 入力監査の固定手順（2026-09-30）

対象は a2a36029c7981c3a41f13cf68f0d3e725b8ac762 の新規合成holdout8例。前回240 callには追加しない。

1. 推論backend/helper/modelを呼ばず、issue143Prepareとissue142GenericForcedInputで正逆入力を再構築する。manifest全12契約との一致を確認し、holdoutのprompt/schema hashを記録する。
2. JSON decode後のrepository_input.filesをfixture元diffと照合する。raw_diffのbyte完全一致、構造evidenceへの置換、metadataのみ、evidence reductionを分ける。全file IDを確認する。
3. 正逆でsystem/schema/repository_inputが一致するか確認する。候補提示以外の変化を区別する。
4. prompt-visibleな証拠から各候補を支持・反証できる理由、元diffにしかない情報、矛盾・曖昧さを記述する。AI監査者はgold/rationale/前回結果を既読のため盲検ではなく、goldの独立検証とは扱わない。
5. 人間確認用資料は、visible file内容と2候補だけを提示し、gold/rationale/前回モデル結果を含めない。各例の選択、根拠、不明点を人間が記録する。候補を一意に区別できない回答を許す。過去情報を見た人間の場合は非盲検と記録する。
6. 人間判断を受領するまで「入力十分」と確定せず、後続contract推論実験に進まない。人間による説明可能性も一般的な情報十分性の証明ではない。

今回の静的観測と人間確認は分けて記録する。合成file evidenceの抜粋・数値・hashを保存し、全文system/user promptや生成responseは保存しない。依存追加・model download・production変更なし。
