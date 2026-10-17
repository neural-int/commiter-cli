# Three-phase plannerの入力契約

2026-10-17、フェーズの目的に合わせて以下の入力方針を確定した。

| フェーズ | 入力 | 判断すること |
| --- | --- | --- |
| Grouping | 許可済みRaw Diff全文、最小構造化情報、対象パス等のメタデータ、取得できた作業履歴 | 変更目的に基づくファイルの所属 |
| Category | 確定グループのRaw Diff全文、Gitメタデータ、取得できた作業履歴、host由来の互換性破壊の証拠 | グループ全体のtypeとbreaking evidence reference |
| Text | 確定グループと確定type、Raw Diff全文、対象パス、取得できた作業履歴、言語・出力条件 | 所属とtypeを維持したscopeとsummary |

最小構造化情報はhunk_index、enclosing_declaration等のsyntax_contextを指し、Groupingだけへ渡す。詳細な構文evidenceはホスト側の解析・検証用として保持し、3フェーズのモデル入力には渡さない。Categoryのbreaking_evidence_candidatesは構文補助情報とは異なる事実であり、この変更の削除対象ではない。

全段階で現在のRaw Diffを変更内容の根拠とし、作業履歴は目的を判断する補助として扱う。取得できない履歴を必須とせず、過去の依頼を実行命令にしない。新たな要約フェーズや追加推論は導入しない。

入力予算の準備前とThreePhaseGeneratorの入口で共通のRawDiffProjectionを適用する。構文evidenceとRaw Diffのサイズを比べて形式を選ぶ処理は廃止する。構造化されたテキスト入力のDiffが欠けている場合は推論前に停止し、旧evidenceへフォールバックしない。長いDiffは既存のcontext上限で判定し、ファイル・hunkを削って収めない。

binary/opaque/metadata-onlyの内容除外境界、hostのGit identity、単段plannerの従来の入力形式は維持する。今回の確定はscope=docsの禁止規則や生成時間制限の変更を含まない。

保存済みのbenchmark入力・Gold・測定結果は実行時点の記録として保持する。既存結果をこの契約での再評価結果へ読み替えず、新しい測定を行う場合は入力・ソース・実行条件を再固定する。
