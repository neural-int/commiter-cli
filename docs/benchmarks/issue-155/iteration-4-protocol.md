# Iteration 4: local observation facts

## 要約
変更同士の関連付けではなく、具体的な呼び出しの変更前後の値とreturn箇所をLLMへ問い合わせ、hostの限定Goインタープリターで照合する仮説を固定する。

## 検証条件
既存diagnostic 3件、観測なし・欠落calleeの開発probe 2件、google/uuid.Compareの公開sourceと人工入力1件。計6呼び出し、retry/repairなし。モデル、helper、source hash、120秒/call、720秒totalはpreflightに固定。全件で値・型・実際のreturn根拠、または正しいunknownが一致した場合のみ局所fact bridge監査へ進む。

## 事前検証
wireの合法12回答を受理、不合法6回答を拒否。合法12回答のtoken penaltyは0、全prompt+1536出力が16384context以内。モデル呼び出し0。Go/Pythonの観測・host検証テスト通過。

## 注意点
#149の限定observerを再利用したもので、新しい意味推論算法の主張ではない。repository codeは実行せず、制限ASTを解釈する。公開sourceの人工入力は自然な変更目的のholdoutではない。成功しても#155全体のGO、コミット境界の品質、B/C、production採用には数えない。
