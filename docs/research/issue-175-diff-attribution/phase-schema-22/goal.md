# Category/Text入力スキーマ最終反映

## Goal / Decision

ユーザー確定仕様をCategory/Textの現行入力へ反映。file_idはF1・F2形式でGroupingも統一。Category/Textに全体取得・投影状態は送らず、ブロックの照合理由・省略理由で表す。Textには確定type・breakingを必須で送る。

## Implementation

metadata_input.goでgroup_idと共通ファイル・全変更ブロックの入力を構成。three_phase.goで短いIDの割当・host内部ID復元を維持し、Categoryの検証済み判断をTextのbreakingへ渡す。falseも省略しない。unresolvedでTextへ進まない。Categoryにはtype/breaking/summary_languageを送らず、Textには証拠候補を送らない。削除済みfactやevidence_pool_limitationは復活させない。

入力対象グループの全ブロックから参照されるコンテキストの和集合だけを定義する。対象外選択・同一参照の重複排除は省略扱いにしない。候補・本文を実際に省略した場合は影響するブロックへ理由を記録。履歴は元のhost照合結果からフェーズごとに再投影し、先行フェーズの省略を固定しない。Git差分とhost元情報は維持する。関連CLIテストとbenchmark146アダプターを追従し、保存済み測定原本は変更しない。

## Validation

成功: go test -p 1 ./cmd/... ./internal/... ./tools/...、go test -race ./internal/planning ./tools/benchmark146、go vet ./internal/planning ./internal/cli ./tools/benchmark146、CLI build、git diff --check。

実リクエストのフィールド、F1形式の3フェーズ固定と元ID復元、Textのbreaking true/false、履歴の範囲選択と和集合・一回定義、非省略扱い、先行フェーズから独立した再投影、ブロック単位のhistory_incomplete、host元情報維持を確認。benchmark146の4ファイル条件で本体とリクエスト一致を確認。途中の旧ID依存mock/adapter失敗は追従修正後、全コード検証で成功。

## Scope / Remaining

App Server取得→PoC照合→GroupingHistory供給の自動接続は引き続き未実装。hostの照合済み情報を供給した際の全3フェーズ実リクエスト到達を検証した。未供給時はGit差分で進め、history_unavailableをブロック理由に残す。モデル精度・速度比較は未実施。

## Publication

https://github.com/neural-int/commiter-cli/issues/183#issuecomment-6381474698

専用ブランチcodex/issue-182-model-comparisonのローカル反映。commit/push/main反映は未実施。全フィールド・固定ルールはschema.md。
