# Grouping入力スキーマの採用

## Goal / Decision

ユーザー提示のGrouping入力と追加のID・省略理由・解析失敗ルールを採用。Category/Textの新履歴スキーマは未決定のまま維持する。

## Implementation

file_id、raw_diff必須、全変更ブロックとdiff内範囲、履歴コンテキスト・候補参照、取得状態・投影状態をGrouping入力へ反映。任意のagent_history_omission_reasonsは指定14通りを保持する。実入力のIDはhostが割当、回答検証まで対応表を固定し元IDへ復元する。特殊モードは同値でも保持。ブロック解析失敗はLLM呼出し前に停止し、空配列成功にはしない。hunk単位の宣言名はブロックにコピーしない。

host供給GroupingHistoryで照合済み結果を受け取る。diffと範囲・参照を検証し、古い結果を拒否する。履歴は選択済みcontext tierの余白内で投影し、省略の有無と原因をブロックに記録。候補をまとめても曖昧さは昇格しない。Groupingには旧codex_contextを重ねない。

App Server取得→PoC照合→GroupingHistoryの自動接続は今回未実装。未供給時はhistory_unavailableを明示してGit差分のみでGroupingする。新形式の実リクエスト到達はテスト用照合済み入力による検証。実履歴でのモデル精度・速度は未測定。

## Validation

全実行コードのgo test -p 1 ./cmd/... ./internal/... ./tools/...、関連planning/benchmark146のrace、vet、CLI build、diff check成功。初回の全パッケージ並列検証で既存PoCの2秒タイムアウトfixtureが初期化待ちで失敗。fixtureを変更せず、直列の全体検証で成功。docs内の保存原本のpackage衝突は変更しない。

## Publication

https://github.com/neural-int/commiter-cli/issues/183#issuecomment-6380846740

専用ブランチcodex/issue-182-model-comparisonのローカル反映。commit/push/main反映は未実施。関連スキーマはschema.md。
