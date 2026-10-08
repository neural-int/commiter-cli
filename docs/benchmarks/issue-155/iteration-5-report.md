## 要約
Iteration 5ではモデルcall 0で、既存限定observerの確定factとChangeUnitのsource位置対応を監査した。6ケース・11queryの22状態中、既知15/unknown7。63atomの完全再構築とreturn witnessのsource位置を確認した。これはattribution GOではなく、LLMへ値を再計算させないhost-only対照の前提確認である。

## 検証結果
- 専用worktree/branch: codex/issue-155-host-fact-audit。前iterationの計測済み6ケースを使用し、fresh holdoutとして扱っていない。
- Fee/WrapとLeft/Rightは各2queryの値が変化。Count(3)は3→3、人工controlのCount(-1)は-1→0。
- 観測なしとcallee欠落はbefore/afterともunknown。公開Compareはbefore unknown、after -1/0/+1であり、beforeが不明なためdeltaはunknown。
- 元sourceのUTF-8 byte spanとreturn witnessのtextを照合。前iterationのobserver binary hashと一致。全63atomのforward再構築を既存extract/reconstructで確認。
- return witnessへの編集位置の交差はFee/Wrap4、Left/Right4、Count1、公開Compare3。Count(3)のreturnには編集位置の交差なし。これはsyntax intersectionであり因果関係の正解ラベルではない。
- semantic_attribution/commit_partitionはnull。モデルcall・任意repository実行・新依存・production変更は0。
- 既存host検証Python3 testsとGo限定observer testsがpass。

## 考察
具体入力の値・return位置はhostが確定でき、モデルの誤った再計算を採用する必要がない。ただしreturnとの交差なしをindependentへ変換することはできない。条件、引数、calleeの編集によって同じreturn文から別の結果が返る場合もある。値が同じでも観測外の入力で振る舞いが変わり得る。

この対照だけでは「同じ目的の変更を結合し、独立目的を分離する」#155の条件を満たさない。同じ観測の値変化もhard unionの根拠にはならない。次の評価には変更assertionの観測条件と具体入力が自然な実履歴から取得でき、その値だけで解けない残余判断があることを先に示す必要がある。既存使用済みfixtureへの規則追加は独立能力の証明にならない。

## Next Steps
- 未使用実履歴について、具体入力・変更assertion・source mappingが既存観測範囲で取得できるかモデルなしで監査する。人工入力probeを自然なattribution holdoutに置き換えないため。
- host-onlyで解ける局所観測対応と、必要な意味判断が残る対象を分け、positive/negative/unknownのgold根拠とgateを推論前に固定する。モデルの責務が単なる値再計算や旧global groupingになる候補は開始しない。
- 独立評価入力が成立するまで追加モデルcall・B/C・production変更を行わない。今回のsource対応を意味GOや旧#150達成として扱わない。
