## Grouping入力スキーマの採用・コード反映（2026-10-24）

ユーザー提示のGrouping案と追加指定を採用します。本コメントのスキーマをGrouping入力の契約とします。Category/Textは前コメントの現行形式を維持し、新しい履歴入力形式は別途決めます。

### 採用スキーマ

以下は型の定義です。`?`は任意・条件付き、記号なしは必須です。

```yaml
schema_version: 整数（現在1）
task: 文字列
agent_history_contexts:
  - context_ref: 文字列
    user_requests: 文字列配列
    final_responses: 文字列配列
files:
  - file_id: 文字列
    status: 文字列
    raw_diff: 文字列
    old_path?: 文字列
    new_path?: 文字列
    language?: 文字列
    binary?: 真偽値
    vendor?: 真偽値
    opaque?: 真偽値
    old_mode?: 文字列
    new_mode?: 文字列
    change_blocks:
      - block_id: 文字列
        diff_range:
          start_line: 整数
          end_line: 整数
        enclosing_declaration_name?: 文字列
        agent_history_match_status: matched | ambiguous | unmatched | error
        agent_history_match_reasons: 文字列配列
        agent_history_candidates:
          - context_ref: 文字列
            match: exact | partial | exact_and_partial
        agent_history_omission_reasons?: 文字列配列
agent_history_acquisition:
  status: complete | partial | unavailable
  reasons: 文字列配列
agent_history_projection:
  status: complete | partial
regeneration_feedback?: 既存の再生成フィードバック形式
```

### 固定ルール

- `file_id`はcommiterが入力作成時に割り当て、入力内で一意とし、回答の検証まで固定する。内部ファイルIDとの対応表はhostで保持する。LLMは渡されたIDのみを回答に使用する。hostが未知ID・重複・欠落を検証し、元のファイルに復元する。現在の割当実装は`path:src/retry.go`形式を用いるが、形式の解釈をLLMに要求しない。
- `diff_range`は、実際に送信する同じファイルの`raw_diff`内の1始まり・終了行を含む範囲。文脈行で分かれる連続した追加・削除をブロックとする。該当する改行終端マーカーも範囲に含む。
- `change_blocks`は履歴一致の有無を問わず、現在の全テキスト変更ブロックを含む。`block_id`はファイル内で一意。host側の組は`(file_id, block_id)`。
- `old_path`・`new_path`の少なくとも一方が必要。renameでは両方必須。
- 必須の配列は空でも`[]`。最終応答がない場合も`final_responses: []`。
- 全候補の`context_ref`は送信する定義に解決する。参照のない履歴コンテキストは送信しない。
- `exact`だけで一意とは判断しない。同じコンテキストへの候補をまとめても、hostが判定した曖昧さを`matched`へ昇格させない。
- 宣言名はブロックとの一意な対応が確認できる場合だけ格納する。現行のhunk単位の構文情報は無条件にコピーしない。今回の実装ではこの任意フィールドは省略する。
- この形式には`worktree_kind`がないため、symlink・gitlinkでは特殊モード`120000`・`160000`を同値でも保持し、通常ファイルと区別する。
- テキスト差分のない変更は`raw_diff: ""`、`change_blocks: []`とし、パス・モード・該当フラグで表す。
- ブロック解析失敗、差分欠落、hunk件数不一致を空配列の成功にしない。今回の生成経路ではLLM呼出し前に入力エラーとして停止する。
- 照合済み結果の元diff・ブロック範囲が現入力と異なる場合も、`snapshot_changed`としてLLM呼出し前に拒否する。
- Groupingには新形式のみを使い、旧`codex_context`を重ねて追加しない。Category/Textの旧経路は新スキーマ決定まで維持する。

### 省略理由

`files[].change_blocks[].agent_history_omission_reasons`は任意の文字列配列とし、省略なしならキー自体を送信しません。

- `candidate_context_omitted`: 候補コンテキストを丸ごと省略。
- `user_request_omitted`: 残したコンテキストのユーザー依頼本文に省略がある。
- `final_response_omitted`: 残したコンテキストの最終応答本文に省略がある。
- `input_budget_exceeded`: 上記の省略の少なくとも一つが入力予算に起因する。単独では使用しない。

三種類の省略理由の非空の組み合わせ7通りと、それぞれに入力予算の理由を追加する7通りの、ユーザー指定14通りを保持します。同じブロックに複数コンテキストがある場合は影響を集約します。丸ごと消したコンテキストの内側の本文省略は、残したコンテキストの本文省略として重ねて数えません。どの省略が予算原因かの詳細な内訳はhost側に属し、この配列では「少なくとも一つ」を表します。

省略が生じたら`agent_history_projection.status: partial`。候補を消しても元の曖昧さは維持し、唯一の`matched`候補を消した場合は送信側を`ambiguous`へ下げて省略を明示します。参照切れを残しません。

### 今回の実装範囲

専用ブランチ`codex/issue-182-model-comparison`のローカル作業ツリーに反映しました。

- `internal/planning/grouping_input.go`: Grouping送信形式と、実送信diffに対するブロック範囲の抽出。
- `internal/planning/grouping_history.go`: hostから照合済み結果を受け取る型・入力検証・参照解決・省略理由の投影。
- `internal/planning/three_phase.go`: Grouping実リクエストへの接続、ID対応表・回答検証・復元の維持。
- 関連するCLI・benchmark146のアダプター／テストを新Groupingキーへ追従。保存済み測定原本は変更していません。

重要な範囲の区別として、App Server取得→PoC照合→本入力への自動接続はまだ実装していません。既存のパスだけで関連付けた履歴を、ブロック照合済みとして流用しません。今回追加した`GroupingHistory`にhostの照合済み結果が未供給なら、取得状態は`unavailable`、理由は`history_unavailable`として、Git差分によるGroupingを行います。新形式の実リクエスト到達はテスト用の照合済み入力で検証しました。実履歴を自動取得した精度・速度の検証は未実施です。

今回の反映はローカル実装です。commit・push・mainへの反映はまだ行っていません。

### 検証

成功:

- `go test -p 1 ./cmd/... ./internal/... ./tools/...`
- `go test -race ./internal/planning ./tools/benchmark146`
- `go vet ./internal/planning ./internal/cli ./tools/benchmark146`
- CLI build、`git diff --check`

実リクエストでの新Grouping入力、ID復元、参照解決、14通りのhost省略理由の保持、予算・安全性検査による本文省略、候補全体省略時の参照除去、曖昧さ維持、入力tierを増やさずGit差分を保持すること、解析エラーの呼出し前停止、diff範囲、複合rename、特殊ファイルモードを確認しました。

初回の全コードパッケージの並列テストでは、既存PoCの2秒タイムアウトfixtureが初期化待ちで失敗しました。fixtureは変更せず、パッケージを直列にした全体検証で成功しています。`go test ./...`はdocs内に保存された測定原本の複数package同居という既存制約があるため、実行コードのcmd/internal/toolsを検証対象にしています。
