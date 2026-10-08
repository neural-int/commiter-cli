## 要約

前後両側で隙間なく連続する編集をまとめるgold非依存proposalにより、実例285atomを3上位単位へ圧縮。現Cの8unit上限以内になった。ただしparent-onlyの既知境界は8/18で、葉は18/18を保持する。自動refinementの意味品質は未検証でありC GOではない。

## 検証結果

条件は61bc8d72で事前固定。変更spanの両側が接する場合のみ結合し、未変更gapは跨がない。全atomとline child IDを保持し、parentはsplittable proposal。gold/path意味/symbolラベルを入力に使用しない。

既存18件の葉境界・部分stagingは18/18成功。上位単位だけでは8/18、10件は分割が必要。前回のline parent-only10/18より粗い。評価gold oracleは構造検査専用でruntime不使用。

実履歴fd042894はextract.go146atom→1proposal、extract_test.go139atom→2proposal。合計3。parent全適用の再構築・forward/reverse staging成功、一意child coverageと同一入力再現を確認。全atom保持のため保存容量削減は主張しない。モデルcall0。

## 考察

粗い入力単位数をC cap内に抑える構造は得られたが、同一上位単位内の複数intentを自動判断する問題は残る。全parentを無条件に一つの目的とする方式は既知10件の境界を失うため不可。今回の3proposalが一つのintentかどうかも未認定である。

次はmodelが分割要否だけを判定し、hostがatom coverageと分割計画を検証する責務境界を定義する必要がある。分割結果が8unitを超える場合、既知goldに合わせた間引きや上限緩和ではなく拒否として保存する。局所圧縮をscaling/production品質と誤認しない。

## Next Steps

- 新規C専用worktreeで、上位proposalのaccept/refine/unresolved interfaceとatom ID coverage、重複、source mapping、予算超過拒否を事前固定する。階層表現を完全自動・fail-closedな判断へ接続するため。
- 評価goldとは独立したhost validationを先に検証する。oracleの正解分割をruntimeへ移植せず、malformed/partial/refinement超過でGitに進まないことを確認するため。
- その後に固定された未使用入力でrefinementの意味品質を計測し、既存候補再試行とは別集計する。責務変更による改善かを判断するため。B停止とD条件未達は維持する。
