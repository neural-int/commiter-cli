## 要約

固定した式依存evidenceは未使用3件で追加価値を示さず、既知の変数経由assertionにFS回帰を起こした。候補No-Go、B No-Goを保持する。構文coverage修復とsemantic成功を区別する。

## 検証結果

| 範囲 | 条件 | exact | FM | FS |
| --- | --- | --- | --- | --- |
| fresh3 | A-only対照 | 1/3 | 12 | 0 |
| fresh3 | expression追加 | 1/3 | 12 | 0 |
| regression2 | A-only対照 | 1/2 | 3 | 0 |
| regression2 | expression追加 | 1/2 | 0 | 1 |

全10call completed/accepted/complete。input12632/output758tokens、累積139.539秒。fresh-variableは両方exact。fresh-siblingとfresh-reassignedは両方全merge。既知diagnosticはFM3→0でexact化したが、既知variable-mediatedはexact→FS1で不一致化した。fixture/gateは60acf85、runnerは8838418で事前固定し、途中修正/retry/repairなし。

今回のA-only対照は同じrepository graphとexpression container/metadataを持ち、expression観測配列だけ空。旧A-onlyの結果と同一条件とは呼ばない。goldは評価側のみ。model/helper/schema/system/context/output/time予算は既存固定。raw全membershipを保存した。

## 考察

式依存の構造観測で反例を区別できても、モデルが独立fixtureのintentを正しく分割する追加signalにはならなかった。既知1件の改善だけを一般化へ転用できず、FS回帰も事前条件に反する。この候補に合わせたmetadata/prompt変更や出力予算増加を次の主解決策にはしない。

2file syntheticの比較であり、親multi-file品質/実repository一般化を証明しない。局所定義のunsupported、65unit拒否、new file/rename/history/costは未達。production4file制約とC未着手を維持する。

## Next Steps

- この式依存候補の改善ループは終了し、次の専用worktreeでは親/Bの現在の未達条件と過去source別ablationを照合する。局所的観測追加を続けてもindependent改善がなく、同じ候補の調整を繰り返さないため。
- 未評価の独立evidence sourceまたはarchitecture差が実際に残るかを具体化し、残らなければgoal.mdの天井条件に照らして停止判断を記録する。成功条件を狭めず、Cへ不合格入力を渡さないため。
- 判定に必要な外部入力が残る場合は不足内容を特定する。候補No-Goだけをgoal完了と扱わない。
