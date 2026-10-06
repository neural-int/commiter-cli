# Iteration32: no-candidate分岐と残余仮説の監査

現worktreeとlive Issue149を照合。追加推論0、gold/測定/production変更なし。Iteration26監査に以下を追加する。全architectureの不可能性や合理的探索消尽は今回も未立証。

## 追加familyの棄却根拠

| family | 改善 | 未達 |
|---|---|---|
| H19 host complete candidates + Qwen8 selection | cross12 exact/guardrail exact | weak16 unresolved、固定generator10/14包含 |
| H20 同selection + Gemma | 小さい出力で全生成完成 | 全3case一括group、FM120/36/12 |
| H21 single-global-context pair labels + Gemma | 120pair1080tokens/56秒で生成 | weak16 FM120/cross12 U/guardrail invalid JSON |
| H22 同global pair + Qwen8 | guardrailは生成完成 | weak16/cross12 timeout120秒、guardrail U |

JSONLを再読: iteration24 purpose weak16 exact、iteration27 selection cross12 exact、iteration29 generator4欠落、iteration31 timeouts。timeout/invalid/unresolvedはFM0成功と扱わない。内部理由は未特定。H21と#146のlocal-context判定は異なるが、どちらも整合性検証だけで正解を保証しない。

## 残余の条件

未試験というだけでは次案の根拠にしない。既存の主要失敗への作用根拠、責務または観測の差、boundedな独立試験、scope/gold/安全条件維持を満たす必要がある。

- 同task予算/wording/model組合せ総当たり: 新しい作用根拠なし、実施しない。
- hard relation/runtime union、gold候補注入、case別モデル選択、unresolved後の推測recovery: guardrail/対象外に抵触、実施しない。
- embedding/finetuning/新checkpoint: 現在のfailureへの独立改善根拠と取得/依存/学習準備が未確認。単に新model名を挙げて実施しない。
- H23 optional host partition proposals + free global assignment: 検証可能な残余あり。H17自由割当はweak16 exact、H19候補比較はcross12/guardrail exact。固定候補だけでは4欠落、自由割当だけではcross12 FS30。同じglobal callにhost候補をsoft proposalとして保持し、最終partitionを自由に生成する責務境界は未測定。これは2方式の成功を合成すれば成功するという証明ではなく、個別に観測された能力を一つのtaskで両立できるかを測る仮説。

H23はH18のLLM-generated purpose textとは異なり、観測source/test/callsからhostが決定的に生成したcomplete membershipsを提案する。候補は参考情報で、最終G-ID assignmentは新partitionも許す。候補ID selectionの後にfallbackする2段階ではなく、最初から1callの自由assignment。元観測全件を保持、候補内group数/辺をhard constraintにしない。generatorは固定、欠落fixtureへルール追加しない。

## 完了条件の証拠対応

| Issue完了条件 | 現時点の証拠 | 判定 |
|---|---|---|
| 既存Issue確認/主要failure | Iteration26監査 + 本監査、#139〜146 commentリンク | 達成 |
| #146と異なるarchitecture、失敗後改訂 | H1〜H22/Iteration1〜31 reportとTOC | 達成 |
| 5〜8/9〜16評価 | H5 Iteration6〜7、H19〜22資格JSONL | 達成（採用成功ではない） |
| specified failure fixtures | Iteration7 suite、runtime guardrail、Iteration29全14件 | 達成 |
| 選定未使用holdoutのregression | Iteration2/7の事前固定holdout測定 | 達成。以後使用済みで再利用時はfreshとしない |
| exact/FM/FS/complete/unresolved記録 | 各iteration JSONL、nullは未評価 | 達成 |
| calls/tokens/stage wall/timeout/overflow記録 | measured backend + phase JSONL。未取得tokensはnull | 達成 |
| current4file同条件比較 | Iteration10 serialized baseline/H5 metadata/Validate | 達成 |
| 改善/残余/採否理由 | 各reportとIteration26/32一覧 | 達成 |
| candidate選定または現時点なしの結論 | 試験済み候補は全棄却だがH23残余あり | 未達 |
| candidate-found architecture契約 | 採用candidateなし | 条件未発動。架空のcandidate文書で埋めない |
| candidate-found child Issue/実装範囲 | production未変更、採用candidateなし | 条件未発動。不要なchild Issueを作らない |
| no-candidate合理的探索消尽 | H23作用根拠とbounded試験が残る | 未達 |
| iteration因果追跡/Skill不使用 | live TOC + 各Next Steps、Skill未使用 | 達成 |

4未チェックの完了条件を維持する。全Verificationが成立しておらず、最終quality gates/Goal完了は行わない。mainの既存.gitignore変更は保持。現在の試験済み案不採用を、scope全体の解決へ読み替えない。

## H23の最小契約

candidate generator/H17 payload/schemaを固定し、最大4のhost complete partitionsだけをsoft proposalとして追加する。final outputはfile→G-ID/unresolved、new partition/提案の無視/分割/結合を許可。selected IDs exactly-once、unknown/missing/duplicate/invalid/U拒否、final metadata後Validateを維持。

取得済みQwen3-8B native0、既存1536token/16K/120秒call/600秒whole、temp0/top_p1/top_k0/seed144、1call/fixture、retry0/repair0、weak16/cross12/guardrail各1回。全case同task/route。資格通過時は新未使用独立評価を事前固定してrange/baseline/order/metadataへ。失敗時はproposal+free出力を棄却し、wording/budget/候補ルールの調整で取り直さない。
