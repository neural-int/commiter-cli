# 診断用App Serverによる内部区間の計測

## Goal

配布版と同じ版番号0.155.1の公式ソースを基に、本文要求の検証・履歴系統解決・DB検索・変換・JSON生成・送信待ちを測る。

## Build / Conditions

公式rust-v0.155.1、commit be2951ea34f0d295ed0becf97079f92fa5f6950e。Rust1.95.0、dev opt-level1、debug0、incremental無効、jobs2。外部依存1333件のlock内容は変更なし。公式タグのlocal crate版番号152件だけを0.0.0→0.155.1に同期。配布バイナリのcommit一致は未確認。ビルド19分12秒で成功。配布版より低い最適化条件と計測負荷があり、取得速度の直接比較に使わない。

通常codexは変更せず、一時ラッパーで診断版を指定。最大4本文worker・一覧中本文取得・queue16・既存保護・同じ対象/期間/保存Git入力で1回。ログは段階名・終了時刻・時間だけ、プロセス/層ごとに保存。

## Result / Decision

610要求、87thread、443Turn、15,097item、complete。診断版のwall time 8.907秒。配布版と保持証拠/照合結果一致。対象編集0件なので実編集の精度は未評価。本文要求513回のhandler時間合計に対して、履歴系統解決は約92.1%を占める。

|区間|回数|合計ms|中央値ms|p95 ms|
|---|---:|---:|---:|---:|
|rpc.items.total|513|15275.108|10.683|77.404|
|store.items.validate|513|40.685|0.062|0.148|
|store.items.lineage|513|14069.549|9.207|73.880|
|store.items.db_acquire|513|3.521|0.000|0.001|
|store.items.page_query|513|976.726|0.779|4.432|
|rpc.items.convert|513|71.680|0.079|0.510|
|transport.json_encode|615|45.913|0.029|0.313|
|transport.stdout_send_wait|615|74.033|0.032|0.569|

合計は複数workerの重複区間を含む。親子区間を加算しない。transportは本文以外も含む615メッセージ。

## Narrowed cause

本文の履歴系統解決14.070秒に対し、本文ページDB検索0.977秒、応答変換0.072秒。全応答JSON生成0.046秒・送信待ち0.074秒。JSON化やpipe送信より、履歴系統解決が次の調査対象。

一覧を含む系統解決603回の総計14.861秒に対し、最初の現在履歴パス解決は0.244秒。通常/アーカイブfallback探索タイマーは今回記録されなかった。このため、系統解決内のメタデータ読込・継承元パス探索・継承境界検証など、現在パス決定後の処理をさらに分ける必要がある。これら個別処理が主因と確定したわけではない。

## Validation / Limits

同時800件のtimer JSON出力検証成功、全7836イベントをJSON解析できた。binary build成功、実取得正常終了、既存配布版の保持証拠/照合結果一致。ビルドのみで公式全体テストは実行していない。計測はwall timeで、async待ちとCPU処理を分離しない。ログ書込の負荷も入れ子の親区間に含まれる。サブ区間の割合を配布版へそのまま移さない。

## Artifacts / Next

plan.jsonにsource/binary/patchの固定情報、diagnostic.patchに変更を保存。私有原本はtmpのissue182-diagnostic-measure-18、sourceはissue182-codex-diagnostic-source-18。配布版/既存原本保持。次は系統解決の継承元探索・メタ読込・境界検証を分離する候補。自動で最適化/追加計測しない。既定変更・commit/push・Issue投稿なし。
