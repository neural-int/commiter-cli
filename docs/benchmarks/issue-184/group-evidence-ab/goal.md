# H184-005 / group-evidence-ab

## 目的と条件

前回system規則追加済みの共通情報先頭案を固定し、Categoryのgroups[].filesをfile_idのみの参照から、完全なfile情報へ置換する。共有辞書・履歴・共通system・後半のphase指示・Schema・その他設定は保持。再掲した各fileオブジェクトが共有側と全値一致し、それをfile_idへ戻すと元のphase入力へ完全復元できることを検査した。

Qwen3.5-4B affine4bit/group64、Thinking OFF、32768/4096、native KV、Schema/scope制約ON、sampling .6/.95/20、seed144、eager、従来prefill保護legacy。前回と同じ保存済みRelease helper。両群で同じGroupingを先に生成しモデル/文法共通準備を保持してCategoryを1回ずつ生成。Groupingは固定Category入力に伝播させない。合計4生成、references→inline。Textと最終plan validatorは実行しない。KV再利用なし。

## 結果

| Categoryの条件 | 文書 | UI機能追加 | メール修正 | 正解 | 入力tokens | 出力tokens | Category秒 | prefill秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| file_id参照のみ | docs | feat | feat | 2/3 | 4107 | 71 | 15.760 | 13.113 |
| group内へ完全な根拠を再掲 | docs | feat | fix | 3/3 | 6237 | 71 | 22.537 | 19.835 |

両群のGroupingは同じ分割を生成。CategoryのJSON生成は両群とも正常完了し、breaking_evidence_refは全てnone。モデル/共通文法準備の再利用を確認、cacheReusedTokens=0。

| 資源 | 参照のみ | 再掲 |
| --- | --- | --- |
| helper RSS最大MiB | 1525.94 | 1596.91 |
| system swap MiB | 666.62で一定 | 666.62で一定 |
| 熱状態 | nominal | nominal |

監視41サンプル、異常なし。2phase全体は37.979/42.126秒だがGroupingウォームアップにも時間変動があるため、対象Categoryの時間を主に参照する。

## 判断

参照のみのメールfeat誤分類が再現し、group内へ完全な根拠を再掲するとfixへ戻った。今回のfixtureではgroup内に根拠を置くことが分類品質を改善した観測であり、ハーネスの根拠アクセス構造が影響する仮説を支持する。

ただし再掲は根拠の重複を増やし、入力も2130tokens増えた。参照解決を不要にすること、根拠が分類対象へ近づくこと、同じ内容の反復、入力長・token位置の変化は未分離。参照機構そのものが唯一の原因と断定しない。各条件1回で一般精度も断定しない。

再掲でCategoryは約6.78秒長くなった。共通prefix約3052tokensを将来再利用できるとしても、再掲増加分を含む6237tokens全体から計算量を評価する必要がある。今の6237から3052を引くと約3185tokensだが、これは将来のtoken数上の見積もりであり、実KV再利用・短縮秒数の測定ではない。

Category限定では再掲案が有望だが、3phase全体の採用やハーネス昇格は行わない。Text・最終計画評価と、より少ない重複で関係を明示する案の確認が残る。今回の実装・報告は未コミット、原本と先行実験を保持。
