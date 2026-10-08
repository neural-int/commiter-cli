## 要約
Iteration 14では既存抽出処理のsource hashを固定して未計測の3実履歴に適用し、2履歴の新規6自然table行を取得した。並行実行テスト1履歴は現parser/input budgetで拒否し選別除外しない。同calleeでも具体入力に対する変更がno-opになる限定根拠を1件保存した。formal independent attribution GOは未証明、モデルcall0。

## 検証結果
- 新専用worktree/branch: codex/issue-155-independent-input-audit。base22b57613。camel.go履歴6件のAPI順からGo testを変更する先頭3件をsource確認前に選択し、4抽出器のsource hashを保存。抽出規則変更なし。
- 531aaa44de12ec166ceb71d9bdad7c8295e4235f: acronyms/camel/concurrency test。新規test beforeなしで4parserがexit2拒否、attributionはunit_budget拒否。並行実行testは明示的なexpected比較もなく、現方式のsource根拠を取得できない。
- d7993fd391266682cdd3408618e54058f8044f5c: camel test追加2行を固定loop抽出器で取得、attribution入力受理。
- 692d1b89fed27aadf9d3ea07717014dffe558ede: camel/snake test追加4行を同じ抽出器で取得、attribution入力受理。
- 追加ファイルのbefore内容取得は最初404。commit metadataでaddedを確認し、存在しないbeforeを空sourceとして保存。parserの拒否は保持し、取得失敗や意味能力失敗を混同しない。
- 新規6rowのsource ref/span/textを検証。既存比較とのrow signature差で候補を抽出し、すべての変更assertionを網羅したとは主張しない。未変更calleeのsourceがsnapshotにない場合の実装効果は未検証。
- 692dのcamel.goはstandard strings.TrimSpace代入の1行追加だけで、他のsourceが完全一致。自然ToLowerCamelテストの入力some stringは前後に空白がなく、この追加代入は同じ入力を保持する。その入力に限ったunchanged根拠を保存。actual戻り値と作者intentの独立goldは未判定。
- selected SHAの既存docs/benchmark149/153/155検索はhitなし。ただし全過去利用の網羅証明でもformal holdout認定でもない。抽出器を変えて成功へ合わせない。
- 固定4抽出器の回帰Python9tests、diff check pass。モデルcall0、任意repository実行0、新依存0、production変更0。

## 考察
source対応は以前の18行だけに閉じておらず、未計測の自然camel履歴でも6行取得できた。一方、新規ファイル/並行実行でfail-closedになり、coverageの限界も残る。取得2/3は意味精度ではない。

same-calleeの自然テストにno-op inputがあるため、caller/calleeや同commitの一致を全positiveへ変える規則は不適切である。ただし「ある入力に影響しない」と「独立変更目的」は異なる。今回の根拠を後者のnegative goldとして流用しない。

正式能力計測には独立目的negativeとcross-boundary positive、unknown、範囲/費用の事前gateが未準備。新規6行のsource取得から#155 GOやproduction候補を推定できない。prompt/schemaが推論前に固定されていない段階なので、今回をformal holdout passとしない。

## Next Steps
- 取得6rowに対して必要な実装source closureと識別可能な効果goldを監査する。whole-intent goldと具体観測の効果goldを分けるため。
- 明示的比較がない並行実行testはunknown/coverage guardとして保持し、容易なtable例へ置換しない。beforeなしのparser拒否は別の構造条件として記録。
- 同callee no-opを診断対照にし、独立目的negative/cross-boundary positiveを含む未使用評価を確保する。goldのない範囲は未判定のまま。
- データ/根拠/gate/prompt/schema/budget/baseline/native wireが固定できた場合のみ能力計測へ進む。B/C・production・旧#150完了条件は未達。
