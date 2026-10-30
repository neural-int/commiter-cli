# H184-005 / evidence-placement

## 条件と検査

Categoryの根拠を共有辞書だけ、グループ内だけ、両方に配置する3条件。共通system（元の分類規則・信頼境界を追加済み）、最後のphase指示、Schema、samplingは固定。参照を解決して正規化すると全3条件のpayload・履歴・指示が一致。グループ内だけの条件でもfileの全メタデータ・差分・照合情報を保持し、共有側のfilesキーだけを省いた。情報欠落なし。

各条件のGrouping要求はbyte同一。モデル/共通文法準備の再利用状態を揃えるためGroupingを先に生成し、Categoryに固定グループを渡す。独立phase評価で、Text/最終plan validatorは実行しない。計6生成（Grouping3、Category3）、各1回、共有だけ→グループ内だけ→両方。

同じ保存済みSwift Release helper/Go runner。Qwen3.5-4B affine4bit/group64、Thinking OFF、32768/4096、native KV、JSON文法/scope制約ON、.6/.95/20、seed144、eager、従来prefill保護。KV再利用なし。新規依存・バイナリ再ビルドなし。

## 品質と時間

| 条件 | 文書 | UI追加 | メール修正 | 正解 | Category入力tokens | Category秒 | prefill秒 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| shared_only | docs | feat | feat | 2/3 | 4107 | 22.112 | 18.750 |
| group_only | docs | feat | fix | 3/3 | 4072 | 34.295 | 27.387 |
| both | docs | feat | fix | 3/3 | 6237 | 41.893 | 36.822 |

全条件ともCategory出力71tokensでJSON生成完了。break referenceは全てnone。共有だけのメールfeat誤分類が再現し、グループ内だけでもfixを維持した。今回のfixtureで正解するために2回の提示は必要ではなかった。対象と根拠を同じgroup内に置き、参照解決を不要にする構成が有望。位置・参照形式・token順のどれを使ったかという内部機序は確定しない。単一fixture/seedで一般化しない。

## 実行状態と速度の限界

| 条件 | 共通Grouping秒 | RSS最大MiB | system swap MiB | nominal/fairサンプル |
| --- | --- | --- | --- | --- |
| shared_only | 27.048 | 687.7 | 634.62–1229.38 | 21/4 |
| group_only | 30.321 | 1205.1 | 1221.38–1221.38 | 0/32 |
| both | 39.610 | 1150.9 | 1221.38–1221.38 | 0/41 |

全く同じGroupingの時間も27.05→30.32→39.61秒と増えた。共有だけの測定中にswapが増え、後半はfairとなるため、Categoryの22.11/34.29/41.89秒を配置だけによる速度差として解釈しない。監視98サンプル、fatalなし。

## KVとの関係

追加の非推論token監査でGrouping→Categoryの共通prefixを確認。共有だけ3052tokens、グループ内だけ223tokens、両方3052tokens。共有filesを省くと、今回の配置では先頭の共通データが変わり、従来の約3Kprefixは維持されない。グループ内だけが正解することと、3Kprefixを再利用できることは別条件。

現在の3条件からは品質と入力量の候補としてグループ内だけを優先できるが、そのまま同じKV再利用効率が得られるとは主張しない。将来の共通prefixには、どの情報を先頭で共有するかを再設計する必要がある。

## 状態

追加推論の再試行なし。生成0回のtoken監査は配置ごとに実施し、原文・token IDsはprivate保存。公開資料は集計のみ。Textや最終計画の採用検証は残るため、ハーネス登録は変更しない。今回の報告と先行実装差分は未コミット。
