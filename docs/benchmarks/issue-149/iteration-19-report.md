## 要約

H12として4snapshot観測値をsoft evidenceへ追加した。weak16とshared-callee independent guardrailは一致したがcross12はFS24で、従来の6pair分割が残った。observerの情報欠落ではなく、追加された値を含むglobal判断でも採用条件を満たさなかった。

## 検証結果

H10の全file/contract/test/caller情報、global prompt/schema/canonicalization/host gateを維持。観測testのliteral引数で副作用なしobserverを評価し、値とknownを入力へ追加した。fixture名/gold/期待groupをpayloadへ渡さず、値をhard unionにしない。

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 | observer probes/samples/unknown | observer wall秒 |
|---|---|---:|---:|---|---|---:|---:|---:|---:|---|---:|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 1 | 2894 | 192 | 15.256 | 0/0/0 | 0.000081 |
| contract-cross-boundary-12 | false | 0 | 24 | true | false | 1 | 5383 | 176 | 19.368 | 4/16/0 | 0.000480 |
| shared-callee-independent-6 | true | 0 | 0 | true | false | 1 | 2454 | 92 | 10.585 | 2/8/0 | 0.000158 |

全call completed、timeout/context overflow/不完全出力なし。各1回、同時推論なし。固定Gemma 475b9088d29754a3379866cf5aeb6b41acd313c2、native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。observerはstep1024/depth16/integer絶対値2^26/string4096bytes/probe32上限。

observer原始値・既存record/anchor focused testsに加え、入力への変換後も24oracle値が一致し、unknownが値へ変換されず、キャンセルで停止することを検証。benchmark build成功。Go codeの外部実行0。資格不成立のためfresh protocol8、metadata/Validateの追加測定は未実行。production変更なし。

## 考察

cross12の16samplesは全てknownで、観測不足/unknownによるfail-closedではない。元のH10 cross12 input4814に対し5383へ増えたが、FM0/FS24と6pair partitionは同じ。callee/callerの相互作用の数値を追加しただけでは採用根拠を得なかった。guardrailでは共通calleeによる誤結合を観測しなかったが、これをcross12成功と混同しない。

今回globalには4値をそのまま渡しており、個別変更と同時変更の効果を比較する作業もglobal semantic判断へ残る。有限の観測値間の等否はhostで決定的に計算できる。一方、それが同一purpose/commitであるかはhostで決定できない。両者の責務を分けるrepresentationを検証する余地が残るが、効果はまだ未立証。

## Next Steps

- H13として、4snapshot値の等否から観測入力に限った変更効果contrastをhostで導出する。global taskへ残る有限値の比較をdeterministic logicへ移し、意味的purpose判断と分離するため。
- 元の4値を保持し、contrastをjoint-only/caller-only/callee-only/no-effect/other/unknownの観測事実として添える。目的・境界の証明やhard unionを作らず、未対応値を推測しないため。
- 条件を事前登録し、weak16/cross12/shared-callee guardrailを各1回測定する。system prompt/seed/orderの局所tuningではなく、観測比較の責務分割だけを評価し、全資格通過時のみ未測定protocol8へ進むため。
