# Issue #141: 局所 boundary 判定の事前条件

## 目的と比較範囲

前回の edge accept/reject は候補 relation 14件を全件採用した。今回の仮説は、relation の有無を判定対象にせず、**2ファイルの変更内容だけ**を提示すれば、同一変更目的と独立目的を区別できるか、である。前回の結果だけから「relation の存在に引っ張られた」という原因は特定しない。

前回と同じ6主 fixture と2 guardrail、正解 grouping、モデル revision `a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`、1024 output tokens/request、2分/試行、2反復を使う。結果は前回の file-centric、強制縮約、edge decision と比較する。今回は Pass 1 の局所判定だけを測り、Pass 2 と結合しない。

## 候補生成と prompt

- 4ファイル以下は全 unordered pair を候補にする。これにより主比較の小規模 fixture と guardrail では候補漏れがない。
- 24ファイル fixture は、現行 relation の source/test・direct import 候補、basename から数字を除いた stem が同じファイルの隣接 pair、入力順で隣接する pair の和集合を使う。重複を除き file ID 順に固定する。正解ラベルは候補生成に使用しない。
- 各 pair の入力には2ファイルの path、language、raw diff、summary を渡す。relation kind、candidate component、正解 grouping、他ファイルの情報は渡さない。
- 8 pair 以下の batch ごとに `P001...` の boolean key を JSON Schema の required に固定する。Go は raw duplicate key、欠落・未知 key、値型を検査する。失敗時は推測で補完しない。
- `true` pair を Go が union し、孤立ファイルを singleton にして全 file ID をちょうど1回割り当てる。負の判定は cannot-link 制約ではないため、推移的 union が負の判定や正解境界を破る件数も測る。

## 評価

候補の真/偽件数、正解 group 内の連結性、candidate pair recall、判定の TP/TN/FP/FN、positive closure による false merge / false split、exact grouping、完全割当、構造的失敗、calls、wall、prompt/output bytes を記録する。出力 token 数が backend から取得できない場合は `unavailable` とする。

主判定は、主6 fixture で強制縮約の false split 改善を維持し、guardrail の false merge を0にできるか。24-file fixture の候補連結性が不足していれば、モデル判定の失敗とは区別する。false merge が0でも、小標本・合成 fixture を一般化しない。call/latency が大きい場合も production 候補としない。

この検証は benchmark 専用で、production planner、SRS、通常 CLI、新しい依存、モデルの download を変更しない。生の prompt、応答、生成 summary は保存せず、数値・file ID・失敗 code のみ保存する。
