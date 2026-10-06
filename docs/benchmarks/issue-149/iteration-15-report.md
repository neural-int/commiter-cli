## 要約

H10 contract単位の観測recordを検証した。weak16は一致したが、cross12は実装+testの6pairへ分割しFS24。H9 Gemmaと同じsemantic failureが残り、採用条件は満たさない。

## 検証結果

flat観測一覧を、既存の一意callee file/symbolで対応testとcallerを配置したrecordへ変更した。未対応assertion/callは別項目へ保持。全selected file/evidenceを入力に残し、参照をhard groupingにしない。system prompt/schema/canonicalization/host gate/modelはH9 Gemmaから維持。goldはmodel入力へ渡さない。

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 | stop |
|---|---|---:|---:|---|---|---:|---:|---:|---:|---|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 1 | 2849 | 161 | 14.314 | completed |
| contract-cross-boundary-12 | false | 0 | 24 | true | false | 1 | 4814 | 176 | 18.304 | completed |

各1回、同時推論なし。固定Gemma 475b9088d29754a3379866cf5aeb6b41acd313c2、native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。timeout/context overflow/不完全出力を観測しなかった。cross12は6組のsource/test pair。fresh protocol8とmetadataの追加試験は未実行。

対応のあるanchor/assertionの既存focused tests、観測の欠落・重複・捏造を検知するrecord conservation test、benchmark build成功。production codeの変更なし。

## 考察

H9 Gemmaはweak16 exact/cross12 FS24で、H10も同じpartitionだった。既存call/test情報の関連配置だけではsemantic統合の改善根拠を得られなかった。入力tokensはweak16で2640→2849、cross12で4657→4814へ増えた。wallの小差から性能改善を主張しない。

H1のraw per-file抽出はcross12 FS24、H5 assertion IRはforwardのみcross12一致、H8はglobal contract出力のdraftでもFS24だった。各案は、raw抽出時の根拠不足または一つのglobal taskで意味抽象化とassignmentを同時に行う構成だった。観測contract+対応testを根拠として意味deltaだけを抽出する段階と、全体のassignmentを別段階へ分離する仮説は未検証。効果は未立証であり、今回の失敗だけでは探索余地の消尽としない。

## Next Steps

- H11として、最大4観測recordから根拠E-ID付きの変更前後behaviorを抽出し、groupingを行わない抽出段階とglobal assignment段階を分ける。意味抽象化とfile割り当ての同時判断が残るため、責務境界を変えて検証する。
- 抽出IRのE-ID全件一致、未知・重複・欠落拒否をhostで検証し、元の観測と全selected fileをglobalへ残す。LLM抽出や局所batchを不可逆なcommit boundaryにしないため。
- 条件を事前登録してweak16/cross12を各1回測定し、双方資格通過時のみ未推論protocol8へ進む。fixture固有tuningや既否定構成の無根拠な再試行を避け、追加stageの実calls/tokens/wallを独立に評価する。
