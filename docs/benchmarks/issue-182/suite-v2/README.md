# H182-002: モデル別H180-004対応ハーネス（準備）

7ケースの個別JSONを実行設定の正本とする。Phiは対象外。Instract表記は配布モデルの正式名称Instructへ対応させた。

Gemma ON/OFF、Qwen4B ON/OFF、Qwen2B OFF、Ministral Instruct、Ministral Reasoning。
Gemma 1.0/.95/64、Qwen4B .6/.95/20、Qwen2B 1.0/1.0/20、Ministral Instruct .1/.95/20、Reasoning .7/.95/20。seed144。

## 引き継ぐ条件

H180-004のC01/C02のmaterialized Git差分、ローカル保存fixtureの作業履歴、Gold、三段階schema・validator・日本語、計画内model/tokenizer/grammar準備保持、8bit affine group64、chunk512の同期eval、4chunkごと＋末尾の同期・解放を保持する。C16の新しい合成fixtureには置き換えない。

H180と同じく実Groupingを保存・検証した後、Category/Textには固定参照partitionを注入する。これは既存Gemmaの比較条件を揃えるためであり、end-to-end成功として集計しない。Grouping品質と参照partition上のCategory/Text品質を別々に評価する。Goldはagent作成の探索用であり人間承認済みの独立Goldではない。

## モデル固有の対応

| ケース | thinking制御 | 量子化対象 | native保持 |
| --- | --- | --- | --- |
| Gemma ON/OFF | enable_thinkingを全フェーズで明示。ONはchannel100/101、OFFは直ちにJSON制約 | full attention4層 | rotating20層 |
| Qwen4B ON/OFF | enable_thinkingを明示。ONはtemplate内のthink開始と自然な終了、OFFは直ちにJSON制約 | full attention8層 | Mamba/GDN24層 |
| Qwen2B OFF | enable_thinking=false | full attention6層 | Mamba/GDN18層 |
| Ministral Instruct | 通常回答にJSON制約 | attention26層 | なし |
| Ministral Reasoning | THINK開始・終了tokenをtokenizerから解決してJSONへ遷移 | attention26層 | なし |

Ministralの交互role制約に対応し、連続userメッセージのみ内容・順序を保ち結合する。ReasoningにはTHINK形式を使う指示をsystemへ追記する。共有prefix長は固定IDやGemmaの長さを流用せず、各ケース・fixtureの実token列から監査する。実行時にもprefix完全一致とcache offsetを再確認する。

QwenのMamba/GDN cacheは、固定長の再帰状態を持ちますが、使用中のMLX実装では`MambaCache.offset`をprefill token数として更新しません。そのため、再帰層は2個の保存状態配列を確認し、attention層は従来どおりoffsetとprefix長を照合します。

## 無制限の意味

研究用helperではGenerateParameters.maxTokens=nil、NativeThoughtBudgetState.limit=nil。768/512/768や512/384の予算は適用しない。Go overlayで計画deadlineをWithCancelへ変更し、test.timeout=0、subprocess timeoutなし。通常CLIの設定は変更しない。ユーザー中止・OS資源/監視異常による停止は残す。

従来の入力context ceiling64Kと、IPCのrequest1MiB/response4MiBは保持する。これは無限のcontext・無限の通信容量を保証する構成ではない。異常な長文が4MiBに達すると生成/通信は停止し、失敗として記録する。

## 準備・監査と実行の分離

`tools/benchmark182/suite_v2.py prepare`は専用Go overlay/binaryと非推論captureを作る。`audit`はモデルを読み込みtemplate/tokenizer・prefix・計画cache型と量子化層数を検査するが、出力tokenのsamplingやprefix prefillは行わない。`run`のみ生成する。

今回の準備artifact: `/private/tmp/issue182-suite-v2/`。native helper: `/private/tmp/issue182-suite-v2-runtime/commiter-mlx-helper`。

実行時にモデル・helper・overlay・設定のハッシュを固定する。C01/C02各1回、計14cycle/最大42phase要求、retry/repair/judgeなし。Gemma ON/OFFも新規に測定し、過去スコアを実測baselineとして流用しない。監視はMach API CPU、既存のGPU/メモリ/swap/電源等を5秒ごとに収集し、15秒の有効監視途絶またはhelper RSS12GiB以上で停止する。温度・消費電力・プロセス別GPUは未取得と明示する。

非推論の監査成功は、実生成におけるcache再利用の数値同値性・自然なthinking終了・品質・速度の保証ではない。それらは比較実行時の確認事項として残す。

## 実行ログ

実行で得たrequest/response、cell結果、監視記録、実行契約は `raw/` に保存する。公開用コピーではメール形式文字列を `redacted@example.invalid`、絶対ホームディレクトリパスを `$HOME` に置換する。ファイル一覧とSHA-256は `raw/SHA256SUMS.txt` を参照する。環境依存のコンパイル済み `evaluation.test` は含めず、対応するGoソースを含める。
