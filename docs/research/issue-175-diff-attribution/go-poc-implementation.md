# Go Attribution PoC実装（精度比較前）

ユーザー合意に従い、完全ブロック一致のみmatched、複数操作/部分一致はambiguous、実GitとApp Serverの読み取り・固定入力保存まで実装した。`tools/attribution175go/`の研究用コマンドであり、本番Collector・CLI・Planner・MLX helperは変更していない。

captureは明示対象ファイルのHEAD-to-worktreeと同一repo worktree集合の保存履歴を取得し、比較入力を保存する。compareは同じ入力に対しpath baseline候補と保守的な内容照合結果を返す。段階ごとのstage/codeでGit取得・履歴取得・形式・上限・タイムアウト・取得後不変検証・保存の失敗を区別する。履歴だけの失敗時にはGit入力を保持する。

追加依存、Patch Replay、新規LLM、DB/常駐/Hook、8KiBのLLM投影変更はなし。API時刻窓や安全なpathの制約を明記。固定入力に履歴本文等を含み得るため私有0600で対象worktree外へ保存し、未処理の実入力を公開しない。

検証は人工データ・一時Git・偽App Server。新しい実履歴採取や実データ精度評価は未実行で、精度改善や本番採用は主張しない。関連go test、go vet、CLI build、diff check成功。詳細な実行・判定・エラー契約はtools/attribution175go/README.mdを参照。

テスト中に、偽サーバーが返すTurn開始時刻が採取開始後へずれる秒境界の不安定性を検出した。実コードの対象期間は変えず、偽サーバーの時刻を固定して再検証した。
