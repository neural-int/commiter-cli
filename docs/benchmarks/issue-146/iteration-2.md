# Issue #146 Iteration 2

## 要約

Iteration 2 は、metadata の順序を現行実装と揃え、実行可能な変更前後のテストを持つ4 / 6 / 12ファイル入力と、未提示 file 対の全対監査を比較した。4ファイル比較の grouping は現行・候補とも正解だったが、Stage 3 は両方停止した。6ファイルでは全対監査後も FM12 が残り、12ファイルでは局所判断が矛盾した。今回の固定モデル・契約のままで production の4ファイル上限を拡大する案は見送る。

## 検証結果

model / revision / sampling / context は Iteration 1 と同じ。contract suite は commit `e5de8a5` で推論前に固定し、変更前後の Go プログラム6件は全てテスト成功。graph は既存 syntax parser / relation extractor から生成し、gold に沿う edge を注入しない。初回結果を踏まえた追加診断であり、独立 adoption holdout ではない。

| fixture / 方式 | files | exact | FM / FS | complete | unresolved | plan / stop | calls | wall s | input / output tokens |
| --- | ---: | --- | --- | --- | --- | --- | ---: | ---: | --- |
| contract-baseline-4 / current-three-phase | 4 | true | 0 / 0 | true | false | invalid_schema | 3 | 60.400 | 3542 / 1075 |
| contract-baseline-4 / bridge | 4 | true | 0 / 0 | true | false | invalid_metadata_value | 3 | 58.340 | 3542 / 1075 |
| contract-independent-6 / bridge | 6 | false | 12 / 0 | true | false | invalid_metadata_value | 6 | 135.128 | 5632 / 2478 |
| contract-cross-boundary-12 / bridge | 12 | - | - / - | false | true | contradictory_pair | 8 | 251.776 | 5711 / 4328 |
| implementation-tests-8 / audited | 8 | - | - / - | false | true | transitive_contradiction | 5 | 136.275 | 4148 / 2743 |
| contract-independent-6 / audited | 6 | false | 12 / 0 | true | false | invalid_metadata_value | 9 | 248.593 | 8484 / 4179 |
| contract-baseline-4 / bridge | 4 | true | 0 / 0 | true | false | invalid_metadata_value | 3 | 73.602 | 3542 / 1075 |

- 実モデル7 observations、37 calls。全 calls の helper stop は `completed`、実 input / output tokens を保存した。context overflow / timeout / incomplete output は0。各局所判断の duplicate / missing / unknown file ID は0。矛盾時の global complete / exact / FM / FS を成功・0として数えない。
- 4ファイルの source-first path 順と group first-seen canonicalization を現行に揃えた。category / text の全 message、schema、generation options が一致する回帰テスト成功。現行・候補は input3542 / output1075 tokens、3calls、同じ grouping だった。wall の差を回帰・改善と判定しない。
- contract 12ファイルは8回目で `contradictory_pair`。metadata は実行しなかった。
- controlled 8ファイルは3 windows の bridge までは参照一致。5回目の監査で `F001/F003/F002/F008` が同じ group と判断され、既存の `F001 != F005` と `F005 == F008` に矛盾し、`transitive_contradiction` で metadata 前に停止した。
- contract 6ファイルは全対監査後も FM12 / FS0。grouping は整合していたが、固定参照と一致しなかった。Stage 3 の group 数1 / 1、missing / unknown0、strict schema成功、scope最大4文字、summary最大72文字だった。
- contract 4ファイルの数値診断は group 数2 / 2、missing / unknown0、strict schema成功、scope最大11文字、summary最大56文字だった。4 / 6ファイルの Stage 3 停止は48文字上限超過で、membership の品質と独立している。生成本文を保存しなかった。
- oracle の全対監査は controlled 7/7で参照一致・完全割当。window 数は4ファイル1、5ファイル4、8・9ファイル8、12ファイル16、16ファイル20。これは Go の契約検証であり、実モデルの正確さではない。contract graph-only は3件中2件が未確認、bridge の oracle は3/3で確定した。
- graph/selected IDs 不一致は backend 前に停止。window budget、context overflow、cancel、局所不正 ID、直接・推移矛盾、metadata 不正出力が部分 plan を返さない契約テスト成功。

## 考察

代表の橋渡しは未確認 group の候補を確定できるが、別の file 組合せで同じ判断になる保証はない。全対監査は今回8ファイルの隠れた不整合を検出した一方、6ファイルの整合した誤判断を修正・拒否できなかった。したがって、割当・整合性の検証と、意味的な目的別 grouping の正解は分けて評価する必要がある。追加 calls を増やすだけで correctness を保証する設計にはできない。

採用するのは、最大4file Stage 1、selected ID全件検証、soft edge は候補提示のみ、same/different 制約と矛盾時停止、global grouping 後の metadata という実験用契約。見送るのは graph-only の単独確定、代表 bridge のみの production 拡大、全対監査を正解保証とする production 拡大。現行4file上限を維持する。model / backend / prompt / runtime は変更しない。

測定上限は16files、48windows、全体600秒、各局所判断は既存120秒以内。G<=16なら metadata 最大8callsを含め56callsが実験上の上限となる。16K contextと各phaseの output枠を維持し、超過時は停止する。この48 / 600秒は実験 guard であり、production SLA・採用予算ではない。safeな最大実用規模は今回確立できなかった。wallはhelper loadを含み、fixture/graph事前構築・build・モデル準備・Git入力収集を除く。推論中に軽いGo build / 契約テストも行い、CPU・熱を隔離した latency評価ではない。

[設計判断と契約](https://github.com/neural-int/commiter-cli/blob/codex/issue-146-bounded-planning/docs/benchmarks/issue-146/decision.md) と [測定raw artifacts](https://github.com/neural-int/commiter-cli/tree/codex/issue-146-bounded-planning/docs/benchmarks/issue-146) を保存する。Design / Verificationの完了を、通常CLIの多ファイル機能提供と扱わない。

## Next Steps

- この設計・比較・見送り判断を Issue の各完了条件へ対応付け、検証成果として commit / push する。品質条件を緩めた上限拡大を避けるため。
- Issue の検証完了条件を確認した後、goal.md に従い最終 test / vet / build を実行し、レビュー用 PR と CI結果を残す。測定結果だけで検証基盤の配送完了と扱わないため。
- 意味的な membership 改善と将来の上限拡大は親 #145 の後続判断に委ねる。本 Issue の対象外である model変更・prompt改善へ実験を拡げないため。
