# H182-020 Spark Low反復ペナルティ1.0/1.1比較

## Goal

他条件を固定しrepetition_penalty1.0/1.1各1回を同じ新アダプターで比較。保存済対照だけでなく1.0も新規に実行する。

## Result / Decision

全2セル・2phase測定終了。214.497秒（約3分34秒）。監視107件全valid、fatalなし。両方Groupingの思考2048でmax_tokensとなり、最終回答0・Category/Text未実行・validator受理0/2。1.1は同一tokenの長い連続反復を抑えたが、回答完了/正解には至らない。1.1の本番採用は確定しない。

## Conditions / gates

Spark Low/grammarあり、H182019 brief prompt/元Category/Text、同じfixed fixture、32K/2048、native KV、temperature1/top_p.95/top_k0/presence0/frequency0/seed144。weight固定、各1回、warmup/retry/repairなし、時間制限なし。生成済Thinkingの直近20token（使用mlx-lmの既定値）だけにsign-aware repetition penalty。入力promptを除外、EOS1/think開始3/終了4 logitsは元値を保持し、終了後はpenaltyをかけず従来grammarへ接続。20は両群共通で可変要因はpenaltyのみ。
非推論Metalテストで1.0が無変更、prompt除外、20token窓、境界/EOS保護、response無変更を確認。両群の実requestと有効wire messagesは完全一致。source/weight/fixture/Git不変。1.0の実出力token列はH182019 Lowと全2048token一致し、既知の反復を再現。本文/token IDsはprivate0600保存し、公開報告では統計と経緯の要約のみ。

## Repetition / completion

| 指標 | penalty1.0 | penalty1.1 |
| --- | ---: | ---: |
| Thinking出力tokens | 2048 | 2048 |
| 同一token最大連続数 | 1631 | 2 |
| 末尾の同一token連続数 | 1631 | 1 |
| Thinking終了marker | 0 | 0 |
| 最終回答tokens | 0 | 0 |
| 最終Grouping採点 | 不可 | 不可 |

1.0は3つの目的候補を認識した後、417token目以降の末尾1631token（約79.64%）を単一token反復に使う。1.1ではこの症状は見られず、実装とテストの対応や3目的の組合せを検討する文章が継続した。しかしREADMEの独立性、group labelの意味・任意性、許容label数と実group数などを繰り返し再検討し、最終JSONへ移らなかった。これは本文からの定性的要約で、内部状態や根本原因の確定ではない。
同一token反復の抑制と、意味上の再検討の反復/回答遷移の失敗を区別する。ペナルティ1.1の効果は今回の症状に対して観測したが、出力が完成していないため精度向上を主張しない。

## Timing

| penalty | セル秒 | prefill秒 | decode秒 | emitted token/s |
| --- | ---: | ---: | ---: | ---: |
| 1.0 | 102.34 | 23.53 | 74.06 | 27.65 |
| 1.1 | 109.81 | 25.45 | 81.10 | 25.25 |

1.1のpenalty用CPU区間計測合計は約0.246秒。ただし遅延評価を含み得るのでGPUへの因果的総負荷を示す数字ではない。n=1・出力token列も異なるため、全体時間差をpenalty実装だけのオーバーヘッドと断定しない。

## Resources

helper RSS最大2.370GiB、MLX peak3.827GiB（別指標・加算不可）。system CPU busy平均43.05%、GPU範囲[0, 100]%、thermal {'nominal': 53, 'fair': 54}、swap MiB開始/終了/最大[2382.94, 2382.94, 2382.94]、free%[21, 62]、同時adapter最大1。
2秒監視。温度実数/消費電力/プロセス別GPUは未取得。load/template/grammar/prefill/decode/host/CPU user-system/KV配列/本文保存負荷の記録を保持。

## Limits / Next

単一fixture各1回。新ペナルティで単一token反復は消えたが一般再発率/実用精度は未評価。次の候補はgroup labelが意味を持たない識別子であること、許容label数は必要group数ではないこと、根拠が揃ったら確定して回答することの明確化。ただし未実装・未試行。ペナルティ増加、予算増加、強制思考終了は追加しない。元データを保持。
