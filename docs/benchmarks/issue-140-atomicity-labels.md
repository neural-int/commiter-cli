# Issue #140: 独立目的の分離指示を検証する事前ラベル

推論前に以下の比較条件と正解を固定する。現行の production-valid `full` と、`constraints.grouping` の末尾に `Put files with independent change purposes in separate commits.` の一文だけを加えた benchmark 専用 `full+atomicity` を比較する。モデル、backend、出力上限、timeout は同じとし、各 fixture を2回測る。2巡目は arm の順序を逆にする。production prompt と出力 schema は変更しない。

| fixture | 正解 group | 検証する点 |
| --- | --- | --- |
| `multi_commit` | `F001,F002` / `F003,F004` | 既存の分離例 |
| `cross_directory` | `F001,F002` | 既存の結合例 |
| `same_directory_independent` | `F001` / `F002` | 既存の同一ディレクトリ分離例 |
| `mixed_24` | 奇数 ID 12件 / 偶数 ID 12件 | 24-file での分離と完全割当 |
| `holdout_split` | `F001,F002` / `F003,F004` | 同じ `shared/` 内の cache 実装・テストと audit 実装・テストを分離する新規例 |
| `holdout_join` | `F001,F002,F003` | retry 実装・テスト・文書を1目的として結合する新規例 |

最後の2例は、今回の一文の結果を見る前に定義した holdout とする。正解は commit の順序を問わず file ID 集合で比較する。主指標は exact grouping、false merge pair、false split pair。complete assignment、repair calls、backend calls、wall time、prompt bytes も記録する。生成失敗は grouping の欠測として扱い、成功に含めない。

判断時は既存の分離例だけでなく、新規分離例で false merge が減るか、既存・新規の結合例で false split が増えないかを見る。全 file ID の有効な割当を維持することを前提とする。2回ずつの合成 fixture は効果の兆候を調べる用途であり、一般化や production 採用をこの測定だけで確定しない。
