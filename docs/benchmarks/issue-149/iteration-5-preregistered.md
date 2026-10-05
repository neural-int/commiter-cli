# Iteration 5 事前登録: observed Go calls

H4はH3のliteral before/after・既存soft relationに、Go ASTからのbefore/after call factsを追加する。fixture testsの実moduleであるfixtureだけをresolveし、import aliasと一意の選択対象FuncDeclへbindする。未解決/曖昧ならedgeを追加せず、fileを除外しない。local groupやhard unionを作らない。model/profile/context/sampling/global instructionはH3と同じ。

まず同じcontract6/12を各1回。両方exact/FM0/FS0/completeが成立しなければcandidateとせず、missing semantic contractの観測責務へ進む。両方成立なら4file/8/9/16/独立評価/順序/共通callee独立guardrailとmetadata/Validateを評価する。これはGo fixtureの構文prototypeでありproductionの汎用module解決、type checkingや精度保証ではない。
