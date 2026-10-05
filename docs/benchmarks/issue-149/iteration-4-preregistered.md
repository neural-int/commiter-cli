# Iteration 4 事前登録: host-grounded facts

H3はLLMのper-file prose抽出を省き、hostがRawDiffの削除行と追加行をliteral before/after配列へ分離する。file ID/pathとprovenanceを保ち、既存graphのsoft relationだけを追加する。gold由来のedge/目的labelを追加しない。graph componentをgroup boundaryにせず、edgeの欠落でfileを落とさない。N<=16の全件からglobal membershipを1callで決める。profile/model/context/samplingはraw-global対照と同じ。

まずcontract-independent-6/cross-boundary12を各1回。両方exact/FM0/FS0/completeでなければproduction candidateにしない。両方成功なら4file、controlled8/9/16、独立6/16、順序逆転、final metadata/Validateへ進む。metadata前のprobeであり、call budget1を全plan budgetと誤認しない。

typed inputとsoft relation提示を同時に変えるため、単独の因果効果を主張しない。6/12の結果を受けてgoldや抽出規則をfixture固有に調整しない。
