# 360秒の共有期限の有限比較

本番の120秒は変更せず、Goのbuild overlayでCandidateCycleTimeoutだけを360秒にして比較する。SRSの120秒契約を変更した、速度が改善した、採用済みであるとは扱わない。入力・Gold・モデル・helper・profile・出力予算はIteration01と同じ。各2ケース×A/B×1、最大12要求、retry/repair/追加judgeなし。

`timeout-treatment.patch` が唯一の処置差分。`three_phase.go.txt` は実際にコンパイルする置換ソース。`go test -overlay=/absolute/overlay.json ./internal/cli -run '^TestIssue175Evaluation$' -v -count=1` と同じrunner環境変数を使う。overlay.jsonのReplaceは実checkoutのinternal/planning/three_phase.goをこのファイルへ対応付ける。出力は新規ディレクトリへ保存する。固定capture/tokenizationはIteration01を参照する。overlayは実装ファイルを書き換えない。

C02のGrouping約107秒に対し、従来の120秒を3phase分確保するという検証候補である。終了時にはphase/全体時間と最終Validator受理を記録し、長い期限で意味不一致やschema不合格が解決するとは仮定しない。
