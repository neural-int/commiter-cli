# H184-005 / multiline-ab 入力表示A/Bの検証

指定ハーネス・実行環境: **4-HARNESS-001 / revision 1**。既存登録名は **H184-HARNESS-001 / revision 1**。両者の対応を準備メタデータに保持し、登録済みprofile B（Text scope制約あり）を表示A/Bの両群で使用する。

実験番号はユーザー指定によりH184-005へ統一。準備時の仮番号H184-006は廃止。過去のscope比較H184-005は原本を保持し、今回のstudy_key=H184-005/multiline-abで区別する。登録済みhelperとmlx.metallibのSHA-256一致を確認した。

## Goal

ホストの元データとLLM表示を分離する。補完済みfixtureの日本語入力で、A=現行の一行JSON表示、B=文字列中の改行エスケープだけを実改行にする表示を比較する。raw_diffの追加／削除コードへの構造化は今回は行わない。

## State

completed。準備時の非推論contract probe6回・unit test4件・fixture検証34ファイルの確認後、全2セル・3phaseずつ計6生成が完了。両群ともGrouping3/3、Category3/3、最終host validator通過。詳細は[結果](results.md)と[計測値](measurements.json)に記録する。

ユーザー承認によりBの実改行表示をH184-HARNESS-002 revision 1として研究用の次期基準へ採用。精度向上や一般的な速度向上を実証したという意味ではない。ヘッダー削除・added_code/deleted_codeへの分解は採用しない。

## Host and display contract

ホストのfixture、Git差分、履歴、照合情報、ファイル／ブロックIDは保持する。user_requestsを空配列にする処理は両群共通の既存採用候補であり、A/Bの変更点ではない。表示変換はhelperへ送る直前にだけ適用し、戻り値のJSON検証やGit操作経路を変更しない。

Bは厳密なJSONではなく、JSONと同じキー・配列・引用符・エスケープを保ち、文字列内LFだけを実改行にしたLLM向け表示である。ホストが解析するcanonical JSONと外側IPCは有効なJSONのまま。表示専用の文字列をホストのJSON入力として再利用しない。

文字列中の改行エスケープを字句単位で変換する。コードの文字列リテラルに含まれるバックスラッシュn、引用符、インデント、空行、末尾改行を維持する。復元で元のcanonical入力とbyte一致し、緩いJSONデコードで全フィールドが同一であることを確認する。YAMLやMarkdownへの全面変更、フィールド名変更、要約・削除・翻訳は行わない。

## Cases and fixed conditions

| 条件 | 表示 | コード | 履歴 |
| --- | --- | --- | --- |
| A | 現行の一行JSON | raw_diff維持 | 日本語の最終応答、user_requestsは空配列 |
| B | 同じ情報を実改行で表示 | raw_diff維持 | Aと同じ |

C184-02、H184-KIMDEL-REPAIRED-002、5ファイル／3目的／9ブロック／3contexts／Grouping command14件。partial/ambiguous/history_incompleteを維持。Qwen3.5-4B Affine4bit/group64 revision 0e7ffd5c629ef7719d4cbc04069232580bfa9d9c。両群で登録ハーネスrevision 1と同じSwift Release helperを使用。

Thinking OFF、json_schema、Textの追加scope制約は両群ON、native KV、context32768/output4096、temperature0.6/top_p0.95/top_k20、penalty1、seed144、eager、warmup/retry/repair0、時間制限なし。systemと出力Schemaは同じ生成段階で共通。新たな依存なし。

各群1回、A→B、Grouping→Category→Textを新規生成する。各条件の実出力を後続フェーズへ継承し、最終host validatorまで評価。正常に全フェーズへ到達した場合は計6生成。先行失敗時は後続未評価を明示し、追加・再試行をしない。動的なGrouping/typeの差が生じた場合、後続入力も変わるため、その差を表示変換そのものと混同しない。

## Preparation checks

| phase | Aの実改行 | Bの実改行 | A bytes | B bytes | 復元byte一致 |
| --- | --- | --- | --- | --- | --- |
| Grouping | 0 | 129 | 14834 | 14705 | 一致 |
| Category | 0 | 129 | 11202 | 11073 | 一致 |
| Text | 0 | 129 | 11284 | 11155 | 一致 |

上表は保存済み入力量のpreviewで、実行時の動的出力から作る後続入力の計測値ではない。native contract probeでモデル可視systemとSchemaのA/B一致、Bの実改行保持、generation_calls=0を確認。host元fixtureのハッシュ不変。準備済みランナーは明示的な--executeなしでは推論を開始しない。

## Evaluation plan

Groupingの正解分割／完全割当、Categoryのtype／breaking、Textのscope／要約の意味・自然さ・日本語、JSON成立、最終validatorを分けて評価する。入力／生成tokens、prefill／decode、各host/runtime区間、全体時間、MLXメモリ、RSS、CPU／GPU／swap／熱状態を既存監視で記録。native KV dtype/bytes、物理温度、電力は既存ランタイムの未取得項目として維持する。

単一fixture・各1回で一般的優位性を結論しない。モデル可視の表示だけを変更するが、レイアウトによる解釈変化も含めて評価する。生の会話・入力原本は非公開で保持し、今回の資料には転載しない。
