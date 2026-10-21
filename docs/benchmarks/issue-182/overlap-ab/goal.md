# H182-007 先読みとCPU状態更新のA/B比較（準備完了・未実行）

## 目的

ユーザー指示に従い、改善側ハーネスを準備した。生成計測はまだ行っていない。モデル・prompt・判定規則を変えず、同期位置の変更だけを比較する。速度改善は未確認。

## A/B実装

同一Releaseバイナリで研究専用環境変数COMMITER_182_PIPELINEを切り替える。A=eager（既定）、B=overlap。通常CLIは常に既存方式。Bは研究config・benchmark・進捗processor有効時のみ選択可能で、不明なmodeや進捗なしのBは拒否する。

AはdidSample内で即座にitem(Int.self)を呼び、既存状態を更新する。BはOverlappedTokenProcessorで既存processorを包み、didSampleではMLXArrayを1個だけ保留する。次のprocess(logits:)でasyncEval(logits)を呼び、次のforwardを実行予約してから保留トークンを既存didSampleへ渡す。状態を更新した後に既存process(logits:)を呼び、最新のThinking/grammar制約を適用する。次トークンのsamplingより後に状態更新を遅らせない。

CPU観察は毎トークン保持し、ログ頻度も既存と同じ。Thinking中にJSON grammarを追加しない。自然終了・Thinking予算・EOS・最終JSON制約・異常終了の判定規則は維持。追加評価の区切りや実行予約負荷があり、GPU計算との重なりや高速化を保証しない。

MLX TokenIteratorの1トークン先読みを前提に、呼び出し元へ返すトークンはその前に状態更新される。最後の未返却先読みtokenはcommitせず破棄する。キャンセル時も未返却分をflushしない。Aの途中カウンタは先読み分を含む場合があり、中断時のカウンタだけを同じ意味で比較しない。最終応答とカウンタを別に監査する。GPUの既予約forwardはキャンセルやEOS時にも残り得るため、依存の既存stream終了・同期処理を維持する。

## 固定条件と実行予定

Gemma簡潔化C01/C02、Qwen4B現行C01/C02。Thinking ON・各phase1024、最終出力予算/時間制限なし。weight、template、sampling、seed144、KV8/group64、schema、validator、固定partition、進捗/OS監視はH182-006と同じ。A/Bは同一binaryと同一mlx.metallibを使用。Aも新規に測定する計画で、過去速度は対照値として流用しない。

8セル・最大24phase、各arm1回、warmup/retry/repairなし。各モデルC01はA→B、C02はB→A。GPU生成は直列。RSS12GiB・監視失敗・進捗300秒なしの安全停止を維持。生成tokensと時間の両方を保存し、Thinking/JSONの内訳が取れない既存metricsだけで区間別速度を推定しない。今回の最小変更では新しい詳細タイマーを追加していないため、区間別性能の追加計測は別判断とする。

## 準備検証

Releaseで44テスト成功。新規4テストは予算境界とJSON制約、自然終了と早期EOS、未返却tokenの破棄、processorコピーの独立性をGPU配列で確認。最初の実行はtest bundleにmetallibがなく停止した。同じ既存metallibをtest実行先へ配置し、ソース変更なしで再実行して成功した。失敗ログも保持する。

全8セルで生成なしtoken監査に成功し、Debug/Release前回の各採用promptと要求・Category/Text実token列・KV構成が一致した。モデル生成、A/B速度測定、実モデルのstreamキャンセル検証は未実施。準備成功と性能・品質確認を区別する。

## 成果物と再現

原本は/private/tmp/issue182-overlap-ab/。prepareが8セル契約とauditを生成し、runで登録セルを測定する。既存成果物を上書きしない。

- tools/benchmark182/overlap_compare.py prepare：生成なし準備（完了済み）
- tools/benchmark182/overlap_compare.py run：A/B生成測定（未実行）

公開raw/は契約、監査、capture、source snapshot、Releaseテスト成功/失敗ログを含む。weight/binary/metallib/build cacheはhashで識別する。メール形式文字列とホーム絶対pathを公開コピーで置換し、SHA256SUMS.txtは公開コピーhashを記録する。契約内hashは原本の値。公開scriptを再利用する際はpath解決が必要。

## 次のゲート

実モデルA/Bで形式・境界・Grouping・文章品質・生成量・時間・メモリを比較する。Bを速い方式として採用しない。まず同条件で正しく完了することを確認し、出力差や異常があれば保持して報告する。Issue #182全体のGoalは未完了。
