# H182-021 Sparkラベル意味・確定指示の明確化

## Goal

Low/grammarあり/penalty1.1のH182-020を対照に、Groupingだけへラベルは無意味な識別子、全許容labelを使う必要なし、根拠が揃ったら確定してJSON出力という説明を追記する。対照は保存結果を使用し再測定しない。

## Result / Decision

1セル・1phaseの測定を終了。セル103.442秒、監視込み104.566秒、53監視サンプル全valid、fatalなし。今回もThinking2048/終了marker0/最終回答0でmax_tokens。Grouping採点不能、Category/Text未実行。単一token最大連続は2で、長い単一token反復はない。プロンプト追記だけで回答へ移れるという仮説は今回未達。追加試行・条件変更・採用なし。

## Thought analysis (summary only)

本文ではラベルが任意の識別子であること、ラベルを余らせてよいことを説明に沿って扱っていた。README、UI実装+テスト、メール正規化実装+テストの3目的を繰り返し列挙し、正解partitionに相当する具体的なfile→label候補も一度整理している。しかし再び目的の独立性や割り当ての確認へ戻り、自然なThinking終了markerを生成する前に上限へ達した。
したがって、説明が全く届いていない状態ではない一方、その説明に沿う文が本文に出たことを、回答完了や正解JSON生成と同一視しない。生のThinking/会話/token IDsはprivate原本だけに保持し、公開報告は要約のみ。

## Fixed conditions / verification

Spark Low/JSON grammarあり、penalty1.1・生成Thinking直近20tokenのみ・EOS/marker免除・response無penalty、32K/2048、native KV、sampling1/.95/topk0/presence0/frequency0・seed144、固定revision/fixture、各1回・時間制限なし・warmup/retry/repairなし。
Go plannerと元Category/Textの指示は不変。実requestは対照と一致。モデル向けwireはGrouping systemへの追記だけの差で、user入力・Schemaも一致。非推論3phaseのwire監査でCategory/Text無変更を確認。期待する3群・正解割り当てはプロンプトへ注入していない。source/weight/fixture/Git不変、本文保存0600、counts一致を確認。

## Comparison

| 指標 | H182-020 penalty1.1 | H182-021 ラベル説明追加 |
| --- | ---: | ---: |
| Thinking tokens | 2048 | 2048 |
| 同一token最大連続 | 2 | 2 |
| Thinking終了marker | 0 | 0 |
| 最終回答tokens | 0 | 0 |
| セル秒 | 109.81 | 103.44 |
| prefill秒 | 25.45 | 24.03 |
| decode秒 | 81.10 | 74.85 |
| emitted decode token/s | 25.25 | 27.36 |

今回の実inputは6832tokens（対照6709）。文脈容量には十分収まり、入力欠落・文脈上限到達による停止ではない。生成本文は異なり、各1回・別時刻の測定なので速度差を安定効果とは断定しない。

## Resources

helper RSS最大0.944GiB、MLX peak3.827GiB（別指標・加算不可）。system CPU busy平均48.46%、GPU範囲[17, 100]%、thermal {'nominal': 53}、swap MiB開始/終了/最大[2382.94, 2480.81, 2480.81]、free%[19, 63]、同時adapter最大1。
load/template/grammar/prefill/decode/host/CPU user-system/KV dtype・bytesとprivate本文を保持。温度実数・電力・プロセス別GPUは未取得。

## Limits / Next

この入力で正解相当の候補を列挙しても最終回答へ確定しない症状が継続した。次に検討するなら、プロンプト追記の繰り返しではなくThinkingと回答の予算分離や、モデルの終了挙動の比較を別の実験にする候補がある。ただし今回は未実装・未試行。増量すれば必ず正解に至るとはしない。原本と保存対照を保持する。
