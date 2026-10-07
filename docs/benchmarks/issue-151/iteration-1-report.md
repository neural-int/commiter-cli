## 要約

14個の#149回帰fixtureと10個のsynthetic境界ケース、合計24ケース/138file/140unitsで、変更内容の再構築とforward/reverse順のGit index byte一致を確認した。同一file内の離れた2変更は別stage可能で、file-only representationの構造的制約を解消する例を確認した。隣接変更は1unitとなり、AのGOはまだ判定しない。

## 検証結果

| 対象 | cases | files | units | reconstruction / Git staging |
|---|---:|---:|---:|---|
| #149 regression | 14 | 128 | 128 | 全pass |
| synthetic境界 | 10 | 10 | 12 | 全pass |
| 合計 | 24 | 138 | 140 | 全pass |

- 独立した2関数変更を同一fileで2unitsとして抽出、別々のindex中間状態を確認した。
- adjacent-edit / same-lineは1unitで保持。推測による分割や変更欠落はないが、goldの2intent境界を表現できない。
- #149既存fixtureは全fileが単一intentに属する評価定義のため、file baselineでもgoldを表現可能。細粒度化により#149のFM/FSが解消したという証拠は得ていない。
- AST annotationsはGo standard parserから抽出し、抽出境界やsemantic labelsの決定に用いていない。
- LLM calls/input tokens/output tokensは0。semantic prediction未実施のためexact/FM/FS/unresolvedはN/A。gold表現可能性をsemantic exactと混同しない。
- byte、line、unit上限でfail closed。4 regression testsで未知/重複unit、stale source、overlap、budget、binary NUL、Unicode/repeated linesを確認。
- extractionとGit stagingを含むwall/representation bytesはiteration-1-results.jsonに保存。純extraction latencyの分離測定は次iterationへ残す。

## 考察

離れたeditを別commitへ割り当てる能力は確認した。一方、line-diffのreplace blockは隣接する独立宣言を統合するため、partial staging可能な境界を十分に表現するには改善が必要。ASTをgoldなしのsource mappingとして活用する次の仮説を検証する。#149の既存multi-file ambiguityはrepresentationだけで解消したとは判断できない。

## Next Steps

- 旧新でline数が等しいreplace blockのline edit分割を固定し、隣接宣言の分割と再構築を検証する。goldを使わず、実際にstage可能なline境界を保存するため。
- insertion/deleteの複数line block、metadata操作、各budgetの境界と、純extraction latency/coverage指標を確認する。全changed bytesと安全なstage operationへの逆参照条件を確認するため。
- Aの全達成条件を証拠へ対応付けた後にGo/No-Goを判定する。BはGOまで開始しない。
