## 要約

固定したsymbol-history guardrail比較はexact増加なし、FM増加で候補条件を満たさなかった。B No-Goを保持する。この履歴setはgold対応の相関を持たない条件なので、symbol history全体の無効性の証明ではない。

## 検証結果

| 条件 | complete | exact | FM | FS |
| --- | --- | --- | --- | --- |
| 共通graph A-only対照 | 3/3 | 0/3 | 29 | 8 |
| symbol履歴追加 | 3/3 | 0/3 | 42 | 5 |

separate条件FM5→18、FS4→1。batchはFM0/FS4両方同値。sparseはFM24/FS0両方同値。全6call completed/accepted、input20012/output720tokens、累積327.517秒。retry/repairなし。fixture/gate393b619、runner7f1bf2fで計測前固定。

current unit annotationからpath/symbol identityを自動生成し、実numeric変更とrevertをtemp Gitへcommitした。HEADはfixture Beforeへ戻した。goldはschedule生成に使わず評価側だけ。各history scan二回の完全一致をassert。初回抽出は17commit/5.363秒、3commit/0.587秒、3commit/0.609秒。cacheなし、memory未計測。抽出の実commit/eventを保存した。

## 考察

separate/batch/sparse履歴は意図的にgold境界との対応を作らず、すべての別変更/すべての一括/単一symbol観測というabsence/反例条件。数値変更の過去testが全passする履歴は要求しておらず、歴史的intentラベルも付けていない。したがってinformative historyが存在するときの追加価値は未評価で、このsetの失敗を全面的なhistory No-Goの立証に転用できない。

同作者syntheticの既存template派生identityであり、実repository独立holdoutではない。5file評価だがproduction baseline/legacy/new/rename/cost全条件は未達。各Git process timeoutはscan全体のUX保証ではない。Bの不合格をCへ渡さない。

## Next Steps

- 新規専用worktreeで、評価変更より前に成立した意味のあるsource/test履歴を持つ入力を検証する。今回の履歴は境界情報を持たず、有効な追加signalの存在仮説を直接検査していないため。
- 正例履歴の過去before/afterとtestを固定し、評価goldを参照せず観測する。独立変更の一括commit反例とsparse/renameを同じ比較へ残し、historyをground truthにしないため。
- 有効履歴の最小比較とguardrailが両立しなければcandidate無しと停止条件を記録する。prompt/予算調整で今回の失敗を修復しない。
