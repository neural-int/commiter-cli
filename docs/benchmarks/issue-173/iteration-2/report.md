# Issue #173 Iteration 2

## 要約

`update` を選択肢から外しても、既存 Gold が `fix` の 6 グループを安定して `fix` に戻せなかった。アーム A（Iteration 1 候補。11 type に `update` と `change` を加えたもの）の Gold 一致は 0/6、アーム B（`update` だけを外したもの）は 1/6 だった。Iteration 1 で `update` だった 4 グループは、B で `fix` 1 件、`feat` 2 件、`change` 1 件に分かれた。

この結果は切り分けであり、`update` の削除を採用しない。Iteration 1 の NO-GO は覆さない。本番の type 一覧と共通 120 秒 deadline は変更していない。

## 比較条件

比較は推論前に `preregistration.json` へ固定した。変更したのは Category の type 選択肢と、feat と fix の二択に関する fallback の 2 文だけである。グループ単位の Raw Diff、関連テスト、Grouping 依頼、Text 依頼、モデル、temperature 0、top-p 1、top-k 0、seed 144、出力上限、native thought 上限 384、再試行 0 はアーム間で同じである。

アーム A の依頼は Iteration 1 候補のバイト列を複製した。アーム B は system 指示から `update` の定義と「feat と fix が曖昧なら `update`」を削除し、動作変更が feat と fix に限られるときはどちらかを選ぶこと、その回避に `change` を使わないことを書いた。JSON Schema の type enum からも `update` を外した。test-only の enum `["test"]` は F11 のまま残した。

既存 fixture、Gold、Iteration 1 の結果ファイルは変更していない。Gold はエージェント作成の synthetic 評価であり、独立した人間評価ではない。生成は 60 回で、Category 50 回と、既存 A〜E の Grouping 5 回、Text 5 回である。安全上限は全 phase・両アームとも 1 呼び出し 600 秒で、製品の 120 秒 deadline はプロセスに掛けていない。

## 既存 6 グループ

アーム A は Iteration 1 候補と 6/6 で同じ type だった。`update` 4、`feat` 2、`fix` 0 で、Gold 一致は 0 である。

| グループ | Gold | Iteration 1 | アーム A | アーム B |
| --- | --- | --- | --- | --- |
| fixture A / G001 | fix | update | update | change |
| fixture B / G001 | fix | update | update | fix |
| fixture C / G001 | fix | feat | feat | feat |
| fixture C / G002 | fix | feat | feat | feat |
| fixture D / G001 | fix | update | update | feat |
| fixture E / G001 | fix | update | update | feat |

B の `fix` 選択は fixture B の 1 件だけである。fixture D と E は `feat` になり、feat/fix の誤断定は A の 2 件から B の 4 件へ増えた。fixture A は `change` で、二択を `change` で避けている。Category 10 リクエストはすべて completed かつ Schema 適合で、timeout は 0 だった。

## 境界 20 件

境界でもアーム A は Iteration 1 候補と 20/20 で一致した。識別クラスごとに分ける。

識別可能 15 件の Gold 一致は A が 8/15、B が 12/15 だった。差の 4 件は Gold が `fix` の F02、F04、F05、F20 で、A は `update` 3 件と `feat` 1 件、B は 4 件とも `fix` だった。この 15 件で、Gold が `fix` でないのに B が `fix` を選んだ例は 0 である。`docs`、`style`、`refactor`、`chore`、`revert` の一致は両アームで同じだった。`perf` は両アームとも `refactor`、`build` は両アームとも `chore` のままであった。`ci` の F13 は A が `chore`、B が `test` で、どちらも Gold の `ci` ではない。F11 の `test` は両アームとも enum が `["test"]` だけの Schema 制約による。

feat と fix だけが識別不能な F03、F10、F16 では、B の feat/fix 強制断定は 0 件だった。F03 と F16 は A が Gold の `update`、B が `change` で、指定した二択を `change` で避けた。F10 は両アームとも `style` で、Gold の `update` には届いていない。二択が偶然 `feat` または `fix` に当たった、という結果はここにはない。

全体として特定不能な F17 は両アームとも `change` で Gold と一致した。F18 は A が `update`、B が `change` で、B だけが Gold と一致した。B は `update` を出せないので、この 1 件だけでは、複数目的を識別したことと、空いた曖昧ラベルへ移ったこととを分けられない。

境界全体のラベル一致は A が 11/20、B が 14/20 である。この 14 には、上記の識別可能 `fix` 4 件と F18 の `change` が入り、feat/fix 曖昧 3 件の一致は A の 2 から B の 0 へ減っている。14/20 を、`update` 除外の総合的な精度向上とは扱わない。

## 時間、Schema、資源

Grouping 5 件は Schema 適合、Text 5 件は適合しなかった。不適合は次の長さによる。scope の上限は 16、summary の上限は 48 である。

| fixture | scope | summary |
| --- | --- | --- |
| A | 20 | 54 |
| B | 9 | 54 |
| C G001 | 8 | 59 |
| C G002 | 8 | 43 |
| D | 7 | 55 |
| E | 7 | 55 |

C の G002 だけが長さの内にある。Text の不適合は Category の type 比較とは別に残る。

各 phase は別プロセスで、120 秒の共有 deadline では止めていない。安全上限 600 秒による timeout は 0 だった。Grouping、Category、Text の wall time の和は、fixture A〜C では両アームとも 120 秒未満、D は A が 166.971 秒、B が 149.230 秒、E は A が 264.715 秒、B が 261.648 秒だった。E の Grouping 単独は 132.495 秒で、Category に入る前に製品 deadline を超える。A〜C も Text の長さで phase Schema を通らない。deadline を外したことで受理 plan は増えておらず、この時間は Category の一致数とは別に記録する。

helper の最大 RSS は 3.798 GiB だった。pressure level は 1 のままで、1 試行前後の swap 増分の最大は最初の Grouping の 793.00 MiB（3465.00 MiB から 4258.00 MiB）だった。RSS は Metal/GPU の総ピークではなく、swap の増分を helper だけに帰属させない。停止閾値（pressure 4 以上、RSS 12 GiB 超、swap 増分 2 GiB 超）には達していない。

入出力 token 件数は未取得である。固定した本番 helper の応答に token 欄がなく、byte 数を token 数にしていない。依頼は凍結済み JSON を helper へ渡しており、リポジトリは開いていない。

## 判定

事前に固定した規則では、既存 6 グループの Gold 一致が A を上回り、かつ歴史的な `update` 4 件がすべて `fix` に戻ることを、`update` が `fix` を妨げていたという仮説の支持条件にした。今回は一致が 0 から 1 へ増えた一方で、4 件のうち `fix` は 1 件だけなので、仮説は一部しか支持されない。識別可能な境界で `fix` の誤選択は増えなかったが、既存グループでは `feat` の誤断定が 2 件増えている。

したがって `update` の削除は採用しない。境界 15 件中の `fix` 4 件が B で一致したことは、既存 4 件の回復とは別に残し、今回の採否を逆転させる材料にはしない。本番の Category 定義、120 秒 deadline、Grouping、Text は変更していない。main への merge やリリースは行わない。
