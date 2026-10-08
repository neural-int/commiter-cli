## 要約

上位proposalのaccept/refine/unresolved契約とhost検証を追加。欠落・重複・未知atom・古いsnapshot・判断不能・8単位超過を拒否し、同じ分割の列挙順で結果が変わらないことを確認。これはbenchmark専用の構造検証で、意味品質・実Git入口の安全性・C GOは未立証。

## 検証結果

hostはtrusted before/afterからcontiguous hierarchyを再抽出する。modelはparent IDごとにaccept、またはrefineと全child atom IDの分割、またはunresolvedを返す。modelにbyte span/payload変更を許さず、再抽出したsource mappingを使用する。parent coverage完全一致、refineの全child一回割当、空group/未知ID/重複拒否、全体1〜8単位を検証する。acceptは意味的に一目的であると証明するものではない。

新規4testで正常accept/2分割・順序反転、欠落/重複/未知/空group、unresolved/古いsnapshot、9分割超過を確認。既存partition3testと合わせ7test pass、git diff --check pass。budget超過時に切り捨てない。model call0、Git mutation0。

## 考察

refinementの責務をmodelの判断とhostの構造検証に分離できた。一方、hostはsemantic independenceやpartial snapshotのcompile/test成功を保証しない。source mappingの検証と実Git mutation前のproduction validationの接続も未実装。構造passを正式C条件の全達成としてcheckboxへ反映しない。

C初回候補のscore/objectiveは変更していない。今回の階層refinementは新候補の前処理契約で、既存No-Go結果を置換しない。実例3proposalが意味的に適切か、必要な分割をmodelが選べるか、local CLIコストが許容かは次の固定評価で測る。

## Next Steps

- 新規C専用worktreeでaccept/refine/unresolved model呼び出しと未使用の階層境界fixtureを計測前固定する。goldは評価側のみとしmodelへrequirementラベルを渡さず、内部境界判定を測るため。
- 各refinementについてhost acceptance、gold境界、partial snapshotのcompile/test、tokens/latency/reject理由を別記録する。valid JSONとsemantic正しさを区別するため。
- 実履歴fd042894も構造・コスト診断として評価し、正解gold未認定のままexact/FM/FSを算出しない。8超過とunresolvedは拒否として保存し再試行しない。B停止・D条件未達を保持する。
