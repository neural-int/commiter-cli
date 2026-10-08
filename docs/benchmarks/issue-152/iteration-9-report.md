## 要約

counterfactual観測を空にしたcontext-only条件は、使用済み7caseでexact7/7、FM0、FS0だった。前回impact条件のFS回帰も解消した。ただし同一metadataを保持した診断であり、repository source由来の一般化signalと確定せずB Goは保留する。

## 検証結果

追加7callはcompleted/accepted/complete、unresolvedなし。旧A-only exact4/7・FM6・FS0、旧impact exact6/7・FM0・FS2に対してcontext-only exact7/7・FM0・FS0。観測配列を空にした以外は同じfield名/semantics metadata/未変更source、同じsystem/schema/helper/model条件を保持した。新規test probeなし。条件とhelper pinは53fa8b8で固定。6contract testとdiff check成功。

## 考察

counterfactual観測の増分はこの7caseでは必要なく、むしろ旧table-drivenでFSを増やした。全7caseは使用済み入力であり、新たな独立検証へ読み替えない。

旧table-drivenはRepositoryが空であるにもかかわらず改善したため、少なくともその改善を未変更sourceだけに帰属できない。evidence container/semantics metadataの影響が残る。prompt wording反復による合格へ置換せず、sourceとmetadataの要因分離を行う必要がある。new-file/rename/sparse history/memory/cacheと旧65unit拒否は未達を保持する。

## Next Steps

- 同じmetadataを保持しsourceも空にした対照を事前固定する。repository evidenceの増分か共通metadataの効果かを区別するため。
- evidence寄与が支持された場合に限り、未使用入力・#149回帰とcost条件でqualificationする。使用済み7caseの成功だけでB Goとしないため。
- CへはB条件が揃うまで進まない。
