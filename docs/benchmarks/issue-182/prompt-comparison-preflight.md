# H182-004 プロンプト比較の事前監査

2026-10-21。対象はGemma ON、Qwen3.5-4B ON、Qwen3.5-2B OFF。各モデルの現行/簡潔化をC01/C02で各1回測定する。12セル、最大36 phase要求。warmup/retry/repairなし。探索比較であり、反復による安定性や16ファイル性能は判定しない。

## 確認結果

- 3モデルのweight/config/tokenizer/template計15ファイルはH182-002の実行契約SHA256と一致。
- 元の履歴から復元したfixtureでcaptureし、モデル名を除く3phaseの要求がH182-002と一致することを各モデル/C01/C02で確認。
- 簡潔化はprompt-candidate.mdの4ブロックを使用。Git facts、履歴本文、JSON schema、finalized_types、role列、model、生成profile、context ceiling、output設定を保持するassertionを全要求で通過。
- 全12セルで実際のCategory/Text token列をSwiftから取得し、共有prefix長とKV layer構成を監査。prefix長は各セルごとに再計算し、旧値を流用しない。
- 両群は同一Swift Debug binary、seed144、既存モデル別sampling、KV8bit/group64。Thinking/出力予算は無制限、120秒制限なし。今回はThinking1024の終了誘導を混ぜない。
- GemmaもThinking ONに固定。Qwen2B ON、Qwen4B OFF、Gemma OFF、Ministralは今回の対象外。
- 現行群も再測定し、過去の速度値と直接比較しない。C01は現行→簡潔化、C02は簡潔化→現行の順序で順序偏りを抑える。GPU生成は並列化しない。
- 同じ進捗processorを両群に適用。初回/64 tokensごとの進捗、要求の変換前/実送信内容、完了メトリクス、OS監視を保存する。
- Swift39テスト成功、Python構文とgit diff --check成功。通常CLIのpromptを変更せず、研究専用proxyで適用する。

## 停止と評価

RSS12GiB、監視失敗、token進捗300秒なしは安全停止し、中断を成功に含めない。インフラ失敗で結果未生成ならsuiteを停止し、retryしない。生成要求前にruntime/asset/設定/監査のhash一致を再確認する。

Groupingの実出力を保存した後、前回と同じ固定partitionでCategory/Textを評価する。後段受理をend-to-end成功と扱わず、Goldは探索用で独立人手承認なしと明記する。文法/validator成立とsummaryの文章品質を分離する。

## 成果物

実行契約、全12セルの実token列・実送信予定要求、入力不変条件の監査は `/private/tmp/issue182-prompt-comparison/` に保存。実行は `tools/benchmark182/prompt_compare.py run`。同じ実行契約の再実行・上書きはしない。モデル採否、簡潔化案の採用は結果を確認するまで保留する。
