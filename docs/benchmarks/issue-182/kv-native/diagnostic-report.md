# H182-008 D1/D2診断結果

## 要約

ユーザー指示によりD1 Gemma簡潔化C01、D2 Qwen4B現行C01を従来A・KV8bit/group64で実行した。両方plan受理・実Grouping基準一致。全6phaseのJSON回答・CPU計測を取得し、監視62サンプルすべて有効、monitor_failed=false。KV比較8セルは未実行。

ThinkingのCPU状態更新・進捗処理自体は小さく、token.itemの取得待ちが計測内訳の大半を占めた。ただし取得待ちにはGPU計算・runtime待ちが含まれ、無駄な同期や転送時間そのものを示さない。モデル/KV/同期の主因は未特定。KV8対nativeを次の一要因比較として行う意義は残る。

## CPU計測（3phase合計秒）

| 項目 | D1 Gemma | D2 Qwen4B |
| --- | ---: | ---: |
| Thinking token取得待ち（初回を除く） | 73.2818 | 127.7663 |
| Thinking状態更新・進捗 | 0.0456 | 0.1790 |
| Thinking mask準備 | 0.0029 | 0.0105 |
| response token取得待ち | 5.9579 | 8.2814 |
| response状態更新・進捗 | 0.0110 | 0.0058 |
| response mask準備 | 0.2353 | 0.3376 |

初回取得時間は各phaseで別記。Gemmaは0.2366/1.1438/2.6074秒、Qwenは原本summaryの値を参照。phaseラベルはcommit前の状態で決めるため、Thinking終了markerはThinking側。event countはcallback回数で、終了marker/EOS/先読みを含み、生成tokensと同一ではない。mask準備はhost上の準備でGPU適用完了時間ではない。CPU計測からGPU占有率・GPU演算時間・重なり率を算出しない。

## 生成結果

| ケース | cycle秒（診断負荷あり） | 生成tokens | Thinking G/C/T | 状態 |
| --- | ---: | ---: | --- | --- |
| D1 Gemma | 102.88 | 1817 | 360/651/676 | 全phase自然終了、plan受理 |
| D2 Qwen4B | 182.61 | 2449 | 515/1024/781 | Categoryのみ予算で境界を閉じJSONへ移行、plan受理 |

診断計測・Instrumentsの負荷があるため、これらの時間を速度比較の対照に使わない。既存採用prompt、Thinking ON・各phase1024、出力/時間制限なし、seed144、sampling、モデル資産を保持し、再試行・追加生成なし。

## Metal traceの範囲と採取上の問題

D1は専用PIDへattachし、実トレース区間31.199秒（設定30秒、時刻の区間には付随時間を含む）を保存した。Grouping/Categoryにまたがり、GPU区間38971行のうち対象PIDに紐付く24010行を確認。行数はbusy率やCPU/GPU重なり率ではない。

D1のxctraceは30秒の採取終了後も変換・保存が長く、コマンド全体253.68秒を要した。trace watcherが逐次処理のため、その間D2のモデル生成だけが先行した。D2へattachした時点ではTextが進行しており、対象終了で19.869秒の採取となった。GPU区間19004行、対象PIDに紐付く9343行を確認。D2のThinking区間はこのtraceに入っていない。

両recordコマンドはexit0でtraceを保存したが、両モデルのThinking全体を十分に採取できたとは扱わない。CPU/GPUの正確な重なり率・idleの原因・個別kernelの律速はこの集計では未確定。D2のThinkingを追加採取するなら、記録の保存待ちが次ケースattachを遅らせない実行順を別契約で用意する必要がある。今回は自動再試行していない。

traceおよびexport XMLには端末・環境情報を含むため、端末内だけに保持し公開コピーへ含めない。公開するのは抽出した数値・採取範囲と通常の研究ログのみ。

## 実行後監査と判断

全6実要求のprompt変換・不変条件、全実行契約hash、CPU診断field/event countの存在を確認。diagnostics/summary.jsonへ集計、diagnostics/review.jsonに判明事項・不明点を記録した。proceed_to_kv_comparison=trueは次の比較を技術的に進められるというレビューであり、この診断から自動で生成を開始する指示ではない。

次は従来A固定・診断OFFでKV8/nativeを新規比較する。今回の結果だけでKVや同期を主因とせず、実メモリ・生成量・形式/意味品質も確認する。比較8セルの実行はまだ行っていない。
