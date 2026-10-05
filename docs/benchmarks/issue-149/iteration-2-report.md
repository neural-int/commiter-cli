## 要約

per-file Semantic IRは4file基準と独立6fileで参照一致したが、cross-boundary12fileでは2契約を6つのsource/test組へ誤分割した。H1はproduction candidateにしない。次は事前固定済みのbounded batch抽出で費用とglobal groupingを測り、失敗時はraw-global対照で圧縮の影響を切り分ける。

## 検証結果

| fixture / architecture | files | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | ---: | --- | --- | --- | --- | ---: | --- | ---: |
| contract-baseline-4 / current Stage1 | 4 | true | 0 / 0 | true | false | 1 | 1116 / 570 | 28.410 |
| contract-baseline-4 / per-file IR | 4 | true | 0 / 0 | true | false | 5 | 2049 / 2996 | 183.665 |
| contract-cross-boundary-12 / per-file IR | 12 | false | 0 / 24 | true | false | 13 | 6523 / 8014 | 550.350 |
| holdout-independent-6 / per-file IR | 6 | true | 0 / 0 | true | false | 7 | 2960 / 4137 | 278.620 |

全26callsはcompleted。context overflow/timeout/incomplete0。accepted membershipはselected全件を一意割当しunknown/missing ID0、JSON duplicateキー拒否を適用。4fileの意味品質は同じ固定参照に一致したが、呼出数・prompt/taskはarchitecture固有に異なる。metadata未実行のgrouping-only比較でありplan成功ではない。12fileで得られたgroupingはF001/F002、F003/F004、F005/F006、F007/F008、F009/F010、F011/F012。

holdout6は初回結果確認前の24b166bで固定、H1の調整に使用していない。H2もholdout6結果確認前の67e784bで固定した。追加16file評価は8つの著者定義契約（既存5目的＋新規3目的）であり、完全に未使用の16file内容とは主張しない。変更前後の全fixtureプログラムはテスト成功。

測定binaryは24b166b。IR文字数はJSON schemaの240字とhostの960byte guardで制限していた。後続ec56c6dでhostを実240文字検証へ揃えた。この補強後に実行するH2へ、過去の成立を移し替えない。

## 考察

全件割当とFM0は確認できたが、跨directoryの共通契約のjoinは成立しなかった。per-fileの要約が共有契約を落としたのか、global判断が局所目的を過度に分けたのかは、保存していないIR本文とmembership結果だけでは特定できない。source/test単位へ正しく揃ったことを期待groupingの成功へ読み替えない。

4fileのgroupingまで約6.5倍のwall増加を観測した。12fileではmetadata前でも約550秒であり、現行120秒cycleを維持するarchitectureとしては不成立。繰り返し同じ設定を測って成功を探す根拠はない。

## Next Steps

- H2の4file以下のkeyed batch抽出を6/12fileで測る。独立IRを維持したままcallsをceil(N/4)+1へ減らせるかを確認するため。
- H2がcross-boundary joinを失敗したらraw-globalを同じ失敗fixtureで測る。共有契約を判断するraw事実の欠落とglobal semantic decisionを切り分けるため。
- raw事実の再導入が合理的なら、IRを唯一の根拠にせず観測差分・soft dependency evidenceを保持するgrounded representationを次のarchitecture仮説にする。local group固定や多数決へ戻らず、圧縮により消える関係を確かめるため。
- H2が有望なら4file/独立評価/順序逆転/metadataとplanning.Validate接続を測る。groupingだけの成功をproduction採用判断へ飛躍させないため。
