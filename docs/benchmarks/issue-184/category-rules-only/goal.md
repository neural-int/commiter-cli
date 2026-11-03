# H184-005 / category-rules-only

## 条件

直前のSchema一回化版 single_schema の保存済み結果を基準とする。基準再実行なし。新規はCategory専用規則集約版の3phase各1回、計3生成だけ。Grouping/Textのsystemから分類/breaking規則とdocs定義を除外し、Categoryではsystemの同規則を保持して後半user内の同文反復だけを削除。信頼境界、Schema一回化、型説明、差分/履歴/コマンド、その他の設定は維持。入力監査で編集箇所以外の全要求一致、Grouping/Textのsystem同一を確認。削減文字数648/604/648。

混合配置：Grouping/Textの先頭に完全な共通files/history、Categoryはグループ内に完全な差分。Qwen3.5-4B affine4bit/group64 OFF、32K/4096、native KV、JSON文法/scope制約ON、.6/.95/20、seed144、eager、legacy prefill保護、同じ保存済みSwift Release。モデル/共通文法準備保持、KV再利用なし。

固定phaseの独立評価で、生成GroupingやCategoryを後段へ伝播しない。Textには固定正解Categoryを渡す。合成plan validatorはE2E成功を意味しない。

## 結果

| 条件 | Grouping | Category | 分類 | Text | 合成validator | 3phase秒 |
| --- | --- | --- | --- | --- | --- | --- |
| 保存済みSchema一回化 | 3/3 | 3/3 | docs/feat/fix | 完了 | 通過 | 58.667 |
| Category規則集約 | 3/3 | 3/3 | docs/feat/fix | 基準と完全同一 | 通過 | 54.095 |

| phase | 基準入力tokens | 集約入力tokens | 削減 | 基準prefill秒 | 集約prefill秒 | 基準全体秒 | 集約全体秒 | 基準decode tok/s | 集約decode tok/s |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 01 | 4915 | 4802 | 113 | 17.560 | 17.396 | 22.056 | 22.367 | 26.66 | 27.16 |
| 02 | 3811 | 3709 | 102 | 13.482 | 11.558 | 16.358 | 14.299 | 25.13 | 26.36 |
| 03 | 3790 | 3677 | 113 | 13.925 | 12.358 | 20.250 | 17.427 | 19.53 | 20.49 |

出力tokensは両群52/71/78。Category出力同一、Textのscope/summaryも全3グループで同一。メールの「空 Message-ID の正規化」では角括弧の具体性が失われる前段の課題が残り、今回その欠点が改善したという意味ではない。

## 性能と限界

| 条件 | RSS最大MiB | CPU最大% | system swap MiB | thermal nominal/fair |
| --- | --- | --- | --- | --- |
| 保存済み基準 | 1270.9 | 109.5 | 1125.38–1175.19 | 19/11 |
| 規則集約 | 1502.0 | 116.9 | 1127.19–1135.19 | 27/0 |

入力合計12516→12188tokens、328tokens削減（約2.6%）。実行58.667→54.095秒だが過去測定比較で熱/負荷が異なるため4.572秒を変更の効果とは断定しない。新規監視27サンプル全nominal、fatalなし、GPU取得27サンプル。RSSはprocessの周期サンプルでGPU共有メモリ全量ではない。詳細MLX active/peak/cache・各phase内時間はmeasurements.json、GPU/VM等時系列はprivate原本。残helperなしを確認。

このfixtureでは追加の分類/構造スコア低下もText変化も観測せず、次の比較の候補にできる。単一fixture/seed/試行で一般化しない。Categoryから同文反復も削除したため、規則の他phaseからの削除だけの単独試験ではない。Grouping/Textの共有入力配置を保持したがtoken prefix長は再監査していない。KV再利用未実装。

原本保持、基準再試行なし。中断したのはチャットの待機で、測定自体は正常終了した。生履歴/Thinking/個人pathは公開しない。本番コードとハーネス登録は未変更、報告は未コミット。
