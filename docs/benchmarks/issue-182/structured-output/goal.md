# H182-010 Qwen構造化出力ハーネス準備

## Goal

履歴付きの新しい3フェーズ入力を使う次回推論比較に向け、モデルから独立した出力制約と、実Schemaから生成するsystem説明を研究ハーネスへ追加する。今回は準備と非推論検証まで。Grouping・Category・Textの推論精度評価やモデル採用は行わない。

## Result / Decision

`output_constraint: json_schema | none`を実装した。新しい研究設定にこのフィールドを明示した場合、両モードへ同じsystem説明を追加する。説明には期待するフィールド名・型・必須／任意・完全なSchemaの形式制約・現在の動的IDを使った構造例を含む。例は実際の変更に関する回答としてコピーしないよう明示する。

- json_schema / Thinking ON：思考区間ではJSON文法を適用せず、終了境界後の回答に適用する。
- json_schema / Thinkingなし・OFF：最初の回答tokenから文法制約を適用する。
- none：最終回答の文法マスクを適用しない。JSONを求めるsystem指示、Thinking境界の追跡・設定済み予算、生成後のJSON解析とホストvalidatorを維持する。自然文を有効なPlanner回答として受理する設定ではない。
- 無効な設定値は拒否する。旧速度診断のstandard-cap等とは併用しない。新しい制約切替を出力1024上限やThinking制御の無効化と結び付けない。
- 通常CLIおよび既存測定設定はこの新規フィールドを持たないため、従来のsystemプロンプトと制約を維持する。

## Models / Conditions

| モデル | Thinking |
| --- | --- |
| Qwen2.5-Instruct 0.5B / 1.5B / 3B | なし |
| Qwen2.5-Coder-Instruct 0.5B / 1.5B / 3B | なし |
| Qwen3.5-2B | OFFのみ |
| Qwen3.5-4B | OFF / ON |

8モデル・9設定。各設定で制約2モードを選択可能なため18設定を準備した。これは全18設定を推論測定するという決定ではない。公開MLX 4bit weight revisionを固定し、8モデル約10.65GBをローカル取得。LFSファイルは公開metadataのSHA256・サイズと照合し、全ファイルのSHA256を記録した。新しい依存は追加していない。

Qwen2.5のsamplingは準備用仮設定temperature=0.7 / top_p=0.8 / top_k=20。公式既定値と完全一致する実装だとは主張せず、次回の測定条件としては未確定。Qwen3.5は以前のモデル別samplingを維持する。Thinking・出力の新しい有限予算やcycle timeoutは採用していない。seed144、eager、KV8/group64。実行プロキシは現在のSchema・messages・phaseを維持し、モデル、ローカルasset、研究用無制限出力設定を適用する。context上限はmodel設定と65536の小さい方で保護する。

## Verification

- Release helper build成功。モデル推論0回。
- 実E2Eでhelperへ届いたGrouping/Category/Text要求を使い、18設定×3フェーズ=54入力でSchema保持とsystem説明生成を非推論監査。制約ON/OFFのmessagesは全組で一致した。
- 8モデル・9設定のロードとCategory/Text合計18入力のtokenizer・Thinking設定・予定KV構成を非推論監査。最大入力2612 tokens、今回の要求context16384以内。新system追加後の長さを確認した。Groupingのモデル別tokenizer監査は今回の既存token-audit入口の対象外であり、54件のsystem/Schema監査と区別する。
- Swift BoundedGeneration最終40テスト成功。Protocol 10テスト成功。文法あり／なし、自由Thinking、自然／予算終了、コピー独立性、無効mode、動的Schema説明を含む。
- Pythonハーネス2テスト成功。8モデルの保持、ON/OFF間の設定一致、実phaseのSchema/messagesを固定groupへ置換しないことを確認。
- Go planning / mlxテスト成功、git diff --check成功。

## Environment corrections

初回SwiftテストはMetalライブラリの探索に失敗した。テストbundleのMacOSディレクトリへの一時metallib配置で実行を確認したが、次のビルドでcodesign対象に混入した。そのファイルを一時領域へ移し、`swift build --build-tests`を正常終了させた。最終の署名済みテストコードを一時領域へコピーし、同居metallib、XcodeのXCTest framework/library検索パス、SwiftPMのswiftpm-testing-helperで40テストを実行して成功した。テスト・チェックの無効化や依存ソースの変更は行っていない。失敗ログも原本保持。

## Artifacts / Limits

公開可能なモデルrevision・asset checksum・内容を含まない監査集計はpreparation.json。会話本文・コマンド本文・Git差分・ローカル個人pathは収録しない。原本、モデル、設定、helperはローカルの専用一時領域に保持する。準備成功はモデル生成、最終JSON成立率、意味品質、速度、E2E推論成功を証明しない。commit/push/main反映は未実施。

[Qwen公式の構造化出力説明](https://qwen.readthedocs.io/en/v2.5/deployment/vllm.html#structured-json-output)は、推論側の制約とsystemまたはpromptでの形式指示の併用を推奨している。今回はvLLMを追加せず、既存MLXランタイムで実装した。
