# Issue #142: 固定 top-2 selector と小規模候補列挙の事前登録

## 対象と順序

- [修正済み考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5891643582)に従う。fixture は `99dc127` で固定した[新規合成8件](issue-142-capped-structural-fixtures-2026-09-29.json)のみ。score は `6bf7f04`、`bc3bd66` で固定した現行/構造的0.4上限を再利用する。generator、score、fixture、gold、production planner、Pass 2 は変えない。
- 検証コードを先にコミットし、selector → 決定的な候補列挙診断の順に実行する。結果を見て条件・対象・列挙順を変えない。selector と列挙診断の結果は別 JSONL に記録する。

## 固定 top-2 selector

- 対象は `new_stem_doc_diverged` の weighted 候補集合を構造的0.4上限 score で順位付けした上位2件だけ。事前結果の ID は C001（非 gold）、C002（gold）であり、相違すればモデル call 前に中止する。
- 候補ID/grouping、repository input、generic forced-choice system/task、schema enum ID順を固定し、`none` を含めない。提示順は正順・逆順・逆順・正順の4 call。pinned MLX 3B `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48` を再利用する。output 2048 tokens、各 call 2分上限、retry/repair 0。
- 各 call の stop reason、選択 ID、有効 ID、gold 選択、prompt/schema hash、wall を記録する。4 call で一般的な selector 精度、提示順の因果効果、production architecture の成立や失敗原因は推定しない。

## 候補列挙診断

- 対象順は `new_test_pair_diverged`、`new_crossdir_collision`、`new_paraphrase`。各 fixture の weighted 候補集合を元にする。モデル call は0件。
- 3ファイルの fixture では、ID 昇順の各 unordered pair だけを同一 group、残りを singleton にする3 partitionを列挙する。4ファイルでは、最初のIDと残り各IDをpairにし、残り2 IDをpairにする3つの完全マッチングを列挙する。どちらもgold・score・relationに依存しない固定順である。
- 既存 weighted 候補を先に保持し、列挙候補を順に canonicalize・dedupe して cap8 まで追加する。元の候補数、列挙3件、追加 ID/grouping、重複数、cap 到達、gold 候補の有無を記録する。元候補と追加後候補を現行 score/上限 score の両方で順位付けし、total recall、recall@1/@2/@3、gold rank、top-2、追加非 gold 件数を出す。
- prepared document の relation edges は file ID、kind、class、reason のみを記録し、生 path/diff/evidence は結果 JSONL に保存しない。候補列挙は3〜4ファイルの合成例に限る coverage 診断であり、edge削除・意味的関係の発見・24ファイルへの拡張を検証しない。cap 到達や dedupe があればその実測をそのまま報告する。

## 判定範囲

- 合成goldはfixture作成者によるラベル。selectorは1 fixture・4 call、列挙は3 fixtureのみ。新規依存・モデル取得は行わない。score の再調整、generator の production 採用、Pass 2 接続はこの結果から行わない。
