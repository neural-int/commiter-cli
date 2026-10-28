# GroupingへのcommandExecution補助情報追加（2026-10-25）

## Goal / Decision

採用仕様をIssue #183へ先に記述し、コマンド本文での現在変更ファイルへのパス言及をGroupingだけへ追加した。編集証拠・ブロック対応は変更せず、Category/Textへは追加しない。実App Serverからの取得、変換、3フェーズhelper受信までのE2E成功。モデル推論・Grouping精度評価は未実施。

採用仕様: https://github.com/neural-int/commiter-cli/issues/183#issuecomment-6395030169
本文を再取得しローカル仕様と一致確認済み。詳細はspec.md。

## Implementation

- historyattribution/history.go: Collect経路だけでcommandExecutionの本文・cwd・状態・終了コード・元item順序を保持。対象パスへの言及があるTurnはfileChangeがなくても保持。出力は取得応答に存在しても抽出しない。
- historyattribution/commands.go: パスのリテラル照合。正規表現の特殊文字を解釈しない。境界を確認し、別prefix/接尾辞/basenameの誤一致を拒否。repo相対/./はcwd=rootのみ。絶対パスは対象rootに対応するもののみ。個別コマンドの意味解析・実行はしない。
- pipeline.go: シェルだけのTurnにもHを割り当て、thread/Turn/itemをhostで保持。元の照合済みHを再利用。原文が同じでも別itemは別S。同一キーの競合は取得不備とする。
- planning/grouping_history.go: agent_command_execution_contextsの必須配列を追加。hostファイルIDを既存のF対応表で変換。Hをブロック候補とコマンド参照の和集合として保持し、定義は重複させない。
- 補助コマンドは既存tier内でのみ予算化。超過時は後ろからコマンドを丸ごと省略し、不要になったシェル専用Hを除く。Grouping projectionはpartial、hostのResult.CommandOmissionsへ実行参照と理由を保持。シェルだけの省略はブロックの候補・状態・省略理由を変えない。
- 指示文で、コマンドは未信頼データであり実行対象でないこと、成功や編集発生を示さないこと、状態はLLM入力で省略していること、Turn/コマンドの共有だけでグループ化しないことを明示。

## Safety correction

最初のE2Eでは閲覧コマンド3件だけが到達し、目的のsed実行が除外された。原因は既存のJWT検出がSiteFooter.test.jsxをトークン形式と誤認したこと。承認済みの対象ソースパスの既知suffixだけを検査用に区切り直すCommandSafetyTextを追加した。送信原文と既知の機密値照合は維持し、その他JWT/Bearer/password等も拒否する。一般の機密検出全体は変更しない。補足仕様はsafety-note.md。最初の受信原本もprivate tmpへ保持し、成功した最終実行と区別する。

## E2E conditions

commiter-test-my-siteで既存の3ファイル（NavigationHrefLink.jsx / SiteFooter.jsx / SiteFooter.test.jsx）を選択。前回と同じraw_diff SHA256。実App Server、通常generateCommitPlan、実MLXクライアントのstdin通信を通す。helperは受信専用subprocessへ置換し、後続フェーズへ進む返答だけ固定fixtureを使う。モデルロード・推論・分類精度評価はしない。Swift製helper内部・全CLI起動フローは対象外。

Go overlayで取得の2箇所の5秒期限だけ外し、テスト自体も時間無制限とした。通常設定の5秒は変更しない。ページ100・最大4並列・メモリ保護と受信上限は維持。

## Results

安全性補完後の実行PASS。取得complete、期限到達なし。6 RPC / 126,993 bytes。編集操作4件、履歴H1（依頼1件・最終応答1件）を保持。取得・照合0.299798秒、通常生成関数全体1.773572秒、テスト2.49秒。Git前後一致、推論0回。

| フェーズ | コマンド件数 | wire bytes | 履歴本文bytes | matched / unmatched |
|---|---:|---:|---:|---|
| Grouping | 7 | 13,049 | 880 | 10 / 4 |
| Category | 0（キー自体なし） | 10,454 | 880 | 10 / 4 |
| Text | 0（キー自体なし） | 9,953 | 880 | 10 / 4 |

GroupingにはS1〜S7を一回ずつ定義し、すべてH1を参照。目的のsed実行はS7でF1/F2/F3へ言及。既存の4未一致をmatchedへ変更していない。各コマンドの送信キーはexecution_ref/context_ref/command/mentioned_file_idsだけであることを確認。状態・出力・host出典IDは送らない。3フェーズのGit差分ハッシュは一致し、本文もhost取得結果と同一。

## Validation

パス境界、cwd、絶対/./、正規表現特殊文字のリテラル扱い、旧新パスのID重複排除、シェルだけのTurnの抽出、同一キー重複/競合、失敗コマンドの保持、安全性、F変換、Category/Text非送信、host非変更、予算でシェル専用H/Sを除きブロック証拠を維持するテストを追加。既知のテストファイル名の誤検知とJWT/Bearer/passwordの拒否も回帰検証。

最終確認では、helper受信コマンド本文がhost保持の原文と完全一致する検査も追加して再実行した。PASS、7件・10 matched/4 unmatchedを維持。取得・照合0.098378秒、全体1.612964秒、テスト2.33秒。最終集計はmeasurements-final.json。先行実行もmeasurements.jsonとして保持。

成功したコマンド:
- go test -p 1 ./cmd/... ./internal/... ./tools/...
- go test -race -p 1 ./internal/historyattribution ./internal/planning ./internal/cli -run 'Test(Command|Commands|ReadCommandOnly|CLIHistoryETLToHelper|AttributedHistoryCacheAndSnapshotGuard)'
- go vet ./internal/historyattribution ./internal/planning ./internal/cli
- git diff --check
- 実履歴付き時間無制限・推論前E2E（コマンド原文一致検査付き）

## Delivery

採用仕様と実装・結果をIssueへ記録。専用作業ツリーのローカル反映。commit/push/main反映は未実施。fixture・Git状態は変更せず、受信原本はリポジトリ外のprivate tmpに保持。公開報告に会話本文・元出典ID・個人絶対パスは含めない。
