## 要約

#155の最初のモデル診断として、根拠付きattributionとdirect groupingを同じ観測入力で比較する。使用済みFee/Wrap、shared-test、same-call診断の3件と、開発用testなし1件。全4件×2方式=最大8call。独立holdoutではなく、成功しても能力GOやproduction GOにしない。独立holdout、cross-boundary、6file以上の評価は別途必要。

## 検証結果

推論前にschema、prompt、payload、評価gold、source manifest、モデル/helper pin、gateをiteration-3-diagnostic-preflight.jsonへ固定した。モデルへのpayloadにはfixture名、gold、要求注釈を含まない。goldを読まないinline atom抽出・context選択を共通に使う。

attributionの出力はrole・unit_ids・observed_contract_ids・evidence_refs・固定reason。全unitを重複なく一度割り当てる。行の集合はrole/referenceを圧縮するだけで、コミットgroupではない。観測はafterの具体的test失敗条件で、変更された関数contextに含まれるもの。before/after sourceを比較して、その観測が実装変更やassertion編集で実際に変わるか判断する。変わらない観測やdiagnostic-only変更は当該観測に対するindependent。対応する観測がなければunknown。

hostは全unit ID coverage、重複、未知ID、source context参照、test条件のsource位置、実装/test roleの位置対応、role/reasonの整合を検査する。これは意味的な因果関係をhostで証明するものではなく、意味的な誤対応は別のgold評価で計測する。unknownはhost-validになり得るが、positive/negativeの正解として代替しない。

対照は同じpayloadのdirect grouping。attributionは最終partitionを生成しないため、relation/roleの指標と対照のatom pair FM/FSを別々に記録し、異なるtaskのexact率を同一指標として比較しない。

local Qwen3-8B pin `545dc4251c05440727734bcd94334791f6ab0192`、neutral helper SHA `bc61461f860957780394650f8b46fab2b737f9931917d0194a0f736e671b41a4`、profile bounded-routed-grammar-neutral、context16384、output1536、native thought0、temperature0/topP1/topK0/seed144。最大128unit、message32768bytes、120s/call・960s/全体。retry/repair0。case1/3 attribution→direct、case2/4 direct→attribution。予算・条件・goldを結果に応じて変更しない。

モデルを動かさないtokenizer/grammar監査では、実schemaの合法16出力を全件許可、不正role/IDを含む8出力を全件拒否、合法16出力すべてでwhitespace等のlogit penalty0、全promptにoutput1536を加えてcontext16384内に収まることを確認した。モデルweightsは未ロード。既存tmp helperのtestを追加して実行したが、計測に使うrelease binaryのSHAは不変。

## 考察

判断責務を役割と根拠参照へ変更した候補の最初の診断であり、棄却済み整数スコアのprompt調整ではない。全inline atomを保持し、同一file/test内で別の観測に対応する編集も別unitとして入力する。partial output、unresolved、host rejectはquality=nullで保存する。

診断gateはattribution4件のhost-valid、全既知role/contractラベルの一致、positive/negative relationの誤判定0、観測なしで明示的unknown。独立holdoutがないため、このgateを満たしてもBOUNDED ATTRIBUTION GOではなく、独立評価へ進む根拠に限定する。失敗した場合はこの固定candidateを不採用とし、根拠なしのwording/model/閾値sweepへ進まない。

## Next Steps

- 固定8call以内で計測し、全raw completed出力・host拒否理由・quality/costを保存する。
- 結果を親子Issueへ報告し、独立評価へ進めるか、契約/判断責務にどの失敗が残るか判定する。
- 公開repositoryの独立候補を引き続き監査する。現在の入力準備とこの診断は独立holdoutの代替にはしない。
