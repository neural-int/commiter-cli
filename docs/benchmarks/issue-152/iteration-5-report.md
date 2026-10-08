## 要約

未使用の共有test反例で、同じfailed test集合だけではsemantic intentを識別できないことを確認した。TestBothを失敗させる4unitには独立した2intentが含まれる。単純impact一致によるmergeは採用しない。Bは未達。

## 検証結果

baseline/allはtest成功、単独4unitは全て同じTestBothが失敗した。一方、要求由来のLeft変更＋対応assertion、Right変更＋対応assertionの各single-intent状態はそれぞれGo test成功。4unitの全6pairをimpact一致でrelatedとすると、gold同intentは2pair、異intentは4pairとなる。モデル呼出0。事前条件と入力はba99f56で固定した。

## 考察

共有testは複数の独立した変更を同時に検査するため、test名単位のimpactは過剰に粗い。前pilotの15組が整合したことを、一般的なintent判定へ拡張できない。この反例はtest-impact evidence全体の不可能性ではなく、同一失敗test名による同値関係の限界を示す。

## Next Steps

- assertion位置/実行経路まで観測するbounded evidenceを別仮説として固定する。test名だけでは独立intentが混同されるため。
- 非behavior変更とabsenceも未使用入力で確認し、未知をdefault mergeしない。impactを得られない変更も保持するため。
- 独立改善・cost・B残条件の評価を続け、Cへは進めない。
