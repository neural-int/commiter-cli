# H184-005 / layout-ab

## 条件と評価単位

KV再利用なしで、監査済み現行配置と共通Git情報先頭の探索案を比較。両群同じSwift Release helper・Go session、モデルと共通文法準備のみ再利用。Qwen3.5-4B affine4bit/group64、Thinking OFF、JSON Schema/scope制約ON、native KV、32768/4096、sampling .6/.95/20、seed144、eager、従来prefill保護legacy、時間制限なし。current→shared_first_exploratory各3phase1回、計6生成。

各phaseへ確定済み入力を与える独立評価。探索案のGroupingやCategoryの誤りを次phaseへ伝播させていない。Textには両群ともG002=feat等の同じ確定Categoryを渡した。全CLI E2Eではない。保存したCategory/Textを合成したplanに対するvalidator通過も、実際の一連の生成成功を意味しない。

## 変更点

現行はphase固有system指示とSchema説明が先頭。探索案は共通system→共通files/historyのuser→元のphase固有system文・Schema説明と残りのphase入力をuserへ配置。共通files/historyを一度だけ置き、後半files/groupsはfile_idで参照。情報・出力Schema・samplingは保持したが、role・指示位置・参照形式・serialize表記が変わるため、単なる項目順だけの変更ではない。

COMMITER_184_LAYOUT_WIRE=1は研究設定がある場合だけ、完成したwireを再加工せず使う。生成時も元Schemaで文法制約する。token数は監査済み配置と一致した。

## 結果

| 条件 | Grouping | Category | Text JSON生成 | 合成出力validator | 秒 |
| --- | --- | --- | --- | --- | --- |
| current | 3/3 | 3/3 | 完了 | 通過 | 52.638 |
| shared_first_exploratory | 3/3 | 2/3 | 完了 | 通過 | 49.967 |

Categoryは文書docs・メールfixを維持したが、UI機能追加がfeat→testへ誤分類。Textも変化し、探索案のscopeはドキュメント/UI機能/正規化修正となり、要約は短くなった。日本語scope自体は禁止されていないので、それだけを誤りとはしない。意味の全品質は別途確認が必要。

| 条件 | phase | input/output tokens | prefill秒 | decode tok/s |
| --- | --- | --- | --- | --- |
| current | 01 | 5123/52 | 15.992 | 27.50 |
| current | 02 | 4027/71 | 12.199 | 27.26 |
| current | 03 | 3911/83 | 12.142 | 22.70 |
| shared_first_exploratory | 01 | 4940/52 | 15.363 | 27.82 |
| shared_first_exploratory | 02 | 3926/71 | 12.129 | 27.61 |
| shared_first_exploratory | 03 | 3810/68 | 11.512 | 23.20 |

## 判定

品質維持条件を満たさず探索案は不採用。52.638→49.967秒（約5.1%短縮）だが、入力tokensやText出力tokensも減り、role/参照形式も変わる。KV cacheReusedTokensは全phase0であり、cache高速化ではない。1回・順序固定で一般性能は断定しない。

共通prefix約2870tokensが得られることと、生成品質を維持できることは別。今回のデータからsystem→userの権限変更、先頭で与える履歴の影響、ファイル参照形式のどれが誤分類原因かは分離できない。この配置のままKV実装へ進めない。

監視52サンプル、異常なし。新研究分岐を含むReleaseビルド成功。今回の新規モデル生成は6回、追加再試行なし。前回の原本保持。入力と生の会話・token IDsはprivate保存。今回の変更と報告は未コミット、採用済みハーネス登録は変更していない。
