# Issue #142: 構造的加点の上限と新規合成 fixture の比較条件

## 固定する対象

- [修正済み考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5890818599)に従う。検証対象は benchmark 専用 scorer で、production planner、Pass 2、MLX selector、pairwise 比較は実行・変更しない。新規依存やモデル取得はない。
- 既存、二値 lexical、weighted lexical の generator、候補 cap8、candidate ID、canonicalization、dedupe、閾値0.75/1.25、候補順序は前回と同じ。各 fixture の同じ候補集合に現行 score と比較 score を適用する。
- score 仕様と検証コードを、下記8件の fixture 内容の作成・確認前に別コミットとして固定する。fixture 内容と gold を後続の別コミットで固定した後に一度だけ測定する。結果で score、fixture、除外条件を変えない。

## 比較する score

- 現行 score: [前回の事前登録](issue-142-prerank-preregistered-2026-09-29.md)と同一。file pair の soft source-test / `matching_test_path` +4、soft direct-import / `observed_import_path` +3、stem +1、同じ directory +0.1、共有 lexical token の `1/(df-1)` を加える。
- 上限付き score: file pair ごとに、上記 relation・stem・directory の構造的加点を合計して `min(0.4, structural)` とする。lexical token は現行と同じ path と raw diff から抽出し、同じ `1/(df-1)` を加える。structural と lexical は独立な証拠とはみなさない。
- 両 score とも candidate の同一 group 内の全 unordered file pair について `support(pair)-0.5` を合計し、別 group の pair は0。score 降順、同点は candidate ID 昇順。gold、候補 provenance、fixture 名は score に使わない。上限0.4は、構造的加点だけの file pair が正の同一 group score を得ないよう、pair penalty 0.5 未満に固定した値である。これは過去の失敗を見た後の設計であり、未知入力に対する事前知識とは扱わない。

## 後続作成の fixture

8件を次の種類・ファイル数・gold partition で固定する。各 fixture は未使用の合成 path と diff を後から記述する。`F001` から順にファイルを割り当て、gold は作成者の意図した変更目的であり、外部の開発者判断ではない。

| 順 | fixture | ファイル数 | gold partition | 作成条件 |
| ---: | --- | ---: | --- | --- |
| 1 | `new_test_pair_shared` | 3 | F001+F002 / F003 | source/test が同じ変更目的、第三ファイルは別目的 |
| 2 | `new_test_pair_diverged` | 3 | F001+F003 / F002 | source/test path は対応するが、test の変更目的は別。第三ファイルは source と同目的 |
| 3 | `new_stem_doc_shared` | 3 | F001+F002 / F003 | code/doc の basename stem が一致し、同じ変更目的 |
| 4 | `new_stem_doc_diverged` | 3 | F001 / F002 / F003 | code/doc の basename stem が一致するが、変更目的は別 |
| 5 | `new_crossdir_shared` | 4 | F001+F002 / F003+F004 | 別 directory の code/doc に二つの変更目的 |
| 6 | `new_crossdir_collision` | 4 | F001+F002 / F003+F004 | 別目的の file 間にも共有語がある二組 |
| 7 | `new_lexical_bridge` | 4 | F001+F002 / F003+F004 | 二組の間に共有語の bridge がある |
| 8 | `new_paraphrase` | 3 | F001+F002 / F003 | 同目的の二ファイルに言い換え表現がある |

## 観測と判定範囲

- 各 fixture・候補集合・score について、候補 ID/grouping/score、gold ID/rank、top-2 ID、top1-top2 margin、最高位の非 gold ID/score、候補数、追加非 gold 候補数を JSONL に記録する。生 path/diff は fixture 定義ファイルにのみ置き、結果 JSONL には入れない。
- 全8件と gold が候補にある件数の条件付き分母で、total recall と recall@1/@2/@3 を集計する。total recall は scorer に依存せず、generator ごとに共通である。候補数が K 未満なら存在する全候補を使う。
- 旧13件は新 score の調整や採否判定に使わない。新8件も設計者が作る少数の合成例で、実開発者判断、production 分布、selector 精度・コストを推定しない。どちらの score を production に採用するかは今回決めない。
