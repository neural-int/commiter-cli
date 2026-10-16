# Issue #175: Codex の作業記録を任意の補助証拠として使う

## 実装範囲

Three-phase Planner に Codex App Server の読み取り結果を追加する。現在の Git Diff が変更内容の正本であり、ログの不在・取得失敗・未対応・関連不明では既存入力だけで継続する。single-pass、Git 実行時の状態変化による停止、既存 Validator は維持する。自動再計画は後続の独立した変更とし、本実装には含めない。

CLI は収集サイクルの開始時刻を固定する。スナップショットに記録された HEAD のコミット時刻以降、サイクル開始時刻以前に開始した turn を対象にする。進行中の turn も対象とし、取得時点で成功した編集を照合する。各 item の正確な発生時刻は推定しないため、これは編集の発生時刻を厳密に区切る保証ではない。最終回答は任意であり、待機しない。再生成のフィードバックを受けても同じ取得結果を使い、取得範囲を広げない。

## 取得経路と上限

`codex app-server --stdio -c analytics.enabled=false` を子プロセスとして実行し、`initialize` / `initialized` の後に `thread/list`、`thread/turns/list`、`thread/items/list` を利用する。セッションの再開、モデル推論、承認、履歴変更の API は呼ばない。JSONL の直接読み取り、Hooks、commiter 独自の常駐処理、独自 DB、ログ全文の複製は追加しない。

セッション候補は Git が報告する同一リポジトリの worktree 一覧から探す。メインのチェックアウトで始めた Codex が別 worktree を編集できるためである。編集の採用には対象 worktree 内のパスとの対応を要求する。ログの相対パスはセッションの cwd を基準に解決し、別 worktree の同名ファイルを混同しない。state DB だけを参照する列挙を使い、メタデータ修復のための JSONL 走査を要求しない。

| 制約 | 初期値 |
|---|---:|
| Git 補助情報取得から App Server 取得までの時間 | 合計5秒 |
| App Server 標準出力の受信量 | 通知を含めて合計4 MiB |
| 履歴取得のページサイズ | 20 item |
| LLM 向け `codex_context` の JSON | 合計8 KiB、かつ各フェーズの既存入力予算内 |

セッション数・item 総数・レコード数・発話ごとの独立した上限は設けない。ページングは行うが、再試行・ポーリング・期間拡張は行わない。上限に達した場合、取得済みで最後まで読めた turn の証拠だけを候補にする。途中までの応答・turn は採用しない。受信量の上限は daemon 全体のメモリ上限を意味しない。

接続終了時は所有する標準入出力と子プロセスだけを終了・回収する。共有 Codex daemon の stop / restart は行わない。Codex 管理の daemon が残る場合がある。cold start と他クライアントが併存する場合の全ライフサイクルは、読み取り API の成功とは別に検証する。

## 編集との関連付け

起動時に固定した変更スナップショットにあるパスを、重複を除いて取得対象にする。履歴の関連付けのために staged / unstaged / HEAD-to-worktree の差分を再取得しない。既存 collector が完全に相殺された変更を対象外にする動作は変更しない。

同一リポジトリの候補セッション、対象期間内の turn、`fileChange.status == completed` の編集、対象 worktree 内の変更パスとの重なりで関連付ける。変更ブロックの一致は要求しない。A→B→C の編集が最終 Diff の A→C に合成された場合や、途中の編集が取り消された場合も、同じ変更パスに関する履歴を目的の補助情報として扱う。異なる turn が同じパスを編集していても排除せず、新しい turn から既存予算内で採用する。同じ turn / item の継承・再掲は重複排除する。

パスとの関連付けは、履歴中の個々の変更が残っている保証ではない。全3フェーズで、履歴は信頼できない補助情報であり、途中・置換済み・取り消し済みの内容を含み得ること、現在の Git 変更を目的の根拠とし、Diff にない変更や結果を説明へ追加しないこと、矛盾すれば Diff を優先することを明示する。これは LLM が常に正しく判断する保証ではなく、意味評価は別途必要である。

move を含む操作、内容のない metadata-only / opaque / binary の変更は引き続き補助情報の対象外とする。Git に存在する当該変更自体は通常どおり処理する。Git の状態変化による実行停止、既存 Validator、取得失敗時の Diff ベースの処理は維持する。

## 機密情報と入力の信頼境界

既存の sensitive-path ポリシーと追加除外パターンを適用する。対象外 worktree や機密パスへの編集を含む turn は採用しない。既知の機密値、既存 secret detector、発話内の credential assignment の検出に該当する依頼は除外し、該当する最終回答は省略する。この検出は任意の秘密情報を完全に識別する保証ではない。

発話は信頼できない補助データとして JSON 内に置き、system 指示へ補間しない。reasoning、途中報告、system / developer 指示、コマンド本文・出力、暗号化内容は LLM 向けレコードに含めない。App Server の stderr / エラーメッセージを表示・記録しない。取得の成否と件数だけを既存 metrics に記録する。現在の backend は既存のローカル MLX / loopback Ollama の境界を維持する。

## 3フェーズへの入力

各フェーズの最上位に、任意の `codex_context` を一度だけ置く。

```json
{
  "codex_context": [
    {
      "paths": ["scripts/report.sh", "scripts/shipping.sh"],
      "user_request": "レポート出力を追加し、送料条件の不具合を修正して。",
      "final_response": "レポート出力を追加し、送料条件を修正しました。"
    }
  ]
}
```

同じ発話をファイル・グループごとに複製しない。単一の依頼が複数目的を含んでも、それを同一コミットへの所属指定とみなさない。取得元の thread / turn / item ID は LLM 入力へ追加しない。長い本文を機械的に切らず、新しい関連記録から採用する。収まらなければ最終回答を省略し、依頼も収まらなければ記録を除外する。

| フェーズ | 変更点 |
|---|---|
| Grouping | 既存のファイル projection に任意の `codex_context` を追加 |
| Category | 確定グループと共通の `codex_context` を使用。グループの内部 `file_ids` を送信しない |
| Text | Category で検証済みの `type` を各グループに追加。所属と type を再判定させず scope / summary だけを生成 |

元の file ID と所属はホスト側で保持し、最終 plan と Validator に渡す。既存 breaking witness の file ID は LLM に見える path ID に投影し、内部 ID を除外しても証拠と所属ファイルの対応を維持する。Codex 発話から breaking witness は自動生成しない。

フェーズの必須入力だけで選択した context tier の残余に補助情報を収める。ログのために tier を引き上げず、Git 情報・出力枠を削らず、新しい LLM Stage を追加しない。

## 検証の区別

通常テストは元のユーザーログへアクセスせず、LLM を使用しない。必要な箇所を抽出しパス等を正規化した実履歴の再生 fixture を含む。実 subprocess の timeout / 受信上限、API 形式、単一・複数目的、反復・取り消し、親履歴、別 worktree、取得失敗、部分欠落、機密除外、予算、3フェーズ入力、Git 状態不変を検証する。実 App Server の読み取り確認は `COMMITER_CODEX_LIVE_SMOKE=1 go test ./internal/codexcontext -run TestLiveAppServerReadOnly -v` で明示的に実行する。

入力契約・API 読み取りの成功は意味精度の改善を証明しない。Grouping / Category / Text の有益性、feat / fix の改善、誤帰属率、cold start の daemon 挙動は別途評価する。固定 Gold や過去の計測は変更しない。

## 2026-10-16 の実装検証

- `go test ./...`、`go build ./...`、`go vet ./...`、`git diff --check`: 成功。
- `go test -race ./internal/codexcontext ./internal/planning ./internal/metrics`: 成功。
- `.github/scripts/release-notes` の Python unittest: 25件成功。
- 明示的な実 App Server 読み取り smoke: thread / turn / item を各20件取得し、受信153,693 bytes、API確認約0.48秒。モデル推論と本文出力は行っていない。cold start の測定ではない。
- 全体テストの初回では benchmark146 の旧 `groups[].file_ids` 参照と旧 Text 入力契約が失敗した。変更前 main での成功と比較して原因を確認し、入力アダプターを新契約へ追従後、全体テストの成功を確認した。固定 fixture・Gold・過去の計測結果は変更していない。

通常の CLI 連携テストでは人工の Codex 記録と既存の擬似 MLX helper を使用し、3フェーズから最終 plan まで確認した。これは実モデルの意味精度改善の評価ではない。

## 2026-10-16 の関連付け方針変更

実履歴調査で、Issue #167 の反復編集17件が最終差分とのブロック照合を満たさず不採用になった。ユーザー合意により、ブロックの完全一致と同一パスの複数 turn 排除を採用条件から外した。履歴照合専用の差分再取得も除いた。現在の Diff を優先する指示は3フェーズ共通で追加する。

コピー済み実履歴5件の再生では、旧方式の4件採用・1件不採用から、新方式では5件採用となった。Issue #167 は1 record に4パスが関連付いた。元のJSON・manifest・未確定Goldは変更せず、旧方式の不採用記録も testdata の README に残す。これは補助情報の収集範囲の確認であり、意味品質改善の証明ではない。

方針変更後の `go test ./...`、`go build ./...`、`go vet ./...`、`git diff --check` は成功した。全3フェーズへの Diff 優先指示、同一パスの複数依頼の保持、期間・worktree・failed/cancelled・機密情報の除外、取得による Git 状態不変をテストで確認した。実モデルによる意味評価と新データを用いた実 App Server の取得確認は行っていない。
