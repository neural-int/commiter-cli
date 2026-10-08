## 要約

未使用2件のguardrailで、最小包含statement内のcall一致だけではsemantic relationを区別できないと確認した。同statement診断文はFP3、変数経由assertionはFN1。B No-Goを維持し、lexical一致だけでのmodel比較・採用には進まない。

## 検証結果

7512ad1で入力/gold/判定を事前固定した。same-statement-diagnosticはt.Log(Value(1), label)のlabel変更で、最小statementがValueを含むため独立3atomにFP3。variable-mediated-assertionはgot := Value(1)の次statementで期待値を変更し、変更statement内にValueがないためFN1。既存関数relationは存在し、lexical観測で落ちた関係を欠落として保存した。

両case全4状態のGo testを実行。独立case全4成功、連動caseはbefore/after成功で片方だけの2状態失敗。失敗をrawへ保存し再試行なし。gold/旧結果/tool変更なし、model call0。既存Python6testとdiff check成功。

## 考察

statementは複数のcall引数を一括で含み、変更したdiagnosticラベルとValueの計算を区別しない。逆に変数による値の受け渡しはstatement境界を越える。前回の局所位置による診断文区別は、両者が別statementだったことに限定される。lexical scopeだけを一般化semantic signalやhard merge条件として採用できない。

## Next Steps

- 新規専用worktreeで、ASTの変更式と定義/useの依存を最小観測する。今回のFPは兄弟引数、FNは局所変数の値伝播に由来するため。
- まず固定反例を診断し、局所変数・比較式・call引数という明示的なサポート範囲とunsupported/unknownを保存する。完全なtype checkerや新依存は追加せず、boundedな観測に限定する。
- データ依存観測で識別可能な範囲を確認後、未使用fixtureと旧回帰を含む共通container比較を事前固定する。構造観測をB GoやC完了と読み替えない。
