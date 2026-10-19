## 要約
H180-004（8bit）→H180-005（4bit）を最新H180-003 iteration09保存結果と比較。baseline再推論なし。

## 検証結果
推論前。8bit C01→C02、4bit C01→C02、各一巡最大12要求、retry/repair/judgeなし。K/V affine group64を先頭prefillからglobal4層へ適用。rotating20層はBF16。chunk512・同期的eval・4chunkごと＋末尾の同期解放・grammar保持・model/prompt/予算/Goldを固定。KV dtype/shape/bytesを実測。

## 考察
JSON一致・schema/Validator・固定参照partition/category一致・text required meanings/forbidden claimsを分ける。Goldはagent作成未承認の探索用で独立正答率ではない。C02後段は固定group注入。各case1回・過去比較・追加計測負荷・baseline KV bytes未記録が制約。

## Next Steps
helper/Validator/reuse/回数/量子化実適用/入力不一致は当該候補の残り停止し次候補へ。出力差は結果として記録し修正しない。監視10秒途絶・OS計測失敗・OS free<5%・helper RSS>=12GiBで全停止。自動採用なし。
