# Iteration 2: 固定policyのpaired比較結果

## 要約

追加した判断規則は未使用ケースでNO-GO。独立誤統合は6/6から2/6へ減ったが、独立変更の正しい分離0/6で、残り4/6は不適切deferだった。必要対応/補償4/4もdeferへ変わり、許容判断は8/16から2/16、判定可能行の確定coverageは14/14から4/14へ低下した。全52生成call・retry0で契約どおり終了し、prompt/gold/閾値を変更しなかった。

## 検証結果

### 未使用8case×2提示順のpaired比較

| 指標 | baseline | policy |
|---|---:|---:|
| schema/source-ref valid | 16 | 16 |
| 許容判断 | 8 | 2 |
| 独立誤統合 | 6 | 2 |
| 必要対応誤分割 | 0 | 0 |
| defer | 0 | 10 |
| 根拠不足の期待defer | 0 | 0 |
| 判定可能行の確定coverage | 14 | 4 |
| 提示順一致case | 8 | 4 |
| input tokens合計 | 9676 | 13180 |
| output tokens合計 | 208 | 208 |
| TTFT中央値（秒） | 2.099 | 2.444 |
| process walltime中央値（秒） | 2.503 | 2.867 |
| process footprint最大（bytes） | 2602879208 | 2683980032 |
| RSS最大（bytes、非加算） | 1887191040 | 1887600640 |
| MLX peak最大（bytes、非加算） | 2316564552 | 2318810424 |
| 資源Gate通過応答 | 15 | 16 |

根拠不足2応答は両armともmergeで、期待defer0/2。policyのdefer10/16はすべて事前規範で判定可能な行に発生した。必要対応誤分割0/4はkeep_separateを返さなかったという件数であり、必要対応を適切に確定した成功ではない。実装/assertion・補償の全4応答はdeferで、必要merge確定0/4。

| case | 規範 | baseline 通常/逆順 | policy 通常/逆順 |
|---|---|---|---|
| nil-slice-corresponding-assertion | join | merge / merge | defer / defer |
| shared-bound-independent-ui-and-batch | separate | merge / merge | defer / merge |
| same-package-independent-time-and-permission | separate | merge / merge | defer / merge |
| shared-import-independent-search-and-sort | separate | merge / merge | defer / defer |
| boolean-polarity-compensation | join | merge / merge | defer / defer |
| new-round-provider-eb | dependency | merge / merge | defer / merge |
| cache-expiry-multiple-boundaries | multiple | merge / merge | defer / merge |
| unknown-admission-contract | insufficient | merge / merge | merge / merge |

### 既知10caseの探索的診断

#163の固定raw Coder20応答をhistorical baselineとし、新たなpolicy20応答と比較した。未使用の資格Gateではなく、同時期・同負荷の費用対照でもない。

| 指標 | baseline | policy |
|---|---:|---:|
| schema/source-ref valid | 20 | 20 |
| 許容判断 | 14 | 13 |
| 独立誤統合 | 4 | 2 |
| 必要対応誤分割 | 0 | 0 |
| defer | 0 | 3 |
| 根拠不足の期待defer | 0 | 0 |
| 判定可能行の確定coverage | 18 | 15 |
| 提示順一致case | 10 | 7 |
| input tokens合計 | 12216 | 16596 |
| output tokens合計 | 260 | 260 |
| TTFT中央値（秒） | 1.985 | 4.812 |
| process walltime中央値（秒） | 2.384 | 5.249 |
| process footprint最大（bytes） | 2593179952 | 2731133304 |
| RSS最大（bytes、非加算） | 1886339072 | 1890435072 |
| MLX peak最大（bytes、非加算） | 2404464668 | 2313698832 |
| 資源Gate通過応答 | 20 | 20 |

| case | 規範 | #163 baseline 通常/逆順 | policy 通常/逆順 |
|---|---|---|---|
| path-escape-corresponding-assertion | join | merge / merge | merge / merge |
| inclusive-alert-corresponding-assertion | join | merge / merge | merge / merge |
| shared-cleaner-independent-outputs | separate | merge / merge | defer / merge |
| weak-tests-independent-text-behaviors | separate | merge / merge | defer / merge |
| weak-tests-wire-compensation | join | merge / merge | defer / merge |
| weak-tests-unit-scale-compensation | join | merge / merge | merge / merge |
| new-cap-provider-ea | dependency | merge / merge | merge / merge |
| new-wrap-provider-eb | dependency | merge / merge | merge / merge |
| retry-policy-multiple-partitions | multiple | merge / merge | merge / merge |
| external-limit-evidence-insufficient | insufficient | merge / merge | merge / merge |

### 構造・停止・費用・資源

- 全52応答はbackend completedかつschema/source-ref正常。host分類はcompleted39、unresolved13。invalid JSON/schema/assignment/evidence、timeout、context超過、fallbackは0。全39確定案は既存authoritative validation・完全割当・byte/tree再構築・ordered apply/revert tests PASS。分離案0なので、この結果から独立revert能力を主張しない。
- native tokenは全52行で事前監査と一致。source/helper/Metal/AST/validator/model9fileの事前・事後digestは一致。事前登録commit `908e0b34` のpush/remote head一致とGitHub manifestのbyte一致を確認した後に生成した。
- 全体費用308.579秒（準備51.076秒＋生成/結果監査257.504秒）。最大52call、retry0、1800秒以内。未使用policyのinput token増分3504、36.21%、output合計は同じ208。walltime中央値比1.145倍。費用Gate PASS。
- 全process採取最大footprintは5GB以内。未使用baselineのnil/assertion通常順だけpressure level2を観測し、baseline資源15/16、policy16/16で、両arm全通過の資源Gateは不合格。全行swap増分0。現在の開発アプリ共存負荷とサンプリングの限界があり、このwarningをモデル単独の原因へ帰属しない。pressure failureはraw/result-auditへ保持した。
- RSS/MLX/observed lifetime peak/Metal別値を足さない。Metal固有の独立allocationはnull。100msのprocess samplingは終了間際のピークを見逃しうる。既知historical latency比較からpolicyの純粋な費用差を推定しない。
- 構造preflight27件、生成前native token監査52件、paired source/schema同一性・raw再集計・研究scope監査はPASS。main HEADと既存.gitignore差分を保持。新依存・新モデル・production/default/4file・staging・metadata変更0。

## 考察

誤統合件数の減少は、この評価では正しい分離の獲得による改善ではない。policyは独立変更4行でdeferし、必要対応/補償4行でもdeferした。一方、根拠不足2行ではmergeを続けた。誤統合を数えるだけでは、この確定判断の喪失と誤った保留対象を見落とす。必要誤分割0も、必要mergeを正しく選べた証拠にはならない。

未使用caseの提示順一致は8/8から4/8へ低下した。通常順では独立3caseがすべてdefer、逆順では2caseがmergeで、残るshared-import caseは両順defer。API依存と複数許容caseにもdefer/mergeの順序差があった。固定規則の追加はこの小標本で順序安定性を満たさなかった。

既知ケースでも許容14/20から13/20、独立誤統合4/4から2/4だが分離成功0で、期待deferは0/2のまま。探索的診断と未使用paired評価は同じNO-GO方向だが、使用済みデータとhistorical実行を独立holdoutや同条件の費用比較へ読み替えない。

source情報だけで原理的に境界を識別できないと断定する結果ではない。未使用nil/assertionとboolean補償は直接対応/明示roundtrip契約が入力にあり、partial state検査でも対応の必要性が観測された。モデルがそこをdeferした原因を出力schemaのdecision/source-IDだけから特定できない。根拠ID正常性も意味的な根拠理解を証明しない。追加規則の局所改善は立証されず、既存#163 NO-GOを変更しない。

## Next Steps

- 本事前登録はNO-GOとして終了し、同じpromptを結果に合わせて反復調整しない。許容判断・coverage・期待defer・順序・独立分離の条件を満たさなかったため。
- 再開候補はsourceから識別可能な直接対応/独立/補償と、外部契約不足の境界を先に監査する。保留を要求した行の根拠が実際に欠けているのかを切り分けるため。新しい手法を採る場合は別prompt revision、未使用入力、family監査、source/model/helper/予算/Gateの生成前commit/pushを要求する。今回の8/10caseは未使用へ再分類しない。
- 良好な次段局所Gateが成立した後のみ、別契約で全体8/16file・80% exact・metadata・production経路へ進む。今回の2file局所判断、構造PASS、資源値から全体能力へ外挿できないため。
- 全8完了条件を証跡付きで反映し、goal.mdの最終品質ゲートを実行する。検証の完了を候補の改善/採用成功と区別するため。

## 再現用成果物

固定契約は `contract.md`、完全入力/prompt/gold/hashは `preregistered.json`、生成前sourceとtoken監査は `input-audit.json` / `native-token-audit.json`、全raw応答は `results.jsonl`、集計/Gateは `summary.json`、独立再集計と範囲監査は `result-audit.json`。参照#163固定revisionをcheckoutし、既存の同digest helper/model/adapterを明示して `evaluate.py prepare` と生成前commit/push後の `evaluate.py run` を実行する。出力はexclusive-createのため既存rawを上書きしない。再実行は別の専用worktreeと事前登録を使う。
