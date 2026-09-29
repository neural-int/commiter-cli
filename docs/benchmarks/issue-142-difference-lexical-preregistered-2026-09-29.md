# Issue #142: 差分提示と語彙候補の探索的検証条件

## 対象と固定条件

[修正済みの考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5883092375)から selector と generator を別実験にする。前回と同じ9合成 fixture、候補生成器、MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48` を用いる。新規依存・モデル取得、production planner、Pass 2、通常 CLI の変更はしない。今回は新規 holdout がないため、結果は探索的な合成 fixture 内の観測に限る。

## Selector probe

`verify_misleading_relation` C001/C002 を主対象、`verify_join_present` C001/C002 を既知の安定正答対照、`holdout_atomic_feature` C001/C002 を既知の方向差対照とする。前回の候補 ID と grouping、repository input、JSON Schema enum 順を固定する。complete-partition 表示の対照 arm と、両候補で co-membership が異なるファイル pair と各候補の同居/分離だけを表示する difference arm を用意する。共通 pair は省略し、repository input のファイル内容と relation context は両 arm で維持する。両 arm は異なる instruction/prompt となるため、提示形式の複合効果として扱い、情報省略の単独因果効果とは呼ばない。

各 fixture・arm で `A,B`、`B,A`、`B,A`、`A,B` の4 call。arm は各 run で交互に先行させる。3 fixture × 2 arm × 4 = 24 call。output 2048、各 call 2分上限、retry/repair 0。完了・有効 ID、各方向の選択、4回答一致、合成 gold 選好、calls、wall、stop reason、prompt/schema hash を記録する。生の prompt/response は保存しない。過去実測との差を因果比較に用いず、同一セッションの両 arm を並記する。

## Generator probe

元の9 fixture の既存拡張候補を対照として固定する。各ファイルの path と diff から英字を抽出して CamelCase/snake_case/区切り文字で分割・小文字化し、長さ4文字以上の token にする。固定 stopword は `bool, case, const, docs, document, false, for, from, func, go, html, how, if, int, md, new, nil, return, src, string, test, the, true, when`。2ファイル間に1 token以上の一致があれば lexical edge とし、連結成分による complete partition 1件を候補末尾へ追加する。canonicalize/dedupe、既存上限8を維持する。既存候補の順とIDは保持する。語彙候補は生成のみとし、ranking は実行しない。

fixture 別に edge と共有 token、候補数、追加候補の有無、cap、合成 gold 包含を記録する。主対象 `holdout_crossdir_semantic` の他にも全9 fixture を走査して誤連結を確認する。既使用の合成 fixture であるため、未知入力への recall/precision 改善は推定しない。
