## 要約

ユーザーによる環境整備後、前回と同じ一括評価版D-Tを再build・ソース変更せず、C01/C02各1回で再実行した。今回の許可による有限な再検証として前回の結果を保持し、最大6生成要求・追加retry/repair/judgeなしで実施した。前回の大幅遅延は再現せず、KV準備はC01=4.26秒、C02=42.56秒、全体は72.76秒/179.78秒で完了した。入力要求と最終JSONは全6応答で元baselineおよび前回と一致し、両ケースの最終Validatorが受理した。

## 検証結果

| 条件 | 測定 | KV準備 秒 | Category＋Text 秒 | 全体 秒 | MLX peak GiB |
| --- | --- | ---: | ---: | ---: | ---: |
| C01-D-T | 改善後baseline：iteration03 | 4.69 | 44.97 | 78.86 | 4.80 |
| C01-D-T | 前回一括評価：iteration04 | 7.59 | 193.18 | 234.09 | 4.80 |
| C01-D-T | 環境整備後の再実行：iteration05 | 4.26 | 42.70 | 72.76 | 4.80 |
| C02-D-T | 改善後baseline：iteration03 | 66.47 | 129.34 | 248.48 | 5.20 |
| C02-D-T | 前回一括評価：iteration04 | 104.13 | 197.85 | 371.92 | 5.09 |
| C02-D-T | 環境整備後の再実行：iteration05 | 42.56 | 105.56 | 179.78 | 5.20 |

- C01-D-T: 入力bytes対iteration03=[True, True, True]、最終JSON対iteration03=[True, True, True]・対iteration04=[True, True, True]、Validatorエラー=''。Text再利用token=2221。
  実Grouping=1groups、固定参照=1groups。
- C02-D-T: 入力bytes対iteration03=[True, True, True]、最終JSON対iteration03=[True, True, True]・対iteration04=[True, True, True]、Validatorエラー=''。Text再利用token=17971。
  実Grouping=3groups、固定参照=1groups。

- 実生成要求数6。source/helper/runner/MetalとfixtureのHEAD/index/対象ファイルは前回と同一、実行前後でも不変。Gold・prefix監査・Package.resolvedのhashも不変。
- 同じモデル/revision、prompt、seed144、temperature0、thinking予算512/384/384、生成予算、chunk512、同期・解放頻度を維持。検証のみtimeoutなし。旧baselineの再推論はしていない。
- source・実行ファイルに変更がないため、前回成功済みのGo全test/vet/build、Swift37tests、Metal smokeを再実行していない。今回の検証は同一候補の実モデル再実行と完了監査である。
- 実行前後のメモリ情報を確認した。環境整備の具体的な内容は未取得であり、性能差を特定の環境操作に帰属しない。

## 考察

元baselineに対してKV準備はC01約9.2%、C02約36.0%、Category＋Textは約5.1%/18.4%短い観測だった。ただしC02のGrouping入力処理＋初回tokenも67.44→44.14秒へ約34.6%短縮し、今回の処置以外の速度変動がある。前回の大幅悪化だけを一括評価固有の退行とする判断は支持されないが、今回の短縮率を一括評価単独の効果として採用することもできない。

過去baselineとの非同時比較・各1回であり、一般的な速度改善率や一括評価単独の因果効果は確定できない。iteration04と今回の候補は同じ実行ファイルで、速度差は候補コード変更によるものではない。システム状態や実行時負荷の影響と整合するが、その内訳は未特定。

eval(cache)はcacheのinnerState()を評価し、旧eval(entry.state)と評価対象配列の表現が異なり得る。今回この差を修正したり、state配列を一括評価する別条件を追加したりしていない。これは前回と同じ候補の再測定である。

検証はtimeoutなしであり、C02約179.78秒は通常の120秒予算への適合を示さない。開始前/終了後のmemory_pressure報告値は66%/62%で、前回途中の31%とは観測時点も異なる。特定の負荷やメモリ操作の因果証拠にはしない。

Groupingに今回の処置はない。C02は固定参照グループを後段へ注入した比較であり、Validator受理を通常の一貫実行の成功や意味品質改善としない。MLX peakはallocator最大であり、OS RSSの比較ではない。

## Next Steps

登録した一巡で終了。追加生成・事後prompt/Gold調整・別実装への切替は行わない。通常経路の未commit採用コードは所在不明のままで、本検証は保存済みiteration03からの検証経路である。通常経路への接続・採用判断、②③への着手、commit/push、PR作成、Issue更新は今回行っていない。
