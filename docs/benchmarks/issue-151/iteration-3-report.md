## 要約

要求から先にgoldと期待snapshotを構成した新規fixture 12件で、Stage Aを再計測した。ChangeUnitは10/12件、file-onlyは2/12件のgold境界を表現できた。同一lineの独立intent 2件は現在のline-unitでは表現不能であり、今回の全範囲能力gateはNO-GO。親A gateと子#151全体は未達を維持する。

## 検証結果

- 新規専用worktree: `/Users/Natsuki/.codex/worktrees/issue-151-fixture-remeasurement/commiter-cli`、branch `codex/issue-151-fixture-remeasurement`。
- 測定前固定commit: `722f5a3`。入力/gold/全期待snapshotのmanifest SHA256: `554fa7fbdfe47e50bed41fef098651dffaeafeb3dfb9a38334364f948fe194e8`。
- development 6件、independent 6件。各組はseparated / adjacent / source-test / same-line / coupled-protocol / insert-deleteで構成。
- 合計18file / 30unit。両組ともChangeUnit 5/6、file-only 1/6がgold表現可能。
- separated、adjacent、source-test、insert-deleteは両組でfile-only不可・ChangeUnit可。coupled-protocolは両方式で可。same-lineは両方式で不可。
- 12/12でfull byte reconstruction、unchanged spans/changed payloadのcoverage、再抽出の決定性が成功。
- 表現可能10件でintentへのdisjoint complete assignment、全intent subset 36状態と全intent順序18系列のindex byte一致を確認。
- gold由来の期待snapshotを直接patch化した検査では、全12件の全44状態をGit indexへstageできた。これはGitが期待状態を保持できる検査であり、unit方式が12件通ったという意味ではない。
- same-line 2件は各1unit。各R1/R2の単独期待snapshotに一致するunit subsetが0件だった。
- unit JSON合計13,264bytes。抽出合計0.637ms、fixture別median 0.050ms / max 0.074ms。単一実行・小規模syntheticの測定値であり、実運用性能推定には使わない。
- 既存14回帰＋従来synthetic10件を別途再実行し、24/24でforward/reverse staging成功。138file / 141unit。既存goldと過去結果ファイルは維持。
- モデル不使用。call/input token/output tokenは0。semantic predictionのexact/FM/FS/unresolvedはN/A。
- Python既存contract 7test成功。Go exporter test成功、symbol helper build/test/vet成功、Python構文検査とdiff check成功。Go build時にsandbox内のmodule stat cache書込警告が出たが、buildはexit 0で生成helperによる回帰評価も完了。

## 考察

file未満のrepresentationは、今回の8件でfile-onlyが持つ構造的制約を解消した。一方、line-unitでは同一lineの二つの独立操作を分離できない。Gitへ単独期待snapshotをstageできる事実と合わせると、当該2件の制約はGitではなく抽出器の表現粒度にある。

評価側でgold snapshotに一致するunit subsetを探索した結果であり、semantic intentを自動推定できたという結果ではない。coupled-protocolもgoldに従って複数unitを一intentへ割り当てられる検査で、連動関係を推論した証拠ではない。

independent組は測定前固定の別要求・数値・symbolを持つが、同作者・同構造のsynthetic transfer setであり、実repository分布での一般化や最終production holdoutを証明しない。今回評価した全12件は既に使用済み評価データとして扱う。

新規fixtureによる優位性は既知#149 failure改善の証明ではない。親の既知#149条件を置換・達成扱いにせず、過去のA未達とB No-Goを維持する。同一lineを除外して今回gateを合格へ変更しない。

## Next Steps

- 行内の独立操作を表すbounded representation仮説を別iterationで事前定義する。今回の2件でpartial staging可能な境界を表現できない原因が再現されたため。
- 実装を比較する場合、今回の12件は回帰用とし、未使用の構造・要求を持つ追加fixtureを測定前固定する。使用済み評価データへの適合だけでGoとしないため。
- 親Aの「既知#149 boundary改善」と子の構造能力gateの評価関係を明文化する。新規の表現能力を既知failure改善と混同しないため。
- B/Cへは今回の結果で進めない。全範囲Aと親gateは未達であり、前回B No-Goも継続するため。
