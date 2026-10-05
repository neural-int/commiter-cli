## 要約

Go ASTで観測したcaller/calleeを追加したH4も、12fileでFS16が残ったため採用しない。6fileの独立目的は分離できた。次は依存関係に加え、testに書かれたinputとbefore/after期待条件をhostが観測契約IRへ変換するH5を検証する。

## 検証結果

| H4 fixture | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | ---: | --- | ---: |
| independent6 | true | 0 / 0 | true | false | 1 | 1707 / 574 | 29.789 |
| cross-boundary12 | false | 0 / 16 | true | false | 1 | 3995 / 634 | 37.084 |

全2calls completed、context overflow/timeout/incomplete0。unknown/duplicate/missing selected ID0。12fileはF001〜F004、F005/F006、F007/F008、F009〜F012へ分割した。H3のFS20より小さかったがraw-global/batchのFS8より大きく、採用条件を満たさない。相対差を一般的な改善へ読み替えない。

共有calleeがあっても独立したtrim/prefix/delimiter目的を持つ新たな6file guardrailを測定前に固定した。before/afterの合成Goプログラムは全てテスト成功。H4のcandidate資格不成立のため、このguardrailの成功をまだ主張しない。

## 考察

構文上の依存を正しく提示しても、同じ変更契約への所属はまだ成立しなかった。semantic目的をsoft edgeから決定的に導けない既存境界を維持する。rootとconsumerが同じ入力に対して同じ期待値変化を持つrounding契約はtestに存在するが、現在はraw codeの内部に残っている。H5ではそれを明示的な観測IRへ正規化する。モデル内部理由の説明ではなく、未検証のarchitecture仮説である。

## Next Steps

- Go testの直接Fatal分岐で表現されるcall/input/expected conditionをbefore/afterで抽出し、selected callee IDへ一意にbindする。goldのgroup labelを注入せず、観測された振舞い変更をglobal判断へ提示するため。
- 同じ6/12fileを各1回測り、資格が成立したら共通callee独立guardrail・holdout・順序・大規模・metadata/Validateへ進む。新しいIRの既使用fixtureだけの一致を採用根拠としないため。
- 非対応assertionや解析できない差分はraw factsを残し、fileを除外しない。test観測の欠落を意味的な独立や同一の証拠として使わないため。
