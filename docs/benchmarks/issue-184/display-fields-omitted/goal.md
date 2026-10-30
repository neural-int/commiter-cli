# H184-005 / display-fields-omitted

## 条件

直前Category規則集約版の保存済み結果を基準とし、基準は再実行しない。LLM表示の各phase本文からtask/schema_versionを省き、共通履歴のuser_requestsが空配列の場合だけキーを省略。元の固定入力は保持。system、分類指示、出力Schema、runtime grammar、残りの値・各fieldの文字列表現は完全に維持。ホストIPCの正規JSONは維持。今回は全3contextのuser_requestsが空。削除は各phase132/138/138文字。入力監査記録あり。

Schema一回化・Category規則集約を保持。混合配置、Qwen3.5-4B affine4bit/group64 OFF、32K/4096、native KV、grammar/scope ON、.6/.95/20、seed144、eager、legacy prefill保護。同一Swift Release helper/モデル/共通文法準備保持。実KV再利用0tokens。新規3phase各1回、追加再試行なし。

固定phase独立評価。TextのCategoryは固定正解で、生成した前段出力を後段へ伝播しない。合成plan validatorをE2E成功と呼ばない。

## 結果

| 条件 | Grouping | Category | 分類 | Text | 合成validator |
| --- | --- | --- | --- | --- | --- |
| 保存済みCategory規則集約 | 3/3 | 3/3 | docs/feat/fix | 完了 | 通過 |
| 表示field省略 | 3/3 | 3/3 | docs/feat/fix | 基準と完全同一 | 通過 |

Text scope/summaryは全3グループで同一。従来のメールsummary「空 Message-ID の正規化」は角括弧の具体性が欠ける点も同じで、意味品質の課題を解消したという意味ではない。

| phase | 基準入力tokens | 省略後 | 削減 | 基準prefill秒 | 省略後prefill秒 | 基準phase秒 | 省略後phase秒 | 基準decode tok/s | 省略後decode tok/s |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 01 | 4802 | 4773 | 29 | 17.396 | 16.743 | 22.367 | 21.679 | 27.16 | 27.30 |
| 02 | 3709 | 3682 | 27 | 11.558 | 11.764 | 14.299 | 14.394 | 26.36 | 27.60 |
| 03 | 3677 | 3650 | 27 | 12.358 | 11.792 | 17.427 | 16.529 | 20.49 | 22.38 |

入力合計12188→12105tokens、83tokens削減（約0.68%）。出力tokensは52/71/78で同じ。3phase実行時間54.095→52.604秒。保存済み基準との各1回比較なので短縮を変更だけの効果と断定しない。

## 資源と判断

| 条件 | helper RSS最大MiB | CPU最大% | system swap MiB | thermal nominal/fair |
| --- | --- | --- | --- | --- |
| 保存済み基準 | 1502.0 | 116.9 | 1127.19–1135.19 | 27/0 |
| 表示field省略 | 1334.6 | 126.6 | 1119.19–1119.19 | 27/0 |

今回監視27サンプル全nominal、fatalなし、GPU取得27サンプル。残helperなし。MLX active/peak/cache、各phase内時間はmeasurements.json、GPU/VM等時系列はprivate原本。RSSは周期processサンプルでGPU共有メモリ全量ではない。

このfixtureでは省略による分類/構造スコア低下やText変化は観測せず、表示簡素化の候補にできる。ただし削減量は小さく大きな速度改善の根拠にはならない。単一fixture/seedで一般化しない。今回の投影はuser_requestsが全て空であることをassertしてから省略した。非空履歴の処理と評価は今回の対象外。共通prefixのtoken長は再監査せず。

本番コード・ハーネス登録は未変更、報告は未コミット。元入力/応答/監視原本保持。生履歴/Thinking/個人pathは公開しない。
