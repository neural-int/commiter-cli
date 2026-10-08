## 要約

Counterfactual test-impact方法のpilotを実行できた。使用済み5fixtureで、goldなしにsource/testの単独変更が失敗させるtest名を観測した。B Goは未判定。

## 検証結果

5fixture × baseline / 6unit単独 / all の40probeを実行。baseline/allの10probeはtest失敗なし、単独30probeは各1test失敗。compile errorによるunknownは0。各fixtureで同じtestを失敗させるsource側unit/test側unitの組が3組観測された。モデル呼出0。結果JSONを保存した。

## 考察

変更のimpactをgoldから生成せず実行で取得できた。ただし使用済みsyntheticであり独立改善や同一intentの証明ではない。共有依存・非behavior・absenceでは同じimpactが誤った関連を示す可能性が残る。

## Next Steps

- 未使用fixtureを固定してA-onlyとの比較を実施する。pilotへの適合だけで採用しないため。
- shared dependency / 非behavior / absenceを含め、compile failureやbudget超過をunknownとして保持する。unsafeなdefault mergeを避けるため。
- Bの残る独立改善・cost条件が揃うまでCへ進まない。
