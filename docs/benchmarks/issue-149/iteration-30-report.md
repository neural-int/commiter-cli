## 要約

H21は全観測を同一global contextへ保持して全pairを一度に判断。最大120pairは予算内で生成できたが、weak16 FM120、cross12未解決、guardrail不正JSONで採用不成立。

## 検証結果

| fixture | exact | FM | FS | complete | unresolved | host結果 | input tokens | output tokens | total wall秒 |
|---|---|---:|---:|---|---|---|---:|---:|---:|
| weak-edges-independent-16 | False | 120 | 0 | True | False | completed partition | 6522 | 1080 | 56.013 |
| contract-cross-boundary-12 | None | None | None | False | True | unresolved_pair | 6607 | 734 | 42.319 |
| shared-callee-independent-6 | None | None | None | False | True | invalid_json | 2541 | 225 | 14.978 |

各case1call/1回。全backend stop completed、timeout/context overflow/incomplete backend outputなし。cross12とguardrailはgroupsなしで停止、exact/FM/FS nullを0成功として数えない。weak16は全16file1group。120pair出力1080tokensは固定1536内。cross12はU判断を含む。guardrailはstrict JSON検証失敗、生成内容を保存していないため詳細JSON不備は未特定。

fixed Gemma revision475b9088d29754a3379866cf5aeb6b41acd313c2、native0/output1536/context16K/call120秒/whole600秒、temp0/top_p1/top_k0/seed144/retry0/repair0。hostは全pair coverage、unknown/invalid/U、S推移closureとD矛盾を拒否。soft syntaxをmust-linkにしない。候補partition集合なし、local window/bridge/correctionなし。

focused安全gate tests、`go test ./tools/benchmark149`（42.529秒）、`go vet ./tools/benchmark149`、`go build -o /tmp/benchmark149-globalpairs ./tools/benchmark149`、diff check成功。weak16推論とbenchmark testsが一部並行したためwallは隔離latency比較に使わない。raw prompt/思考/生成pair内容を保存せず、model内部理由は推測しない。追加取得/依存/production変更なし。

## 考察

#146とは異なる同一global contextでも、整合した誤結合と未解決が残る。pair出力へ分けるだけで意味的正解を保証できない。候補欠落は避けられるが、構造/整合gateは正解の代替にならない。予算内で実行できたことはproduction実用性の証明ではない。

Gemmaは複数taskでweak16を誤結合する一方、Qwen8はE-root/直接G-IDでweak16を分離し、candidate comparisonでcross12/guardrail一致の観測がある。現在の同一global pair taskはQwen8未評価。候補制限なしの境界判断という責務差を持つ組合せを一度確認する根拠はあるが、120pairで同じcall予算を完了できるかは不明。予算超過を増額/再試行で回復しない。

## Next Steps

- H22としてH21のglobal input/pair schema/gatesを固定して、取得済みQwen3-8B native0でweak16/cross12/guardrailを各1回測る。独立分離とcross12候補比較の改善観測があるcheckpointで、候補制限なしのglobal boundary taskを独立評価するため。
- 1536token/120秒call/600秒whole、retry/repair0を維持し、timeout/invalid/U/contradictionを拒否する。予算sweep・case別routing・未解決recoveryを避けるため。
- 全資格通過時のみ新独立評価を事前固定。失敗ならH21/H22を監査へ追加し、global/local責務・observations・generation/task/modelの残余と具体的再開prerequisiteを再評価する。有限probeを普遍的不可能性とせず、既存失敗の別名反復を防ぐため。
