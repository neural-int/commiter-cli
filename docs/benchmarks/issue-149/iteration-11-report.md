## 要約

H8のobserved-contractsはweak16でexact/FM0/FS0となったが、cross12でmemberの観測根拠が不足し停止した。品質条件やcoverage gateを緩和して採用せず、契約文・引用の生成をhostへ戻して、観測契約anchorへのglobal assignmentに責務を絞る次の仮説へ進む。

## 検証結果

| fixture | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | ---: | --- | ---: |
| weak16 | true | 0 / 0 | true | false | 1 | 1393 / 1351 | 50.361 |
| cross12 | null | null / null | false | true | 1 | 4005 / 776 | 36.341 |

2callsともcompleted、context overflow/timeout0。cross12はmissing_member_evidence。構造gateで停止し、部分partitionを返していない。停止時にsemantic正解やFM0を推定しない。weak16はunknown/duplicate/missing selected ID0。metadata未実行。

観測根拠spoof、別memberの根拠、duplicate/missing assignment、unknown subject、unresolvedを拒否するhost testsがpass。また、構造的にgroundedでも独立entityを誤統合したpartitionはFM1になり得るtestがpassし、groundingの構造とsemantic正解を分離した。

fresh holdout-protocol-and-health-8はweak16通過後、cross12結果確認前に固定。header/envelope/accepted versionのv1→v2という異なるentityの共通contractと、独立health変更を含む。before/afterとも実行可能なGo programとしてtest pass。H8の資格不成立によりまだ推論していない。goldをpayloadへ渡していない。

## 考察

観測されたsubjectとbefore/afterを明示する方式は独立定数の誤統合を避けたが、membershipに加えて根拠coverageを生成する契約を満たさなかった。Gemma native0/profile・input representation・output schemaを同時に変更したため、改善を単独要因に帰属させない。出力に含まれる自由文契約の真偽を構造検査だけで証明したとも扱わない。

入力fixture全体の観測根拠はhostに存在するので、モデルに同じ観測の引用・契約文を再生成させることが不可欠とは言えない。soft edgeをhard unionへ変えることなく、hostが保持するbefore/afterの実体へLLMが所属を割り当てる責務分割を試す余地がある。

## Next Steps

- H9 observed-anchor assignmentを実装し、hostの観測契約E-IDへ各fileを直接割り当てる。契約文の創作・引用coverageの再生成をsemantic taskから除き、観測rootへのglobal所属判断に限定するため。
- rootの実在、root file自身の同じanchor所属、unknown/duplicate/missing IDをhost検証する。soft relationのunionや多数決による修復を行わず、cycle/contradictory anchorは停止するため。
- 弱いedge16/cross12を各1回資格試験し、通過後に未推論fresh holdout・各レンジ・順序・metadataを測る。構造的な簡略化だけでsemantic改善を仮定しないため。
