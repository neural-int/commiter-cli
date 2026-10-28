# H182-003 Ministral公式tokenizer再検証

## 要約

4ケースの再測定を完了した。InstructのC01/C02は前回と同じunresolved_breaking_evidenceで停止。Reasoning C01は3フェーズを正常に終了しvalidatorがplanを受理した。Reasoning C02はGroupingが探索Goldと一致したが、Categoryのunresolved_breaking_evidenceによりTextへ進まず停止した。plan受理は1/4、実Groupingの探索Gold一致は4/4。後段は固定partition方式であり、end-to-end成功率ではない。

Reasoningの全5phaseでTHINKを自然終了し、有効なJSONへ移行した。前回C01のnative_eos_before_closeは今回観測されなかった。ただしC01 Textは8334 Thinking tokens、C02 Categoryは7821 Thinking tokensと長く、回答品質・待ち時間の課題は残る。モデル採用判断は行わない。

## 実行条件

H182-002と同じweight revision、sampling、seed144、KV8bit/group64、C01/C02、schema/validator、固定参照partitionによる後段比較。出力・Thinking予算無制限、120秒cycle制限なし。簡潔化プロンプトは未適用。

公式mistral-common 1.8.6による入力token列をSwift/MLXへ渡す。Reasoningは公式system promptを併用し、最終JSONを明示する。開始THINKを強制せず自然に生成させる。個々の変更の因果効果を単独で測る実験ではない。

## 検証結果

| ケース | 状態 | cycle秒 |
| --- | --- | ---: |
| Instruct C01 | Categoryでunresolved_breaking_evidence | 22.60 |
| Instruct C02 | Categoryでunresolved_breaking_evidence | 289.68 |
| Reasoning C01 | 3フェーズ正常終了、plan受理 | 986.41 |
| Reasoning C02 | Categoryでunresolved_breaking_evidence、Text未実行 | 1645.48 |

Reasoning C01 Groupingは2283 tokens（Thinking 2224）、生成148.78秒、約15.34 tokens/sで正常終了。EOS前にTHINK終了とJSON成立を確認した。後段は固定partitionであり、これをend-to-endの成功とは扱わない。

Categoryは1492 tokens（Thinking1473）、生成94.26秒、約15.83 tokens/s。Textは8379 tokens（Thinking8334）、生成720.05秒、約11.64 tokens/s。全3phaseでnativeThoughtForcedClose=false。summaryは「修正:静的 import の URL を Next.js で正規化し、2つの component」で終わり、validator受理を文章品質の保証と解釈しない。

Reasoning C02の実Groupingは4ファイル1群で探索Goldと一致。前回の3群から変化したが、公式system prompt等も含めた変更でありtokenizer単独の因果効果を示さない。生成3253 tokens、Thinking3176 tokensで正常終了した。

### Phase別実測

G/C/TはGrouping/Category/Text。各値は生成tokens（Thinking tokens）/ decode tokens/s。Thinkingは生成数の内数で、既存helperと同じprefixカウンタを使用する（開始markerを含み、終了markerを含まない）。tokens/sはhelperのruntimeGenerationSecondsから計算し、prefillを含めない。

| ケース | G | C | T |
| --- | ---: | ---: | ---: |
| Instruct C01 | 58 (0) / 19.38 | 28 (0) / 19.43 | — |
| Instruct C02 | 74 (0) / 10.06 | 27 (0) / 8.85 | — |
| Reasoning C01 | 2283 (2224) / 15.34 | 1492 (1473) / 15.83 | 8379 (8334) / 11.64 |
| Reasoning C02 | 3253 (3176) / 8.23 | 7850 (7821) / 8.39 | — |

実際のモデル生成は9 phase要求、計23444 tokens（Thinking23028）。推論前に終了した2起動は含めない。全9要求の完了応答を取得した。Reasoningの全5phaseでnativeThoughtForcedClose=false。InstructではThinkingを有効にしていない。

### 前回との比較

| ケース | H182-002 | H182-003 |
| --- | --- | --- |
| Instruct C01 | Categoryでunresolved、24.92秒 | 同じ停止、22.60秒 |
| Instruct C02 | Categoryでunresolved、276.20秒 | 同じ停止、289.68秒 |
| Reasoning C01 | Groupingでnative_eos_before_close、46.11秒 | 3phase正常終了・plan受理、986.41秒 |
| Reasoning C02 | GroupingはGold不一致。固定partition後のplan受理、929.53秒 | GroupingはGold一致。Categoryでunresolved、1645.48秒 |

Reasoning C01のsummaryは48文字、scopeは16文字で両方上限に達した。summary末尾が「2つの component」であり、形式受理を文章品質の保証としない。C02 Categoryはtype=feat、breaking_evidence_ref=unresolved。未解決判定をnoneへ書き換えたり、Textを追加実行したりしていない。

### OS監視

元run66件、復旧後run527件、計593件の監視サンプルがすべて有効。安全停止・monitor failure・runtime hash driftはなし。最低memory free表示は元run22%、復旧後25%。2つのrunの実行時間は326.57秒と2636.52秒であり、途中の復旧作業を含む連続したwall timeとして合算しない。温度や電力の実測値とは扱わない。

## 環境障害と復旧

Instruct終了後、元の一時fixtureのrepos.jsonとGit管理情報が失われ、Reasoningの最初の2起動は推論前に終了した。これらをモデル失敗や生成試行に数えない。旧runの失敗ログは保持する。

履歴から別directoryにfixtureを復元し、C01/C02×3 phaseの要求内容が前回とモデル名を除き完全一致することを非推論captureで確認した。最初の復旧captureはText向け環境変数が未設定で一致しなかったため、正しい条件で再captureし全6件一致を確認した。推論のretry/repairは行っていない。完了したInstructは再実行していない。

## 成果物

- 元run: `<PRIVATE_TMP>/issue182-ministral-retest/`
- 復旧後Reasoning: `<PRIVATE_TMP>/issue182-ministral-reasoning-recovered/`
- 専用runtime・asset manifest・復旧一致audit: `<PRIVATE_TMP>/issue182-ministral-runtime/`
- ハーネス適合性と再現手順: `README.md`
- 公開用の全テキスト実行ログ・要求/応答・監査・実行契約・source snapshot: `raw/`。元ログは保持し、公開コピーではメール形式文字列と絶対ホームパスを置換した。`raw/SHA256SUMS.txt` は公開コピーのchecksumであり、実行契約内の実行時原本hashとは区別する。

## 開発用チェックとGoal監査

新規scratch directoryでSwift Debug buildと39テスト成功。全語彙のbyte照合、両モデル×C01/C02のCategory/TextについてSwift実入力と公式token列の完全一致、Python構文検査、git diff --check成功。

実行後、runnerがresult未生成のインフラ障害でも次セルへ進む挙動を修正し、以後は停止させる。選択したcase数に応じて要求上限も算出する。これらは記録済みモデル結果を変更しない。各runで実際に用いたrunnerのsource snapshotは保存し、実行契約のSHA256との一致を確認した。

この小規模単発比較でモデルを採用しない。16ファイル評価・独立承認Goldは未実施。Issue #182全体のGoalは未完了。Gemmaの現行/簡潔化比較、続くQwen4B ON/Qwen2B OFFの比較も未実行。

## Next Steps

今回の4ケースは終了し、失敗phaseのretry/repairや追加生成は行わない。次のプロンプト比較とThinking予算比較はfollow-up-plan.mdに従い別契約で実行する。Qwen2BはOFFのみ。今回確認したMinistralの自然終了を、有限予算での終了誘導対応の証拠と解釈しない。
