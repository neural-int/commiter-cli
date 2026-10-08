## 要約

モデル生成0の前提監査は24合法replay許可・8不合法拒否・8prompt全文保持/token数一致で通過。positive scoreが構造的に禁止される仮説は支持されない。ただし追加の機械的traceで、現在の空白penalty -200が、Qwenの空白付き0/+1/+2に作用し、空白付き負値には作用しない非対称を確認した。次の合理的仮説は、signed scorerの空白biasだけを取り除く対照。今回のtraceだけでiteration21の全negative原因を確定せず、意味品質改善/正式C GOにしない。

## 検証結果

新規C専用worktree issue-153-score-wire-auditで条件とtestを0e697a18へ固定。iteration21の実4schema/8prompt、実Qwen tokenizer（vocab151669、EOS151645）、現GrammarSamplingStateを使用。各schemaの-2/-1/0/1/2均一値とmixed +/-2/unresolved=true計24合法JSONをtoken単位にreplayし、全token mask有限かつEOS終端成功。範囲外-3/3計8JSONは拒否。元/追加のsystemとuser全文がchat template表示に保持され、各token数がiteration21のhelper実測値に一致。

旧positive controlは別profileなので今回へ転用していない。weightロード/モデル生成0、追加package/version/download0。GrammarSampling.swift/TokenBudget.swiftのworktreeと現helperソースのSHA一致。実helper binaryは監査前後とも3b50561e987fb166a794bbe7d6a6dd8aca00514a3619a391a523f001f8c64f80。production source/manifest変更なし。

初回はyyjson module参照不足、次はtools引数不足でtest本体未到達。既存product参照のtest配線とtools:nilを修正。次のruntimeはMetal資源探索不足で停止し、既存default.metallibをtest Mach-O隣接へ配置して同条件成功。setupの各失敗はgrammar/モデルの不合格として数えない。配線と詳細はiteration-22-setup.md、patchはscore-wire-test-wiring.patchに保持。最終trace付selected Swift testは1test/1suite、9.245秒で成功。

合法到達性が分かった後の追加trace（事後機械的診断、独立semantic評価ではない）:

| 空白付き値 | Qwen token pieces | 空白penaltyの対象token数 |
| --- | --- | ---: |
| -2 | Ġ- / 2 | 0 |
| -1 | Ġ- / 1 | 0 |
| 0 | Ġ / 0 | 1 |
| 1 | Ġ / 1 | 1 |
| 2 | Ġ / 2 | 1 |

token IDsは-2=[481,17]、-1=[481,16]、0=[220,15]、1=[220,16]、2=[220,17]。WhitespaceTokenBiasは純空白に-200、GrammarSamplingStateは許可tokenのmaskへその値を加える。schemaの意味ラベルは読まない。既存6unit/15pairの通常空白付きJSON replayは-2/-1がpenalty0token、0/1/2が各15token。実maskで確認した。mixed traceも結果JSONへ保持。

## 考察

構造上positiveは到達可能で、policy全文もtemplate段階で消えていない。一方、通常の「colonの後に空白を出す」形式では、正値と0に入る際の空白が抑制され、負値は空白とminusが結合したtokenにより抑制を回避する。分割の意味とは独立したtokenization差がscore選択へ作用し得る根拠となる。

これを実modelのscore変化の証明にしない。空白なしpositiveは別経路で許可されること、モデルlogits/選択の因果効果は未計測であることを保持する。replayはteacher-forcedの合法性確認であり、モデルの理解や選択確率、全decoderの無偏向を保証しない。全negativeをモデル能力不足だけへ帰属する結論も保留する。

iteration21のpolicy No-Goと全過去結果を維持する。新たに観測したnonsemanticな符号非対称が、generation contractとsemantic scorerの責務を分ける次比較の根拠になる。元promptを固定して空白biasだけを変えれば、wording sweepや新しいgold由来constraintを追加せず、この機序の作用を調べられる。JSON grammar/EOS/context/output/fail-closedを保持して有限penaltyのみ比較する。

## Next Steps

- 新規C専用worktreeで、signed scorerの空白bias -200対0の単一比較を事前固定する。符号と無関係な出力形式の非対称がスコアを変えるか測るため。
- iteration21の元system/payload/schema/model/profileの他条件を保持し、使用済み4caseを因果診断として一回ずつ比較する。fresh generalizationに数えず、gate/条件を結果依存で変更しない。
- semantic改善が得られた場合だけ新規未使用データと全段/旧回帰へ進む。biasが存在する事実と、品質改善の実測を分けるため。
- A bounded GO/B No-Go/C正式条件未達/D未正当化/production4file/Goal未達を保持する。構造監査の成功をproduction採用の代替証拠にしないため。
