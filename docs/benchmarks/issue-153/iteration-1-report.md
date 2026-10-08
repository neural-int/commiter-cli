## 要約

A-only C qualificationはdirectとscore+partitionともexact0/2。FM24→12、FS8同値で、事前条件のexact改善は未達。solverは一意な最適解を再現したがscoreはgoldに対し各10pair誤り。さらに隠れたadapterを変更するとgoldだけ変わりA-only入力は同じになるため、このfixtureの情報不足を確認した。C方式全体のNo-GoやD必要性の証明とは扱わない。

## 検証結果

| 条件 | complete | exact | FM | FS |
| --- | --- | --- | --- | --- |
| A-only direct | 2/2 | 0/2 | 24 | 8 |
| pair score + exact partition | 2/2 | 0/2 | 12 | 8 |

全4call completed/accepted。input9064/output1112tokens、合計127.149秒。各score28pair、各solver4140状態、一意最適解。入力順/score順を反転して同じpartitionを確認。最適objective34/29、gold objective2/7。goldは評価側のみで最適化に使用しない。schema/systemの責務変更を含む比較でoptimizer単独効果と呼ばない。

事後counterfactual診断では、未変更adapterのC0/C1 routingを入れ替え、評価側source/test goldを実routingへ対応させた別状態を作った。元fixtureは変更していない。A-only payloadは完全一致するがgoldは異なる。payload SHA256と診断値をiteration-1-audit.jsonに保存。追加model call0。3solver contract test/diff check成功。

## 考察

solverは入力scoreに対する一意のglobal optimumを返し、goldはそのobjectiveで劣るため、今回の誤りを最適化探索の取りこぼしとは説明できない。score自体はsource同士を結合しsource/testを負とした。ただしsource/testの真の対応は未変更adapter内にあり、今回A-onlyでは除外される。異なる正解が同じ入力になることから、このsetでは確実な対応回復に必要な情報が欠ける。これを汎用scorerだけのボトルネックと呼びDへ進む根拠はない。

事前gateの未達はそのまま保持する。counterfactualは事後診断で、qualificationの成功へ書き換えない。最大8unitのsolverはproduction scalability/staging/source mapping/legacy全域の証明ではない。B No-Goとproduction4file制限は保持する。

## Next Steps

- 新規専用worktreeでA-only入力にsource/test対応が実際に含まれる未使用fixtureを事前固定する。今回のhidden routingによる情報不足と、scoring/partition責務分離の効果を分けるため。
- 同一file独立変更・5file以上のsource/test連動・shared dependency反例を含め、今回のsolver/score尺度/予算は変更しない。goldへ閾値やcandidateを合わせないため。
- 入力に情報がある条件の固定比較結果を確認後に旧回帰/最終holdoutへ進む。成功や不成功をC全体/production/Dへ早期一般化しないため。
