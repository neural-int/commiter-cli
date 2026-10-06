## 要約

H12の前提検証として、synthetic caller/calleeの4snapshotを実行し24観測を得た。cross12の4callerでは両方変更した場合だけ出力が変わり、shared-callee independent guardrailの2callerではcallee変更と無関係にcallerだけで出力が変わった。既存IRにない実行上の相互作用を観測できたが、grouping改善やproduction candidateの立証はまだ行っていない。

## 検証結果

既存call/test AST観測からsource caller/calleeとliteral引数を取得。4snapshotの他sourceはbeforeに固定。test期待値やgoldで呼出し入力・snapshot・出力を生成していない。repository定義の2synthetic fixtureのみをGoで実行し、model calls=0。24 go runのwall合計12.531秒、focused test全体12.62秒。全観測の実行は成功し、timeout/invalid JSONなし。

| fixture / 呼出し | 引数 | callee前/caller前 | callee後/caller前 | callee前/caller後 | callee後/caller後 |
|---|---|---|---|---|---|
| cross12 Authorized | 10,10 | true | true | true | false |
| cross12 Restore | 10,10 | true | true | true | false |
| cross12 FormattedCents | 1.999 | 199 | 199 | 199 | 200 |
| cross12 PersistedCents | 1.999 | 199 | 199 | 199 | 200 |
| guardrail trace.Label | "A" | "a" | "a" | "trace:a" | "trace:a" |
| guardrail render.Label | "A" | "a" | "a" | "[a]" | "[a]" |

実行数/各wall/各value/file IDsはiteration-17-runtime-interactions.jsonに保存。fresh protocol8、model推論、metadata、production変更は未実行。

## 考察

観測した引数に限れば、cross12の変更はcallerのafter出力に対してcallee/caller両方の変更が必要だった。guardrailはcallee変更が観測caller出力へ影響しなかった。共通callee参照の有無だけより区別できる情報が増えた。

有限の引数・2fixtureの観測を全入力への証明とは扱わない。jointな実行効果も、その変更を必ず同じcommitへ入れるべきことを証明しない。以前のpassing-intermediate auditとも矛盾しない。依存順で6pairを適用すればtestは通るが、今回の4snapshotはその異なる組合せの値を示している。

このGo実行probeはsynthetic fixtureの調査に限定する。任意の変更コードをplannerで実行するproduction方式として採用しない。追加情報を安全に得るには、閉じた副作用なしの式・関数だけを扱い、未知を未知として残すobserverの検証が必要。

## Next Steps

- H12のprototypeで、限定Go ASTの純粋な式・関数を副作用なしで観測する方法を事前登録する。任意のrepository codeの実行をproduction経路へ持ち込まず、今回得た追加情報を利用できるか確認するため。
- 未対応syntax、型、参照、再帰・評価budget超過をunknownにし、24実行値との一致を確認する。未知の値を推測せず、限定observerの対応範囲と正確性を確認するため。
- 一致した場合のみ、4snapshotの値をsoft runtime evidenceとしてglobal入力へ加え、weak16/cross12とshared-callee guardrailを固定条件で測定する。runtime evidenceをhard unionにせず、semantic品質を独立に評価するため。資格通過後のみ未測定holdoutへ進む。
