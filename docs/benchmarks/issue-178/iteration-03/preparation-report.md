## 要約

現行worktreeを一巡だけ再検証する。ユーザーの確認依頼に基づき、既存2ケース×ログなし/あり×1回、現行の共有120秒、最大12要求、retry/repairなしで条件を固定した。KV再利用・タスク統合・scope規則変更は含めない。

## 検証結果

- HEAD=f2317c84に加え、未commitのRawDiffProjection共通化・旧evidence選択廃止を含む全関連ソース148件のSHAを固定した。tracked patchと新規ソースも保存した。
- go test ./...、go vet ./...、go build ./...が成功。既存release helperとMetalは前回と同じSHAで、metal_okを確認した。
- 現行の機械キャプチャ12件は前回キャプチャと全件bytes一致した。この2ケースは旧選択処理でも既にRaw Diffになっていたため、入力契約変更が今回のLLM入力を変えるケースではない。
- Textのgrammar接続は前回の実測にも含まれる。前回の2ケース4条件は360秒overlayで完了しており、今回はoverlayを使わない現行120秒の動作確認とする。
- fixtureと探索的Goldは保持し、固定前のHEAD/index/worktree bytesを保存した。実行条件はexecution-contract.json、機械入力比較はpayload-comparison.jsonに保存した。

## 考察

コード上の契約修正と、この2ケースでの実入力差分は区別する。今回の結果を入力変更による品質改善や退行の因果証拠にしない。C02は120秒で後続phaseに到達しない可能性があるが、条件を途中で延長せず現行仕様の結果として保存する。Goldは独立した人の確定評価ではなく、従来の探索的参照のままである。

## Next Steps

- C01-A/B、C02-A/Bの順に一巡実行する。各cycleは失敗時に止め、残りの事前登録cycleだけを続ける。4cycle後に追加生成しない。
- Grouping、Category、Textの意味、schema/最終Validator、phase/全体時間、取得可能なresource、Git不変性を別に報告する。runnerのGo test成功をケース成功とは扱わない。
- 前回の360秒結果も参照として保存し、今回の120秒結果と区別してIssueへ追記する。
