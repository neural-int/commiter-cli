# 本文RPCと一覧用App Server終了時間の分離

## Goal / Conditions

最大4本文worker、一覧中本文取得、queue16、既存メモリ保護、同じ対象・期間・保存Git入力で1回実測。接続別の独立計測で本文/IDを追加ログへ保存しない。

## Results

数値はsummary.json参照。全セルcomplete、443Turn、15,097item、610要求、対象編集0件。本文RPCは513回。応答待ち受信とJSON解析、一覧cleanupを分離した。本文待ちは4worker合算でwall timeへ加算しない。一覧cleanupはclose/kill/Waitを含む。

## Decision / Limits

一覧cleanupは約3msであり、前回の本文終了後約0.30秒を終了処理と推測する根拠はない。今回は一覧の残りRPC等も含む時間として扱う。本文取得RPCの待ちが主な候補で、サーバー内部読込/JSON化/pipe受信は未分離。実編集精度は対象操作0件のため未評価。単発・負荷変動を含むため過去との差を計測オーバーヘッドと断定しない。

## Validation / Artifacts

関連3パッケージテスト、計測有効fixtureのrace検査、vet成功。原本は私有tmpのissue182-rpc-cleanup-14。公開用は本goalと数値summaryのみ。追加再試行、既定変更、commit/push、Issue投稿なし。
