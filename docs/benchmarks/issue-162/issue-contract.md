## 作業種別
Verification / Design

## 概要
**与えられた根拠から、独立してレビュー・revertできる妥当な分割を作る。** 作者の過去commit境界の一意再現を目的にせず、機械的baselineとLLM追加の価値を分離して測定する。#156のexact33.3%・NO-GO、過去gold・閾値・production4file契約は変更しない。

## 背景
#142の二択/tournament、#146の局所統合、#149のIR/global判断、#153のpair-score/solver、#155のbehavior attribution、#156のA2/B/Cを参照。新規性は評価対象を「作者の目的推測」から、入力根拠に基づく必要な同時レビュー、独立性、方向付き依存、識別不能へ分けること。スコアだけのhard-linkや推移統合を繰り返さない。

## 検証順とGate
1. **評価契約**: complete exactly-once、byte/tree reconstruction、必要な対応変更の分断、独立変更の混入、依存順序、複数妥当/根拠不足を区別。Git上のrevert可能と動作上の独立性を別指標にする。機械でsource確認できる条件と人間のレビュー判断を混同しない。
2. **機械的baseline**: file-only、および同一symbol/import/callからの関係候補。構造関係だけで同目的・must-linkと認めない。
3. **限定LLM判断**: 同一入力・schema・予算で固定GemmaとQwenを比較。`together / separate / depends_a_on_b / depends_b_on_a / unknown`と実source根拠参照を返す。最初は新規controlled8ケース（対応変更2、独立2、方向依存2、多義/不足2）×提示順2×モデル2=32call上限の能力診断。4fileのbefore/afterを全提示し、対象2fileの判断を測る。自己作成データであり独立採用holdoutではない。gold/sourceは推論前固定し、作者理由を入力外から当てさせない。
4. **逐次構築**: 3の限定能力Gateを満たす場合だけ、少数候補への仮配置・取り消し・再統合を実装して同情報の一括判断と比較。要約でsourceを失わず、候補欠落、提示/投入順、bridge、累積誤りを評価。Gate未達なら本段階をgated-outとして原因と再開条件を保存する。
5. **独立16file評価**: 4成立時のみ未使用public履歴/別作者holdoutで妥当性・安全性・total latency/tokensを評価。前段失敗なら性能成功を主張せず、必要な評価入力と能力を明記する。

## 初回固定条件
既存local MLX helper、既存cache内のGemma-4-E4B-it-4bitとQwen3-8B-4bitの固定revisionを利用。新依存/モデルdownload無し。neutral grammar、context16384、output1536、temp0/top_p1/top_k0/seed144/native thought0、per-call30秒、total1200秒（推論＋結果検証）、max32call、retry0。入力hash、gold、model/helper/source hash、schema、timeoutを実験前commit・push。モデルfail/unknown/null tokensを成功や0へ置換しない。

初回Gate: 全32応答completed/schema/source-reference-valid、識別可能6ケース×2orderで各モデル12/12の関係正解、曖昧2ケースは4/4 unknown、順序一致、negative誤統合0、1200秒以内。この小標本Gateは次段を始める条件に限り、production品質の証明ではない。各モデル別にGateを判断する。

## 安全・非目標
- production/default model/4file上限、SRS、authoritative planning.Validateを変更しない。
- SQLite・新依存・cloud推論・外部へのdiff送信を導入しない。状態はメモリ、記録はJSON/JSONL。
- benchmark用docs/toolsと専用worktreeのみ。既存main変更を保持。
- 外部repo code/testを実行しない。初回は自己作成sourceを読み取り/AST解析するだけ。モデル応答の根拠参照の有効性は意味の正しさと別。
- Skill/無制限探索/閾値事後調整無し。初回failでも根拠付き結果と次の切り分けを保存。

## 成果物 / 完了条件
- [ ] レビュー・revert妥当性、複数許容境界、方向依存、unknownの評価契約
- [ ] 機械的baseline、事前固定入力/gold/source hashと識別可能性監査
- [ ] Gemma/Qwen同条件の根拠付き関係診断、順序・失敗・cost・追加価値
- [ ] Gateに従う逐次構築比較、またはgated-out理由・再開条件
- [ ] Gateに従う独立16file評価、または未立証能力と必要なholdoutの記録
- [ ] GO/NO-GO、production採用とは異なる判断、再現資料・適切な品質チェック

## 報告方針
各iterationは要約・検証結果・考察・Next Stepsを日本語でIssueへ記録し、専用branchへcommit/pushする。未達を過去成功へ読み替えない。

推論前のhelper source確認によりneutral profileの固定出力上限は1536と判明した。512の初稿は未実行のdraftとして保存し、1536へ修正してから事前登録する。モデル結果を見た予算変更ではない。
