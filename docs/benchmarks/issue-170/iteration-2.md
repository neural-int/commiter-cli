## 要約
Iteration 2: **Gate 1 NO-GO**。BはAよりpurpose過剰分割を8→5に減らしたが、具体的coverageは11/12→10/12に低下し、目的リストexactは4/8のまま。Cは過剰分割8、exact3/8、根拠のない説明2件で実質改善を示さなかった。Phase 2の直接Groupingは未評価とし、製品採用なしで有限研究を終了する。

## 検証結果

| 新規holdoutの指標 | A | B | C |
|---|---:|---:|---:|
| 具体的coverage | 11/12 (91.7%) | 10/12 (83.3%) | 11/12 (91.7%) |
| 目的リストexact | 4/8 (50.0%) | 4/8 (50.0%) | 3/8 (37.5%) |
| purpose過剰統合 | 0 | 0 | 0 |
| purpose過剰分割 | 8 | 5 | 8 |
| grounding | 20/20 | 17/17 | 18/20 |
| 根拠のない説明 | 0 | 0 | 2 |
| 順序で品質/粒度が変わるfixture | 2/4 | 3/4 | 3/4 |
| schema / 小入力admission | 8/8 / 8/8 | 8/8 / 8/8 | 8/8 / 8/8 |
| input / output tokens合計 | 7580 / 5046 | 7652 / 4773 | 7316 / 4876 |
| wall p50 / p95 秒 | 50.12 / 53.95 | 46.35 / 53.02 | 48.46 / 61.60 |
| helper peak RSS GiB | 3.227 | 3.369 | 3.180 |
| swap増分最大 MiB | 151.62 | 81.75 | 350.44 |
| pressure最大 | 1 | 1 | 1 |

- 新規holdoutは自作合成4fixture・2順序の8paired cell。coverage分母12は計6goldを両順序で照合した観測であり、独立12目的標本ではない。
- N02 normalのBはsender/receiverのwire-keyを一つに統合したが、reverseでは3目的に分割。N03のBは両順で独立slug/visibilityの2目的を保った。N01は各armで粒度または具体的説明が順序に依存した。
- Cの根拠のない説明はN01 normalのn>nという比較、N04 reverseのbatch単位pages。曖昧な値のplaceholderだけの説明はtarget-onlyとした。Cのこの2評定を仮に許容してもexactは最大4/8・過剰分割8でGate未達であり、正式score/rubricは変えていない。
- known回帰12callは採用根拠と分離。coverageは各arm6/7、exact A1/4・B2/4・C2/4、過剰分割A3・B2・C3。
- 全36生成はcompleted/schema valid、preflightとの実input token一致100%、fixture snapshot/index/worktree非変更100%。試行喪失0、retry/repair0、追加model0、Grouping0。全frozen source/runtime/request pinsが一致。
- 16GiB MacのRSS/pressure/swapを全条件で記録。C N02 normalのswap増分350.44MiBは事前登録の256MiB基準を超過。pressureは全観測1。ホストのswapは開始前から約5.5GiBを使用していた。
- S16はA47146/B47155/C47113 tokensで全条件context超過。生成しなかった。16file約80%の達成を主張しない。
- mainのHEAD/index/既存.gitignore変更を保持。変更範囲は#169最終headから研究用tools/benchmark170とdocs/benchmarks/issue-170のみ。#169は現在もClosed/completedであり変更していない。
- [最終研究報告](https://github.com/neural-int/commiter-cli/blob/codex/issue-170-prompts/docs/benchmarks/issue-170/report.md)、[固定Gate集計](https://github.com/neural-int/commiter-cli/blob/codex/issue-170-prompts/docs/benchmarks/issue-170/phase1-summary.json)、[全文照合評価](https://github.com/neural-int/commiter-cli/blob/codex/issue-170-prompts/docs/benchmarks/issue-170/holdout-evaluation.json)、[資源/Git監査](https://github.com/neural-int/commiter-cli/blob/codex/issue-170-prompts/docs/benchmarks/issue-170/integrity-audit.json)。

## 考察
Bには同目的の統合を改善する局所観測があるが、coverageと順序安定性を改善せず、absolute exact>=80%・過剰分割0等の基準を満たさない。Cも実質改善を示さず、資源基準にも不達だった。したがって直接Groupingを追加する根拠はない。purpose過剰統合0はfile partitionのFM0を意味せず、GroupingのFM/FS・complete assignment・fallback/singleton逃げは未評価。

差はPrompt全体のpaired比較であり、各文や否定文1行の単独因果効果ではない。評価は同じagentによるoffline input/gold照合で、独立人手評定ではない。新規holdoutは今回の初回生成という意味で、別作者/実repositoryの無作為標本ではない。N01の共通page-size契約も設計者が事前固定したgoldであり、直接caller参照は含まれない。適用範囲をこの合成goldに限定する。

wall/RSS/swap/pressureはこのホストの順次観測であり、短いpeakや他appの因果を分離しない。小標本のp95や一時的なpressure1だけで、Mac16GiBの長時間共存を立証しない。

## Next Steps
- Phase 2と製品採用を行わず有限研究を終了する。事前登録した絶対品質・実質改善に到達しなかったため。
- 研究成果物と評価の全履歴をcommit/pushし、Issueの完了条件8項目を証拠付きで確認する。NO-GOという研究結果と未完了を混同しないため。
- 完了条件とチェックボックスを確認した後だけ、goal.mdの最終test/vet/build等を実行する。全pass後にIssueをcompletedとして閉じるため。
- 再研究は別Issueで実repository/別作者holdoutと独立評定を先に定義する。本Issueの局所観測を一般化しないため。今回は実施しない。
