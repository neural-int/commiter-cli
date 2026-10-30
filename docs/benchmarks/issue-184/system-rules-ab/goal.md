# H184-005 / system-rules-ab

## 条件

共通情報先頭の探索案を対照に、共通systemへ元のCategory分類規則・信頼境界を追加。Categoryにのみ適用する条件付き分類/breaking/perf規則、docs定義、信頼境界を保持。追加内容はsystem-addition.txtへ保存。元の後半user規則も残すので反復と入力長も変わる。共有files/history、file_id参照、メッセージ順、後半指示、Schema、その他設定が同一であることを全3要求で検査。

Qwen3.5-4B affine4bit/group64、Thinking OFF、32768/4096、native KV、Schema/scope制約ON、sampling .6/.95/20、seed144、eager、従来prefill保護、同一helperでモデル/共通文法準備再利用、KV再利用なし。前回と同じ保存済みRelease helper/Go runner。各3phase1回、6生成、control→system_rules。

各phaseの独立固定入力評価。Textは両群とも既存の確定Category（UI=feat、メール=fix）を受け取り、今回のCategoryを伝播しない。保存した出力の合成plan validator通過をE2E成功と呼ばない。

## 結果

| 条件 | Grouping | Category | 文書 | UI追加 | メール修正 | Text/合成validator | 秒 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| control | 3/3 | 2/3 | docs | test | fix | 完了/通過 | 53.689 |
| system_rules | 3/3 | 2/3 | docs | feat | feat | 完了/通過 | 53.617 |

| 条件 | phase | input/output tokens | prefill秒 | decode tok/s |
| --- | --- | --- | --- | --- |
| control | 01 | 4940/52 | 17.507 | 27.05 |
| control | 02 | 3926/71 | 12.411 | 26.95 |
| control | 03 | 3810/68 | 11.735 | 22.14 |
| system_rules | 01 | 5121/52 | 15.826 | 26.37 |
| system_rules | 02 | 4107/71 | 13.217 | 29.33 |
| system_rules | 03 | 3991/92 | 12.557 | 22.68 |

## 判断

対照のUI=test誤分類が再現。規則追加でUIはfeatへ戻った一方、メールがfix→featへ誤分類され、全体2/3のまま。この変更だけでは品質維持を達成できず不採用。system追加が分類挙動に作用した観測は得たが、roleのみを原因と断定できない。

Textは追加条件で「空の角括弧を含むMessage-IDを正規化」等へ変化したが、独立した固定Category入力なので生成Categoryを修復したという意味ではない。

追加は全phase181tokens。生成0回のtoken監査で入力5121/4107/3991、共通prefix G→C3052、C→T3054tokensを確認。共通prefixは維持したが、実KV再利用なし。53.689/53.617秒の差はほぼなく、単一試行から性能優位を主張しない。

監視54サンプル、異常なし。新規モデル推論6回、追加再試行なし。原本はprivate保存、会話やtoken IDsは公開しない。設定と入力の変更だけで、今回新規のモデル/Schema/ランタイム変更はなし。採用済みハーネス登録を変更せず、報告は未コミット。
