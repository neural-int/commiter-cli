## 要約

H18としてbounded provisional purpose discoveryとglobal assignmentを分離。全caseで2段階の出力は完成したが、weak16 FM120/cross12 FS30、guardrailのみ一致。採用しない。

## 検証結果

| fixture | exact | FM | FS | complete | unresolved | purpose候補数 | E-ID参照数 | total wall秒 |
|---|---|---:|---:|---|---|---:|---:|---:|
| weak-edges-independent-16 | false | 120 | 0 | true | false | 1 | 16 | 48.681 |
| contract-cross-boundary-12 | false | 0 | 30 | true | false | 3 | 12 | 103.196 |
| shared-callee-independent-6 | true | 0 | 0 | true | false | 3 | 6 | 56.424 |

| case | phase | input tokens | output tokens | wall秒 | stop |
|---|---|---:|---:|---:|---|
| weak16 | global-purpose-discovery | 1444 | 137 | 17.135 | completed |
| weak16 | global-purpose-assignment | 2984 | 210 | 31.545 | completed |
| cross12 | global-purpose-discovery | 4146 | 174 | 37.724 | completed |
| cross12 | global-purpose-assignment | 5095 | 158 | 65.469 | completed |
| guardrail6 | global-purpose-discovery | 2062 | 127 | 29.922 | completed |
| guardrail6 | global-purpose-assignment | 2346 | 80 | 26.501 | completed |

各fixture1回、2calls。timeout/context overflow/incomplete backend outputなし。weak16は全16fileを1group、cross12は12 singleton、guardrailは3implementation/test pair。候補text/raw prompt/思考は保存せず、内部理由や候補の意味正確性は観測値から推定しない。候補数がgoldに近いことを成功指標にしない。

fixed Qwen3-8B-4bit revision545dc4251c05440727734bcd94334791f6ab0192、native0/output1536/context16K/call120秒/whole600秒、temp0/top_p1/top_k0/seed144、retry/repair0。候補1〜16/各purpose120文字/参照1〜32、unknown/同候補内duplicate/empty/同一text重複/unresolved拒否。候補membershipは要求せず、元観測全件を後段へ保持し追加/split/mergeを許可。これは構造gateであり意味gateではない。

`go test ./tools/benchmark149`（42.796秒）、`go vet ./tools/benchmark149`、`go build -o /tmp/benchmark149-discovery ./tools/benchmark149`、`git diff --check`成功。新規testはunknown/duplicate/missing/unresolved等の拒否と跨る根拠共有を確認。追加取得/依存/production変更なし。fresh protocol8/metadataは未実行。

## 考察

#140で主に未完了だったdiscoveryを今回の小さい出力で完了できたが、grouping改善には繋がらない。H17に対してweak16は一致からFM120へ悪化、cross12 FS30/guardrail一致は同じ。暫定候補を不可逆boundaryにしなくても、正しい目的区別を保証できない。構造的成功・意味的成功・費用を分離する必要がある。

H11はper-entity意味抽出、H18はglobal目的候補抽出という責務差があるが、双方でcross12改善は得られていない。call増加だけを採用根拠にしない。bounded-window再実装、soft evidence hard union、gold修正、多数決、wording/budget反復で成功扱いにする選択肢は対象外。

## Next Steps

- 次iterationでは追加推論に先立ち、#143/#139〜#142/#146とH1〜H18の試験済みfamily、棄却根拠、観測不足を一覧にする。似た方式を別名で繰り返さず、合理的な探索残余を根拠付きで判定するため。
- 利用可能checkpoint/generation契約と未検証architectureの組合せを監査し、未検証というだけでなく既存failureに作用する根拠があるか判定する。根拠があれば小さい次仮説を事前登録、なければ現在capabilityでの検証限界と再開prerequisiteを具体化する。有限の失敗を普遍的不可能性としないため。
- その監査とIssue完了条件を照合し、candidate不成立・追加探索消尽の立証・単なる未完了を区別する。全Verification達成前の最終品質ゲートやcheckbox完了宣言は行わない。
