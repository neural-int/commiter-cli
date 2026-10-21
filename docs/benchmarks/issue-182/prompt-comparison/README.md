# H182-004 プロンプト比較の保存資料

結果は[goal形式の報告](../prompt-comparison-goal.md)、条件は[事前監査](../prompt-comparison-preflight.md)を参照。

## 保存範囲

`raw/`には元実行とQwen2B再開の全テキスト成果物を保存する。モデルweight、実行binary、Metal library、build cacheは含めず、実行契約のSHA256で識別する。原本は実行端末の`/private/tmp/issue182-prompt-comparison/`に残す。

- `contract.json`：元12セルの凍結契約、資産・source hash
- `audit/`：実際のtoken列、prefix長、KV構成と実送信予定要求
- `capture-*/`：生成を伴わないfixture要求
- `gemma-on-*/`、`qwen4-on-*/`：要求、応答、部分進捗、結果
- `qwen2-resume/`：未実行Qwen2B OFF4セルだけの再開契約、再capture、要求、応答、結果、OS監視、completion
- `user-interruption.json`（各中断セル内）、`suite-interruption.json`：ユーザー指定の中断記録
- `final-summary.json`：中断を明示的に分類した全12セル集計、実Grouping比較、生成区間token/s
- `source/`と`checks/`：実行時sourceとSwift39テストのログ
- `resume_qwen2.py`、`finalize_report.py`：再開と集計に実際に使ったscript

`NN-request.json`は元要求、`NN-actual-request.json`がproxy変換後にSwiftへ送った要求。後段に固定partitionを注入するため、最終plan受理だけでは実Groupingの正しさを評価できない。実Groupingは`01-response.json`を別に確認する。

## 再開と解釈

元driverは既存出力があると停止する設計。Qwen2B再開は凍結済みdriverのrun関数をそのまま使い、未実行4セルと保存先だけを選び直した。再開前に元契約の全hashと、現在のfixtureから再captureした全要求の一致を確認した。既存セルを上書き・再測定していない。

元suiteはユーザー指定で停止したためcompletionがなく、元resultsのEOFエラーは中断を自然失敗として表現できない。各中断記録を併読する。Qwen2B再開のcompletionは4セル・monitor_failed=falseであり、元suite全体の自然完了を意味しない。

公開コピーではメール形式文字列を`redacted@example.invalid`、絶対ホームパスを`$HOME`へ置換した。保存scriptの絶対pathも置換されるため、再利用時は環境に合わせたpath解決が必要。実行時原本のhashは契約内に保持し、公開ファイルのhashは`SHA256SUMS.txt`で別に記録する。

各条件1回の探索比較。元監視と再開監視の間には中断期間があり、連続した実行時間として合算しない。温度・電力の直接実測はない。追加推論、retry、repair、prompt採用は行っていない。
