## 要約

親/B契約と既存実装を照合した。式依存候補は終了するが、全evidence探索の天井は未立証。既存historyはsynthetic housekeepingのfile-level co-changeで、同一file内symbol-level変更履歴の増分は未評価だった。次はこの独立仮説を最小検証する。B No-Go/C未着手を保持する。

## 検証結果

GitHubの親#150はAチェック済み、B/C未チェック。B#152は7条件中deterministic extraction/gold isolationの2項目のみチェック済み。fresh一般化・regression回避・absence・costは未達。親はB independent改善なしを停止/見直し条件に持つ。goalは全条件達成前の最終gate実行を許さず、天井時の停止条件を持つ。

既存tools/benchmark152/evaluate.py history()はfileへhousekeeping commentを追加するsynthetic commit scheduleを作り、initial/restoreも含む最大64commitのdiff-tree name-onlyでfile対co-changeを数える。symbol変更を抽出せず、現在のinline atomとの対応も評価しない。旧history失敗をsymbol-history仮説の失敗として転用できない。

symbol-callは独立改善gate未達。test impactは相殺/共有assertion反例とFS回帰、context成功はmetadata-onlyで再現。正規化graph比較はexact増加なし。statement/式観測は構造識別を改善したがiteration20はfresh差なし/FS回帰。production候補なし、65unit拒否、new-file/rename/sparse/dense/cost未完了。既存結果を変更しない。

## 考察

同じ式依存候補を繰り返す根拠はない。一方、file-level historyは同一file内の独立intentを区別できない情報形式であり、symbol-level履歴の評価はarchitecture上の差を持つ。これは新候補が成功する推定ではなく、未評価項目の特定。housekeeping scheduleをgoldと同じになるよう設計して成功させてはならず、初期一括commit/restoreの交絡、sparse/new/renameのabsenceを先に検証する必要がある。

## Next Steps

- 新規専用worktreeでboundedなsymbol-touch履歴抽出を最小検証する。before/after AST spanと実commit差分からsymbolのtouchを観測し、同一fileの別symbolを区別できるか確認するため。
- history scheduleと評価goldを独立に固定し、一括commit/housekeeping/初期snapshot/rename/no-historyをguardrailに含める。co-changeをground truthと扱わずunknownを保持するため。
- 観測contract成立後のみ共通containerのモデル比較へ進む。仮説が追加価値を示さなければそのNo-Goと停止条件を記録し、Cへ不合格入力を渡さない。
