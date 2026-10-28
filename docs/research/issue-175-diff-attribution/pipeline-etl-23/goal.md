# App Server取得からhelper送信までのETL接続

## Goal / Decision

App Server取得→固定Git差分のブロック照合→hostの照合済み情報→Grouping/Category/Textの新スキーマ→helper stdin送信を通常CLIへ接続。前回残っていた自動供給の未接続を解消した。

## Implementation

既存Go PoCの取得・照合エンジンとテストをinternal/historyattributionへ共通化し、tools/attribution175goのcapture/compareコマンドは同じ実装を呼ぶ入口として維持。pipeline.goで現在の全テキスト変更ブロックを照合し、H1参照へ変換する。元のthread/Turn/item/change indexはhost側に保持する。同じTurnへ候補をまとめても曖昧さを一意へ昇格させない。

internal/cli/attributed_history.goが呼び出し、安全性ポリシー、スナップショット再確認、同一固定入力での取得再利用を担当。planning.goが取得結果をThreePhaseGeneratorへ供給する。新wireスキーマを追加変更せず、各フェーズの独立した履歴投影・入力予算処理を利用する。

取得は現在ワークツリーだけ、固定HEAD時刻〜サイクル開始時刻、100件ページ、本文最大4並列、一覧取得中の本文取得、queue16・接続再利用・段階的メモリ保護。通常CLIの履歴取得は5秒以内。1応答300MBとメモリ保護を維持し、研究用原型の旧4MB接続累積上限は通常経路に適用しない。追加依存・履歴ファイル直接読取・Git変更は導入しない。

## Failure / Integrity

取得不能・取得不足はブロックに理由を残してGit差分へフォールバック。進行中Turnはhistory_incompleteで曖昧として扱う。入力の不正・固定Gitの変更は停止する。取得前後と生成後にHEAD/branch/index/選択変更を確認する。renameの旧新パスの両方に追加安全性ポリシーを適用する。履歴本文・出典ID・サーバー診断の秘密情報をメトリクスや公開エラーに含めない。

## Validation

成功:
- go test -p 1 ./cmd/... ./internal/... ./tools/...
- go test -race -p 1 ./internal/historyattribution ./internal/planning ./internal/cli
- 最終の旧パスポリシー補完後も関連取得/変換/旧パス拒否テストのrace成功
- go vet ./internal/historyattribution ./internal/cli ./internal/planning ./tools/attribution175go
- 通常CLI・PoCコマンドのbuild、git diff --check

別プロセスのApp Server/helper fixtureで通常CLIから全経路を通した。matched、同一記録重複、同一Turnの異なる競合編集、取得失敗のfallback、取得中Git変更でhelper0回停止の5ケース成功。同じ入力の再生成で再取得しないこと、元Git状態が変わらないことも確認。追加・rename・複合diff・モード変更・不正履歴・本文省略・進行中Turn・4MB超の応答・キャッシュ/安全性ポリシー変更・branch変更・キャンセルも検証。

CLI ETLテストはmacOSの実メモリ監視を使用。共通RPCテストはsamplerを注入して他環境でも実行可能。個人の実履歴・実モデルによる精度/速度の測定は未実施。接続と処理の成立をfixtureで検証した結果であり、モデルの意味的精度の保証ではない。

## Publication / Delivery

Issue #183: https://github.com/neural-int/commiter-cli/issues/183#issuecomment-6381766303

本文を再取得し一致を確認。全報告はreport.md。専用ブランチcodex/issue-182-model-comparisonのローカル実装・検証まで完了。commit/push/main反映は未実施。保存済み採取原本・測定成果物は変更しない。
