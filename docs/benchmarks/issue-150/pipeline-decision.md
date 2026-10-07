## 要約

Issue #150 pipelineは **A: 表現能力の限定GO・親の既知failure改善未立証 → B: 補助評価NO-GO → C未着手**。production candidateなし、D必要性未立証。親Issue/Goal全達成には到達していない。Aの親条件が未立証であり、さらにBの補助評価も独立改善がないという定義済み停止条件に該当するため、現在の研究pipelineを停止する。

## 検証結果

| stage | 専用新規worktree / branch | 結果 | 証拠 |
|---|---|---|---|
| A / #151 | issue-151-change-units / codex/issue-151-change-units | 子#151の表現能力GO・親#150条件未立証 | 24cases、138files、141unitsのstage/reconstruction成功 |
| B / #152 | issue-152-semantic-evidence / codex/issue-152-semantic-evidence | NO-GO（補助評価・固定evidence設計） | 6cases × 3source = 18calls、independent2caseで改善なし |
| C / #153 | 未作成 | 未着手 | B Go gate未達のため |
| D | 未作成 | 必要性未立証 | Cを経たscorer-only bottleneckの証明なし |

Aは同一fileを複数line editsへ分けてstage可能で、file-only representationの構造的制約を解消する例を確認した。24casesのうち#149回帰14とsynthetic境界10。source snapshot/range/byteの保存、forward/reverse stage、Go AST annotationsとbudgetを検証。same-lineはatomic、metadataはwhole-operation契約。実CLI integrationとsemantic品質はAのGOから結論しない。

Bの集計は以下。exact/FM/FSはunit pairの比較であり、production精度分布を推定しない。

| source | 全6case exact | FM | FS | complete | fresh2case exact | fresh FM | fresh FS |
|---|---:|---:|---:|---:|---:|---:|---:|
| A-only | 2/6 | 6 | 72 | 6/6 | 1/2 | 0 | 3 |
| + repository | 1/6 | 15 | 51 | 6/6 | 0/2 | 12 | 3 |
| + history | 2/6 | 6 | 29 | 6/6 | 0/2 | 0 | 6 |

全18行unresolved0、backend completed、host accepted。input token計49,243、output token計2,004、model call wall計543.219秒。modelは既存cached Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192、helper pin/current cached manifest/file sizes確認、追加downloadなし。temperature0/top_p1/top_k0/seed144/native0、context16K/output1536/call120秒、各condition1回、repair/retry0。

初回schema未伝達の6失敗行、1中断attemptと最小診断1callは保存し、修復後18行のaggregateへ混在しない。研究全体の開始attemptは26、中断費用はunavailable。messagesへのschema追加は既存benchmark149.invokeの契約に合わせた修復で、semantic結果を改善するwording tuningではない。

Aの最終quality gatesはGo test -p1/vet/build、Python7tests、diff check成功。最初の並列Go testで既存MLX起動待ち2件失敗、単独とpackage直列で成功し、閾値/skip/production codeを変更していない。BはPython3contract tests、go test/vet ./tools/benchmark152/...、diff check成功。B達成条件が未達のため、Goalの最終quality gate開始条件を満たしていない。

## 考察

Bのqualification集計でFS改善があっても、freshでは改善を再現できず別regressionが発生した。unchanged common helperの追加で12pairの誤統合を観測した。call/co-changeの構造的関連はshared semantic purposeを保証しない。モデルの内的理由やrepository/history一般の無効性は断定しない。

AのGOは子#151のrepresentation能力に限定する。親#150の「既知#149 boundaryをfile-levelより適切に表現する」という条件は未立証。既存14caseではfile-onlyでもgoldを表現でき、128files全て1unitだった。B開始判断でこの親の追加条件の照合が不十分だったことを記録し、Bを親pipelineの正式Go通過後評価とは扱わず補助観測として保存する。元の測定/gold/子#151の構造的能力は維持し、parent-gate-audit.mdに判定の境界を明記した。Bのsource-gateを満たさずCへ進むと、未選定evidenceを前提にoptimizerを評価することになり依存契約に反する。DはCの結果からdedicated scorerの必要性が分離証明される場合に限るため、今回は切り出さない。

親#150はA/B/C全段階、current production baseline comparison、C最終fresh holdout、最終partition品質の条件が未達。new-file/rename/sparse-history/実規模memory/cacheのB未検証項目も残る。これらを条件未発動として全達成へ書き換えたり、production candidateなしだけでGoalをcompleteにしたりしない。

## Next Steps

- 現在の固定方式はA親条件未立証/B補助評価No-Goで停止する。追加budget/fixture-specific routing/wording/gold調整でgateを通さないため。
- 再開には、既知#149でfile granularityがboundary表現を妨げた具体的case、または親A gateを見直す明示的な判断が必要。A親条件が整理された後、boundedな新evidence仮説、既存observabilityに対して追加される情報/責務の根拠、未使用independent評価を必要とする。現方式の同じ入力/同じモデル反復だけを改善根拠にしないため。
- parent/child Issue本文に報告の目次と停止理由を反映し、commit/pushとremote headを確認する。local測定・remote配送・Goal達成を区別するため。
- production4file上限/default/Git mutation契約の変更やmergeは行わない。今回の研究gateからproduction採用の根拠は得られていないため。

## Aチェックリストの範囲監査による補足

[子#151の範囲監査](https://github.com/neural-int/commiter-cli/issues/151#issuecomment-6041328409)により、partial staging全範囲と既知failure boundary改善の2条件を未達へ訂正した。一部の構造的能力は確認済みだが子Issue全体は未達で、親A gate通過も未確定。過去の測定/gold/達成条件は維持する。人間へ、親gate対象の既知fixture名またはIssueコメントの入力を確認中。現在のfile-ID goldから未記録のfile内境界を作り替えたり、同一条件の推論反復でgateを通過させたりしない。追加model processは稼働しておらず、C/Dを開始しない。
