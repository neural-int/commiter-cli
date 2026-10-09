# #156 検証完了条件の照合

**設計・検証の結論はproduction NO-GO、80%未達。** この完了監査は数値目標達成という意味ではない。対象Issueの完了条件は未達時のNO-GOと残余課題保存を明示しており、その条件で検証trackを完了する。#145の4file超拡張は解決扱いにしない。

|Issue / 成果物|根拠|結論・残余|
|---|---|---|
|#157 fallback契約・分類・metadata|planner.py、coarse.py、architecture-decision.md、iteration1/5|unknown継続とmixed未確定を明示。A2構造限定GO、意味品質GOではない|
|#157 CU→file lossless map/staging|source spans/digest、coverage、replay、24/24初回、6/6 A2再診断、15/15独立D|byte欠落/重複なし。16file成立。unsupported/mode/path/予算停止も保存|
|#157 naive singleton比較|iteration1 known18、file exact0/18 FM211/FS20、CU exact8/18 FM0/FS216|既知診断を新holdoutへ再分類しない。file-onlyのFM/FS限界を記録|
|#157 compose互換性・予算・GO/NO-GO|iteration1 invalid_type、iteration8 invalid_schema body、固定source/model/入力hash|compose採用NO-GO。ユーザー確認・新依存・production変更なし|
|#158 ranking/final grouping責務、evidence/unknown schema|selective.py、iteration2/8 prereg|scoreはordinal、soft候補、全coverage/margin確認。推移閉包なし|
|#158 独立baseline/naive比較、order/費用|iteration2 primary1/4、naive3/4だがnegative FM648、iteration6 partial-file対照|B独立小規模NO-GO。threshold fitting無し|
|#158 mixed吸収・bridge回帰、6/8/16適用境界|test_selective.py、iteration8 transfer全8観測|大規模はtimeout/presentation拒否、改善なし。639.897秒>600もfailure|
|#159 selective refine/groupと分岐仕様|adaptive.py、coarse.refine、architecture-decision.md、iteration6|first model-mixedのみrefine。goldでruntime割当/起動しない|
|#159 raw stage/metadata/compose互換性|iteration6 raw14、partial8 validator拒否、iteration8 6/16 oracle stage、type/body検証|C統合NO-GO。大規模意味推論は互換性gate未達により非採用、成功未立証|
|#159 同一行/別hunk/同symbol/UTF8/partial-cross-file/unknown|独立controlled7＋公開mixed別集計、既知#151回帰、unsupported安全回帰|混在分類gold10file観測、mixed TP7/FN3、false mixed0。最終mixed FM改善なし|
|#159 baseline差分/cost/GO|iteration6 final4/14対baseline2/14、FM70同値/FS36対44、raw FS200、27calls|control改善は保持。raw正解を最終成功へ数えない|
|#160 independent holdoutと固定比較|iteration7 prereg、corpus-d、gold-audit、fixed baseline/H23 adapter|15入力、12 primary、3事前別集計。component相関とcontrolled解釈の限界を明示|
|#160 exact/FM/FS/coverage/cost、各層・16file|iteration7 raw/summary-clarified/secondary|A2 4/12、9〜16 1/3、coverage15/15。publicのみ2/7も80%未達|
|#160 4file baseline品質差/H23|D同条件A2 0/4対baseline2/4、H23 0/4、iteration4履歴も保持|未完成pair null、停止の詳細/telemetry欠落を保持。回帰なしとは主張しない|
|#160 A/B/C独立寄与とproduction条件|全iteration報告＋architecture/gold監査|A2構造限定GO、B/C・意味候補NO-GO。implementation Issue/PRは今は作成しないと決定|
|全体境界・安全|差分限定、fixed helper/cache、temp-indexのみ、Skill無し|外部repositoryコード/test実行無し。source formatterはdata-only。cloud推論・gold tuning無し|

「single/mixed/unknown」は仮説でありモデル出力を事実にしない。Cの6/16file補助監査はoracle structural capabilityのみで、大規模semantic performanceを捏造しない。gated-out CやBをDへ全面統合することはIssueの順次能力評価契約に反するため、到達可能なA2のみを評価した。

metadata不足・目的分類の不安定・runtime予算管理は残余課題として確定した。このIssue内でproductionのvalidationを緩和する理由にはならない。将来の別Issueには目的帰属の安定性、CU/file所有権schema、partial-stage検証、type/body/release互換性、全cycle予算管理が必要である。

最終repository品質ゲートはこの照合とGitHubチェック更新の後に実行済みで、全8項目PASS。初回diff-check失敗とbyte-preserving修正後の影響範囲再検証はfinal-quality-gate.json / final-quality-gate-recheck.json / iteration-9-report.mdに保存した。機能削除が無いため既存test削除は不要。追加7研究回帰はsource corruption/割当不正/path/mode/resource/UTF8 stage/mixed false absorb/bridge/実際の予算停止/実際のvalidator互換性を確認し、自明なtestを追加していない。
