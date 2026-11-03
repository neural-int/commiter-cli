# H184-005 / schema-dedup

## 比較条件

現行の混合配置（Grouping/Text共通先頭、Categoryはグループ内完全差分）を対照に、各phaseのThe required JSON Schema is:に続く最初の完全なSchemaだけを削除。後段のExact JSON Schema、フィールド型/必須説明、ランタイム文法用Schema、指示、payload、その他全設定を維持。削除前の両SchemaとランタイムSchemaの意味的一致、編集箇所以外の要求全一致、基準要求と前回mixedの一致を確認。入力監査はmeasurements.jsonへ記録。

Grouping463文字、Category1027文字、Text760文字を削除。Qwen3.5-4B affine4bit/group64、OFF、32K/4096、native KV、grammar/scope ON、sampling .6/.95/20、seed144、eager、legacy prefill保護。固定Swift Release helper/モデル/文法準備再利用。KV再利用なし。current→single_schema各3phase1回、計6生成、再試行なし。

独立固定phase入力。Textは共通の固定正解Categoryを受け取り、生成したCategoryを伝播しない。合成plan validator通過をE2E成功と呼ばない。

## スコア

| 条件 | Grouping | Category | 分類docs/feat/fix | 合成validator | 3phase wall秒 |
| --- | --- | --- | --- | --- | --- |
| current | 3/3 | 3/3 | docs/feat/fix | 通過 | 55.758 |
| single_schema | 3/3 | 3/3 | docs/feat/fix | 通過 | 58.667 |

| phase | 現行入力tokens | 整理入力tokens | 削減tokens | 現行出力tokens | 整理出力tokens |
| --- | --- | --- | --- | --- | --- |
| 01 | 5121 | 4915 | 206 | 52 | 52 |
| 02 | 4072 | 3811 | 261 | 71 | 71 |
| 03 | 3991 | 3790 | 201 | 92 | 78 |

## Text本文と意味の差

| グループ | 現行summary | 整理summary |
| --- | --- | --- |
| G001 | 品質チェックと失敗時の確認セクションを追加 | 品質チェック手順を記載 |
| G002 | 期限日のツールチップと日付計算を実装 | 期限日ツールチップ追加 |
| G003 | 空の角括弧を含むMessage-IDを正規化 | 空 Message-ID の正規化 |

scopeは両群ドキュメント/UI/正規化で同じ。整理版は簡潔になった一方、文書の失敗時の確認セクション、メール修正の角括弧という具体性を落とした。メールの「空 Message-ID」は適用範囲が広く読める。UIの従来の「日付計算を実装」は整理版で消えた。Textを同一品質と断定せず、独立した意味評価の数値化はしていない。構造・分類スコアが維持されたことと、意味精度を分ける。

## 時間と資源

| 条件 | phase | phase秒 | load秒 | prepare秒 | grammar秒 | prefill秒 | decode秒 | decode tok/s |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| current | 01 | 21.282 | 1.563 | 0.065 | 0.722 | 16.159 | 2.098 | 24.79 |
| current | 02 | 15.529 | 0.000 | 0.048 | 0.005 | 12.879 | 2.595 | 27.36 |
| current | 03 | 18.943 | 0.000 | 0.054 | 1.190 | 13.344 | 4.286 | 21.47 |
| single_schema | 01 | 22.056 | 1.611 | 0.067 | 0.778 | 17.560 | 1.951 | 26.66 |
| single_schema | 02 | 16.358 | 0.000 | 0.042 | 0.003 | 13.482 | 2.825 | 25.13 |
| single_schema | 03 | 20.250 | 0.000 | 0.061 | 2.153 | 13.925 | 3.995 | 19.53 |

| 条件 | helper RSS最大MiB | CPU最大% | system swap MiB | thermal nominal/fair |
| --- | --- | --- | --- | --- |
| current | 1608.9 | 126.9 | 1125.38–1125.38 | 28/0 |
| single_schema | 1270.9 | 109.5 | 1125.38–1175.19 | 19/11 |

監視58サンプル、fatalなし、GPU状態取得58サンプル。詳細のCPU/MLXメモリ・phase内時間は集計JSON、GPU/VM/thermal時系列はprivate原本。RSSは周期サンプルのprocess値でGPU共有メモリ全量ではない。整理版の後半はfairとswap増加があり、入力削減しても時間は短縮しなかった。順序固定・各1回なので、整理そのものが遅くしたと断定しない。

## 結論と保存状態

このfixtureではGrouping/Category/validatorの低下は観測せず、入力計668tokensを削減。Text本文が変わり一部具体性は低下するため、全スコア無影響とまでは結論しない。Schema一回化は入力削減候補だが、全面採用や次段の変更を自動適用しない。元入力/応答/監視原本をprivate保存。追加推論なし。本番コードとハーネス登録は未変更。報告は未コミット。公開資料に生履歴・Thinking・個人pathを含めない。
