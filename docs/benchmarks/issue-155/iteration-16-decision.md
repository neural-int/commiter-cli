## 要約
Go固有の解析拡張を凍結する。Iteration 3のattribution診断4件、Iteration 4の局所fact診断6件は全件host不合格であり、以降のsource binding改善から意味能力の改善は立証されていない。現候補はNO-GO（限定能力GO未立証）。新規6自然観測のeffect-only草案を#155の受入実験として採用しない。

## 検証結果
- 専用worktree/branch: codex/issue-155-reference-contract。凍結基準commit: 865692bf5ce9b5e5ba78088e6286bfb6036449b4。
- 既存bindingは18/18行、新規履歴では2/3履歴から6行を取得。並行実行testの拒否をcoverageとして保持する。これらはsource対応の件数で、semantic正答数ではない。
- Iteration 3: attribution 4/4 backend完了、host合格0/4。Iteration 4: 局所fact 6/6 backend完了、host合格0/6。棄却結果の意味品質はnull。
- Iteration 16の追加モデルcallは0。手書きASCII参照モデルはCamelの4参照と一致し、Snake2件はsource経路の参照に留まる。3人工state controlは自然holdoutに加算しない。独立Go runtime観測ではない。
- effect input/goldは草案で、prompt/schema/budgetを固定した正式計測ではない。同一目的positive、独立目的negative、cross-boundary、全ChangeUnit attributionの評価条件を満たさない。

## 考察
特定入力で戻り値が変わらない回帰テストでも、実装修正と同じ目的を支えることがある。unchangedをindependentへ変換すると別の問題を測る。したがって6観測のchanged/unchanged/unknown測定だけではAの中心仮説を判定できない。

sourceの対応付けを改善した事実は保存するが、意味判断の改善を示さずにGo構文対応を追加し続ける合理性は現時点でない。今回のNO-GOは現候補・準備案の採用判断であり、Behavioral Attribution全方式の不可能性を証明したものではない。

## Next Steps
- Go抽出器・値計算・参照モデルの拡張を停止する。意味能力を実測する前の追加開発を避けるため。
- 再計測する場合は既存抽出器だけを使い、未使用実履歴のpositive/negative/unknown、shared-test、cross-boundaryを含む小規模セットの識別可能性とgold根拠を先に確定する。抽出不能・gold曖昧は除外せずcoverageへ記録する。
- 中心仮説の入力/出力契約、モデル/helper、予算、受入条件を一度だけ事前固定し、同じ観測入力のbaselineと限定比較する。効果分類のみへの縮小、結果後のprompt調整、追加解析での追試を行わない。必要なケースを既存機能で準備できなければ、そのcoverage不足をNO-GOとして終了する。
- Aの限定GOが出るまでB/Cを開始しない。依存変更や新仮説は別の判断として扱う。productionの4ファイル上限、旧#150未達を保持する。
