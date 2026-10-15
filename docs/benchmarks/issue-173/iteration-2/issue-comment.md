## Iteration 2：`update` 除外の feat/fix 二択比較

### 要約

切り分けは一部しか支持されませんでした。既存 Gold が `fix` の 6 グループで、アーム A（`update` あり）の一致は 0/6、アーム B（`update` なし）は 1/6 です。Iteration 1 で `update` だった 4 件は、B で `fix` 1、`feat` 2、`change` 1 に分かれました。`update` の削除は採用せず、Iteration 1 の NO-GO は覆しません。本番の type と 120 秒 deadline は変更していません。

### 条件

推論前に比較を固定しました。変えたのは Category の type 選択肢と、feat/fix の二択に関する fallback 文だけです。Raw Diff、関連テスト、Grouping、Text、モデル、sampling、再試行 0 は同一です。アーム A は Iteration 1 候補の依頼そのもので、今回の A は既存 6/6、境界 20/20 で Iteration 1 の候補出力と一致しました。生成は 60 回、timeout は 0 です。検証ハーネスでは共有 120 秒 deadline を掛けず、各呼び出しの安全上限を 600 秒にしました。

### 既存 6 グループ

| グループ | Gold | アーム A | アーム B |
| --- | --- | --- | --- |
| fixture A / G001 | fix | update | change |
| fixture B / G001 | fix | update | fix |
| fixture C / G001 | fix | feat | feat |
| fixture C / G002 | fix | feat | feat |
| fixture D / G001 | fix | update | feat |
| fixture E / G001 | fix | update | feat |

B の feat/fix 誤断定は 2 件から 4 件へ増えました。fixture A は `change` で二択を避けています。Category 50 件はすべて Schema 適合です。

### 境界

識別可能 15 件の一致は A が 8/15、B が 12/15 でした。増分は Gold が `fix` の F02、F04、F05、F20 で、B は 4/4 が `fix`、この 15 件に `fix` の誤選択はありません。`docs` / `style` / `refactor` / `chore` / `revert` の一致は同じです。`perf` は両方 `refactor`、`build` は両方 `chore` のままです。F13 の `ci` は A が `chore`、B が `test` で、どちらも不一致です。

feat/fix だけが識別不能な F03、F10、F16 では、B の feat/fix 強制断定は 0 です。F03 と F16 は A の `update` から B の `change` へ移り、二択を `change` で避けました。F10 は両方 `style` です。F17 は両方 `change`、F18 は A が `update`、B が `change` で Gold と一致しました。B は `update` を出せないため、F18 だけでは識別とラベルの置換を分けられません。境界のラベル一致 11/20 と 14/20 は、この内訳を崩して精度向上とは扱いません。

### 時間と Text

Text 5 件は scope 16 文字または summary 48 文字を超え、phase Schema に適合しませんでした。A〜C の 3 phase 合計は 120 秒未満でも Text で不適合、D は 166.971 秒と 149.230 秒、E は 264.715 秒と 261.648 秒で、E の Grouping 単独が 132.495 秒です。受理 plan は増えておらず、この時間は Category の一致とは別です。

helper RSS 最大は 3.798 GiB、pressure は 1、swap 増分最大は 793.00 MiB でした。RSS は Metal/GPU 総ピークではなく、swap を helper だけに帰属させません。token 件数は helper 応答に欄がないため未取得で、byte 数では代用していません。依頼は凍結 JSON で、リポジトリは開いていません。

### 判定

既存 4 件が `fix` に戻る、という仮説は一部しか支持されません。境界の識別可能な `fix` 4 件が B で一致したことは別枠の観察として残し、採否は逆転させません。`update` は候補に残し、main への merge とリリースは行いません。

証跡は作業ツリーの `docs/benchmarks/issue-173/iteration-2/`（`preregistration.json`、`results.jsonl`、`analysis.json`、`report.md`）にあります。このコメント時点では branch へ push していません。
