# Issue 184 ハーネス登録

現行のscope制約比較ハーネスを **H184-HARNESS-001 / revision 1** として登録する。実験番号H184-005、ケース番号C184-02とは別の識別子で、実装・固定設定・測定バイナリの対応を表す。新たな条件変更ではrevisionまたは番号を更新する。

- [固定設定・ソースおよび測定バイナリSHA-256](H184-HARNESS-001.json)
- [検証報告](../text-scope-grammar/goal.md)

Bが今回の改善版、Aは同一バイナリの対照。ScopeValueConstraintの追加マスクは研究用のbounded-textにのみ接続し、本番の既定動作は維持する。プロンプトと出力SchemaはA/B共通。JSON Schemaのnotを有効化した実装ではなく、既存JSON文法にscope値の検査を重ねる。

実行には、Release helper、モデル資産、保存済みText要求、ケース設定、禁止語配列を指定する。ケース設定はmanifestのモデル・sampling・Thinking・output_constraintに一致させ、禁止語配列はGoのallowedTypes定義と一致させる。

```sh
export COMMITER_182_CASE_CONFIG="$CASE_CONFIG"
export COMMITER_182_NEUTRAL_EXAMPLES=1
export COMMITER_180_KV_BITS=native
export COMMITER_182_PIPELINE=eager
export COMMITER_184_SCOPE_POLICY="$SCOPE_POLICY"
export COMMITER_184_SCOPE_CONSTRAINT=1
"$HELPER" --benchmark-single < "$TEXT_REQUEST"
```

AはCOMMITER_184_SCOPE_CONSTRAINT=0。プロンプトの確認には同じ要求を--benchmark-output-contractへ渡す（生成0回）。BのscopeConstraintRejectedTokensは除外した候補の延べ数で、生成トークン数ではない。

測定に使用した実履歴・要求・token IDs・バイナリは非公開原本に保持し、公開Gitには含めない。公開情報だけで同じ実履歴入力を再構成できるとは主張しない。
