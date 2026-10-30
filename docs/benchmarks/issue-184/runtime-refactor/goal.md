# H184-005 / runtime-refactor

## 実装

実改行表示はinternal/llm/display.goへ移し、MLX Backendがbounded profileのuser本文を作る境界で適用。共通plannerはcanonical JSONを保持、外側IPCはjson.Marshalで正規JSONを維持。Python proxyは本番経路に不要。

ThreePhaseGeneratorはMLXのBeginCycleを使い1サイクル専用helperを開閉する。--three-phaseが3要求を順に処理し、モデルとGrammarPreparationのみ保持。GrammarConstraint/matcher・生成状態・KVはphaseごとに新規。失敗・キャンセル・終了時に子プロセスを回収。

scope検査は引用符候補かを先に判定し、その候補だけ履歴token列を作る。解析結果の増分再利用は未実装。実入力8192tokens以下では32K設定時の毎chunk同期・解放を省き、8192超は従来の保護を使用する。

## 条件と評価

C184-02の補完済み固定fixture、5ファイル/3目的。Qwen3.5-4B 4bit、Thinking OFF、grammar/scope制約ON、native KV、32K/4096、sampling .6/.95/20、seed144、時間制限なし。baseline→improved各3phase、新規計6生成。Pythonは監視の起動管理だけで使用し、推論proxyには使用しない。

両群の3phase実要求が一致し、3phaseの生成JSONも全て一致。後続の固定入力を使う前に先行結果と元結果の一致を確認。新しいGit/App Server取得や動的入力投影を含む全CLI E2Eではなく、固定phase入力のモデル生成比較。

| 条件 | Grouping | Category | 本番最終validator | 3phase秒 | helper RSS最大MiB | system swap MiB | 熱状態 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| baseline | 3/3 | 3/3 | 通過 | 61.743 | 2085.6 | 0.00–0.00 | fair/nominal |
| improved | 3/3 | 3/3 | 通過 | 74.373 | 2899.4 | 0.00–866.62 | fair |

| 条件 | phase | prefill秒 | decode秒 | tok/s | load秒 | 文法共通準備秒 | input/output tokens |
| --- | --- | --- | --- | --- | --- | --- | --- |
| baseline | 01 | 16.778 | 2.146 | 24.24 | 1.977 | 0.699 | 5123/52 |
| baseline | 02 | 13.297 | 2.812 | 25.25 | 1.073 | 0.676 | 4027/71 |
| baseline | 03 | 13.832 | 4.834 | 17.17 | 1.101 | 0.707 | 3911/83 |
| improved | 01 | 20.270 | 1.992 | 26.11 | 1.463 | 0.685 | 5123/52 |
| improved | 02 | 16.901 | 4.159 | 17.07 | 0.000 | 0.000 | 4027/71 |
| improved | 03 | 19.091 | 6.108 | 13.59 | 0.000 | 0.000 | 3911/83 |

## 判定

機能は成立、速度改善は不成立。61.743→74.373秒で約20.5%悪化。ロードと共通文法準備はCategory/Textで再利用されたが、prefillは43.907→56.262秒、decodeは9.791→12.259秒へ増加。improvedでsystem swapが0→866.62MiBとなり、熱状態はfair。保護軽減とモデル保持がメモリ圧迫へ関係する仮説はあるが、3変更同時・固定順・1回のため単独原因を断定しない。

cacheReusedTokensは全phaseで0。improvedのprefillProtectionEnabledは全phase false、入力5123/4027/3911tokensで閾値以下であることを確認。phaseをまたぐKV再利用は実装していない。

Textは両群とも同一。UI scopeが実ファイル名とは異なるwidget.pyを含み、メール修正に「追加」という表現が残る。正しい分類やvalidator通過は、文章の全意味が正確である保証ではない。

今回の変更は未コミットの比較候補として保持。採用済みH184-HARNESS-002 revision 1の登録は上書きしない。性能改善版としての昇格は見送る。

## 検証と限界

Swift Releaseビルド、Swift53テスト、Python11テスト成功。Go全体はパッケージ直列実行で成功。旧EOF待ちのCLI疑似helperを継続応答へ更新し、JSON本文を解析する他backendとの互換性はMLX境界への表示移動で維持。既存の200ms起動タイムアウトテストは並列負荷下で起動前に失敗したため直列で確認した。

測定後、Goの最終phaseでは正常応答に加えてhelperの正常プロセス終了も確認する処理を追加。追加のモデル推論は行っておらず、掲載時間はmeasurements.jsonに記録したrunnerバイナリのもの。helper起動開始前と最終Closeの時間は含まれない。長入力8K超の推論や順序反転・反復試行は未実施。raw履歴・入力・応答はprivate原本へ保存し公開しない。
