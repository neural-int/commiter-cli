## 要約
Iteration 6の自然実履歴coverage監査では、dustin/go-humanizeの3commitから新規の自然test入力9個を取得したが、既存host observerの18状態は全unknown。attribution入力も2/3でunit_budget拒否。独立能力評価を開始できる入力契約は成立していない。モデルcall 0、意味品質とholdout GOは未判定。

## 検証結果
- 専用worktree/branch: codex/issue-155-natural-observation-audit。
- bytes.go履歴をAPI順に6件取得し、bytes_test.goも変更する先頭3件をsource確認前に選択した。成功による再選択なし。selection artifactに全6SHAと規則を保存。
- 2ccb40ff40263df6ede3d42093de3b7b0eaeb4ae: 丸め誤差の自然入力2件、known0/4状態、attributionはunit_budget拒否。
- 71f653b2f435ca3898cffe7097697aa59227a87e: ParseBytes整数精度のtable入力4件、known0/8状態、37atom、supported failure predicate1。loopのtable入力と直接callを区別して保存。
- 5ce95c58a9e11ad9d59f28d1d489f34989b473ec: 表示精度の自然入力3件、known0/6状態、attributionはunit_budget拒否。
- after test内のliteral rowのUTF-8 span/textを照合し、beforeには同一call/expected rowがないことを確認。期待値は自然testに書かれた値であり、author intent goldではない。
- 実sourceにはuint64、複数戻り値、fmt.Sprintf、strconvなどが含まれる。既存限定observerの対応範囲外で、これらを実行する拡張はしていない。unit制限も維持。
- 既存docs/benchmarksとbenchmark149/153/155のsourceでrepository名と選択SHAを検索しhitなし。ただし全過去使用を網羅した未使用証明ではない。正式holdout認定なし。
- snapshots、source hashes、query/期待値/unknown、入力拒否を保存。semantic_gold/attribution/partitionはnull。モデルcall、任意source実行、新依存、production変更は0。
- 既存Python host検証3testsがpass。監査は最初にunit_budgetで停止し、その拒否を記録するよう処理を修正した。評価用上限を緩和していない。

## 考察
人工的なscalar例の成功だけでは自然履歴へのcoverageを示せない。今回の3件ではhost確定factがないため、値の検証を前提とするattribution候補を測っても受入根拠にならない。unknownを成功扱いすることも、モデルに未対応計算を代行させてauthoritative factへ昇格させることもできない。

3件は同一repositoryの主にpositive bug/feature履歴であり、negative独立目的やcross-directory統合も未準備。これは全実repositoryで不可能という証明ではなく、この固定標本と現observerのcoverage不足である。fixture固有にfmt/uint64/loopを追加するだけでは、変更目的を識別する新アーキテクチャの証明にはならない。

## Next Steps
- 別repositoryの固定小標本をモデルなしで監査し、今回だけのsource特性か、対応範囲の一般的不足か切り分ける。成功結果による選択や同じrepositoryの容易な例への置換は行わない。
- 自然入力と変更assertionを含むpositive/negative/unknownが確保できるか先に確認する。observerが解ける値だけを選ぶ正式gateは採用しない。
- coverage不足が継続する場合は必要な観測能力と実装費用を明記し、現A候補の継続可否を判断する。formal GO、B/C、production、旧#150完了へ進めない。
