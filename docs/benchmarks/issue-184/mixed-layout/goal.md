# H184-005 / mixed-layout

## 目的と条件

Grouping・Textは共通情報先頭の入力を維持し、Categoryだけ共有filesを省いてグループ内に完全なfile情報・raw_diffを置く。基準は全3phaseの共有辞書配置。元データ、履歴、分類規則、Schema、samplingは同一。Categoryは参照展開後の正規化payloadの一致を確認。Grouping/Text要求はbyte同一。

Qwen3.5-4B affine4bit/group64、Thinking OFF、32K/4096、native KV、grammar/scope制約ON、.6/.95/20、seed144、eager、従来prefill保護。保存済み同一Swift Release helper。各条件内でモデルと共通文法準備を保持。新規依存・本番コード変更なし。shared_all→mixed各1回、計6生成。

今回は入力配置の品質確認。実KV再利用は0tokens。各phaseの固定独立入力で、Categoryへ生成Groupingを、Textへ生成Categoryを伝播しない。Textは両条件とも同じ正解Categoryを入力する。生成結果を合成したplanのvalidatorは評価するがE2E成功とは呼ばない。

## 結果

| 条件 | Grouping | Category | 文書/UI/メール | Text | 合成plan validator | 3phase合計秒 |
| --- | --- | --- | --- | --- | --- | --- |
| shared_all | 3/3 | 2/3 | docs/feat/feat | 完了・両群同一 | 通過 | 59.469 |
| mixed | 3/3 | 3/3 | docs/feat/fix | 完了・両群同一 | 通過 | 56.940 |

| 条件 | phase | 入力/出力tokens | load秒 | 準備秒 | grammar秒 | prefill秒 | decode秒 | decode tok/s | phase全体秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| shared_all | 01 | 5121/52 | 1.654 | 0.058 | 0.689 | 19.913 | 1.976 | 26.32 | 25.021 |
| shared_all | 02 | 4107/71 | 0.000 | 0.045 | 0.003 | 13.291 | 2.647 | 26.82 | 15.989 |
| shared_all | 03 | 3991/92 | 0.000 | 0.056 | 1.166 | 12.882 | 4.300 | 21.40 | 18.460 |
| mixed | 01 | 5121/52 | 1.153 | 0.056 | 0.686 | 16.928 | 1.974 | 26.34 | 20.833 |
| mixed | 02 | 4072/71 | 0.000 | 0.039 | 0.002 | 13.851 | 2.648 | 26.81 | 16.542 |
| mixed | 03 | 3991/92 | 0.000 | 0.057 | 1.293 | 13.872 | 4.294 | 21.42 | 19.564 |

## 資源状態と解釈

| 条件 | helper RSS最大MiB | helper CPU最大% | system swap MiB | thermal nominal/fair | GPU取得サンプル |
| --- | --- | --- | --- | --- | --- |
| shared_all | 677.8 | 160.8 | 1173.38–1189.38 | 30/0 | 30 |
| mixed | 2408.0 | 143.6 | 1165.38–1173.38 | 16/13 | 29 |

RSSは周期サンプルのprocess RSSで、MLXのGPU共有メモリ全量や厳密な瞬時ピークとは別。MLX active/peak/cache、CPU内訳、phase時間の実値はmeasurements.json、GPU/VM/thermal等の時系列はprivate原本へ保存。監視fatalなし。後半thermalがfairへ変化し、同じGrouping要求でも25.021→20.833秒なので時間差を配置の効果と断定しない。

## 判断と残課題

Categoryのみグループ内配置で3/3となり、Groupingは3/3、同じ固定入力のTextは両群で完全同一だった。Categoryのメールfeat誤分類が基準で再現し、比較群でfixへ改善。今回の固定phase評価では混合配置で品質を維持できた。Textの同一出力は完全な意味品質の保証ではなく、validator通過もCategoryの正解を保証しない。単一fixture/seed、各1回で一般化しない。

Grouping/Text要求は保存済み共通先頭形式を維持した。既存token監査のG→C3052/C→T3054から、これらのGrouping/Text先頭3052tokensの一致は保持される。新しいtoken監査やキャッシュ計算は行っていない。Categoryは独立prefill。実際のGrouping時点の状態保存・Category中の保持・Text復元は未実装であり、速度短縮・メモリ費用は未評価。次はこの入力配置を固定し、全cache/recurrent状態の正しい分岐を実装してKVなし/ありを比較する。

## 保存状態

原本保持、追加再試行なし。会話・Thinking・token IDs・個人pathは公開資料に収録しない。報告と既存の研究変更は未コミット。採用済みハーネス登録は変更しない。
