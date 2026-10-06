## 要約

H9 observed-anchorsの入力・schema・host gateを維持し、groupingのみ固定Qwen3-8B-4bitへ変更した。weak16は一致したが、cross12は全ファイルを単独groupへ分割しFS30。production candidateの資格不成立で、未測定holdoutへは進まない。

## 検証結果

modelはmlx-community/Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192。ユーザー承認後に9ファイル・4,623,782,544bytesを取得し、digest検証とReady成功を記録。初回取得は約45分でincomplete fileとして失敗。取得専用timeoutを2時間へ変更した再取得は6006.445秒で成功した。推論budgetは変更していない。

experimental helperのrelease buildと24テスト、benchmark buildは成功。production source/defaultは変更していない。

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 | stop |
|---|---|---:|---:|---|---|---:|---:|---:|---:|---|
| weak-edges-independent-16 | true | 0 | 0 | true | false | 1 | 2591 | 210 | 32.312 | completed |
| contract-cross-boundary-12 | false | 0 | 30 | true | false | 1 | 4232 | 158 | 37.522 | completed |

各1回、同時推論なし。native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。両方ともtimeout/context overflow/不完全出力を観測しなかった。cross12の最終partitionは12単独group。goldの2groupと一致しない。資格不成立のためfresh protocol8、metadata、最終Validateの追加試験は実行していない。

## 考察

checkpoint変更だけでcross-boundary contractを回復する根拠は得られなかった。GemmaのH9 cross12 FS24に対し、今回FS30。weak16成功やcomplete assignmentをsemantic成功と混同できない。パラメータ数だけでは採用を支持しない。

H5のtest contract IRはforward cross12一致、reverse FS24、H6 canonical IRはFM36、H9 anchor assignmentはFS24/30だった。観測情報にsource/testが存在しても、file一覧からcontractと対応testの関連をglobal modelに再構成させる表現が残る。これが原因かは未立証。contractを単位とした入力representationを検証する余地は残っており、探索完了とは判定しない。

## Next Steps

- H10としてhostの観測contract recordに、対応するtestの引数とbefore/after期待条件を参照として配置するrepresentationを事前登録する。file単位の並び替えとは異なり、global modelが関連を再構成する負担を減らす仮説を検証するため。
- ASTで一意に対応を観測できる参照だけを関連付け、曖昧・未知は明示する。参照をhard unionにせず、全selected IDのglobal判断とcomplete/fail-closed gateを維持するため。
- 新representationでweak16/cross12を固定条件で各1回測定し、双方成功した場合のみ未測定protocol8へ進む。goldやfixture名に依存せず、反復tuningを避けて改善根拠を得るため。
