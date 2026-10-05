## 要約

Iteration 1のper-file Semantic IR→global membershipは、#146で整合したFM12が残った6ファイルfixtureで固定参照と一致した。production candidateの選定はまだ行わない。4-file回帰、大規模入力、holdout、順序安定性、metadata/Validate接続は未確認。

## 検証結果

| architecture | fixture | files | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | wall s |
| --- | --- | ---: | --- | ---: | ---: | --- | --- | ---: | ---: | ---: | ---: |
| per-file Semantic IR → global | contract-independent-6 | 6 | true | 0 | 0 | true | false | 7 | 3,065 | 4,226 | 240.865 |

全7callはcompleted。context overflow/timeout/incomplete0。抽出6callは202.623秒、global1callは38.240秒。raw JSONLへ各phaseの実tokens/wall/stopを保存。prompt/IR/思考本文/生成テキストは保存していない。初回はcommit98167adの計測コード（JSON重複キー検証追加前）で実行し、後続で24b166bに停止契約を強化した。初回結果を後続コードの成立証明にはしない。

#139〜#142/#143/#146を確認した。#146の同fixtureのauditedはFM12/FS0・9calls・248.593秒だったが、今回とはprompt/task/call構成が異なる。architecture全般の優位性や精度推定とはしない。抽出前後の合成Goプログラム6件はテスト成功。追加の不正割当/duplicate JSON/incomplete extraction停止test成功。

## 考察

局所groupを固定せず、個別の変更内容を全体判断に渡す方式で、この既使用fixtureの独立3目的を分離できた。IRが原因という単独因果は未証明で、global promptや入力構成も異なる。N+1callsで6fileでも約241秒を要し、現行120秒サイクルの採用予算を満たしたとは言えない。metadata未実行のためplan成功にも数えない。

## Next Steps

- 同じ固定条件で4file現行Stage1とIRを測る。baseline regressionを比較するため。
- cross-boundary12fileをIRで測る。複数directoryの単一契約を誤分割しないか、大規模global判断が成立するかを確認するため。
- 事前固定済みholdout-independent-6を測る。既使用3修正に限る成功かを独立入力で確認するため。
- 失敗原因とcostに応じて、最大4fileの抽出batchまたはraw factを保持するIRを比較する。prompt wordingだけの局所tuningへ戻らず、抽出call数と情報欠落の境界を検証するため。
