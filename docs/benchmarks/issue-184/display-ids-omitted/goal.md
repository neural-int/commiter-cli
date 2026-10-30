# H184-005 / display-ids-omitted

## 条件

表示field省略版の保存済み結果を基準に、LLM表示からblock_id/execution_refだけを省略。基準は再実行しない。ホスト元データは維持。Groupingではblock_id9件/execution_ref14件、Category/Textではblock_id各9件を省略。残りのfield値・文字列表現・参照・system・指示・出力Schema・runtime Schema一致を監査。削減文字数443/144/144。file_id/group_id/context_ref/mentioned_file_ids/diff_rangeなどの参照は維持。

Schema一回化・Category専用規則集約・task/schema_version/空user_requests表示省略を引継ぎ。混合配置、Qwen3.5-4B affine4bit/group64 OFF、32K/4096、native KV、grammar/scope ON、.6/.95/20、seed144、eager、legacy prefill保護。同じSwift Release helper、モデル/共通文法準備保持、KV再利用0tokens。新規3phase各1回のみ。

固定独立phase評価。前段出力を後段へ伝播せず、Textには固定正解Categoryを入力。合成plan validator通過をE2E成功と呼ばない。

## 結果

| 条件 | Grouping | Category | Text | 合成validator | 3phase実行秒 |
| --- | --- | --- | --- | --- | --- |
| 保存済み表示field省略版 | 3/3 | 3/3 | 完了 | 通過 | 52.604 |
| block_id/execution_ref表示省略 | 3/3 | 3/3 | scope/summary全3件が基準と同一 | 通過 | 49.978 |

Categoryはdocs/feat/fix、breaking_evidence_refは全noneで同一。出力tokensは52/71/78。Textのメールsummary「空 Message-ID の正規化」は従来の角括弧の具体性不足を引き継ぐため、同一出力を完全な意味正解の保証とはしない。

| phase | 基準入力tokens | 省略後 | 削減 | 基準prefill秒 | 省略後prefill秒 | 基準phase秒 | 省略後phase秒 | 基準decode tok/s | 省略後decode tok/s |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 01 | 4773 | 4630 | 143 | 16.743 | 14.860 | 21.679 | 19.437 | 27.30 | 27.49 |
| 02 | 3682 | 3628 | 54 | 11.764 | 11.626 | 14.394 | 14.297 | 27.60 | 27.12 |
| 03 | 3650 | 3596 | 54 | 11.792 | 11.556 | 16.529 | 16.241 | 22.38 | 22.43 |

合計12105→11854tokens、251tokens（約2.1%）削減。実行52.604→49.978秒だが過去結果比較/各1回なので変更の純粋効果と断定しない。

## 資源と判断

| 条件 | helper RSS最大MiB | helper CPU最大% | system swap MiB | thermal nominal/fair |
| --- | --- | --- | --- | --- |
| 保存済み基準 | 1334.6 | 126.6 | 1119.19–1119.19 | 27/0 |
| ID表示省略 | 1313.8 | 304.7 | 1119.19–1119.19 | 25/0 |

新規監視25サンプル全nominal、fatalなし、GPU取得25サンプル、残helperなし。詳細MLX active/peak/cache・phase内時間はmeasurements.json、GPU/VM等時系列はprivate原本。RSSはprocess周期値でGPU共有メモリ全量ではない。

このfixtureでは表示省略による分類/構造スコア低下、Text変化を観測しない。ホスト内の識別/追跡は削除しない。本検証は2フィールド同時の省略で、将来入力や出力で参照を追加する場合は再評価が必要。単一fixture/seed/試行で一般化しない。共有prefixのtoken長は再監査しておらずKV速度効果は未評価。

原本保持・再試行なし。公開資料に生履歴/Thinking/個人pathを含めない。本番コード・ハーネス登録は未変更。報告は未コミット。
