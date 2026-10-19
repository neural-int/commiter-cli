## 要約
候補H180-002（grammar準備データ保持）を採用済みH180-001のiteration05と比較する。H180-001を現行baselineに維持し、候補を自動採用しない。

## 検証結果
推論前。C01→C02各1回・最大6要求、retry/repair/judgeなし、検証のみtimeoutなし。model/prompt/Gold/予算/chunk512/一括eval(cache)/同期・解放頻度/KVコピーを維持。baseline再推論なし。Groupingで不変データを構築し、Category/Textで再利用する。各phaseのschema constraint/matcher/native stateは新規。再利用フラグは各計画false/true/trueを要求する。

## 考察
保持対象はGrammarTokenizerが所有する語彙表と空白/停止token集合等。生の語彙配列を重複保持せず、mutable grammar matcherは保持しない。単発経路や別計画には持ち越さない。n=1・過去baseline比較のため一般的効果は未評価。

## Next Steps
helper/validator失敗・再利用パターン不一致時は残り停止。出力差は記録し救済しない。時間/再利用/最終JSON/メモリ/fixture不変を監査して一巡で終了し、③を追加しない。
