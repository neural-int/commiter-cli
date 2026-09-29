# Issue #142: 二方向 pair 比較と決定的集約の事前条件

## 対象と固定条件

[修正した考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5882653296)の最初の2手を診断する。前回の9合成 fixture と同じ候補集合を再生成する。既使用5件と前回 holdout 4件は別々に集計する。今回は新規 holdout を設けないため、未知 fixture への精度は評価しない。production planner、候補 generator、Pass 2、通常 CLI は変更しない。

固定 MLX model は `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。既存の model と helper を再利用し、新規依存・モデル取得はしない。各 call は以前の forced-choice system/prompt/schema、output 2048 tokens、2分上限、repair/retry 0。候補 ID と grouping の対応を保つ。

各 fixture の全 unordered candidate pair を候補 ID 昇順で列挙する。pair `A,B` について、提示順を `A,B`、`B,A`、`B,A`、`A,B` として4 call 行う。JSON Schema の enum は4 call とも `A,B` に固定する。したがって候補提示順以外の意図的な入力変更はしない。pair 間の呼び出し順は固定し、モデルの残る非決定性や時間的変動は排除したとみなさない。全9 fixture で候補数が前回同様に2または3なら21 pair、84 callを予定する。候補数が異なれば実数を記録する。

## 判定と記録

4 call がすべて `completed` かつ有効IDで、4回答が同じ候補なら `stable` とする。同じ提示順の2回答がそれぞれ一致し、両提示順の回答が異なる場合は `direction_disagreement`。同じ提示順で回答が異なる場合は `repeat_variation`。未完了または無効IDを含む pair は `incomplete`。`direction_disagreement` は順序に関連した差を示すが、位置が単独原因であることまでは示さない。

Go 側の集約では `stable` な pair だけを有向 edge とする。全ての他候補に stable に勝つ候補が1つ存在するときに限り確定する。確定しない場合は ambiguity。安定 edge の有向 cycle を別記する。合成 gold 一致は確定した行と全9行の両方を分母として示す。候補包含、pair status 内訳、call の stop reason、候補ID、prompt/schema hash、calls、wall、確定IDを記録する。生の prompt、response、repository content は保存しない。

結果は前回 tournament の calls、wall、最終一致と併記するが、異なる日時の実行なので直接の因果比較はしない。少数の合成 fixture から production accuracy、実開発者判断、位置効果の大きさ、採用可否を推定しない。結果を見て今回の条件や gold を変更しない。
