## 要約

同一container/metadataで未使用3件を比較した結果、repository条件は誤結合を36から16へ減らしたが、exactは両条件0/3、誤分割は両条件12だった。事前条件のexact増加を満たさず、B No-Goを保持する。

## 検証結果

| 範囲 | 条件 | complete | exact | false merge | false split |
| --- | --- | --- | --- | --- | --- |
| 未使用3件 | A-only | 3/3 | 0/3 | 36 | 12 |
| 未使用3件 | repository | 3/3 | 0/3 | 16 | 12 |
| #149回帰4件 | A-only | 3/4 | 0/4 | 10 | 45 |
| #149回帰4件 | repository | 3/4 | 0/4 | 10 | 45 |

回帰のexact分母4には事前拒否1件を含み、拒否のexact値はnullのまま保存した。受理3件もexact0/3。14比較行、実推論9call、同一request再利用3行、65unitの事前拒否2行。実推論は全completed/acceptedで、input19829/output1106tokens、累積252.562秒。再利用は独立測定として数えない。request SHA256とmembershipをraw結果へ保存した。

fixtureと条件は32c0fe5で事前固定した。各新規fixtureは5file/4intentで、未変更adapterの実際のcall経路を追加する。goldは評価側だけに保持する。model/context/output/time上限、旧64unit上限、gold、過去の結果は変更していない。

## 考察

repository追加による誤結合削減の限定的なsignalはあるが、sourceと対応testの4組を完全には回復できない。fresh-2は全8unitを分割してFM0/FS4、fresh-3は誤った対応でFM4/FS4だった。情報量の増加だけでは意味的partitionの成立を支持できない。

同じ作者によるsynthetic3件であり、実repositoryの独立holdoutではない。new file/rename/sparse history/memory/cacheと65unit拒否も未解決で、Cへ進む条件は満たさない。productionの4file制約は維持する。

## Next Steps

- 新規専用worktreeで、モデルへcall graphの読解を委ねる方式と、hostが解決したunit間経路を提示する方式の差を最小検証する。fixture固有の対応表を実装せず、既存ASTの一般的な経路解決だけを使う。
- まず今回の失敗で、正しいsource/test対の経路がunit_relationsに存在するかを確認する。抽出不足とselector失敗を切り分け、実際の欠落が確認できるまでコードを変更しない。
- exact改善と回帰回避が立証されるまでB No-Goを保持する。
