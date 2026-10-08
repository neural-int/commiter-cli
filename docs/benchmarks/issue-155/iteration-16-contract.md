# 観測効果taskの入力草案

6自然観測のsource/state/assertionと、状態を未指定にした3人工controlを別区分で保持する。モデル入力draftにgold、before参照結果、effect、fixture名、目的label、commit SHAは含めない。期待値は自然testのsource根拠であり評価effect goldではない。評価側draftは別fileに保存する。

出力案は全query IDを一度ずつ扱い、changed/unchanged/unknown、source evidence refs、unknown理由を返すもの。hostは構造・source位置・complete coverage・budgetを検証し、意味正答は別の参照と照合する。source ref validを意味validとしない。

3人工controlは同じ引数/sourceでもmutable mapの初期stateが未指定で、constructed counterexampleによりeffectを一意に定められない条件である。自然holdout件数へ加算しない。

この草案はChangeUnitの全attribution、同一目的の統合、独立目的の分離、cross-boundaryを未評価。source reasoningの参照とbounded手書きモデルの一致は独立Go runtime観測ではない。caseの一部を観測効果taskへ限定することを#155全体のGOや旧#150のGoal達成に置換しない。

prompt/schema、source evidence ID契約、baseline、token/time/call gate、native grammarは未固定。これらを完成・事前登録するまでモデル計測しない。既定model/helper、production4file上限、B/C依存は維持する。

## 採否更新

このeffect-only草案は#155受入実験には採用しない。解析拡張を凍結し、中心仮説の条件不足をiteration-16-decision.mdへ記録した。草案と参照は準備証拠としてのみ保存する。
