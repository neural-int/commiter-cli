# Issue #142: ranking と verifier の検証条件

## 目的と固定条件

[考察コメント](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5881996734)を修正後、モデル推論前に本書と benchmark code をコミットする。固定 Louvain generator（Gonum v0.16.0、seed `142,1`、resolution `[0.25, 0.5, 1, 2, 4]`）、前回の prepared repository input と forced-choice ranking contract、MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を維持する。新たなモデル取得や依存追加はしない。

新規の合成 fixture は `verify_join_present`、`verify_split_present`、`verify_join_absent`、`verify_split_absent`、`verify_misleading_relation` の5件とし、gold grouping はコードに固定する。名称の `present` / `absent` は設計時の想定であり、実際の candidate recall は generator を動かしてから記録する。想定と異なる場合も fixture や generator を後から調整しない。gold label と fixture 名はモデルへ渡さない。

## 測定手順

1. 各 fixture の generator 出力から候補数と gold 包含を記録する。候補数が2以外なら、その条件は二択対象外として中断し、理由を報告する。
2. 各 fixture 2反復で、前回と同じ forced-choice ranking を行う。1回目は元の候補順、2回目は提示順のみ反転し、候補IDと grouping の対応を維持する。ranking は2048 output tokens、2分上限、repair/retry 0。
3. 有効な候補IDを返した場合だけ、その選択候補と repository input を別の verifier call に渡す。別候補、gold、ランキング結果の正誤は渡さない。verifier は JSON Schema enum `accept` / `reject` の一語判断とし、2048 output tokens、2分上限、repair/retry 0。`accept` は選択候補を最終候補とし、`reject` は `none` とする。未完了・不正応答は判定不能とし、`reject` と数えない。
4. gold 候補が含まれる fixture では1反復目に限り、同じ verifier contract に gold 候補を直接与える対照 call を1件追加する。これは verifier の gold 誤 reject を selector の正誤から独立に観測するためであり、通常の最終出力には使わない。選択候補と同じでも別 call として記録する。

結果は candidate recall（fixture 分母）、ranking の合成 gold 一致（全試行・有効候補応答分母）、選択候補が gold だった場合の verifier 判定、非 gold 候補の verifier 判定、gold 対照の判定、最終候補の合成 gold 一致、stop reason、failure、calls、wall、prompt/schema hash、output token telemetry に分ける。正解候補がない条件での `none` は最終 exact grouping とは数えず、reject として記録する。各段階の分母0は率を算出しない。

既使用 fixture や前回の非同時測定との数値比較は記述的に扱う。5件の合成 gold と少数反復から、実際の開発者判断、production 発生率、`none` 分離の因果効果、採用可能な精度を推定しない。結果で contract を変更せず、end-to-end の Pass 2 や通常 CLI は実行しない。生の prompt、response、repository content は保存しない。
