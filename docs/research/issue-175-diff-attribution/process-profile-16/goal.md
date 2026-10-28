# 配布版App Serverの外部プロファイル

## Goal / Conditions

最大4本文worker、一覧中本文取得、同じ保存Git入力/対象/期間、既存保護。配布版は変更せず、今回のハーネス配下5プロセスだけをsample各2秒・2ms間隔で観測。psでCPU/状態を記録。

## Results

610要求・443Turnがcomplete。プロファイル下の全体16.221秒は観測負荷と実行状態を含み、従来の約4秒と性能比較しない。5プロセスでsample成功。fs_usageはroot必須で失敗し、ファイルI/O時間は未取得。

条件変数/セマフォ/keventの待機を観測。一部にはRolloutFile::from_pathとstat、tokio::fs::read_dirとディレクトリ列挙/openが見え、履歴探索・ファイル確認を調査候補にできた。待機スレッドが多いことは正常な仕事待ちも含み、要求処理が停止している割合とはみなさない。未解決シンボルも多く、JSON化の割合は算出不能。

## Limits / Next gate

CPU処理とファイルシステム操作の存在は確認できたが、CPU/I/O/非同期仕事待ちの時間比率、RPCとの因果は未確定。短い観測窓と計測負荷がある。root権限のI/O計測は自動で行わず未取得として保持。次は権限を用意したFile Activity計測か、対応ソースを確認した診断用区間計測。追加再試行なし。

## Artifacts

原本は私有tmpのissue182-process-profile-16。生sampleには個人path等があるため公開しない。公開用は本goalと匿名数値summaryのみ。既定変更・commit/push・Issue投稿なし。
