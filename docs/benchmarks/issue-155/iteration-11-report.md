## 要約
Iteration 11で未対応7行のcontrol根拠をsource位置付きで取得し、次のattribution評価契約を作成した。ParseBytesはerr/continue後の値比較、ignoreはcolumn guardによる追加引数とdiagnostic branchの区別が必要。binding11/18は変わらず、意味能力と独立holdoutは未証明。モデルcall0。

## 検証結果
- 新専用worktree/branch: codex/issue-155-control-contract。
- ParseBytes4行の根拠はerr != nil、continue、got != p.exp。errを返すcallのmulti-result対応と比較への到達経路は未検証。
- ToSnakeWithIgnore3行ではlen(i)==3でignoreを設定し、result != outで失敗を判定する。failure表示のistrにも別のlen(i)==3分岐がある。before/after引数型の差を省略しない。
- 標準ASTからassignment/branch/control conditionを取得し、全7行の各source span/textを照合。source近接をbindingや効果の確証にはしていない。
- 次の契約ではverified source binding、未検証control、実測値、LLM意味仮説を別に保持。source refの妥当性を意味正答と呼ばない。
- 独立negative/cross-boundary positiveなどのデータ・gold・受入閾値・推論budgetは未固定で、契約は準備段階。モデル計測事前登録と偽らない。
- Go build/vet、既存source位置/偽comment検査test1件pass。モデルcall0、任意repository実行0、新依存0、production変更0。

## 考察
値計算候補の失敗後に、その計算をLLMへ戻す必要はない。確定sourceのassertionを入力にし、実装効果の仮説を別の評価対象として扱う方が#155の責務に合う。ただし今回取得したのはcontrol syntaxだけで、path/argument bindingと因果効果は未証明である。

新契約は独立能力条件を省略せず、現在のsource監査データへの過学習をGOへ変換しない。残るsource bindingの閉包と独立意味評価の両方が必要で、どちらか片方だけで#155完了とはしない。

## Next Steps
- 7行のsource controlを限定的な規則で検証できるか評価する。error guard、column guard、値に影響しないdiagnostic分岐を区別し、未解決はunknownに保持する。
- 未使用のpositive/negative/unknown、cross-boundaryを含む評価入力を確保し、変更目的と部分状態の根拠を記録する。既存6履歴をfresh holdoutにしない。
- データ・gold・budget・baseline・wire監査が成立した場合のみ一つの候補を事前登録する。source近接からhard union、B/C、production、旧#150達成へ進めない。
