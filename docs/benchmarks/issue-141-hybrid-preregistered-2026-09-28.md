# Issue #141: relation seed と file-centric 割当の事前比較条件

この文書と benchmark code を推論前にコミットする。[前回の file-centric 比較](issue-141-file-centric-2026-09-28.md)と同じ6 fixture・正解 group、同じ repository input / relation context を使い、次の3条件を同じ実行内で比較する。

| 条件 | Pass 1 に渡す単位 | seed 対象 |
| --- | --- | --- |
| file-centric | 全 file ID を個別に割当 | なし |
| hybrid-source-test | source/test と対応した file を1単位に縮約 | `source_test`、`Soft`、`matching_test_path` の edge のみ |
| hybrid-source-test-import | source/test または直接 import で対応した file を縮約 | 上記に加えて `direct_import`、`Soft`、`observed_import_path` の edge |

現行 extractor の `source_test` と `direct_import` はどちらも soft edge であり、一般的な must-link ではない。縮約は benchmark 内だけの仮説検証であり、candidate component、同一ディレクトリ hint、changed identifier、未解決・曖昧な観測は seed にしない。seed は prepared Document の edge と file ID だけから決定し、正解 group は seed 選択や prompt に渡さない。union による推移的な結合も含めて、縮約後に正解 group と照合し seed の true / false pair を記録する。

各 unit に `U001` 形式の ID を付け、unit に属する元の file ID を入力に明示する。モデルは unit ID ごとに group label を1つ返す。既存 file-centric と同じ JSON Schema と Go の重複 key・未知／欠落 ID・不正 label 検証を unit ID に適用し、Go が file ID へ展開する。Pass 1 が完全割当かつ正解 grouping の場合だけ、前回と同じ batch Pass 2 と `planning.Validate()` を実行する。構造的失敗には最大1回の修復を許し、semantic grouping failure は修復しない。

## 事前の seed census と反例

下表は推論前に既存 extractor と prepared Document から決定的に確認した edge 件数である。`source_test` 単独 / `source_test + direct_import` の順に示す。今回の6 fixture では seed に起因する false pair は両条件とも0。ただし、この観測から一般的な安全性を推定しない。

| fixture | source/test edge | direct import 追加 edge | seed が扱える正解 pair |
| --- | ---: | ---: | ---: |
| multi_commit | 1 | 0 | implementation / test の1 pair。README / docs の結合は seed 対象外 |
| cross_directory | 0 | 1 | import の1 pair。source/test 単独では改善を強制できない |
| same_directory_independent | 0 | 0 | なし |
| mixed_24 | 0 | 0 | なし。24単位のまま |
| holdout_split | 2 | 0 | 2つの implementation / test pair |
| holdout_join | 1 | 0 | implementation / test の1 pair。docs との結合は seed 対象外 |

別の2つの guardrail fixture はモデルに渡さず、seed 選択と正解 group の照合だけを行う。同名の source/test が独立した変更目的である例と、変更された JS file が別目的の変更 file を import している例で、既存 extractor がそれぞれ edge を作り、縮約が false merge を1 pair強制することを確認する。これは soft edge を production で無条件に must-link にする案への反例であり、主比較の6 fixture や36試行の分母に混ぜない。

## 実行条件と判定

- 6 fixture × 3条件 × 2反復の36試行。2巡目は arm 順を逆にする。前回の file-centric 生データも照合するが、主比較は今回の同一実行内の3条件を使う。
- model は `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。既存ローカル cache を使い、追加ダウンロード・依存追加・クラウド API は使わない。
- 各 request は1024 output token、plan cycle は2分 timeout。fixture、正解、backend、repository evidence、relation context、Pass 2、修復上限、採点法は前回と同じ。unit list・schema・task 文は新条件として追加されるため、差を seed だけの純粋な因果効果とは扱わない。
- 主指標は完全割当、exact grouping、false merge / false split pair、per-file accuracy、seed true / false pair。構造的失敗は grouping 指標を欠測にする。特に結合2 fixture の false split、分離4 fixture の false merge と `mixed_24` の完全割当を別に示す。
- 補助指標は Pass 1 / repair / Pass 2 / retry / 総 calls、wall、prompt/output bytes、stop reason。output token count が取得できない場合は `unavailable`。Pass 2 失敗は Pass 1 と分離して集計する。

採用候補となるには、file-centric の完全割当12/12と false merge 0を維持し、false split 16 pairを減らし、結合2 fixture の recall を改善する必要がある。seed の false pair が出た場合はその時点で hard 縮約の安全性を満たさない。guardrail fixture は両 policy に反例となるため、主比較が改善しても、このまま production の hard must-link へ進めない。2反復・合成 fixture・1モデルだけで production 決定や pairwise 方式への移行を確定しない。

推論前の prompt bytes は `mixed_24` で file-centric 15,281、両 hybrid 16,166。大きい入力による timeout や出力品質への影響も含めて計測する。production planner、通常 CLI、SRS は変更しない。
