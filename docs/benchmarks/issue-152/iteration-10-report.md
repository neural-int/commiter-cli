## 要約

repository sourceとcounterfactual観測を両方除いたmetadata-only対照もexact7/7・FM0・FS0だった。今回の7caseにおけるcontext-onlyの成功は、追加repository evidenceの増分として支持されない。B Goは未達を維持する。

## 検証結果

固定7callは全completed/accepted/complete、unresolvedなし。metadata-only exact7/7・FM0・FS0はcontext-onlyと同じ全caseのmembershipだった。旧A-onlyはexact4/7、旧impactは6/7。metadata-onlyは両evidence配列を空にし、同じcontainer/semantics metadataだけを保持した。条件はdd2a469で事前固定。6contract testとdiff check成功。gold/旧結果は変更していない。

## 考察

成功をrepository/test観測に帰属するために必要な対照差がなかった。情報を追加しないcontainer/metadataの差で出力が変わっており、旧A-onlyとの比較には入力形式/共通policy表現の交絡があった。この差をBの意味的追加signalへ読み替えず、metadata wordingをさらに調整して合格を狙わない。

今回の対照は使用済み7caseの診断であり、repository evidence全体の不可能性も、metadata-onlyの実repository品質も立証しない。Cに進む条件、new file/rename/sparse history/memory/cache、旧65unit拒否は未達として保持する。

## Next Steps

- 次のevidence比較では共通container/metadataをbaselineにも同一に持たせる。追加情報以外の差を比較から除くため。
- その固定baselineで未使用のsource/test対応と#149回帰を測る。今回の使用済み7caseに合わせたpolicy採用ではなく、repository情報の増分を検証するため。
- Bの独立追加価値とregression回避が立証されるまでCへ進まない。
