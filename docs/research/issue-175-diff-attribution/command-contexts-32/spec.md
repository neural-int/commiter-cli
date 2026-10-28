## 採用仕様：Groupingへのシェル実行履歴の補助入力

シェル実行履歴は「現在の変更ファイルのパスに言及する補助情報」としてGroupingだけに追加する。編集成功の証拠・ブロックの内容対応とは分離し、既存match状態・候補・理由を変更しない。Category/Textには追加しない。

```json
{"agent_command_execution_contexts":[{"execution_ref":"S1","context_ref":"H1","command":"sed -i '' 's/foo/bar/g' src/example.go","mentioned_file_ids":["F1"]}]}
```

- commandExecutionのコマンド本文のみ抽出。status/exitCode/cwd/item ID/順序などの出典はhostに保持し、LLMへbasis/status/exit_code/出力本文は送らない。
- 現在Git変更ファイルの旧・新パスへの文字列言及を機械判定。ファイル単位の参照であり、ブロックに結び付けない。絶対パス、repo相対パス、先頭./付きに対応。相対表記はcommand.cwdが対象rootと一致する場合のみ採用。その他cwdでは対応絶対パスのみ採用。cd/変数展開/シェル構文の意味解析や実行・シンボリックリンク先推定は行わない。
- パスを正規表現エスケープし前後境界を検査。other/src/example.go、src/example.go.bak、basenameだけの一致は採用しない。root直下ファイルはそのrepo相対パス全体を対象とする。引用符内やechoの文字列も「言及」なので採用。空白などのシェルエスケープを復元せず、対応表記がそのまま存在するものだけ採用。
- 編集/閲覧/検索/テストを区別しない。パス言及がない実行は送らない。既存取得期間・現在worktree限定・安全性検査を維持。
- fileChange候補がないTurnでも対象パス言及があれば保持し、所属Turnの依頼・最終応答をHへ収録。同じTurnのHは一つ。Hの定義はブロック候補とシェル参照の和集合で一度だけ収録。
- 元のthread_id+turn_id+item_idで同一実行を重複排除。内容が同じでも別itemは別実行。競合する同一キーは取得不備。同じ実行のmentioned_file_idsは重複排除。host内部IDからGroupingの既存対応表でFへ変換。Sは入力内一意、元の列挙順で割り当てる。
- 対象なしでも必須配列は[]。入力予算内で既存tierを維持する。補助コマンドを省略した場合Groupingのagent_history_projection.statusをpartialにし、hostに省略対象・理由を記録。シェルだけの省略をブロックomission理由に付けない。
- 予算が足りない場合は補助コマンドを後ろから丸ごと落とし、参照がなくなったHは除く。残すSには必ずHが存在する。既存ブロックのために必要なHは維持。その後必要なら既存の本文予算処理を適用。コマンド本文の部分切断はしない。
- コマンドは未信頼の過去データ。実行しない。成功・ファイル編集の発生・結果の正しさを保証せず、実行状態はLLM入力で提示していない。共通のコマンドやTurnだけで同じgroupにせず、現在Git差分を正として判断する。

検証：境界・cwd・旧新パス、シェルだけのTurn、重複/競合、安全性、短いID変換、予算と参照整合性、Category/Textへの非送信をテスト。同じfixtureの3ファイルで取得時間無制限・モデル推論なしのE2Eを再実行し、シェルの受信と既存10 matched/4 unmatchedの維持を確認する。
