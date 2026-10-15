# Issue #173 固定検証結果

## 要約

候補版は採用不可（NO-GO）。既存5条件のCategory Gold一致は現行版・候補版とも0/6グループで、候補版は4件を誤って`update`、2件を誤って`feat`へ分類した。境界20件は11/20、実際の3段実行で受理された完成planは0/5だった。Group-scoped Raw Diffと二段階フォールバックの実装は検証ブランチに保持し、mainへの採用・mergeは行わない。

## 検証結果

- 基点はPR #172の`0f9278c33c2f477372bc7dd07c7616a738dd6268`。既存5 fixtureと過去Goldを変更せず再利用し、新規境界20 fixtureのGoldを推論前に固定した。すべてエージェント作成のsyntheticデータであり、独立した人間評価ではない。F11の`test`は既存のtest-only Schema制約で強制される。
- Category単独30回＋実際の3段実行13回＝生成43/45回。再試行・修復・追加の品質判定LLM呼び出しは0。Category単独測定は研究用比較であり、製品の3段に追加stageを設けたものではない。
- モデルはGemma E4B 4bit、revision `475b9088d29754a3379866cf5aeb6b41acd313c2`。temperature 0、top-p 1、top-k 0、seed 144。Category native thought上限384、出力上限512。3段の出力上限768/512/768、共通deadline 120秒、context ceiling 65536、4-file上限を維持。
- 実機はarm64/macOS 26.6.2、RAM 16 GiB。Categoryの全30リクエストはcompletedかつSchema適合。helper RSS最大は約3.686 GiB。観測pressure levelは1、Categoryの1試行前後のswap増分最大は630.88 MiB。これはhelperのMetal/GPU総ピークやswap変化の因果帰属を示さない。
- 入出力token件数は未取得。既存tokenize-onlyバイナリは推論前の3件の互換性確認を`malformed_request`で拒否した。本番helperも件数を返さない。byte数と未取得のtoken数を混同しない。

| 既存5条件・6グループ | 現行Category | 候補Category |
| --- | ---: | ---: |
| Gold一致 | 0/6 | 0/6 |
| feat/fix誤断定 | 6 | 2 |
| update誤選択 | 0 | 4 |
| wall time中央値 | 25.753秒 | 22.640秒 |
| wall time最大 | 31.677秒 | 102.313秒 |
| 入力byte数合計 | 38231 | 76511 |

wall timeはホスト上の観測値であり、統計的な高速化の証明ではない。初期試行と回帰テストの時間帯が一部重なったため、その点も性能比較の限界である。

| 境界20件のfallback | 選択 | 正当選択 | 誤選択 | 本来選ぶべきケースの見逃し |
| --- | ---: | ---: | ---: | ---: |
| update | 6 | 2 | 4 | 1 |
| change | 1 | 1 | 0 | 1 |

| 境界fixture | 固定Gold | 実際のtype | 一致 |
| --- | --- | --- | --- |
| F01 | feat | feat | yes |
| F02 | fix | update | no |
| F03 | update | update | yes |
| F04 | fix | update | no |
| F05 | fix | feat | no |
| F06 | refactor | refactor | yes |
| F07 | perf | refactor | no |
| F08 | docs | docs | yes |
| F09 | style | style | yes |
| F10 | update | style | no |
| F11 | test | test | yes |
| F12 | build | chore | no |
| F13 | ci | chore | no |
| F14 | chore | chore | yes |
| F15 | revert | revert | yes |
| F16 | update | update | yes |
| F17 | change | change | yes |
| F18 | change | update | no |
| F19 | feat | feat | yes |
| F20 | fix | update | no |

候補版の26判断（既存6＋境界20）に`fix`は0件だった。Goldが`fix`の10判断のどれも正答していない。断定回避を正答として扱っていない。

| 実際の3段実行 | Grouping Gold | calls | wall time | 受理plan | 停止 |
| --- | --- | ---: | ---: | --- | --- |
| A | 一致 | 3 | 48.256秒 | no | Text invalid_schema |
| B | 一致 | 3 | 74.410秒 | no | Text invalid_schema |
| C | 一致 | 3 | 88.435秒 | no | Text invalid_schema |
| D | 一致 | 2 | 121.506秒 | no | Category interrupted_by_product（共通120秒deadline） |
| E | 一致 | 2 | 121.379秒 | no | Category interrupted_by_product（共通120秒deadline） |

Text長制約の実測：

- A：G001: scope=20, summary=54（上限16/48）。
- B：G001: scope=9, summary=54（上限16/48）。
- C：G001: scope=8, summary=59、G002: scope=8, summary=43（上限16/48）。

- 完了した各試行のHEAD、index登録内容、worktree diff、statusのdigestは前後一致。GroupingとTextの実リクエストは、到達したphaseについて固定リクエストと一致。Categoryの実リクエストも固定入力と一致した。
- Category投影はグループ所属を変更せず、完全な選択済み差分と意味に関係するfile stateを保持する。同一グループの既存source/testパス対応だけを役割の観測として利用し、参照解決や同目的の証明とは扱わない。対応不明・inlineテストはcontentsに保持する。未変更テストは追加読込しない。opaque/binaryの内容は投影しない。
- `update`/`change`は共通Schema・最終Validator・single-passで許容し、type名scopeは引き続き拒否。breaking evidenceの候補・none/unresolved・test-only・scope/summary長・Git安全性は維持した。
- 回帰確認：pass。初回`go test ./...`は旧benchmark146のCategory packetアダプターで失敗した。基点の同テストPASSを確認し、削除済みfile_idsへの依存をパス→既存ホストID対応に更新した。このアダプター修正では過去計測・Gold・固定実験バイナリ・固定済みCategoryプロンプトを変更していない。

## 考察

候補版ではfeat/fixの誤断定6件が2件に減ったが、残り4件はGoldがfixのケースをupdateに置き換えたもので、Gold一致は増えていない。境界例でも明確な修正をupdate/feat、性能改善をrefactor、UI色変更をstyle、build/ciをchoreとする誤分類を観測した。この条件では二段階fallbackを正確に運用できたとはいえない。

今回はCategory入力・意味判断プロンプト・type enumを合わせて変更した比較であり、各要素の個別効果を切り分けていない。Raw Diffの導入だけが失敗または成功の原因であるとは断定できない。テストの有無とtypeにも単純な因果関係は主張しない。

最大入力EのCategory単独実行は102.313秒で、Group-scoped Raw Diffの保持は入力負荷を増やした。3段の共通deadlineと各phaseの実測を別に評価する必要がある。scope/summary違反はCategoryの成功から自然に解消しない。Textを変更せず全完成planを受理させる完了条件は今回満たせず、Text・deadlineの変更はIssueで分離を指示されている。これ以上その修正へ進めることは現在の検証範囲を越えるため、goal.mdのスコープに関する停止条件を適用する。

## Next Steps

- Textのscope/summary制約遵守と共通deadline内のphase予算を別Issueで扱う。完成planの失敗をCategoryの調整だけでは解消できないため。
- Categoryを再検証するなら、既存契約が差分の旧実装と矛盾する場合の根拠利用と、複数独立目的のchange判定を新しい有限契約で比較する。今回の誤分類から得た仮説であり、現fixtureの反復改善を一般的な精度向上としない。
- token計測はhelperの計測契約を確認してから別runで固定する。今回の未取得値を推測で補完しない。
- Issueは未完了のまま保持し、未達のtoken計測・完成plan条件にチェックを入れない。mainへの採用・merge・リリースは行わない。

証跡： [preregistration.json](preregistration.json)、[Gold](gold.json)、[Category結果](category-results.jsonl)、[3段結果](integrated-results.jsonl)、[診断](integrated-diagnostics.json)、[品質確認](quality.json)、[計測前後のhash照合](measurement-integrity.json)。

公開上の処理：準備スクリプトの環境固有helperパスだけを計測終了後に引数化した。計測時hashと公開版hashを[publication.json](publication.json)に別記し、原本はローカルに保持した。条件・入力・Gold・生の測定結果は変更していない。過去のpreregistrationが保持するprepare.pyのhashは公開版と意図的に異なる。
