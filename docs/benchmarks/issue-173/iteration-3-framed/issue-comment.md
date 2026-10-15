## 要約

[指定された短縮版Category Prompt](https://github.com/neural-int/commiter-cli/issues/173#issuecomment-6188352282)を既存25条件で各1回検証しました。既存5条件のGold一致は **0/6のまま**、境界20例は **11/20→13/20** でした。限定的な改善はありますが、既存fix条件の改善基準は未達のため採用を見送ります。

## 検証結果

- 意味判断用System Promptだけを差し替え、Raw Diff・関連テスト・group membership・update/changeを含むSchema・breaking evidence契約・安全指示・モデル・samplingを維持しました。Grouping/Text再推論と本番コード変更は0です。
- 既存6グループは前回と同一：誤ったupdateが4件、誤ったfeatが2件。
- 正答改善は **F04: update→fix、F13: chore→ci**。F05はfeat→updateですが、Goldはfixのため誤選択のままです。その他のtype判断は前回と同じでした。UI色変更F10もstyleのままで不一致です。
- 境界例のupdateは選択6・正当2・誤選択4・見逃し1、changeは選択1・正当1・誤選択0・見逃し1で前回と同じです。境界例の誤ったfeat/fix断定は1→0件でした。
- 修正後の分類用推論は25回、Category Schema適合25/25。F11は既存test-only Schemaによる強制です。token件数は未取得です。
- 最初の25送信は、こちらの送信処理でIPC末尾改行を欠落させたため、全件が推論前にmalformed_requestで拒否されました。原記録をiteration-3に保存し、送信形式だけを修正した別記録iteration-3-framedで推論しました。両者の固定requestファイル25件は同一です。helper送信は計50件、分類用推論は25件です。
- Category単独call上限120秒。製品の共通deadline・完成planは今回再検証していません。前回候補の履歴との比較であり、今回同時に実行した対照群や未使用holdoutではありません。
- go test ./... / go vet ./... / go build ./...、Python構文、git diff --checkはPASS。CIは未実行です。

[全条件の比較・計測結果](https://github.com/neural-int/commiter-cli/blob/codex/issue-173-category/docs/benchmarks/issue-173/iteration-3-framed/report.md)、[固定条件](https://github.com/neural-int/commiter-cli/blob/codex/issue-173-category/docs/benchmarks/issue-173/iteration-3-framed/preregistration.json)、[生の結果](https://github.com/neural-int/commiter-cli/blob/codex/issue-173-category/docs/benchmarks/issue-173/iteration-3-framed/results.jsonl)を保存しています。

## 考察

短縮版は今回の境界例2件で正答を増やし、1件で誤ったfeat断定をupdateへ変えました。一方、既存修正例6グループは改善せず、update/changeの使い分けも改善していません。全体的なfix識別やfallbackの問題を解消したとは判断できません。既知のsynthetic fixtureに対する歴史的比較であり、一般化性能や完成planの改善は主張しません。

## Next Steps

- 短縮版を採用せず、今回の原結果と送信失敗記録を保持します。既存fix条件とfallbackの問題が残るためです。
- 続ける場合は、既存契約の利用とupdate誤選択に対象を絞った別の有限契約を先に固定します。今回のGoldや失敗結果を書き換えず、結果に合わせた無期限のPrompt探索は行いません。
