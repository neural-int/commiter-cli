# H182-005 保存資料

結果は[goal.md](goal.md)、採用promptと事前条件は[../prompt-adoption.md](../prompt-adoption.md)。

研究用`thinkingBudget`を既存Swiftのnative境界制御へ接続し、1024到達後はEOS打ち切りでなくJSON回答を続ける。通常CLIの設定・promptは変更しない。Qwen公式仕様への参照と、本実装がSwift連続decodeである限界は採用条件文書に記載。

`tools/benchmark182/thinking_compare.py prepare`が生成前の要求・token・KV一致を監査し、`run`が登録4セルを順次実行する。既存出力は上書きしない。無制限側はH182-004の採用prompt結果を使い再実行しない。

`raw/`は契約、監査、生成なしcapture、実送信要求、応答、メトリクス、進捗、監視、集計、source snapshot、Swiftテストログ、準備ログを含む。weight/binary/metallib/build cacheは含めずhashで識別する。`raw/source/`は実行時のハーネスと関連Swiftソース。

- contract.json：4セルの契約、原本hash、保存済み無制限基準への参照
- audit/：全4セルの実token列とKV監査
- *-thinking1024/：実要求、応答、結果、進捗。予算到達はnativeThoughtForcedCloseで識別
- summary.json：全12phaseの予算・入力不変監査、無制限基準、metrics
- completion.json：4セル完了・monitor_failed=false
- invalid-budget-audit.json：不正0予算の生成前拒否

公開コピーではメール形式文字列をredacted@example.invalid、ホーム絶対pathを$HOMEへ置換する。scriptを再利用する場合は環境に合うpath解決が必要。SHA256SUMS.txtは公開コピーhashであり契約内の原本hashとは区別する。

無制限基準はcommit e955e96ad69046c363b21f9912c9ed53de7d0d71の`docs/benchmarks/issue-182/prompt-comparison/raw/`。元の中断記録を保持する。後段plan受理は固定Grouping下の結果で、実Grouping・意味品質とは別に確認する。各1回で、速度差の因果効果や一般的な品質改善を示さない。
