# 完了検証

Issueの8完了条件とチェックボックスをGitHubから取得して確認した後、goal.mdの最終品質ゲートを実行した。

- 推論前事前登録: prompts-full.json、preregistration.json、frozen-inputs、fixtures.json、input-audit.json。
- Known/holdout: H01-H04回帰12call、新規N01-N04 holdout24call。全36receiptを保存し、gold/Prompt/閾値を変更しなかった。
- 判定: Gate1 NO-GO。Phase2は条件不達で未評価、製品採用なしで有限研究終了。
- 資源/安全: 全tokens/wall/RSS/swap/pressureとGit保持をintegrity-audit.json、measurementsへ保存。Cのswap基準超過を隠していない。
- 最終品質: go test -count=1 ./...、go vet ./...、go build ./...、Python32検証、Python構文/pins、git diff --checkがpass。
- 初回go test ./...は既存timeout testのchild did not startで失敗。500ms起動期限と負荷の可能性を記録し、単独および通常並列の全suite非cache再実行がpass。元の失敗log hashもfinal-quality.jsonへ保存し、コード/testは変更しなかった。
- テスト整理: product更新・削除なし。既存testsを削除せず、入力同一性/hash改変/Phase2起動拒否の3研究契約だけ追加。
- Scope: #169最終headからtools/benchmark170、docs/benchmarks/issue-170のみ。internal/cmd/mlx-helper/go.mod/go.sum/.githubはmainとも同じ。mainの既存.gitignore変更とindex/HEADを保持。

報告はIteration1/2/3としてIssueへ記録。branchをpushし、remote head一致とIssue closed/completedを再取得してからGoalを完了する。mainへのmergeやproduct移行はこの研究に含めない。
