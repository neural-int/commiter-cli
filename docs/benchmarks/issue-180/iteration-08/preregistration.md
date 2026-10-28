## 要約
H180-002を同じ実行ファイルで1巡再実行し、2秒間隔でCPU/GPU/熱状態を観測する。

## 検証結果
推論前。C01→C02各1回・最大6要求、timeoutなし、retry/repair/judgeなし。H180-001再推論なし。入力・model/予算・ソース等はiteration07と同一。実行前にpowermetricsサンプルの更新と有効性を確認した。

## 考察
摂氏温度とプロセス別GPU時間はこの環境のpowermetrics出力には存在せず、未取得として扱う。thermal pressureを温度値に置換せず、全体GPU稼働率から個別プロセスへGPU使用率を割り当てない。観測自体の負荷と過去baselineの非同時計測は制約。underflow表示は記録し、不正値は除外する。

## Next Steps
有効サンプルが10秒途切れたら自分が起動したbenchmarkをキャンセルし、追加生成しない。helper/validator/reuse gateはiteration07と同一。実行前後の不変性と実測を報告し、一巡で終了。
