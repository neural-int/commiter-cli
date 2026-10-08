## 要約

Joint impactをsoft featureとしてA-onlyと14call比較した。未使用synthetic transfer3件はexact2/3→3/3、FM1→0、FS0→0で追加価値を観測した。一方、既知table-drivenケースのFS0→2が事前条件に反するため、今回の候補はNO-GO。B全体は未達、Cへ進まない。

## 検証結果

既知4件＋未使用3件、各A-only/impact。全14call completed/accepted/complete、unresolvedなし。独立改善はfresh comment-only absenceのFM1→0、他の独立2件は両条件exact成功。既知compensating valuesはexact改善、既知table-drivenはA-only all-merge FM4/FS0、impact all-split FM0/FS2。入力/goldはa5dff44、評価器/観測/pinsはbc3bc9eでmodel測定前に固定した。6contract testとdiff check成功。全membership/stop/token/wallは結果JSONLへ保存。

## 考察

独立評価での限定的な追加価値は得られたが、FMをFSへ置き換えるregressionを全体exact増加で相殺しない。独立3件は同作者・同構造のsynthetic transferであり、実repository一般化を証明しない。

impact conditionはcounterfactual結果と未変更test sourceを合わせて追加した。改善した独立caseはbehavior impactがないコメント変更であり、probe自体が改善の原因とは断定できない。未変更sourceのみとのablationが必要である。相殺反例をscorerは分離できたが、共同回復のpredicateがsemantic同値関係になったわけではない。

new-file/rename/sparse history/実規模memory/cache、旧65unit拒否は未解決として保持する。旧No-Goや今回gold/判定条件を変更しない。

## Next Steps

- 未変更repository sourceのみの条件を固定して比較する。高価なcounterfactual probeが必要なsignalか、context追加の効果かを区別するため。
- table-drivenのFS増加を回帰条件に残す。独立成功だけでGoへ変更しないため。
- Bの追加価値・regression回避・cost条件が揃うまでCへ進まない。
