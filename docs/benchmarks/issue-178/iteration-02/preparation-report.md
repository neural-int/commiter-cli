## 要約

Textへのgrammar接続候補のrelease buildとMetal smokeが成功したため、実モデル比較条件を固定した。実行前の新規推論は0回。元の2ケースと探索的Gold、prompt、schema、出力予算を維持する。

## 検証結果

- Goの全test/vet/build、Swift全27test、release build（214.09秒）が成功した。初回の明示的Xcode build-systemでのターゲット解決失敗を残し、オプションを外した同じ固定依存で成功した。
- release helper SHA256: 1f68284a462d48449353a5d3679b8a89086b2d504d4df6ba5be72771b069d12f。Metalは前回と同じSHA256で、分離したruntimeでもmetal_okを確認した。
- execution-contract.jsonに全関連ソース、実helper、Metal、入力capture、Gold、評価用360秒overlayのハッシュを固定した。fixtureのHEAD/index/bytesは前回の不変状態と一致した。

## 考察

今回の処置はhelperの再ビルドとTextへのgrammar接続であり、promptを変更しない。長さ制約の適用、最終Validator、意味、phase時間を分けて確認する。scope=docsの禁止や意味の正しさはmaxLengthでは表現されず、不合格が残る可能性がある。前回helperのcompiler versionは記録されていないため、速度差をgrammarだけの因果効果として断定しない。

## Next Steps

- 固定順C01-A/B、C02-A/Bを各1回、最大12要求、共有360秒/cycle、runner25分上限、retry/repair/追加judgeなしで実行する。C02 Textまで到達する条件で処置を評価するため。
- 失敗したcycleは保存して次の事前登録cycleだけへ進み、4cycle後に報告する。追加の条件変更を未使用枠の再試行として実施しないため。
- 本番の120秒は変更せず、意味不一致・scope禁止・文字数・最終planを独立に記録する。評価用期限延長や単体テストをproduction採用の証明にしないため。
