## 要約

C02の入力受理失敗には、構文evidenceによるサイズ増大と、非圧縮64k入力を試す前に共通Prepareが圧縮するという二つの原因があった。Three-phaseに限り、ファイルごとに構文表現と許可済みDiff全文の小さい方を選び、設定されたcontext上限で非圧縮のままPrepareする変更を用意した。モデル、timeout、出力予算、fixture、Gold参照は維持する。

## 検証結果

- C02の元promptは1,047,993 bytes。projectionのみでは32kに収める圧縮が残った。RequireUncompressedとの組合せで52,551 bytes、summary=0、compression=noneとなり、A/Bとも機械応答による3phase captureを完了した。
- C01もsummary=0。Categoryの事前token数はA=2,083/B=2,540で、親の7,951/8,408から減少した。Groupingの入力は親と同じである。
- C02のnative tokenizer計測は約17–18k token。計測専用の旧helper自身の16k上限ではinput_too_largeとなるが、token数＋出力予約は実生成の64k枠内だった。この計測は本番helperによる64k受理の証明ではない。実生成時にも本番helperがtoken budgetを独立に検証する。
- 初回preflightの停止条件を確認し、count-only判定を追加して再計測した。旧helperはソースが現存せず、実行時binary hashで固定した。
- 元の変更パスと承認済みDiffのSHAを照合した。構文解析・relation抽出はprojection前に実行し、hostの結果は保持する。opaque/binary/missing-sourceは既存表現を維持する。
- go test ./...、go vet ./...、go build ./...は成功。追加テストは実際のGo構文解析での膨張、元データの非変更、metadata-only境界、64k非圧縮受理と上限超過拒否を検証する。既存のsingle-pass圧縮テストは既存動作を担保するため維持する。

## 考察

変更は既存のraw_diff表現を利用する。全Diffを残すためファイル・hunk削除による成功には当たらない。一方、モデルに渡す表現は変わるため、意味品質が維持されるかは実モデルで別に評価する必要がある。C01の入力減少が120秒内のText完了につながるかは現時点では未確認である。

## Next Steps

- 元の2ケースとGold参照をそのまま使い、2ケース×A/B×1、最大12要求、120秒/cycle、retry/repairなしの契約を推論前に固定する。入力改善の効果と意味品質を同じ条件で確認するため。
- 最終Validator、各phaseの意味、時間、取得できるresource情報、Git不変性を別々に記録する。受理成功だけを意味品質成功に数えないため。
