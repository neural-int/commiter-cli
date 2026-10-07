# B / Iteration 1 事前登録

A gate: #151 Iteration2 GO、line/source mapping/staging検証成功。B専用新規worktree: issue-152-semantic-evidence、branch codex/issue-152-semantic-evidence。

仮説: 変更されていないadapter/functionを含むrepository source evidenceは、選択ChangeUnitsのみよりcross-directory関係の可観測性を改善し得る。Git historical co-changeは別signalであり、shared helperや偶然のco-changeをhard mergeには用いない。

固定比較: A-only（ChangeUnits + selected file before/after + AST symbol annotations）、A+repository（追加unchanged source、syntax-derived calls）、A+history（bounded co-changeのみ）。同一semantic model/task/schemaとunit ID、sampling、予算を使用し、sourceの効果を分離する。final optimizerはB内で導入しない。

qualification: #149 same-directory-independent-5 / cross-directory-single-intent-9 / multiple-intents-boundary-12 / shared-callee-independent-6。independent: fixtures.pyのfresh152-unchanged-adapters-6 / fresh152-common-helper-independent-6を結果閲覧前に固定。各条件各1回、全18calls、再試行/repair/wording調整0。goldは評価側だけ、model payloadに含めない。既使用#149 fixtureをfreshとして扱わない。

local model: 既存cached Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192と既存計測helperを使用、downloadなし。context16K/output1536/native0/temperature0/top_p1/top_k0/seed144、call120秒。prompt/output/thoughtはRAMのみ、保存はID grouping/numeric telemetry/stop分類だけ。

historyはsynthetic temporary Git repositoryの実commitから取得。既存fixturesにはindex順の決定的な混合co-changeを作り、freshには明示したmixed schedulesを固定。現在goldからhistoryを生成しない。これはproduction history分布でなく、偶然co-changeへのrobustness検査。最大64commits/64files/1MiB context、absenceはunknown。rename mappingは過去名のalias、new file/no-historyはabsenceを保持。partial typing/overload/unknown moduleは解決を推測しない。

GO: 少なくとも1sourceが新independentでA-only比の測定可能な改善（exact増加またはFM/FSの減少）を示し、他fixtureのFM増加や総誤り増加がそれを上回らない。全cases complete/fail-closed、bounded cost、gold leakなし。改善なしならNo-GoとしてCを開始しない。single runの結果を一般populationの精度/因果証明としない。
