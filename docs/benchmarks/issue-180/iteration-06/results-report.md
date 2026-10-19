## 要約

ユーザーの明示指示に従い、元方式（不要logits評価省略済みD-T、各層のeval(entry.state)）をbaselineとして再実行した。保存済みiteration03 helperのSHA一致を確認して使用し、一括評価版のソース・実行ファイル・過去結果は変更していない。2ケース・計6要求を完了し、入力と最終JSONは全6応答で過去baselineおよび一括評価版と一致、両ケースの最終Validatorが受理した。

## 検証結果

| 条件 | 測定 | KV準備 秒 | Category＋Text 秒 | 全体 秒 | MLX peak GiB |
| --- | --- | ---: | ---: | ---: | ---: |
| C01-D-T | 過去baseline：iteration03 | 4.69 | 44.97 | 78.86 | 4.80 |
| C01-D-T | 今回元方式baseline：iteration06 | 4.59 | 42.31 | 72.55 | 4.80 |
| C01-D-T | 直前一括評価版：iteration05 | 4.26 | 42.70 | 72.76 | 4.80 |
| C02-D-T | 過去baseline：iteration03 | 66.47 | 129.34 | 248.48 | 5.20 |
| C02-D-T | 今回元方式baseline：iteration06 | 51.93 | 121.02 | 197.28 | 5.20 |
| C02-D-T | 直前一括評価版：iteration05 | 42.56 | 105.56 | 179.78 | 5.20 |

- C01-D-T: 入力要求bytes対元baseline=[True, True, True]、最終JSON対元baseline=[True, True, True]・対一括評価版=[True, True, True]。Validatorエラー=''。Text再利用token=2221。
  実Grouping=1groups、固定参照=1groups。
- C02-D-T: 入力要求bytes対元baseline=[True, True, True]、最終JSON対元baseline=[True, True, True]・対一括評価版=[True, True, True]。Validatorエラー=''。Text再利用token=17971。
  実Grouping=3groups、固定参照=1groups。

- C01-D-T→C02-D-T各1回、実生成要求数6、上限6。検証のみtimeoutなし、retry/repair/追加judgeなし。
- fixtureのHEAD/index/対象ファイル、baselineソースsnapshot/helper、runner、Metalライブラリを推論前に固定し、完了後の不変を確認。現worktreeの一括評価版ソースとGold・prefix監査・Package.resolvedも不変。
- モデル/revision、prompt、seed144、temperature0、thinking予算512/384/384、生成予算、chunk512、同期・解放頻度、固定参照グループ注入は一括評価版と同じ。runnerとMetalは同一bytesで、helperだけを元方式に切り替えた。
- 保存済み実行ファイルを再利用し、コード変更・再buildをしていないため、既存Go/Swiftテストの再実行はしていない。今回の検証は元方式の実モデル再実行と完了監査。git diff --checkは再確認。
- 実行前後のメモリ情報を確認した。詳細snapshotは公開報告に含めない。

## 考察

新baselineに対し、一括評価版はC01のKV準備を約0.33秒（7.3%）短縮したが、Category＋Textは約0.39秒（0.9%）長く、全体の差は約0.21秒だった。C02はKV準備を約9.38秒（18.1%）、Category＋Textを約15.47秒（12.8%）、全体を約17.50秒（8.9%）短縮した観測。両ケースでMLX peakは一致した。全ケースの高速化ではなく、大きい入力C02の短縮という結果である。

今回は環境整備後の追加baselineであり、過去baselineだけを参照した比較より条件差を減らせる。ただし各1回・逐次測定であり、候補とbaselineのhelperビルド時点も異なる。反復・交互順序の対比較を行っておらず、一般的な改善率や一括評価単独の因果効果を確定しない。

eval(cache)と旧state評価は対象配列の表現が異なり得る。今回、その差を追加修正したり、別の評価条件を実行したりしていない。Groupingは処置対象外なので、その変動をcache評価の効果としない。

C02の後段は固定参照グループによる比較であり、Validator受理を通常の一貫実行や意味品質の成功と扱わない。MLX peakはallocator最大であり、OS RSSではない。timeoutなしの完了を通常120秒予算への適合と扱わない。

## Next Steps

元方式baselineの登録した一巡を終了。追加生成・retry/repair・prompt/Gold調整・別実装への切替は行わない。通常経路の未commit採用コードは所在不明のままで、本検証は保存済み検証経路に限定する。通常経路への採用・②③への着手・commit/push・PR作成・Issue更新は今回行っていない。
