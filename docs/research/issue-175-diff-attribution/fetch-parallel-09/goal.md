# 逐次100件と2接続並行100件の比較

## Goal

初回取得を短縮するため、現在worktree・同じ期間・同じ保存Git入力・100件ページ・1応答300MBで、逐次と独立Turnの同時2件取得を比較する。

## Result / Decision

今回の単発測定は逐次11.908秒→並行7.399秒、37.9%短縮（約1.61倍）。受信量はどちらも約42.10MB、選択証拠・照合結果は一致。2接続並行方式は改善候補だが、既定paged100や本番実装は変更しない。

今回は実行前からメモリ圧迫監視値2、測定中も2で、system-wide Swapoutsが6,092増加している。並行化だけが原因とは判断できず、実行順/cache/システム負荷も統制していないので、改善率と常用時の負荷の再現性は未確定。

## Implementation

一覧列挙を1回行い、そのApp Serverを閉じてから2つの独立App Serverを起動し、Turnの共有queueを処理する。最大同時workerは2。同一App Server接続のRPC多重化とは別方式。各Turnのcursorは逐次処理。結果は列挙順に戻す。1応答300MB上限を全接続へ適用。3回のinitializeになるため、要求数・起動回数は少し増える。

Turn処理を共通関数へ分離。item_orderをTurn内の全item位置へ統一し、worker割当てや全体の取得順に左右されないようにした。同じTurn内の前後関係は保持する。フィルタ・欠落/失敗の扱い・diff・出典IDは共通。worker失敗でcancelし、完全取得済みTurnだけを保持する。

## Frozen conditions

前回の主checkout .gitignoreのquery root/since/until/保存Git入力を使用。現在cwdだけのthreadが対象。逐次→並行の順で各1回、100件要求、時間/累積受信量無制限・1応答300MB、内訳/種別統計は両方無効。source/binary/queryを実行前planへ固定。モデル生成・履歴変更・再試行なし。

並行では待ち時間の加算が壁時計時間と一致しないため、今回単一接続用の詳細タイミングは使用しない。メモリを約1秒間隔で監視し、critical=4なら測定プロセス群だけを停止する方針。今回は発動せず正常終了。

## Results

| 指標 | 逐次paged100 | 2接続parallel100 |
| --- | ---: | ---: |
| 取得時間 | 11.908秒 | 7.399秒 |
| 要求数 | 606 | 608 |
| 受信量 | 42,099,399bytes | 42,100,021bytes |
| thread / 対象Turn / item | 87 / 443 / 15,097 | 87 / 443 / 15,097 |
| 選択編集操作 | 0 | 0 |
| coverage / 照合 | complete / unmatched | complete / unmatched |
| harnessのサンプル最大RSS | 47.64MB | 49.91MB |
| App Server群のサンプル最大RSS | 135.28MB | 274.32MB |

要求が増えても独立Turnの待ちを重ねることで壁時計時間が短くなったという観測。サーバー内部のread/JSON化等の寄与を分離したものではない。300MB近辺の巨大応答に対する負荷試験でもない。

メモリは単発の約1秒間隔サンプルで、瞬間peakではない。phase分割はプロセス経過時間と逐次の計測時間を使った概算で、各最大値が同時刻に出た保証はない。server群はlauncher等の子孫も含む。並行側のRSS増加は観測できたが、全体のswap増加を特定プロセスに帰属させない。全20サンプルでpressure=2、開始前/終了後も2。Pageoutsは342、Swapoutsは6,092増加。

## Quality / validation

実データの選択操作・Turn文脈・coverage・Comparison全体が方式間で一致した。ただし今回の対象編集は0件であり、実編集の精度や取り逃がしを立証しない。非空の6Turn×104item fixtureで逐次と並行のsource/順序/照合結果が同じことを確認した。fake serverの要求開始/終了イベントで逐次最大1、並行最大2の重なりを検証。失敗時のfallbackも確認。

関連3パッケージのgo test、並行テストのgo test -race、PoC go vet、測定バイナリbuildが成功。追加依存なし。本番Collector/Plannerへ接続していない。

## Artifacts / Next gate

私有原本は<PRIVATE_TMP>/issue182-fetch-parallel-09/。plan、2セルの履歴/比較/metrics、memory時系列、外部timeを保存。公開用は本goalと本文/ID/ホームpathを含まないsummary。過去原本は保持。commit/push・Issue投稿は今回行っていない。

2接続は有力な改善候補として保持。常用採用を決めるなら、メモリ圧迫の低い状態で実行順を入れ替えた再確認と、現在cwdの非空実編集を持つ標本での確認が必要。今回は自動で追加実行しない。
