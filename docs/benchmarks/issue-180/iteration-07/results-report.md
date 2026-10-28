## 要約

採用済みH180-001をbaselineとして、候補H180-002（grammar準備データ保持版）を実装・検証した。GrammarTokenizerが所有する語彙情報、空白token ID、停止token集合、EOS/unknown metadataを計画内で保持し、schema constraint/matcher/native stateはphaseごとに新規作成する。③同期・メモリ解放頻度は変更していない。2ケース・計6要求で計画内再利用と計画間分離を確認し、入力要求/最終JSONは全応答でH180-001と一致、両ケースの最終Validatorが受理した。grammar準備全体はC01で約2.04秒、C02で約1.50秒短縮した観測だが、全体時間の改善は両ケースで揃っていない。

## 検証結果

| ケース | 構成 | grammar準備全体 秒 | うちCategory＋Text 秒 | Category＋Text全体 秒 | 全体 秒 | MLX peak GiB |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| C01 | H180-001（iteration05） | 3.219 | 2.664 | 42.697 | 72.757 | 4.796 |
| C01 | H180-002（今回） | 1.183 | 0.568 | 41.265 | 72.145 | 4.796 |
| C02 | H180-001（iteration05） | 3.287 | 2.733 | 105.556 | 179.784 | 5.197 |
| C02 | H180-002（今回） | 1.787 | 1.248 | 109.720 | 195.934 | 5.197 |

- C01: grammarPreparationReused=[False, True, True]、不変準備秒=[0.6089947083964944, 2.086162567138672e-07, 3.3294782042503357e-07]、入力要求bytes一致=[True, True, True]、最終JSON一致=[True, True, True]、Validatorエラー=''。
  実Grouping=1groups、固定参照=1groups。
- C02: grammarPreparationReused=[False, True, True]、不変準備秒=[0.5378997083753347, 5.834735929965973e-07, 2.0815059542655945e-07]、入力要求bytes一致=[True, True, True]、最終JSON一致=[True, True, True]、Validatorエラー=''。
  実Grouping=3groups、固定参照=1groups。

- C01→C02各1回、実要求数6、上限6。検証のみtimeoutなし、retry/repair/judgeなし。H180-001のiteration05保存値を使用し、baselineの再推論はしていない。
- Swift release build成功。先行呼び出しの出力・実行ファイル更新が直ちに確認できなかったため、SwiftPMの直接呼び出しで確認した。共有scratchの先行ビルド待機もログに保持。最終ビルドの完了と候補helperの更新を確認してから固定した。モデル生成前の準備であり追加生成ではない。
- Swift既存37tests成功。共有tokenizerでschema別matcherが独立する既存テストを含む。関連Go mlx/planning tests、Metal smoke、git diff --check成功。依存追加なし。
- model/revision、prompt、Gold、seed144、temperature0、thinking予算512/384/384、生成予算、chunk512、一括cache評価、同期/解放頻度、KV snapshot/copyを維持。生成処理本体はH180-001からbytes不変を推論前に確認。
- source/helper/runner/Metal/参照契約とfixtureを固定し、実行前後の不変を監査。memory情報は測定し、詳細snapshotは公開報告に含めない。

## 考察

Category＋Textのgrammar準備はC01=2.664→0.568秒、C02=2.733→1.248秒。不変データ構築を各計画3回から1回へ減らし、後段では1µs未満の取得時間となった。一方、C02では変更していないKV準備も42.56→50.72秒へ伸び、全体は179.78→195.93秒だった。この増分を不変データ保持に帰属する証拠はないが、今回の一巡で一貫した全体高速化が得られたとは扱わない。

grammarSetupSecondsは不変データの準備に加えてschema constraint/matcher/native stateの構築等を含む。grammarPreparationSecondsは今回追加した不変データ準備だけの計測で、baselineには同じ独立内訳がない。未取得値を推定で補わない。

各計画の再利用フラグfalse/true/trueはGroupingで構築しCategory/Textで再利用した観測であり、次の計画のGroupingもfalseであることにより計画間で持ち越していないことを確認する。mutable matcherは保持対象外で、出力一致は今回のfixture内の観測に限る。

各条件1回・過去baselineとの非同時比較で、全体時間の差をそのまま不変データ保持の因果効果としない。Groupingや生成速度の変動を分ける。C02の後段は固定参照group注入で、通常の一貫実行や意味品質の改善を証明しない。MLX peakはallocator最大でCPU側grammarオブジェクトやOS RSS全体を表さない。timeoutなしの完了を通常120秒予算への適合と扱わない。

## Next Steps

事前固定した一巡で終了する。H180-002は候補として保存し、採用判断前にH180-001のbaselineを上書きしない。③・追加生成・事後prompt/Gold調整・通常経路接続を行わない。コードと全raw/監査/報告は専用worktreeに保持し、commit/push・Issue更新・PR作成は今回行っていない。
