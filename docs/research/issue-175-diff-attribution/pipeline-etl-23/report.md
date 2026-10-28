## App Server取得からhelper送信までのETL接続（2026-10-24）

前コメントで未接続だった「App Server取得→Git差分ブロック照合→hostのGroupingHistory供給」を通常CLIの3フェーズ生成経路へ接続しました。新スキーマの受け口だけを持つ状態から、取得・照合した履歴をGrouping・Category・Textへ自動で渡せる状態になりました。

### 実行経路

1. 通常CLIが現在のGitスナップショットと入力用の完全diffを固定する。
2. 取得前にHEAD・branch・index・選択変更の同一性を確認する。
3. App ServerのJSON-RPCを通して、現在ワークツリーで記録された履歴を取得する。
4. 履歴の編集内容と、固定した現在の全テキスト変更ブロックをPoCと共通の照合エンジンで比較する。
5. 出典のthread/Turn/item/change indexをhostに保持し、履歴コンテキストをH1形式の短い参照へ変換する。F1形式のLLM用ファイルIDと内部GitファイルIDの対応は既存のhost対応表で維持する。
6. 取得後にGitスナップショットを再確認する。変更していたらsnapshot_changedで停止し、helperへ送らない。
7. 元の照合済み情報から各フェーズの入力を独立に作り、新スキーマをhelperのstdinへ送信する。
8. 生成後にもスナップショットを確認し、変わっていれば計画を返さない。

### 取得条件・失敗時の扱い

- 現在のワークツリーだけを対象とし、別ワークツリーで記録された会話は取得対象に含めない。
- 対象期間は固定HEADのコミット時刻から処理サイクル開始時刻まで。テキスト変更ブロックがある、承認済み対象ファイルの旧新パスを使う。
- 採用済みの100件ページ、本文最大4並列、一覧取得中の本文取得、queue16、接続再利用、段階的メモリ保護を使う。
- 通常CLIの履歴取得期限は従来どおり5秒以内。1応答300MBの上限とメモリ保護を維持し、研究用原型の旧4MB接続累積上限は通常経路には使わない。親contextの期限・キャンセルは維持する。
- 同一ソースキーの重複は排除し、同じキーで内容が競合する場合は取得不備として扱う。内容が同じでも別の編集記録なら候補として保持し、同一Turnへまとめても曖昧さをmatchedへ昇格させない。
- 取得不能・取得不足はブロックの理由に残し、Git差分による生成へフォールバックする。Git入力の欠落・不正、固定スナップショットの変更は停止する。公開エラーにサーバーの診断本文を含めない。
- 進行中Turnに由来する候補は一意と断定せず、該当ブロックをambiguousとしhistory_incompleteを残す。Turn状態が欠落・不正な場合はhistory_parse_failedとして扱う。
- 再生成時に、root・HEAD・index・入力ファイル・安全性ポリシーが同じなら取得済み結果を再利用する。入力・ポリシーが異なる場合は再利用しない。各フェーズの投影は毎回元情報から作る。
- source IDや編集記録の位置はhost側に保持し、LLMには新スキーマの短い参照と必要な本文だけを渡す。履歴を直接ファイルから読む経路や追加依存は導入していない。

照合の意味は、対象期間・取得範囲内での内容対応です。patch replayによる因果的な作成元の証明は追加していません。

### コード反映

- `internal/historyattribution/`: 既存のGo PoCのエンジンと回帰テストを共通化。pipeline.goが取得結果→新スキーマのhost入力への変換を担当。
- `internal/cli/attributed_history.go`: 通常CLIからの呼び出し、固定スナップショット確認、取得再利用、安全性ポリシー、メトリクスへの状態・件数記録。
- `internal/cli/planning.go`: 共通エンジンの結果をThreePhaseGenerator.GroupingHistoryへ供給。
- `tools/attribution175go/main.go`: 従来のcapture/compareコマンド入口を維持し、共通エンジンを利用。READMEの実装・テスト実行先も更新。保存済み採取原本・測定成果物は変更していない。

Grouping/Category/Textの確定したwireスキーマは維持しています。

### ETL検証

`TestCLIHistoryETLToHelper`は、App Serverとhelperを別プロセスのfixtureとして起動し、通常CLIのgenerateCommitPlanから実際のJSON-RPC、Git内容照合、新スキーマ生成、MLX helperのstdin送信まで通します。helper側でも受信したF1・H1、本文・候補参照・照合状態、Textのtype/breaking、Category/Textの除外フィールド、出典IDの非送信を検査します。

| ケース | 結果 |
|---|---|
| 一致する編集履歴 | 全3フェーズへmatchedと必要本文を送信し、元GitファイルIDへ復元：成功 |
| 同じソース記録の重複 | 重複を除き、履歴本文を一回だけ収録：成功 |
| 同一Turnの異なる編集記録が同じ内容 | コンテキスト定義は一つ、照合状態はambiguous、host出典は二つを維持：成功 |
| App Server取得失敗 | 理由を保持して全3フェーズをGit差分で処理：成功 |
| 取得中にGit変更 | snapshot_changedで停止し、helper呼び出し0回：成功 |

同じ固定入力で再生成してもApp Serverを再起動・再取得しないこと、HEAD・index・選択変更をETLが変更しないことを確認しました。別途、入力キャッシュと安全性ポリシー、branch変更・キャンセルの拒否も検証しています。

共通エンジンでは変換15ケースとRPC取得4ケースを追加し、追加・rename・複合diff・モード変更を含むdiff・取得不足・取得不能・一致なし・出典競合・本文省略・進行中Turnなどを検証しました。4MBを超える有効応答も、通常経路の1応答上限内で取得・照合できることを確認しています。

成功:

- `go test -p 1 ./cmd/... ./internal/... ./tools/...`
- `go test -race -p 1 ./internal/historyattribution ./internal/planning ./internal/cli`
- `go vet ./internal/historyattribution ./internal/cli ./internal/planning ./tools/attribution175go`
- 通常CLIとPoCコマンドのbuild、`git diff --check`

CLIのETLテストは実機と同じmacOSのメモリ監視を使います。共通RPCテストは監視samplerを注入して他環境でも実行できます。個人の実履歴や実モデルを使う追加推論・精度比較・速度測定は実施していません。上記はfixtureによる接続と処理の検証です。

### 反映状態

専用ブランチcodex/issue-182-model-comparisonのローカル実装・検証まで完了。commit・push・main反映は未実施です。過去コメントの「通常CLIへの自動接続は未実装」という残作業は、この実装で解消しました。
