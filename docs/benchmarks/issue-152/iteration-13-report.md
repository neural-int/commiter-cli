## 要約

固定した独立intent反例で、未変更adapterを介した2hop relationが独立変更間にも1組生成された。call pathをhard merge条件にする仮説を棄却する。さらに同一fileの直接呼出をgraph extractorが除外するため、ChangeUnit上の構造関係が欠落することを特定した。B No-Goは維持する。

## 検証結果

3件を14a1cf9で計測前固定した。direct dependentはrelation0、2hop dependentは独立unit間のrelation1、shared calleeはrelation0だった。全3件のbefore/after/各単独intent状態、計12状態のgo test ./...はexit0（testなし、compile検証）。独立性のgoldは個別public API要件として事前固定し、compile成功をsemantic goldの証明とは扱わない。モデルcall0、graph/relations/gold変更なし。

直接呼出を持つsame-fileケースでrelationが出ない原因はtools/benchmark152/graph/main.goの `targets[0] != f.src.ID` 条件だった。旧file graphで同一file edgeを除いており、同一fileの別symbolも除かれる。2hopケースは未変更別fileを経由するためrelationを生成する。これは前回3件のsource/test対が完全抽出された事実と矛盾しない。

## 考察

call graphの構造的到達性は共有intentを保証しない。Base係数とPublic加算の独立要件は、途中のRouteが不変でも2hop関係を持つ。hard mergeへ変更すれば少なくともこの反例を誤結合する。旧shared-callee抑制は今回も維持された。

same-fileの欠落はChangeUnitへの移行に伴う抽出契約の不足であり、観測可能な再現条件と原因を特定した。ここを修復しても前回のモデル分割失敗やsemantic FPを解決するとはいえない。関係をsoftのまま扱い、構造coverage修復とsemantic品質を区別する必要がある。

## Next Steps

- 新規専用worktreeでsame-file別symbolのcall edge除外を最小修復し、direct/2hop/shared-calleeを回帰確認する。ChangeUnitが同一fileの独立symbolを扱えるのに構造情報が欠落するため。
- 関係をhard mergeへ変更せず、同一symbol self-callは重複unit関係を生まないよう既存contractを検証する。coverage修復をsemantic改善と混同しないため。
- 次の一般化比較では修復済み構造coverage、独立intent反例、source/test対応を含める。既知fixtureへのprompt調整や予算拡大は行わない。
