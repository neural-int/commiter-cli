# 64K prefill保護の順序均等化比較

状態: 実行開始。結果・採用判断は未確定。

## 目的

コンテキスト設定を65,536 tokensに固定し、prefill同期・キャッシュ解放のON/OFFだけを比較する。16K自動切替や本番のコンテキスト予算はまだ変更しない。

## 固定条件

- Qwen3.5-4B Affine 4bit、既存固定revision、Thinking OFF、JSON grammarあり
- temperature 0.6 / top_p 0.95 / top_k 20 / seed 144
- 出力4,096 tokens、native KV、フェーズ間prefix再利用なし
- 固定Grouping入力のみ。3フェーズ伝播E2Eとは扱わない
- 実入力約12,288 / 16,384 / 20,480 tokens。既存fixture本文を診断用参照として反復して長さを調整する。新しい編集履歴を捏造しないが、自然な長入力の代表性は保証しない
- Chat Template適用後の実token数をhelperの監査経路で確認する。目標との差は8 tokens以内

## 比較と実行順

Aは既存のlegacy保護、Bは研究設定限定のdiagnostic-off。同じReleaseバイナリ・同じ要求を使用する。

各入力長にABBA/BAABを3組、各条件6回、合計36回。入力長の順序は組ごとに12/16/20K、16/20/12K、20/12/16Kとする。実行前に順序をplanへ固定する。

両条件とも短い既存入力でウォームアップを1回行う。各測定は新しいhelperと空のKVから実行する。初回ロード時間はprefillと分離して記録する。

## 開始条件・監視

thermalState nominal、低電力モードOFF、1分load averageが論理CPU数の半分以下、memory_pressureのfree percentageが20%以上の状態を60秒維持してから開始する。30分以内に回復しない場合は停止する。電源状態は監視ログへ保存する。

CPU、GPU、プロセスRSS、MLXメモリ、メモリ圧迫、swap、熱状態を取得する。必須監視失敗、critical熱状態、helper RSS 12GiB到達時は停止する。熱状態がfair/seriousに変わっただけでは結果を除外しない。

## 評価

prefill/prepare/load/decode/全体時間、生成量、停止理由、Grouping出力と構造検証、資源推移を原本へ保存する。対比較・中央値・ばらつきと実行順を併記し、速度差がノイズと区別できなければ未確認とする。監視とロードを含む全体時間を、GPU prefillだけの時間と混同しない。

これは保護ON/OFFの診断であり、64K上限超過・履歴省略の境界検証、32K対64K比較、自然な長入力での精度検証は別途必要。4ファイル制限は変更しない。

## 追加切り分け：同期を維持し、解放のみ変更

前記36回は終了。次は同じ20,475 tokensの要求を使い、legacy（同期ON・解放ON）とdiagnostic-sync-only（同期ON・解放OFF）を各1回、新しい同一Releaseバイナリで比較する。両方のウォームアップは集計対象外。通常設定は変更しない。

最初の2回で大きなメモリ挙動差が出るかを確認する。GPU全体の割り当てピークが1.5倍以上、かつswapout差が10倍以上となる場合を、明確な差の目安とする。速度の小差のみ・背景負荷・不整合などで判断困難な場合だけ逆順の2回を追加し、最大4回で止める。この回数で小さな速度差の統計的確定は行わない。GPU割り当てはhelper専用ではなく、swap/圧縮はシステム全体のカウンターである。

input/output・native KV・sampling・開始条件・監視は前記と同一。prefillSynchronizationEnabledとprefillCacheClearEnabledを応答へ記録し、設定の適用を検査する。
