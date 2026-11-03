# Harness 004 revision 3：実token判定と12K境界

3-Phaseのコンテキスト枠を全フェーズ65,536 tokensへ固定する。Groupingで小さい枠を選び、その枠をCategory／Textへ引き継ぐ処理を削除した。各フェーズの出力予約4,096 tokensを含めて上限を検査するため、最終入力は最大61,440 tokensとなる。

| 条件 | 動作 |
|---|---|
| 実入力が12,000 tokens以下 | 毎chunkに追加していたGPU待機・未使用allocator cache解放を省く |
| 実入力が12,000 tokens超 | 従来の毎chunk同期・未使用allocator cache解放を行う |
| 入力＋出力予約が65,536以下 | 生成可能 |
| 必須入力＋出力予約が65,536超 | 生成せずErrTooLargeで停止 |
| 任意の履歴候補が上限を超える | 当該最終応答の記録全体を省略 |
| 計数自体が失敗 | 0 tokens扱いや黙った履歴省略をせず停止 |

12Kは12,288ではなく**12,000**を指す。12,000ちょうどは軽量側、12,001は保護側。以前の実入力12,284 tokensの条件は保護側になる。12,000が最適な境界であることや、速度改善は、この実装だけでは証明されない。

## 最終入力と計数の一致

Goは同じhelperへ`count_only: true`を送り、生成と同じLLM表示、System／Phase指示、出力Schema、モデルのchat templateを適用したtoken数を受け取る。外側IPCは正規JSONを維持する。モデル用の実改行表示も生成・計数で同じBackendの処理を通す。仮の固定テンプレート予約やUTF-8バイト数を入力token数として使わない。

モデル・tokenizerは同一プロセスに保持する。計数要求は文法準備、prefill、sampling、prefix snapshotの作成・更新、フェーズ進行を行わない。候補の計数結果は当該フェーズ内で同一最終メッセージに限り再利用する。最終採用入力を生成時にもtokenizeし、Swift側のTokenBudget検査を残す。計数要求はResult.Callsの推論3回へ加算しない。

履歴はfinal-only、選択順・重複排除・機密情報除外を維持する。安全な非空の最終応答が収まらなければ記録全体を落とし、ユーザー依頼へ戻さない。Groupingで採用した共有データをTextで再利用する。Categoryは従来どおり独立入力であり、各最終入力を個別に検査する。64K以内なら、後段の入力がGroupingより多くても、旧16K/32K枠のために停止しない。

保護判定には、キャッシュ再利用で省いたsuffixだけではなく**再利用するprefixを含む全入力**を使う。Groupingの手動prefix準備と通常のprefillの両方へ同じ判定を適用する。prefix状態を確定させるための`eval(cache)`と、必要なKVコピーは保持する。

## 範囲

通常3-Phaseの4ファイル上限、サイクル120秒、summary48文字、各phase出力4096、Thinking OFF、文法制約、native KV、sampling／seed、型別prefix再利用を維持する。既存の`llm.context`／`llm.max_context_tokens`による段階選択は旧single-pass経路だけに残し、3-Phaseの枠選択へ使わない。

GoのIPC最大1MiB、応答4MiB、履歴の保持バイト上限は資源制限として維持する。コンテキストtoken推定とは別の制限である。通常3-Phaseの前で旧Prepareが行っていたバイト予算による拒否・縮約は迂回する。旧Renderer・制約オブジェクト自体の撤去、履歴取得の統一、旧Planner廃止は[Issue #187](https://github.com/neural-int/commiter-cli/issues/187)の別スコープとする。

## 検証

- Go：64K固定、64KiBを超える表示でも実token数で受理、後段30K入力、61,440／61,441境界、任意履歴の採否、計数失敗時停止、Grouping／Text共有データ不変。
- Go–helper：計数を各phaseで複数回行っても同じプロセスで正しいphase順を維持し、推論を3回だけ実行する。実改行表示とホスト元データ保持、計数応答の型・失敗分類も検査する。
- Swift：11,999／12,000／12,001／12,284の保護境界、64K内の4,096出力予約を検査する。

これらの構造検査と、速度比較・多ファイル意味品質の再評価は別である。

### ローカル確認結果（2026-11-03）

Swift Release helperで、保存済みのSHA固定モデルを使った公開合成1ファイルの疎通を各1回実行した。App Serverや実Gitリポジトリへの書き込みは行っていない。計数結果は、生成時にhelperが計測した全入力token数と全phaseで一致した。

| モデル | Grouping入力tokens | Category入力tokens | Text入力tokens | Text再利用tokens | 接続・最終validator | テスト本体経過 |
|---|---:|---:|---:|---:|---|---:|
| Qwen3.5-4B 4bit | 733 | 702 | 701 | 191 | 全3phase完走・受理 | 14.99秒 |
| Gemma E4B 4bit | 752 | 725 | 720 | 200 | 全3phase完走・受理 | 12.58秒 |

経過時間はテスト本体の観測値であり、順序・冷却・反復を揃えた性能比較ではない。短い合成入力での疎通であり、12K付近の実入力におけるメモリ安全性・短縮効果や、保存7/9/11ファイルfixtureの意味品質を新たに実証したものではない。

Swiftの既存57件に加え、今回の境界・出力予約2件が成功した。初回の既定SwiftPMビルドは、このMacのMetal Toolchain未導入で失敗した。依存の固定revisionは変更せず、SwiftPM nativeによるReleaseビルドと同じ固定依存用の既存Metalライブラリで全テストと上記疎通を実行した。追加依存やCI設定変更、CI成功の主張はない。

最終状態で`go test ./...`、`go vet ./...`、CLIのビルド、`git diff --check`が成功した。旧benchmark146のmock／oracle／templateアダプターにも計数インターフェースを接続し、実測バックエンドでは実計数を必須にした。合成値を実測token数として記録しない。新しいRelease helperをローカルの`dist`にも配置した。ソースのcommit／push、PR更新、mainへのマージ、製品リリースはこの作業では行っていない。
