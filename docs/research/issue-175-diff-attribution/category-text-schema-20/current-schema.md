## 現行入力スキーマの変更：Category・Text共通のファイル投影（2026-10-24）

ユーザー指定の変更を作業ワークツリーへ実装しました。**以下を、以後の新スキーマ検討で比較元とする現行Category・Text入力スキーマとします。** これは先に確定した履歴共通スキーマをフェーズへ接続する変更ではありません。

### 変更範囲

- Category・Textの`groups[].files[]`を最大13キーの専用送信形式へ変更。
- `head_identity`・`worktree_identity`・`change_hash`・`size`・`staged`・`unstaged`を送信入力から除外。
- 通常更新の同一`old_path`、未変更モード、falseフラグ、パスから一意に判断できる言語を省略。
- `breaking_evidence_candidates[].fact`をCategory・Textの送信入力から削除。
- `evidence_pool_limitation`をCategory・Textの送信入力から削除。
- Groupingの入力、各フェーズの出力スキーマ、最終validator、Git操作の安全性は変更していません。
- 元のGit識別情報と`BreakingEvidence.Fact`はホストの検証用データには保持します。今回削除するのはLLM送信フィールドです。

### 現行：Category・Text共通のgroups[].files[]

```text
id:
  ファイル・証拠候補の対応に使う参照ID
  （必須・文字列、例: "path:src/retry.go"）

status:
  現在のGit変更状態。既存の値を維持
  （必須・文字列、例: "modified" / "added" / "deleted" / "renamed"）

old_path:
  変更前パス。通常更新でnew_pathと同じなら省略。
  削除では旧パスを残し、renameでは両端を残す
  （条件付き・文字列、例: "src/old_retry.go"）

new_path:
  変更後パス。削除では省略
  （条件付き・文字列、例: "src/retry.go"）

old_mode:
  モード/種別が変わる場合の変更前モード
  （条件付き・文字列、例: "100644"）

new_mode:
  モード/種別が変わる場合の変更後モード
  （条件付き・文字列、例: "100755"）

worktree_kind:
  通常ファイル以外の種別。通常fileと削除状態deletedは省略
  （条件付き・文字列、例: "symlink" / "gitlink"）

language:
  パスの拡張子だけで言語が一意に確定しない場合、または
  拡張子からの推定とホスト判定が異なる場合に保持
  （条件付き・文字列、例: "Go" / "Python"）

binary:
  テキスト差分として扱えないバイナリ。trueだけを送信
  （条件付き・真偽値、例: true）

vendor:
  外部依存として扱うファイル。trueだけを送信
  （条件付き・真偽値、例: true）

opaque:
  内容を提供せずメタデータだけで扱う状態。trueだけを送信
  （条件付き・真偽値、例: true）

mode:
  提供している変更情報の表現形式
  （必須・列挙文字列: "raw_diff" / "metadata_only"）

raw_diff:
  現在の完全なGit差分。raw_diffモードで収録し、metadata_onlyでは省略
  （条件付き・文字列）
```

通常更新の例：

```json
{
  "id": "path:src/retry.go",
  "status": "modified",
  "new_path": "src/retry.go",
  "mode": "raw_diff",
  "raw_diff": "@@ -10,1 +10,1 @@\n-retries := 1\n+retries := 3\n"
}
```

### 条件付き省略の実装契約

- old_path/new_pathの少なくとも一方を必須とし、renameでは両方を必須とします。
- 両モードが同じならold_mode/new_modeを省略。両方あり異なる場合は両方を送信します。
- 追加/削除で片側モードだけがある場合、通常の100644は省略し、100755/120000/160000等の意味があるモードを残します。
- 既存依存の言語判定を使用。拡張子から一意に同じ言語へ判定できる場合だけlanguageを省略し、拡張子なし・曖昧・判定不一致は保持します。
- binary/opaque/metadata_onlyにはraw_diffを載せません。通常のraw_diff入力で完全差分が欠ける場合は、黙って内容なしにせずエラーとして扱います。
- 元のホスト入力は変更せず、送信時の投影だけを変更します。

### 現行の外側フィールド

```text
Category:
  task
  groups[]: id, files[]（上記13項目の条件付き投影）
  breaking_evidence_candidates[]: id, file_ids
  summary_language
  regeneration_feedback（任意）
  codex_context（既存方式・収録できる場合だけ）

Text:
  task
  groups[]: id, files[]（Categoryと同じ形式）, type（Categoryで確定）
  breaking_evidence_candidates[]: id, file_ids
  summary_language
  regeneration_feedback（任意）
  codex_context（既存方式・収録できる場合だけ）
```

`breaking_evidence_candidates[].fact`と`evidence_pool_limitation`は両方から除去済みです。`breaking_evidence_candidates`自体とid/file_idsは維持します。Groupingには今回の変更を加えていません。

### 状態と検証

実装は研究用ワークツリーで完了。commit/push/main反映はまだ行っていません。今回のコメントは、ローカルで実装した現行入力の比較基準を記録するものです。

通常更新5項目、rename/削除、モード変更、symlink/gitlink、拡張子なし/曖昧な言語、binary/opaque/vendor、ホスト情報の保持、実際のCategory/Text送信からの不要キー除去、Textへの確定typeの引継ぎを回帰テストで確認しました。検証結果は後記。

検証結果：

- `go test ./cmd/... ./internal/... ./tools/...`：成功。
- 関連Planner/入力/履歴処理、および現行Plannerと送信要求を比較するbenchmark146のrace検査：成功。
- 関連`go vet`、通常CLI build、`git diff --check`：成功。
- benchmark146は同じ投影関数を使用するよう追従し、通常Plannerとの要求一致テストが成功。保存済みfixture原本・測定結果は変更していません。
- `go test ./...`には、既存の`docs/benchmarks/issue-182/suite-v2/raw`へ複数Goパッケージの原本が同居するためのsetup失敗が残ります。原本は変更せず、実行コードのcmd/internal/tools全パッケージで検証しています。
- モデルによる精度比較や追加の実履歴取得は今回は行っていません。
