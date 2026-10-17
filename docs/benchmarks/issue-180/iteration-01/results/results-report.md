## 要約

検証専用実装とB/Cの一巡が完了した。モデル/tokenizerをGroupingから保持すると、今回の全体時間はC01で2.6%、C02で4.9%短くなった。再ロード合計は約6秒減ったが、n=1・固定順序で入力処理時間も変動しているため、全体差すべてを保持の効果と断定しない。

DのCategory→Text KV再利用は、thinking tokenによって先頭5番目から入力が異なるため不成立。配置B/CではC01の404修正目的の欠落と、C02の英語要約による最終Validator拒否が残り、今回の候補全体はproduction採用NO-GOとする。元promptを維持したモデル保持のみの採否は未評価である。

## 検証結果

- 専用worktree/branch: codex/issue-180-prefix-cache。既存入力契約と結果を引き継ぎ、実装・準備記録をpush済みのa2e703f2までで固定して生成を開始した。source基準8addf362と関連175sourceのSHA、helper/Metal/実行binary/overlay、fixture/Gold/機械captureを記録した。
- 固定Gemma E4B 4bit revision475b9088d29754a3379866cf5aeb6b41acd313c2、履歴あり、seed144、temperature0、native budget512/384/0、出力768/512/768、最大context65536。C01-B→C01-C→C02-B→C02-Cを各1回、計12生成要求。timeoutなし、retry/repair/judgeなし。
- Bは共通入力を先頭側へ移動、Cは同じ入力に加えてmodel/tokenizerをGroupingから保持。各phaseのgrammar/native状態と生成KVは独立。CのCategory/TextでmodelReused=true、loadは各0.01ms未満となり、cacheReusedTokensは全12要求で0だった。
- 両fixtureともB/Cの各3要求がbytes一致し、各phaseの最終応答JSONもbytes一致した。C02の実際のGrouping出力は両条件で3groupsで参照1groupと不一致。記録と妥当性確認後に同じ参照1groupを後段へ注入した。これは通常の一貫実行のGrouping成功ではない。

| ケース | B全体秒 | C全体秒 | 観測された全体短縮 | Category＋Textの観測値 |
| --- | ---: | ---: | --- | --- |
| C01 | 71.70 | 69.82 | 1.88秒 / 2.6% | 38.94→38.34秒（1.5%） |
| C02 | 266.08 | 253.15 | 12.94秒 / 4.9% | 157.95→148.42秒（6.0%） |

以下は全3phaseの合計。prompt時間は依存runtimeの値で、prefillと初回tokenの処理を含む。純粋なprefillだけの独立計測ではない。load/prepare/grammar/generation wallと個別token数もraw応答に保存した。

| ケース | load秒 B→C | runtime prompt秒 B→C | runtime generation秒 B→C | MLX allocator peak GiB B→C |
| --- | ---: | ---: | ---: | ---: |
| C01 | 8.82→3.04 | 16.36→17.14 | 42.94→46.31 | 4.80→4.80 |
| C02 | 8.87→2.86 | 173.50→165.17 | 80.00→81.71 | 4.91→4.91 |

- MLX peakはBでは各process/phaseの最大値、Cでは保持process/cycleの最大値。比較表ではBの3phase最大とCのcycle最大を示した。OS RSSやシステム最大メモリではなく、メモリ保持時間や一般的な16GB安全性の証明ではない。
- システムsnapshotは開始時free60%・swap使用6871.69MiB、終了時free45%・swap使用9038.56MiB。途中最大値は未取得、他作業は制御しておらず、増分をCに帰属しない。
- C01-B/Cは固定グループのtype=fix、最終Validator受理。ただし要約は両方「静的アセットのインポートをURL文字列に正規化するヘルパーを導入」で、参照が求める既存404/表示不具合修正の目的が欠落した。#178の履歴あり要約には404修正が含まれていた。
- C02-B/Cは固定グループのtype=featで参照と一致。要約は両方 `Phase1: File-first structural validation of HEAD`、scope=gitstate。長さは契約内だが日本語指定を満たさず、最終Validatorはinvalid_summary_languageで拒否した。既知のscope=docs拒否は今回の直接の不合格原因ではない。
- Goldは既存のagent作成の探索的参照を保持した。文章の意味もagentによるrubric照合で、独立した人のGold判定ではない。
- Dの実token化は両ケースとも先頭4tokens `[2,105,9731,107]` のみ一致。5番目はCategory=98、Text=10896。共通system本文・Raw Diff・履歴より前で分岐しており、Dはnot_evaluated_prefix_gate_failed。KV実装やD生成での救済は行っていない。token監査rawと監査時scriptを保存した。初版scriptの256token閾値を含むreceiptは変更せず保持し、今回の判定は閾値ではなく実際の先頭5番目の分岐に基づく。
- 完了監査で固定175source・helper/Metal/binary/overlay・Gold/captureの変更なし、両fixtureのHEAD/index/対象ファイルbytes不変、生成要求12件を確認した。
- Go全test/vet/build、Swift protocol10＋bounded generation27tests成功。過去ソース複製がGo探索対象になった初回failureは内容を保持して保存先を_source-snapshotへ移動し、修正後の全体成功logと両方保存した。

## 考察

モデル保持は成立し、2回分の再ロードを省く効果を確認できた。C01のload合計は8.82→3.04秒、C02は8.87→2.86秒だった。ところがC01では生成時間が約3.37秒増え、全体差は1.88秒に縮んだ。C02ではprompt時間も約8.33秒短くなっており、全体12.94秒の差を再ロード削減だけの因果効果と呼べない。固定順序n=1の観測値であり、平均や信頼区間・一般的な短縮率は算出しない。

Dの本来の対象だった入力処理はC02で大きいが、同じJSONを含むだけでは、その前にあるthinking markerの相違を越えてKVを共有できない。4tokensだけ再利用する実装を追加してRaw Diffの再計算削減と呼ぶ意味はない。今回はthinking/templateの新条件を追加しないという停止条件に従った。

B/C間で全最終JSONが一致した点は、この2fixtureではモデル保持による出力変化が観測されなかった根拠になる。ただしBの配置はC01の目的表現とC02の言語契約に課題があり、Bを経由したCの品質をそのまま採用基準にはできない。KVの成立利得がない現状で、配置変更Bを採用する根拠もない。

Aは#178の履歴あり結果を再実行せず参照した。Aは120秒制限、今回B/Cは無制限で、C02には固定partition注入もある。Aの未取得Textや未取得時間内訳を補完せず、A→B/Cの総時間改善率や完全な品質差を主張しない。今回C01は現行120秒以内だったが、C02は約253〜266秒かかり、製品の120秒予算を満たす結果にはならなかった。

## Next Steps

- 今回の事前登録4cycleは終了する。候補全体を採用せず、productionのprompt・timeout・helper起動方式は維持する。今回追加したモデル保持は明示的な検証専用経路のままとする。
- 次の比較を行う場合は「現行promptのままモデル/tokenizerだけ保持」を独立条件として事前登録する。Dが不成立なら共通prefix配置へ変更する理由がなく、Cの純粋な有用性を現行の品質を維持して確かめるため。今回の追加生成は行わない。
- Dを再開する場合は、thinking/template契約を変更してよいかを先に決め、新条件の品質比較から再登録する。設定を黙って統一しない。
- #178のGrouping/言語/目的表現と既知scope規則の問題は別に保持する。固定グループを与えた今回のtype一致を、Groupingや全体意味品質の解決とは扱わない。
- #180は未完了として維持する。KV再利用の成立・prefill削減・候補の採用品質は満たしておらず、全checkboxを完了扱いにしない。実装と全失敗を含むraw結果・監査・報告をcommit/pushして引き継ぐ。
