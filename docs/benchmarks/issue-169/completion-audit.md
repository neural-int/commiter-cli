# 達成条件の照合

10条件を研究・条件分岐・停止判断の記録として照合した。Gate 1 NO-GOでPhase 2は未評価。チェックは後段実施・意味成功・production採用を示さない。

| # | 達成条件 | 証拠 / 判定 |
|---|---|---|
| 1 | 実schema/sample/projection/上限を生成前固定 | input-schema-A/B/C.json、frozen-inputs/、preregistration.json/md、2417ac1a・07f9c433。Cは2fieldsのみ |
| 2 | Before/After ASTと追加/削除/置換等の決定的検査 | context_test.goの追加/削除/置換・複数hunk/宣言・曖昧省略等の境界検査＋35宣言cap監査、source-mapping.jsonのcontext_extraction。metadata件数bugを生成前に修正 |
| 3 | 同一変更のpaired input/gold/model/prompt/Gate登録 | fixtures.json、frozen-inputs/manifest.json、preregistration.json。生成0回のIssue報告後に開始 |
| 4 | known/独立holdout分離、欠落/予算監査 | cohort-audit.json、input-audit.json、tokenization-preflight.jsonl。holdout4件は新規合成、別作者実履歴は含まない。S16全arm予算不達、raw diff欠落0 |
| 5 | Phase 1 coverage/exact/grounding/過不足とGate 1 | known/holdout-evaluation.json、phase1-summary.json。27生成保存、Gate 1 NO-GO |
| 6 | Gate 1 GO時のみPhase 2 | NO-GO条件分岐で直接Grouping/FM/FS/exact/fallback未評価、生成0。条件付き実行を守った |
| 7 | arm別tokens/latency/RSS/pressure/swap | 各measurements.jsonl、phase1-summary.json、integrity-audit.json。全観測あり、Bのswap基準超過を保存 |
| 8 | 絶対品質とC−B増分/費用を事前基準で判定 | phase1-summary.json/report.md。双方NO-GO、Cの採用根拠不足。未到達Grouping絶対品質は未評価 |
| 9 | 元Git状態、製品未接続、未到達Phase | integrity-audit.jsonでmain/6fixture保持、27/27receipt一致。製品src変更0、4file上限維持、Phase 2/Metadata/commit実行未評価。公開pathのみ匿名化をpublication-auditへ分離 |
| 10 | 有限結果のGO/NO-GOとNext Steps | report.md、Issue Iteration 3。固定候補終了、品質Gateへ進む。prompt/model調整・追加生成0 |

既存機能と既存テストの削除/変更はないため、不要テストの整理対象はない。研究テストはAST/hunk対応・曖昧省略・上限・不正ID/完全割当拒否・index/source変更検出・実helper予算契約を対象とし、意味成功の証明には用いない。
