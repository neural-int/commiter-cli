# Issue #140: intent discovery の事前ラベル

この表は intent probe の推論前に固定した。既存の `fixture.reference` を正解ファイル集合とし、意図名は合成 diff に合わせた説明用ラベルである。モデル出力との照合では、意図の説明文だけでなく `evidence_file_ids` も確認する。ID の完全割当を Phase A には要求しない。

| fixture | intent label | evidence file IDs |
| --- | --- | --- |
| `normal` | parser の normalize 実装と対応テスト | F001, F002 |
| `multi_commit` | parser の normalize 実装と対応テスト | F001, F002 |
| `multi_commit` | format オプションの説明文書 | F003, F004 |
| `japanese` | 日本語ヘルプ表示と対応テスト | F001, F002 |
| `large_diff` | format 処理とそのテスト・説明 | F001, F002, F003 |
| `cross_directory` | login 実装と別ディレクトリの対応テスト | F001, F002 |
| `same_directory_independent` | login 関数 | F001 |
| `same_directory_independent` | report 関数 | F002 |

注意: `multi_commit` のコード差分に format オプションは現れない。既存 grouping ラベルを踏襲するための事前定義であり、実際の intent 解釈の強い正解とみなさない。
