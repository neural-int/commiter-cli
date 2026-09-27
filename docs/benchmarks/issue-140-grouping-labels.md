# Issue #140: grouping 追加検証の事前ラベル

以下の正解 grouping と判定基準は推論実行前に固定した。commit の順序は問わず、file ID 集合で比較する。false merge / false split は誤った file pair の数として数える。

| fixture | 変更意図 | 正解 group | 役割 |
| --- | --- | --- | --- |
| `multi_commit` | parser 実装・テストと format 文書 | `F001,F002` / `F003,F004` | 異なる意図の分離 |
| `cross_directory` | login 実装と別ディレクトリのテスト | `F001,F002` | 関連ファイルの結合 |
| `same_directory_independent` | login と report | `F001` / `F002` | 同一ディレクトリ内の分離 |
| `mixed_24` | login rule 12ファイルと report rule 12ファイル | `F001,F003,...,F023` / `F002,F004,...,F024` | 24ファイルの完全割当と過剰結合を同時に評価 |

`mixed_24` は `shared/` に login / report ファイルを交互に並べる。各ファイルは対応する名前の関数を追加する。意図をファイル名に明示した合成 fixture であり、実リポジトリの曖昧な変更集合への一般化は行わない。既存 `multi_commit` のラベルも従来の fixture 定義を継承する。

各 fixture について `baseline`、`guidance-only`、`statistics-only`、`guidance+statistics`、production で有効な `full` を同じ model / backend / output budget / wall-time limit で各2回測る。2巡目は arm 順を逆にする。部分 relation context は要因探索専用であり、production 採用候補は `full` の結果を別に見る。

主指標は exact grouping match、false merge、false split。complete assignment、repair、backend calls、wall time、prompt bytes も記録する。invalid plan は grouping を採点できないため、欠測として失敗分類を残す。baseline 自体が誤る fixture では「baseline より悪化しない」だけで品質改善とは判断しない。`full` が正解 grouping を維持・改善するかを production 候補の判断材料とするが、この少数の合成 fixture だけで採用を決定しない。
