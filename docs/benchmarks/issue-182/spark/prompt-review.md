# H182-017 Sparkプロンプト見直し（非推論レビュー）

## Conclusion

分類と文章の指示には改善候補がある。Low/HighのThinking上限は簡潔化指示で改善する可能性はあるが、promptだけで予算内の自然終了を保証しない。実装変更・追加推論は行っていない。

## Evidence and candidates

| 対象 | 観測 | 現行指示の弱い箇所 | 比較候補 |
| --- | --- | --- | --- |
| Category | OFF grammarありはメール修正をfeat、なしは全群feat | docs/feat/fixの短い定義だけで、優先順位と新規コード追加との区別がない | 既存の期待動作の修復は、新たな入力ケースや回帰テストを追加してもfix。featは独立した新規利用者機能と説明 |
| Text | OFF grammarありでテスト用日数らしい断片を不自然な文章へ混在 | actual change and purposeという指示だけで、テストの細目・実装表現をsummaryへ入れる判断が曖昧 | 利用者/開発者への主な変化を一文。コード断片・テスト用データ・コマンドを根拠なく持ち込まず、48文字に詳細を詰め込まない |
| Thinking | Low/High全4条件が2048 thought tokens、marker/final=0 | runtime共通説明はthink freelyで、簡潔な終了方針や証拠の参照順を指定していない | think freelyを構文自由という意味に限定。簡潔なtask-focused reasoning、まず現在diff、曖昧な場合に関連履歴を使い、十分な根拠があれば終了 |
| JSON形式 | OFF none TextでJSONDecodeError | JSONのみ/Schema指示は既に存在 | promptの不足と断定しない。制約ありのgrammarがこの形式問題を防いだ事実を優先 |
| Grouping OFF | 両条件3/3正解 | 目的分離/実装とテストの組合せは成立 | この基本ルールは変更しない |

## Causal interpretation

現在のCategory入力には、空の山括弧が欠落値へ正規化されずmid:キーとなり、既存のloc:フォールバックへ揃える修正であることを示す依頼がある。差分と回帰テストも提供されている。したがってこの例では原因を単純な履歴欠落と扱わない。ただしどの情報をモデルが重視したかは未確認。

Thinkingの本文・token列は保存されず、progressの数と境界有無のみが残る。何を迷ったか、反復したか、どのファイルで詰まったかは断定不可。低いreasoning_effortは自然終了のtoken上限ではなく、今回も共通2048上限で終了した。入力は約6.7K tokens、出力予算2048で、32K入力不足や文脈上限到達ではない。

Textは確定typeを固定して受け取るので、Categoryのfeat誤りをTextでfixへ修復することは現仕様に含めない。型・文字数のgrammarは意味の正解まで保証しない。

## Candidate wording (not applied)

Category:

```text
Classify the purpose, not the presence of newly added code or tests.
Use fix when restoring expected existing behavior, including an
incorrectly handled edge case. Adding a regression test or handling
that edge case does not by itself make the change a feat.
Use feat for a new user-visible capability beyond correcting existing behavior.
Use docs for documentation-only changes.
Ground the decision in current diffs; use relevant history only as supporting context.
```

Text:

```text
Summarize the primary change in one natural Japanese sentence.
Do not paste test inputs, code fragments, or command options into
ordinary prose unless they are essential to describe the actual change.
Prefer a clear complete sentence over packing details into 48 characters.
Keep the supplied membership, type and breaking judgment unchanged.
```

Thinking:

```text
Use concise, task-focused reasoning. Start with current Git diffs.
Consult relevant history only when needed to resolve purpose or ambiguity.
Stop deliberating once the required decision is supported, then emit the final JSON.
The thinking section is free-form; the JSON contract applies only to the final answer.
```

## Proposed next comparison

保存したH182-017を対照に、まずOFF/grammarありでCategory/Textの指示だけ変更し、32K/2048・weight・sampling・native KVを固定して比較する。次に別実験でLow/grammarありの簡潔な思考指示だけを変更する。全6条件の一括再試行や入力予算・sampling同時変更は不要。Thinkingと最終回答の予算分離や強制境界は別のハーネス変更として扱う。過去結果は上書きしない。
