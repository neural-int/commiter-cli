# Issue #173 Category分類の仮採用

2026-10-15、ユーザーの仮採用指示に基づき、Category専用のグループ差分入力、関連テストの役割整理、update/changeを含む13種類の分類規則、Iteration 3短縮Promptを取り込む。

## 採用する変更

- finalized groupにgroup_id、contents、nullable related_tests、意味に関係するfile_metadataを提示する。ホストのファイルID・Git identityはGo側に保持する。
- 同一グループの既存source/testパス対応だけを利用する。参照解決や目的の証明とはみなさず、不明・inlineテストはcontentsに保持し、未変更テストを追加読込しない。
- feat/fixだけ識別不能ならupdate、具体的Categoryを合理的に選べなければchange。欠落テストだけをfallback理由にしない。
- Iteration 3の意味判断用System Promptを製品categorySystemへ反映する。breaking_evidence_ref契約、安全指示、動的Schemaは維持する。
- 共通Schema・Validator・single-passの型受け入れ、scope制約、利用側commitlint/リリース規約の説明、benchmark146の入力アダプターを含める。

## 変更しない境界

Grouping、Text、3段構成、追加stageなし、4ファイル上限、共通120秒deadline、出力上限、test-only制約、breaking evidence、安全なファイル選択、完全割当、Git安全性、最終Validatorを維持する。Iteration 2のupdate除外方式は採用しない。

## 検証結果と未達

Iteration 1のNO-GO、Iteration 2の切り分け結果、Iteration 3の固定結果は当時の判定のまま保存する。この仮採用は検証Gateの達成宣言ではない。

- Iteration 3：既存Gold一致0/6のまま、境界11/20から13/20。F04のfixとF13のciが正答化した。F05のfeatからupdateへの変化はGoldのfixと不一致。
- update/changeの正当選択・誤選択件数は未改善。既知synthetic fixtureの歴史的比較であり、未使用holdoutや独立した人間評価ではない。
- Iteration 1の実際の3段実行は完成plan0/5。Iteration 3はCategory単独評価で、Text・共通deadline・完成planを再検証していない。
- token件数は未取得。Textの文字数制約と共通deadlineを別の課題として残す。
- Iteration 3最初の送信はIPC末尾改行欠落で推論前に拒否された。修正後の分類25回と拒否記録を別々に保持する。

Issue #173は未完了のまま保持する。Text・deadlineへの対処、token計測、受け入れ可能な完成planと未使用fixtureでの精度確認を後続の検証事項とする。

## 証跡

[Iteration 1](../benchmarks/issue-173/report.md)、[Iteration 2](../benchmarks/issue-173/iteration-2/report.md)、[Iteration 3](../benchmarks/issue-173/iteration-3-framed/report.md)。
