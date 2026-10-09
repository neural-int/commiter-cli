## 要約

**次試験の入力・複数許容分割・機械/LLM責務・予算を固定し、推論前監査を完了した。** 依存方向はhostが抽出し、Gemmaは対象2fileのmerge/keep_separate/deferを提案する。初回NO-GO/goldは変更しない。16file/production導入は本試験の対象外。

## 検証結果

新規自己作成Goの8 workload、selected filesは4/6/8、対象pairは2file。対応実装/test2、独立2、方向依存2、多義的policy2。各caseのbefore/after endpoints全てGo tests PASS。対応2件はonly-A/only-B双方fail、独立/多義4件は双方PASS、新API2件は一方のみfailで実際の依存方向と一致した。準備監査20.818秒、model calls0。

same-package-independentのbeforeでstrings未使用importを検出し、発生条件/箇所/ログを再現後、そのimportのみ修正した。元入力・失敗・未実行preflightは保存。モデル結果を見ての入力変更ではない。

新Go標準AST抽出器は両方向の新API依存を検出。6研究回帰PASS。実planning.Validateを変更せず呼ぶadapterを用意し、依存順と複数許容境界を確認。詳細はiteration-2-contract.md / iteration-2-input-audit.json / iteration-2-preregistered.json。

## 考察

API addition/callerは同一commitも依存順付きsplitも許容する。timeout/flagも複数妥当partitionを認めるため、初回の5分類正解率とは同じ指標で比較しない。機械的な静的依存baseline、file-only、test-assisted baselineと、Gemma＋static hostを比較する。test-assistedは追加のtest実測情報を使うため、純粋なLLM因果効果とは呼ばない。

自己作成controlled入力であり独立作者/自然履歴/production/全8file目的推論の証明ではない。tests PASSはfixtureの観測範囲のみ。依存順のreverse revertと任意groupの単独revertを別測定する。

## Next Steps

- 本事前登録をcommit/push後、固定Gemmaで2提示順×8case=16callを実行する。結果を見て契約を変更しないため。
- baseline監査＋推論＋authority/byte/temp-index/適用/revert testsを1800秒内に計測する。per-call30秒、retry0、監査予約120秒、モデル追加/依存導入無し。
- 固定Gateに従い局所境界能力のGO/NO-GOと機械方式との差を保存する。次段/16file採用へ自動昇格しないため。
