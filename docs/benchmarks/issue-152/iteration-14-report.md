## 要約

同一file別symbolへのcall edge除外を最小修復した。direct独立反例の呼出関係が0→1になり、旧source/test対12組は維持された。構造coverage修復であり、semantic追加価値のGoではない。B No-Goを保持する。

## 検証結果

変更はgraph/main.goのedge条件で、同一fileでもcallerとcalleeが異なる場合に許可する。same-symbol自己呼出は引き続き除外する。回帰テストは実graph実行で別symbol2edgeと自己呼出除外を確認し成功。既存Python6test、graph package go test、graph build、diff checkも成功した。

固定反例: direct-dependent-independentはsemantic FP1、two-hop-dependent-independentはFP1、shared-callee-independentはrelation0。旧normalized3件はTP各4、FP/FN各0で計12正関係を保持。gold/過去の結果は変更せず、新結果をiteration-14-results.jsonに分離保存。model call0。以前のhelperは上書きせず別helper /tmp/issue152-samefile-graphを使用した。

## 考察

Aの同一file ChangeUnitをBのgraphが表現できない欠落は修復された。しかしcall依存と共有intentの区別は残り、hard mergeへの変更は独立反例を誤結合する。旧model exact0/3を今回の構造検証から改善済みと読み替えることはできない。

Go packageのfocused検証を実施したが、親IssueのVerification未達のためgoal最終品質ゲートは未実施。production処理、4file制限、モデルpolicy、unit上限は変更していない。

## Next Steps

- 新規専用worktreeで修復済みgraphの関係を型別に観測し、source/testとproduction caller/calleeの追加signalを独立fixtureで比較する。今回の反例は一律call relationの意味的曖昧さを示すため。
- 関係の型はtest declaration等の実source構造から決定し、goldやfixture名/pathの特例で分類しない。test関係もhard mergeには使わず、共通containerを揃えたbaselineで増分のみ評価する。
- 未使用ケースへsource/testの独立intent反例も含め、改善対象とguardrailを両方固定する。旧B No-Goと未解決のnew file/rename/history/cost条件を保持する。
