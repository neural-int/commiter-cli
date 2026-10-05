# Iteration 3 事前登録: bounded batch extraction

Iteration1のNext Stepsと4file比較の観測（現行Stage1 28.410秒、per-file IR 183.665秒）に従い、抽出call数を改善するH2を固定する。Iteration2の12file/holdout結果は本書作成時点で未確認。

H2: 最大4fileの独立IRをkeyed batchで抽出し、全batchのIRからglobal membershipを決める。batchはirreversibleなgroupではない。per-file IRと同じbefore/after/changed_contract/symbols、各240文字上限。抽出はbounded-text/native0/出力768、globalはbounded-grouping/native512/出力768。generation profileも異なるためbatchだけの単独効果とは主張しない。model/context/samplingは固定、retry/repair0、120秒/call、600秒/fixture。N<=16でceil(N/4)+1calls（metadata未実行）。

まずcontract-independent-6とcontract-cross-boundary-12を各1回測る。どちらかがexactでなければproduction candidateとせず、raw-globalを同じ失敗fixtureで1回比較し、情報欠落と全体意味判断の区別を試みる。exactなら4file、ordering逆転、未使用独立評価を追加し、metadata/Validate接続へ進む。

選定にはexact/FM0/FS0/completeを要求する。構造失敗はFM0扱いにせずnullとして記録。最大16fileでmetadataを含む実測costに基づく採用予算の根拠が必要で、grouping only probeの成功をproduction candidate成立とはしない。
