## 要約

assertion位置は旧共有test反例を区別できたが、新規table-driven反例では独立intentを混同した。位置一致によるdefault mergeは採用しない。Bは未達。

## 検証結果

2fixture×baseline/4unit単独/allの12probeを実行。baseline/allは4状態とも成功。旧fixtureの単独4unitはboth_test.goの4行目/5行目へ2unitずつ分かれた。新規table-driven fixtureの単独4unitは全て9行目で失敗した。各fixtureのLeft/Right intent単独状態4件はGo test成功。位置一致による全pairの評価では旧fixtureは同intent2pairのみ、新規fixtureは同intent2pairと異intent4pairとなる。モデル呼出0。測定前固定commitは60f9ffb。

## 考察

同じassertionが複数のdata rowを評価する場合、test名とsource位置では実行対象を区別できない。細粒度の位置情報を追加しても、その一致をsemantic intentの同値関係にはできない。生成Go syntheticのみの観測であり、production採用やBの独立品質改善を立証しない。

## Next Steps

- 複数unitの共同適用でtestが回復するかを、boundedな別仮説として事前固定する。単独impactの一致より強い観測を取得するため。
- 非behavior/absenceと共有依存guardを保持し、test成功だけでsemantic intentを確定しない。異なるintentでも同時適用が成功し得るため。
- costと独立A-only比較を確認し、Bの未達条件が揃うまでCへ進めない。
