# Iteration 2: 原sourceを共有した局所モデル比較

## 要約

初回2モデルとも局所NO-GO。Coderは今回の局所入力で資源Gateを通過したが、独立変更の誤統合と根拠不足での保留条件を満たさなかった。Gemmaは局所品質に加え、process footprintが5GB上限を超えた。8/16fileの全体分割はgated-out、生成0call。これは検証としてのNO-GO確定であり、4file拡張・production採用の達成ではない。

## 検証結果

| 指標 | Gemma-4-E4B固定4bit | Qwen2.5-Coder-3B固定4bit |
|---|---:|---:|
| completed/schema/source-ref valid | 20/20 | 20/20 |
| 許容判断 | 15/20 | 14/20 |
| 独立変更の誤統合 | 3/4 | 4/4 |
| 必要対応の誤分割 | 0/8 | 0/8 |
| 提示順一致 | 7/10 | 10/10 |
| 根拠不足caseの期待defer | 0/2 | 0/2 |
| process footprint採取最大 | 6,127,194,960 bytes | 2,593,179,952 bytes |
| RSS採取最大（footprintとは別） | 3,475,570,688 bytes | 1,886,339,072 bytes |
| MLX peak（他指標へ加算しない） | 5,149,225,852 bytes | 2,404,464,668 bytes |
| TTFT中央値（helper handle起点） | 3.943秒 | 1.985秒 |
| process walltime中央値 | 5.320秒 | 2.384秒 |
| 資源Gateの通過観測 | 0/20 | 20/20 |
| input / output tokens計 | 13,440 / 481 | 12,216 / 260 |
| resource / local / global | NO-GO / NO-GO / gated-out | 局所入力GO / NO-GO / gated-out |

40生成call、retry0、全応答completed、invalid JSON/schema/assignment・timeout・context超過0、defer0。全40確定案でauthoritative validation、完全割当、byte/tree再構築、順序付き適用/revertのfixture testsはPASSした。構造・testsの成功でもレビュー妥当性の失敗を解消しなかった。単独revertは依存順付きrevertと別にraw auditへ保存した。

全体費用は326.238秒（source/token監査146.681秒＋生成/結果監査179.558秒）。1800秒予算内。download/buildの準備費用とは別。未知の独立Metal割当値はnullで、MLX allocator値を別のMetal全量へ読み替えない。

Gemmaの試行中にpressure dispatch level1/2/4とswap増分を観測した。準備中にもwarningがあり、既存アプリ負荷を終了/固定再起動していないため、pressure/swap変化全体をモデル単独の因果効果とは断定しない。process footprint上限超過はそれだけでもresource NO-GOの根拠。Coderは全20試行でpressure1、swap増分なし、footprint上限内。3GB未満を失格とはしない。

機械的baselineのレビュー規範受理はfile-only 5/10、static-new-api-order 5/10。順序付きtestsを含む妥当性はそれぞれ4/10、5/10。対応assertion2caseの両順では両モデルがmergeしstatic baselineを改善したが、独立変更のnegative条件を満たさなかった。

両モデルのinput token長は事前native template監査と全40行で一致。固定source hash、responseのmodel pin/profile、Gemma cached重み/tokenizer/template digest、Coder9file digest、既存helper binary保持を監査してPASS。元のmain/.gitignore変更を保持し、成果物はbenchmark docs/tools限定。

## case別の判断

| case | 規範 | Gemma通常/逆順 | Coder通常/逆順 |
|---|---|---|---|
| path-escape-corresponding-assertion | join | merge / merge | merge / merge |
| inclusive-alert-corresponding-assertion | join | merge / merge | merge / merge |
| shared-cleaner-independent-outputs | separate | merge / merge | merge / merge |
| weak-tests-independent-text-behaviors | separate | keep_separate / merge | merge / merge |
| weak-tests-wire-compensation | join | merge / merge | merge / merge |
| weak-tests-unit-scale-compensation | join | merge / merge | merge / merge |
| new-cap-provider-ea | dependency | merge / merge | merge / merge |
| new-wrap-provider-eb | dependency | merge / merge | merge / merge |
| retry-policy-multiple-partitions | multiple | merge / keep_separate | merge / merge |
| external-limit-evidence-insufficient | insufficient | merge / keep_separate | merge / merge |

## 考察

CoderはGemmaより小さいfootprintと短いlatencyを示したが、この少数controlled入力の許容判断は14/20で、Gemmaの15/20を上回らなかった。独立変更4応答をすべて誤統合したため、コード特化やresource GOからレビュー分割能力のGOを導けない。両モデルは直接対応・補償変更の必要統合を保った一方、shared helper/同packageの独立性と根拠不足を区別できなかった。

Gemmaの提示順不一致3caseには、APIや複数妥当caseで両方の判断が許容されるものも含む。その妥当性と、事前登録した同じpartition/decisionを返す順序安定性を別の指標にした。根拠不足2応答は両モデルとも確定判断を返し、defer規範未達だった。

入力は未推論の自己作成10caseだったが、対応/assertion・API・roundtrip familyには相関がある。公開自然履歴、別作者、未使用repositoryの採用holdoutを立証していない。15/20や14/20を過去の作者境界exact指標・#156の80%目標と比較しない。8/16file全membership、metadata、production経路、任意group単独revertの全体資格は未評価。

## Next Steps

- 本契約はresource/local/globalを分離したNO-GOとして終了し、8/16fileへ進めない。negative誤統合0と局所Gateが未達のため。
- 再開する場合は新しい根拠付き構成・未使用資格入力・prompt/schema/予算を別に事前登録する。今回の10caseをholdoutへ再分類したり、結果後にgold/閾値を変更したりしないため。
- 全体資格の再開には、局所Gate成立後の未使用8/16file全source入力とrepository/family独立性監査を要求する。2file判断や構造PASSを全体能力へ外挿しないため。
- Issue完了条件の証跡を反映し、goal.mdの最終品質ゲートを通して検証完了を確認する。NO-GO確定とproduction機能達成を区別するため。
