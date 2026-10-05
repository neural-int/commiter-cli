## 要約

host-grounded before/after factsと既存soft relationを提示しても12fileでFS20が残り、H3は採用しない。次は変更関数への呼出関係を、fixtureの観測moduleとGo ASTから追加するH4を検証する。

## 検証結果

| H3 fixture | files | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | ---: | --- | --- | --- | --- | ---: | --- | ---: |
| contract-independent-6 | 6 | true | 0 / 0 | true | false | 1 | 1473 / 574 | 29.208 |
| contract-cross-boundary-12 | 12 | false | 0 / 20 | true | false | 1 | 3257 / 634 | 36.465 |

全2calls completed。context overflow/timeout/incomplete0。membershipは全件一意でunknown/duplicate/missing selected ID0。12fileはF001〜F004、F005/F006、F007/F008、F009/F010、F011/F012に分割した。typed before/afterはsource literalだけから生成し、goldに合わせて変更していない。metadata未実行。

LLM0の追加観測では、既存contract12 graphの12edgesは6組のsource/testそれぞれにchanged_identifierとsource_testが付いたものだった。root関数からconsumerへのcross-directory call関係は、この12edgesに含まれなかった。既存extractorはmodule resolutionを提供していない。fixture testsが書く実際のGo moduleはfixtureである。

追加Go AST prototypeはbefore/after各versionで一意の選択対象関数へbindする20call factsを観測した。gateway/storageからauth.Expired、receipts/ordersからbilling.Centsへの各4組がbefore/after双方で存在した。宣言が曖昧な場合はedgeを生成しない契約test成功。モデル評価はまだ行っていない。

## 考察

soft graphにmissing call関係があることは確認したが、これがFS20の原因だったとはまだ証明しない。H3はtyped差分とgraph提示の両方を変えており、raw-globalとの差を単独要因へ帰属できない。

H4では変更関数のcaller/calleeをhostが観測し、意味判断の入力へ渡す責務を追加する。構文の依存関係は同じ変更目的を証明しないためsoftのままとする。今回のmodule名はgold由来でなく実行可能fixtureのgo.modに基づく。productionの汎用Go module resolutionやtype checkerを完成した扱いにはしない。

## Next Steps

- H4のobserved call factsを6/12fileで各1回測る。missing dependency提示が共通契約のgroupingに寄与するかを検証するため。
- 有望な場合、共通calleeを持つ独立変更のguardrailを追加し、hard union相当の過剰結合にならないかを確認する。依存の存在を目的の一致へ読み替えないため。
- H4でも失敗する場合、test assertionのbefore/afterを観測契約IRとして抽出する責務を検討する。rootとconsumerの呼出関係だけでなく、変更された振舞いの対応が必要かを分離するため。
