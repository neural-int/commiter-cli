## 要約

2026-10-08ユーザーの参照報告に従う再開指示で、A-only C探索を開始する。B No-Goと追加evidence探索停止は保持し、C先行評価を正式B統合/production gate通過と混同しない。

## 検証結果

新規専用worktree codex/issue-153-a-only-partitionを作成。親#150とC#153の依存契約をA GO後の限定C探索許可へ更新した。旧結果/チェックを変更していない。標準Python exact signed optimizerは2〜8unitの全partitionを列挙し、8unitでは4140状態。欠落score、bool等不正score、同点最適解は拒否。source/goldをsolverへ渡さない。3contract testとdiff check成功。

使用済み5file/8unit2件を4call比較として固定した。A-onlyはchange_units/selected_file_contextだけでrepository/historyは追加しない。直接membership生成と、LLM全pair score→host partitionを比較する。schema/systemの責務変更も含むarchitecture比較であり、optimizerだけの因果効果とは扱わない。条件とrunnerを計測前commit/push済み。

## 考察

以前はevidenceを増やして最終membershipをLLMへ委ねた。今回の仮説は判断責務をrelation評価へ限定し、全体整合したpartitionをhostへ分離するもの。これはユーザー承認された検証設計変更で、過去BのNo-Goを取り消さない。solver testは構造contractの証拠でsemantic品質の証拠ではない。最大8unitの診断はproduction scalingの実装ではない。

## Next Steps

- 固定4callを実行し、exact/FM/FS/completeとscore行列・曖昧さ・costを保存する。直接生成から責務分離する方式の追加価値を評価するため。
- qualification改善時だけ未使用holdoutと旧回帰へ広げる。使用済み成功をproductionへ一般化しないため。
- 失敗時はscore品質と最適化失敗を分け、goldや閾値を結果へ合わせない。B停止/production4file制約/D条件を保持するため。
