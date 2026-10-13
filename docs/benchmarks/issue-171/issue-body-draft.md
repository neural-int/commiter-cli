## 目的
**Direct Groupingの入力情報（既存入力／Raw Diff）とSystem Prompt（既存／#170暫定B）が、最終的なコミット分割品質に与える個別効果と組み合わせの影響を、2×2の4条件で検証する。**

既存Three-phase Plannerの**第1段階のGrouping判断だけ**を対象とする。Purpose-first、Purpose Discovery/Assignment、後続Category・Text生成、Production仕様変更は対象外。

## 背景（既存結果を維持）
- #160：16file統合候補 A2 exact 4/12、80%未達。
- #162〜#166：関係分類、局所判断、明示規約、段階分割、schema調整は一部局所改善のみ。False Merge抑制だけでは正しい分割と両立しない。
- #167：file-firstの構造保持36/36に対し完成plan 0/36。構造安全と意味品質・完成率を分ける。
- #168：Structural Evidence中心のPurpose-firstはGate1 NO-GO。
- #169：Raw Diffで具体的purpose coverage 13/14へ改善したがpurpose list exact 2/8、直接Groupingは未実施。
- #170：プロンプトB（詳細版）は目的の過剰分割を8→5に抑制したがcoverage・順序安定性が悪化、exact 4/8でBaselineと同等。**Bは研究用の暫定基準**であり、製品正式採用ではない。Gate1 NO-GOは維持する。

## 今回の優先課題
1. **目的の過剰分割と必要な同目的統合**：関連する実装・テスト・consumerを不当に分離しない。
2. **独立目的の誤統合回避**：全件統合・全件singletonによる見かけの改善を拒否する。
3. **入力順序依存**：Go側の安定ソートなど、追加LLM呼び出しなしの決定的な対策を検討する。
4. **情報欠損と推論判断の切り分け**：Raw Diffから同一契約・独立性を裏付けられるかをケース別に監査する。

## 1. 比較する4条件（2×2要因比較）

| Arm | LLM入力 | System Prompt |
|---|---|---|
| **A：Baseline** | 既存Grouping入力 | 既存Groupingプロンプト |
| **B：入力変更のみ** | Raw Diff中心の入力 | 既存Groupingプロンプト |
| **C：プロンプト変更のみ** | 既存Grouping入力 | #170暫定Bの意味判断規則 |
| **D：組み合わせ** | Raw Diff中心の入力 | #170暫定Bの意味判断規則 |

- A→B：既存プロンプトにおける入力変更の効果。A→C：既存入力におけるプロンプト変更の効果。
- B→D、C→Dも比較し、入力とプロンプトの交互作用を確認する。**DとAだけを比較して一方の変数の効果と断定しない。**
- Baselineの実入力・実プロンプトは既存コードと実リクエストから特定し、想像した旧構成で代用しない。
- #170の暫定Bは意味判断規則を維持し、**Purpose ID・説明の列挙を要求する出力指示だけをGroupingタスクへ適合**させる。適応内容を全文記録する。
- 4条件とも**共通のファイルID→グループID形式のGrouping専用JSON Schema**を用いる。既存Three-phase Planner第1段階のSchemaを再利用するが、これはSchemaを固定した条件下での比較に限定される。**Schemaとの交互作用、新方式が別Schemaで到達し得る性能、Production全体の優劣は評価しない。**
- 入力構造がRaw Diffを保持できることと、モデルにとって意味情報が十分であることは別。各条件の実ペイロードを固定し、フィールドの有無・圧縮・入力予算・ID対応を監査する。

## 2. 固定する条件

- **同一fixture、Git snapshot、選択ファイル集合、Gold、評価器、モデルcheckpoint/revision、推論backend、seed、temperature、decoder設定、context/output上限、retry/timeout**。
- **同一のGrouping出力Schema、ファイルID、Grouping以外の共通task指示・安全指示**を使用する。プロンプト差は比較対象の意味判断規則に限定し、不可避の適応差分を記録する。
- 入力の通常順・逆順を**4条件すべてに同じように適用**し、paired比較する。Go側安定ソートの有無を特定armのみに適用しない。
- Raw Diffと既存入力の生成経路、最小共通メタデータ、context超過時の処理を事前固定。入力変更以外の差分を隠さない。
- fixture数、反復数、call上限、資源上限、品質Gateの数値、プロンプト全文とhash、Schemaを**初回モデル生成前に凍結**し、結果による調整を行わない。
- Goldは[作成・採点規則 v1.0](https://github.com/neural-int/commiter-cli/blob/codex/issue-171-direct-grouping/docs/benchmarks/issue-171/commit-grouping-gold-rules.md)を使用し、1file複数目的の正当なFallback、複数許容解、未確定ケース、arm別入力識別可能性を区別する。

## 3. 評価指標

**主指標**
- **Exact Partition**：Goldと最終ファイル分割が完全一致したfixtureの割合。
- **False Merge（FM）**：Goldで別グループのファイルペアを誤統合した件数・発生率。
- **False Split（FS）**：Goldで同一グループのファイルペアを誤分割した件数・発生率。

**副指標**
- 入力順序反転時の結果一致率、complete assignment・Schema validity、全件merge／全件singletonへの偏り。
- Fallback／unresolved、入力からの識別可能性、tokens、wall time、RSS、memory pressure／swap、context超過。
- **Purpose列挙の正確さやSchema適合のみをGrouping成功とは扱わない。** 意味品質と構造安全性を別に報告する。

## 4. 実施方針

- 既存Three-phase Plannerの**第1段階Groupingのみ**を、同一の研究用ハーネスで4条件実行する。後続のCategory・Text生成やPurpose-firstとの比較は行わない。
- **Phase 0**：実リクエストからBaselineを特定し、4条件の入力・プロンプト・共通Schemaの静的互換性と最小限の受理条件を確認。Gold、fixture、Gate、資源上限を凍結する。
- **Phase 1**：4条件を同一fixtureと通常／逆順でpaired実行し、入力要因・プロンプト要因・組み合わせの違いを報告する。
- **判定**：Exact改善のみではGOとせず、FM悪化、FS、全件merge／singleton、完全割当、安全・資源を同時に評価する。事前固定GateでGO/NO-GOを決定し終了する。
- 既存の実装と研究ハーネスを再利用し、**新Planner、新LLMステージ、Schema要因を増やす追加ablationを作らない**。小規模・合成fixtureでの結果をProduction品質と一般化しない。

### Phase 0：事前監査（詳細）
- 既存Three-phase/Direct Grouping/validatorの利用可能範囲を確認。**新しいPlannerやLLMステージを作らない**。既存の実装・研究ハーネス再利用を優先する。
- #168〜#170の既知fixtureは診断専用。今回の評価用goldと未使用ケースを事前固定。実装＋テスト、producer/consumer、cross-directory同目的、同packageの独立目的、1file混在、順序反転を含み、根拠不足のケースを明示的に分離する。
- 各caseを「Raw Diffだけで関連性・独立性が識別可能／根拠不足／複数境界許容」に事前分類。作者意図や元commit messageをLLMに漏らさない。可能なら実repository別作者の未使用変更を含めるが、その収集自体は先送り方針を尊重し、当初の小規模切り分けでは**一般化・正式採用の結論を出さない**。
- Raw Diff、file ID、各file共通fields、共通Grouping schema、各System Prompt全文、MLX Gemma E4B 4bit revision（#170と同じ候補）、seed/temperature/output/context、入力順序、モデルcall上限、品質/資源Gateを**推論0回で凍結**する。
- Go側入力順序の安定化は**単独変更要因として切り分ける**。4条件すべての入力順を同じ決定則で揃えた主試験に加え、診断として逆順入力を保持。標準化により結果を一意にすることと、意味的に正しいことを混同しない。

## 安全・資源・研究範囲
- Local-only、元Git HEAD/index/worktree不変更、file IDの所有権、byte/tree一致、既存validator、秘密情報の安全境界を維持。
- Input/output tokens、wall p50/p95、RSS、memory pressure/swap、context超過を比較。16GB Apple Siliconで他プロセスとの共存を意識する。16file約80%達成は本研究の小標本から主張しない。
- Production default、既定4file上限、Git commit mutation、#169/#170のNO-GO、#170 Bの**研究用暫定採用**は変更しない。
- Purpose-first比較・Purpose Discovery/Assignment、追加モデル・LLM多数決・無期限プロンプト調整・二次LLM検証・複雑な依存解析エンジンは対象外。新しいPlannerは作らない。
- Goldの作成・採点は[Gold作成規則 v1.0](https://github.com/neural-int/commiter-cli/blob/codex/issue-171-direct-grouping/docs/benchmarks/issue-171/commit-grouping-gold-rules.md)を基準とし、具体的fixture・許容解・閾値を初回生成前に固定する。

## 完了条件
- [x] 既存Direct Groupingの実装・入力・出力契約と有限Gateを生成前に記録
- [x] Baselineの実入力・実prompt/schema、TreatmentのRaw Diff・暫定B全文prompt/schema、同一モデル・gold・入力順序契約を固定
- [x] 誤統合・誤分割・識別不能を分けたケース別の根拠監査
- [ ] Phase 1で既存Direct GroupingとRaw Diff＋暫定BのDirect Groupingを同一fixtureでpaired比較し、exact/FM/FS・順序依存・完全割当・資源を評価
- [x] 結果を意味品質・安全・費用に分けてGO/NO-GO、未到達項目、次の実装条件を明記
- [x] Productionと既存Git状態を変更せず、有限研究として終了

## 関連
#160、#162〜#170。特に[#169](https://github.com/neural-int/commiter-cli/issues/169)（Raw Diffの情報価値）と[#170](https://github.com/neural-int/commiter-cli/issues/170)（暫定Bプロンプト）。

## 検証報告目次

### [Iteration 1：fixture妥当性確認と資源Gate停止](POSTED_COMMENT_URL)
- Gold・入力・有限Gateを生成前に固定し、初回callのswap上限超過で停止。Phase 1のpaired比較は未到達。
