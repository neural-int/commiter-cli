## 要約

全体判断専用の予算拡大とgrouping-only routingを検証したが、採用条件を満たすrouteは得られなかった。Gemmaの予算拡大でも誤統合・誤分割が残り、Qwen3.5/Graniteではweak16を全統合、Phi4miniではstrict JSON契約で停止した。

## 検証結果

| route / fixture | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | ---: | --- | ---: |
| Gemma native1024 / weak16 | false | 120 / 0 | true | false | 1 | 3500 / 1082 | 54.154 |
| Gemma native1024 / cross12 | false | 0 / 24 | true | false | 1 | 6119 / 1146 | 61.368 |
| Qwen3.5 native0 / weak16 | false | 120 / 0 | true | false | 1 | 3450 / 232 | 21.325 |
| Phi4mini native0 / weak16 | null | null / null | false | true | 1 | 2824 / 1487 | 91.327 |
| Granite4.2 native0 / weak16 | false | 120 / 0 | true | false | 1 | 2879 / 163 | 19.352 |

全5callsのhelper stopはcompleted。timeout/context overflow0。Phiはhostのstrict JSON検証でinvalid_json、部分割当を成功扱いしていない。Phiの生成内容は保存しておらず、どのJSON不備かは未特定。その他4observationsはunknown/duplicate/missing ID0。metadata未実行。qualification失敗のためrouting側cross12は事前登録通り実行していない。model downloadなし、production helper/source変更なし。helper copyのSwift testsは成功した。

## 考察

固定canonical入力のままnative思考枠を512から1024へ拡大しても、主要failureは解消しなかった。ここから全ての思考予算が無効とは結論しない。別routeも品質条件を満たさず、速いQwen/Graniteをlatencyだけで選定できない。modelとgeneration contractを同時に変更した比較なのでmodel単独効果は不明。採用を見送ったrouteへmetadata生成を実施してもglobal semantic failureは解決しない。

## Next Steps

- 保存済みの未評価3checkpoint（Nemotron、Ministral Instruct、Ministral Reasoning）を同じweak16 qualificationで各1回評価する。既存capability内のgrouping-only routing探索に未評価範囲が残るため。通過時のみcross12以降へ進む。
- weak16のgoldと観測入力の識別可能性を監査する。異なる定数変更が独立作業か一括設定変更かを、diffだけで一意に区別できる根拠があるか確認するため。既存goldの変更や有利なgold選択は行わない。
- 同一入力から矛盾する意図が成立する場合は、追加の信頼できる変更要求・独立goldの必要性を明記する。有限probeを普遍的なarchitecture天井値へ読み替えないため。
