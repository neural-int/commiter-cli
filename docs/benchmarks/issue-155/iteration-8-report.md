## 要約
Iteration 8のscope監査で、全戻り値のhost計算は#155の必須条件ではないことを確認した。過去の29unknownは値計算候補のcoverage不足であり、assertion根拠の取得不能を意味しない。既存6自然履歴・18table行をASTで調べ、13行では同じ関数内の失敗条件sourceを取得、5行ではvalidator本体がsnapshotにない。これはsource根拠の存在確認であり、意味attribution GOではない。

## 検証結果
- 新専用worktree/branch: codex/issue-155-observation-scope-audit。
- 現#155スコープは「変更前後のassertion/behavior、関数・symbol・span」を入力とする。全natural callの値を限定interpreterで再現する条件はIssue gateにない。
- 実装・gold・既存計測は変更せず、使用済み6履歴の18after table行をGo標準ASTでsource関数に対応付けた。全行で唯一の囲む関数を確認。
- ParseBytes整数精度4行とstrcase9行は同じ関数にErrorf/Fatal selectorを含むif条件がある。table行と条件のsource byte span/textを照合した。
- go-humanizeの残り5行はtestList.validateに委譲し、取得済みsnapshotにはvalidator本体がない。条件取得不能として保持する。
- 失敗条件selectorの存在はtesting receiverの証明でも、table列とpredicateのdata-flow bindingの証明でもない。13行もsource_evidence_only_binding_unverifiedで保持。causal_attribution/behavior_value/semantic_goldはnull。
- コメント内の偽if/Errorfを無視し、日本語を含むsourceのUTF-8 spanが正しいことをテストで確認。Python test1件とGo vet/build pass。モデルcall0、任意source実行0、新依存0、production変更0。

## 考察
前Iteration 7の「現host-fact候補終了」は保持する。一方、その候補が必要とした全値の確定を#155全体の再開前提へ一般化した部分は過剰だった。元Issueはsource-grounded assertionも認める。値計算coverageのために汎用Go interpreterや実行sandboxを開発する必要性は現時点で証明されていない。

Aで必要な最小拡張は、table行の入力・期待値がloop内のcall/result/predicateへどのように対応するかをsourceで検証する範囲である。検証済みassertionをLLMの根拠にすることと、実装がそのassertionを実際に成立させると主張することは別であり、実装attributionは未解決。単なるcalleeや同じtestの一致をpositiveへ昇格させない。

## Next Steps
- モデルなしでtable→loop→call→failure predicateのsource bindingを固定範囲で検証する。影響のある変数再代入、共有条件、receiver未解決はreject/unknownにし、実コードの実行や値再計算をしない。
- validator本体がsnapshotにない5行は必要な公開sourceを追加で取得してclosureを監査する。取得不能を容易な例への置換で回避しない。
- 実装の意味対応と独立negative/cross-directory positiveは別gateとして残す。source binding単独で#155 GO、B/C開始、旧#150完了としない。過去の14モデルcallの失敗と29unknownは保持する。
