## 要約

canonicalなentity/contract IRは入力presentationの依存をhostで除いたが、12fileでFM36、weak16でFM120を観測した。決定的な入力は意味的な正解を保証せず、H6は採用しない。次はglobal groupingのsemantic task専用generation contractとgrouping-only model routingを分けて測る。

## 検証結果

| H6 fixture | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | ---: | --- | ---: |
| cross-boundary12 | false | 36 / 0 | true | false | 1 | 6119 / 634 | 40.592 |
| weak-independent16 | false | 120 / 0 | true | false | 1 | 3500 / 674 | 36.006 |

全2calls completed、context overflow/timeout/incomplete0、unknown/duplicate/missing selected ID0。両方とも全fileを1groupへまとめた。metadata未実行。incoming Filesとedgesの順を反転してもcanonical Files/edges/call/assertion/declaration JSONが同一になるhost testが成功した。constant16のbefore/after宣言32件はkind/name/valueを保持するhost testが成功した。

## 考察

canonical化はpresentation入力を固定したが、固定した順序で正解になる根拠は得られなかった。pathやIDの特定の順を成績に合わせて選び直さない。観測IRを増やしても、同一global判断task・固定native512のままでは不十分だった。思考枠の不足とmodelのsemantic grouping能力のどちらが原因かは、この結果だけでは分からない。

## Next Steps

- H6と同じ入力でGemmaのglobal専用native1024/output1536 contractを最小probeで測る。4file用native512/output768を全体判断へ転用した予算が十分かを分離するため。samplingは変更しない。
- cached Qwen3.5-4B/Phi-4-mini/Granite4.2-3Bのgrouping-only native0/output1536 routeを小さなqualification probeで比較する。metadataのGemma構成を維持したままglobal判断の責務だけを別checkpointへ渡せるかを確認するため。
- cacheに存在するrevisionだけを使いmodelを取得しない。runtimeは計測helperのコピーへprofileを追加し、production default/backend/runtimeを変更しない。
- 各routeが資格を満たした場合だけ、独立評価・大規模・順序・metadata/Validateへ進む。cacheにあるという理由で採用しないため。
