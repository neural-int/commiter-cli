# Iteration 3: requirements-first fixtureによるA再計測

## 仮説と評価範囲

既存#149 goldはfile IDのpartitionであり、同一fileの異なるintentへの分配を測れない。新規fixtureはfile-onlyの構造制約を測る補完評価とし、既知#149 failure改善の証明へ読み替えない。過去の未達判定とB No-Goを保持する。親A gate自体の変更・通過は今回の測定結果だけで宣言しない。

抽出器は既存line-unit方式を変更せず使用する。モデルなし、標準ライブラリのみ。Goldも要求文も抽出器へ渡さない。固定したbefore/afterのみを入力し、生成unitからgoldの期待snapshotを表現できるか評価側で調べる。

## 固定データと事前条件

`tools/benchmark151/create_fixtures.py`は抽出器をimportせず、先に定義した変更要求とold/new editからgoldおよび全intent subsetの期待snapshotを保存する。development 6件、independent 6件: separated / adjacent / source-test / same-line / coupled-protocol / insert-delete。independentは別の要求・数値・symbolで構成するが、同じ作者・同じ構造のsynthetic transfer setであり、実repository分布のholdoutとは呼ばない。今回は双方を一度評価し、測定後の抽出器調整に用いない。

`fixtures/manifest.json`が全入力/gold/期待snapshotのSHA256を固定する。unit数・IDに正解を依存させない。同一lineケースを除外しない。

## Go / No-Go（今回の補完能力gate）

全12件でgold intentの独立snapshotを、重複なく完全なunit assignmentから表現できること。全intent subsetと全commit順序でGit indexが期待snapshotとbyte一致すること。complete reconstruction・determinism・changed byte coverageを全件で保持すること。file-onlyが表現できないケースで改善を示すこと。1件でもgoldが表現不能なら全範囲Goにはしない。連動protocolは全unitが一つのintentに属することを確認するが、semantic推定能力は測らない。

評価器の探索は各file 16unit・全intent 3件を上限とし、超過は失敗扱い。これはgold feasibility oracleの上限で、抽出器のproduction budget変更ではない。未知/重複ID・stale snapshot・overlap・byte/unit予算の拒否は既存contract testsで再検証する。

## 測定項目

file-only/ChangeUnit gold representability、単一intent選択の一致、complete/disjoint assignment、全subset staging、全intent順序staging、full reconstruction、source span byte coverage、repeat extraction determinism、unit数、JSON overhead、extraction latency。semantic exact/FM/FS/unresolvedはN/A、call/tokenは0。

既存14caseと従来synthetic 10caseも既存評価器で再実行する。新規fixtureの結果と分けて保存する。既存gold・既存測定ファイルは変更しない。

測定前manifest SHA256: `554fa7fbdfe47e50bed41fef098651dffaeafeb3dfb9a38334364f948fe194e8`
