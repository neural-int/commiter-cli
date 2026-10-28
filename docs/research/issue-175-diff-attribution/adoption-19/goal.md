# 取得改善の採用と探索終了

## Decision

内部探索と追加計測を終了。4.134秒/4.235秒を記録した最大4本文worker・一覧取得中本文取得・接続再利用を研究用capture CLIの既定pipeline100x4へ反映する。通常commiter CLI/既存Collector/Plannerへの統合はこの変更に含めない。

## Adopted contract

100件ページ、本文最大4worker、一覧用接続を含むメモリ監視、queue16、Turn単位の重複防止、item/操作の重複と競合検出、列挙順の保持、1応答300MB、1/1.5/2GiBとpressureによる保護を維持。警告時は接続再利用。配布版App Serverを使用。診断用ビルド・詳細計測・8並列は既定に採用しない。過去の測定原本は保持。

研究用capture helperはfixtureの逐次baselineを維持。単一接続のタイミング計測はpaged100を明示するテスト/文書へ更新。通常captureの既存時間・累積受信量の上限は変更しない。過去の4秒台の測定はunlimited-history指定であり、その値を全入力や通常の制限下で保証しない。

## Validation / Scope

既存の非空6Turn fixtureでpipeline100x4の結果・順序一致、重複/競合、最大同時数の検査を使用。追加の実履歴取得・モデル生成・App Server内部探索は実施しない。実データで対象編集が非空の精度検証は未完了であり、本番統合の完了とは扱わない。commit/push・Issue投稿は今回行わない。

関連3パッケージのgo test、並列fixtureのgo test -race、PoC go vet、採用版バイナリのgo buildが成功。
