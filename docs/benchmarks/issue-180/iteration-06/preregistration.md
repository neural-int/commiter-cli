## 要約
ユーザー指示に従い、元方式（iteration03改善済みD-T、層別eval(entry.state)）のbaselineを再実行する。一括評価候補や過去の測定を上書きしない。

## 検証結果
推論前。C01-D-T→C02-D-T各1回、最大6生成要求、timeoutなし、retry/repair/judgeなし。元方式の保存済みhelperをSHA一致確認して使用し、runnerとMetalは候補と同一bytes。model/prompt/Gold/生成・thinking予算/chunk512/同期・解放頻度を維持する。

## 考察
同じ検証環境での追加baselineだが、各1回・別時点の逐次測定である。旧helperと候補helperはビルド時点も異なる。環境起因の差を完全に排除した因果推定とは扱わない。

## Next Steps
helper/validator失敗時は残り停止。入力と出力JSONの一致、KV再利用、時間内訳とmemoryを確認し、追加生成や別条件は行わず一巡で終了する。
