# Issue #142: bounded candidate set と pairwise tournament の事前条件

## 固定する設計

[考察コメント](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5882414396)を修正後、holdout fixture 作成とモデル推論より前に、この文書と benchmark code をコミットする。production planner は変更しない。

既存 Louvain generator（Gonum v0.16.0、seed `142,1`、resolution `[0.25, 0.5, 1, 2, 4]`）が返した候補を先に維持する。次に、ファイル名の stem 一致または `matching_test_path` の source-test relation または `observed_import_path` の direct-import relation を結合した connected components を1候補として追加する。最後に all-joined と all-split を追加する。各 complete partition を canonicalize して重複を除き、先着8件に制限する。gold grouping は生成に使わない。8件を超える場合は後続を採用しない。

2候補以上を提示し、固定 ID のまま pairwise forced-choice を行う。1回目は昇順の候補列、2回目は降順の候補列で、先頭を暫定勝者とし、次の候補と比較する。各 call は既存 forced-choice prompt/schema を使い、`none` を選択肢に含めない。未完了または無効応答でその試行を中止し、最終候補を確定しない。修復・再試行はしない。両順序の差は bracket 順序の差であり、候補位置、ID、内容の各効果に分解しない。

同じ MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を再利用する。call あたり output 2048 tokens、2分上限。新規依存やモデル取得はしない。Pass 2、absolute verifier、通常 CLI は実行しない。

## Fixture と評価

既使用の verification fixture 5件を設計確認用に再測定する。設計を固定した後に、別コミットで新しい labeled synthetic fixture 4件を追加する。設計確認用と holdout の結果は混ぜずに表示する。holdout の観測後に generator、候補順、prompt、上限、gold を調整しない。

fixture ごとに既存/拡張 candidate count、合成 gold 候補の包含、各試行の bracket IDs、pair call の stop reason、valid ID、calls、wall、prompt/schema hash、選択 ID、最終合成 gold 一致、false merge / false split を記録する。gold 候補なしでも forced-choice は候補を返すため、その最終 grouping は gold 不一致として数え、`none` や reject と解釈しない。未完了試行は最終精度の分母から分ける。候補包含は fixture 分母、最終一致は全試行と完了試行の両方を示す。

少数の合成 fixture と2つの bracket 順序から production 分布、実利用者の正解、方式間の因果的優劣、採用可能な精度は推定しない。生の prompt、response、repository content は保存しない。
