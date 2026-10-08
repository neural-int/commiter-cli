## 要約

実Git commitのbefore/after snapshotから、同一file内のsymbol touchを区別できる最小観測を確認した。初期/new snapshotはunknown、関数外housekeepingはtouchなし。historyの意味的追加価値は未評価でB No-Goを維持する。

## 検証結果

temp Git repositoryで初期、Leftのみ変更、関数外comment、一括変更、renameをcommitした。旧file-levelでは同じfileの変更だが、symbol内容hash比較はLeftのみ、空、Left/Right両方をそれぞれ取得した。初期/new fileのbefore欠落はunknown_snapshot_boundary。rename内容一致はtouchなしだが、rename identity追跡は未実装と明記した。gold/モデルは使用せず、model call0。

抽出は既存Go symbol helperを利用しsource各1MiB、symbol256上限。重複symbolやparse/budget失敗はunknown。既存Python6testとdiff check成功。実commit IDと観測をraw保存。内容hashはsymbol内部format/commentにも反応し得る。削除/additionのsnapshot欠落を既知の無関係として扱わない。

## 考察

file-level historyとは異なり、同一fileの別symbol変更を観測できる。一括commitは複数symbol touchを示すだけでshared intentを証明しない。初期commitを全symbolのco-changeとして算入しない契約は保てるが、symbol rename/move、merge commit、同名method、旧current symbolへの対応とscanning costは未解決。観測成立をhistory一般化やB Goへ転用しない。

## Next Steps

- 新規専用worktreeで固定commit窓からsymbol touchを集計し、初期/merge/rename/missingをunknownとして保持するbounded history contractを検証する。snapshot観測だけでは現在のunitへの情報にならないため。
- 同一fileの独立symbol履歴と一括commit反例をgoldから独立に固定し、co-changeとco-separationの観測を両方保存する。単なる共変更をintent正解として採用しないため。
- contract成立後に未使用fixtureで共通containerのhistory増分を比較する。production制約、B未達条件、C未着手を維持する。
