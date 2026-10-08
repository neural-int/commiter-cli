## 要約
Iteration 15で新しい6自然table行に必要なsnake実装sourceを補完し、具体観測の効果参照案を3changed/3unchangedとして整理した。これはsourceを読んだ参照案で、repository実行による実測でも、作者の独立変更目的goldでもない。状態前提と未達条件を明記し、formal GOは判定しない。モデルcall0。

## 検証結果
- 新専用worktree/branch: codex/issue-155-effect-gold-audit。
- 既存snapshotを保持し、d7993fd3/692d1b89のcommit/parentでsnake.goを取得。両履歴でbefore/afterが完全一致。
- ToSnake→ToDelimited→ToScreamingDelimitedのplain package callを既存AST observerで確認。変更されたCamel処理へのplain callはない。callee source refsとbyte span/textを保存。
- d799: CONSTANT_CASEのToCamel参照はCONSTANTCASE→ConstantCase、ToLowerCamelはcONSTANTCASE→constantCase。反復大文字を小文字化するprevIsCap/hasAcronym branchが差分根拠。
- 692d: ToLowerCamelのsome stringはsomeString→someString、先頭空白ありはSomeString→someString。追加TrimSpaceと既存capNext/byte loopが根拠。
- 692dのToSnakeの2入力はsome_string→some_stringの参照案。実装とcallee sourceが同一で、Camel編集はその呼び出しpath外。
- Camel参照には初期acronym mapがID:idで外部ConfigureAcronym変更なし、before/afterで同じ初期状態という前提が必要。実際のtest suite実行時のglobal状態は観測していない。
- 参照afterが自然tableの期待値literalと一致すること、source位置/hashを確認。runtime_measurementとwhole_intent_goldはnull、source_reasoned_reference_not_runtime_measuredを明示。
- source再監査、固定binding回帰Python9tests、diff check pass。モデルcall0、任意repository実行0、依存追加0、production変更0。

## 考察
新しい自然tableには変化する観測と変化しない観測の両方があり、同じcalleeという情報だけでは効果を識別できない。初期stateとsourceが明示された範囲の効果参照を評価対象にできる可能性があるが、この6行を独立変更目的のpositive/negativeへそのまま置き換えられない。

表のexpectedはafterテストが要求する値であり、その値が実行で観測されたことは証明しない。sourceの参照案にも独立照合が必要。before結果を実測値と報告せず、状態が未確定ならunknownに保持する。

正式な#155能力gateにはwhole-intentの独立negativeとcross-boundary positive、unknown、事前固定された受入閾値/モデル入力/予算が不足する。source closure補完と参照案だけで#155完了を宣言しない。

## Next Steps
- 6行の参照案を状態前提付きで独立に照合し、具体観測の効果goldとして使用可能な範囲を確定する。source参照案を実測へ昇格しないため。
- source/初期state/根拠と、モデルが答える効果仮説を分離した入力契約を固定する。gold/参照結果/fixture名をモデルへ渡さない。
- 独立変更目的negative、cross-boundary positive、明示的比較のないunknownを含む評価入力を確保する。これらが不足したまま6行の効果taskだけをformal GOとしない。
- データ/gold/gate/budget/baseline/native wireが成立した場合だけ候補を事前登録し、B/C・production・旧#150達成は未達のまま保持。
