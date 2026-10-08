## 固定前提監査

iteration21の全negativeを意味能力の不足へ帰属する前に、現bounded-grammar契約がpositive scoreを構造的に禁止していないこと、policyがtokenizer chat templateで保持されることを確認する。新規C専用worktree issue-153-score-wire-auditでzero-model監査を実施する。旧iteration3のpositive controlは別generation profileなので、現profileの可到達性の代替証明にしない。

固定iteration21の4schema、実使用Qwen tokenizer/vocab、現在helperと同一GrammarSamplingStateソースを使う。各schemaでuniform -2/-1/0/1/2とmixed +/-2・unresolved=trueの6合法応答、範囲外-3/3の2不合法応答をtoken単位にreplay。計24合法/8不合法。各実promptをenable_thinking=falseでtokenizeし、全文保持と既存input token数を比較する。学習weightロード/モデル生成0、依存変更/追加ダウンロード0。

testはtools/benchmark153/ScoreWireGrammarTests.swiftを既存/tmp helperのtest targetへコピーし、swift test --configuration release --skip-update --disable-automatic-resolution --filter ScoreWireGrammarTestsで実行。production source/manifestを変更しない。helper binaryのSHAを実行前後確認する。helperの構造ソースとworktree sourceのSHA一致も確認。失敗時は再現場所と原因を特定し、semantic結果の解釈を保留する。全通過はlegal outputの可到達性・tokenizer表示保持の証拠であり、モデルがその値を選べる意味能力、実logit経路の確率、全制約の無偏向を証明しない。

結果を見てscore enum/schema/promptを調整しない。all-negative全体の原因をこの監査だけで特定しない。正式C/B/production条件、過去No-Go/4file制限/Goal未達を保持。現model scorerで独立positive/negative能力の新根拠がない場合、追加モデル再計測を行わない。
