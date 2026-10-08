## 要約
Iteration 10では残る13自然table行のうち6行で、入力列→call→result→failure predicateと期待値列のsource bindingを検証した。前Iteration 9の5行と合わせ11/18行。残る7行はerror branchや追加引数の条件付き処理で未対応。モデルcall0、実装意味attribution/独立能力GOは未証明。

## 検証結果
- 新専用worktree/branch: codex/issue-155-loop-binding。
- 6fab6530のtoSnakeに追加された6rowは、入力row[0]、期待値row[1]、result=ToSnake(input)、result != expectedが同一loopの単純な定義列であることをASTで確認。全根拠のUTF-8 span/textを照合。
- 受入範囲は標準testingのT/TB param、[][]string literal、1range、3個の新規単一変数定義、単一引数のplain call、1failure if/Errorf。変数再定義、追加control、未知receiver等は受け入れない。
- ParseBytesの4row、ToSnakeWithIgnoreの計3rowはunsupported_loop_shape。未対応を独立、positive、GOへ変換していない。
- 前Iteration 9の5行と今回6行を足した11/18はsource bindingのcoverageのみ。値、実装編集の効果、同一intentは別の未判定条件。
- source再実行の結果が保存JSONと一致。Python2tests pass、Go build/vet、diff check pass。Unicode位置・未定義calleeを実行しないこと・再代入/追加分岐/偽testing拒否を確認。
- モデルcall0、任意repository source実行0、依存追加0、production変更0。対象は使用済み監査sourceで正式fresh holdoutではない。

## 考察
自然tableのassertion sourceは限定的なAST検証で取得できる。型付きtableのvalidatorと単純loopに別の受入範囲があり、値計算coverage29unknownとは別の指標である。11行の根拠取得はモデル入力の事実を明確にするが、実装効果の正答や独立目的の分離を証明していない。

残る7行を認めるにはerror/continueやignore引数の条件を含むcontrolの対応を確認する必要がある。単に同じ関数の条件とtable行を結合する処理や、callee一致を同一intentへ変換する処理は使用しない。

## Next Steps
- 残る7行のcontrolと引数対応を監査し、限定source規則で検証可能な範囲とunknown境界を固定する。直線loopの規則を条件付き処理へ無条件に適用しない。
- 検証済みassertionを観測根拠にする入力契約と、未検証implementation効果を答えるtaskの出力契約を分離して事前登録する。過去の値計算/global assignment候補の再試行にはしない。
- 独立positive/negative/unknown、cross-boundaryの評価入力とgold根拠を確保してから能力計測する。source coverage11/18で#155 GO、B/C開始、旧#150達成とはしない。
