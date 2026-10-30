# Issue 184 ハーネス登録

| ID | revision | 用途 | 採用状態 |
| --- | --- | --- | --- |
| [H184-HARNESS-001](H184-HARNESS-001.json) | 1 | Text scopeの追加マスク比較。profile BがON | 親ハーネスの保存仕様 |
| [H184-HARNESS-002](H184-HARNESS-002.json) | 1 | 3phase共通でraw_diffを保持し、LLM表示だけ実改行化 | 旧研究用基準（保存） |
| [H184-HARNESS-003](H184-HARNESS-003.json) | 1 | 混合配置＋採用済み表示整理1〜4 | 現在の採用研究用基準 |

H184-HARNESS-002の短縮表記は **4-HARNESS-002 / revision 1**。実験番号H184-005、ケースC184-02とは別の識別子である。旧H184-HARNESS-001は上書きしない。製品mainの既定経路を変更した登録ではない。

## H184-HARNESS-002の保存仕様

- モデル、量子化、Swift Release helper、Metalライブラリは001と同じ。バイナリのSHA-256と依存の固定revisionは002のmanifestへ記録する。
- Thinking OFF、JSON文法制約ON、Text scope追加マスクON、native KV、入力32768・出力4096、temperature=0.6、top_p=0.95、top_k=20、repetition_penalty=1、seed=144、eager。
- Grouping・Category・Textのuser本文を、同じフィールド・配列・値のまま表示変換する。ホストのcanonical JSONと外側IPCは有効なJSONを保持する。
- LLM表示は文字列内LFを実改行にしたJSON風テキストであり、厳密なJSONとして再利用しない。`restore_view`で元のcanonical文字列へbyte単位で復元できる。
- `raw_diff`のファイルヘッダー、`@@`、行頭`+/-`、コード、diff_rangeはすべて保持する。added_code/deleted_code化、ヘッダー削除は採用しない。
- user_requestsは両群共通条件どおり空配列、final_responsesは日本語のまま保持。Categoryのdocs定義、Textのscope禁止語・文字数指示もH184-005で両群共通だったものを維持する。
- 出力JSONのSchema、検証、先行phaseの実結果の後続への継承は変更しない。モデル入力への命令追加や翻訳は行わない。

## H184-HARNESS-002の使用方法

既存の測定bundleへ`paths.json`を次の形式で用意する。helperの値だけ実資産のパスへ置換する。`display_format`を指定した場合は実改行・raw_diff保持に固定され、残存した`STUDY_ARM=A`などでは切り替わらない。

```json
{
  "helper": "CALLER_SUPPLIED_RELEASE_HELPER",
  "representation": "raw",
  "display_format": "literal_newlines"
}
```

bundleには測定時と同じ`case.json`と`scope-policy.json`も必要。固定されたhelper/Metalのハッシュ、case設定とmanifest、モデルrevisionを実行前に確認する。

001のSwift実装・生成経路への接続・前提となるモデル別研究経路・対応テストは、このブランチの`mlx-helper`へ取り込み済み。取り込み元はcommit `374f8f6da57adc84dc230f997afe070d55f160a6`に固定し、ソース一致と検証内容を[移植記録](runtime-import.md)へ記載する。既存のPackage.swift・Package.resolvedは変更していない。測定時のバイナリSHAは歴史的な測定資産の識別子として保持し、再ビルドしたバイナリとのbyte一致を主張するものではない。

```sh
export COMMITER_182_CASE_CONFIG="$BUNDLE/case.json"
export COMMITER_180_KV_BITS=native
export COMMITER_182_PIPELINE=eager
export COMMITER_STUDY_OUT="$OUTPUT"
python3 -B tools/benchmark184/multiline_view.py proxy --bundle "$BUNDLE" < "$PHASE_REQUEST"
```

このproxyは1phaseあたり非推論wire監査を行った後、実生成を1回呼ぶ。出力先は空のprivateディレクトリを使用する。同じ出力先への上書きはしない。入力・実リクエスト・wireには会話本文が含まれるため、公開Gitに置かない。

3phase全体の評価器と実履歴fixture、モデル資産、測定バイナリは別途用意する。この公開チェックアウトだけで過去の実履歴を取得・再構成できるとは主張しない。

## 根拠と比較記録

- [採用の根拠：元diffで一行JSON／実改行表示](../multiline-ab/results.md)
- [非採用：コード分解C/D](../structured-diff-cd/results.md)
- [非採用：全ヘッダー削除](../headerless-ab/goal.md)
- [非採用：hunkヘッダー保持・ファイルヘッダー削除](../hunk-retained-ab/goal.md)

元diffのA/Bは両群Grouping3/3・Category3/3、validator通過。今回の採用は当該fixtureで品質を維持した表示変更として行う。精度向上・一般的な速度向上が証明されたという扱いにはしない。


## H184-HARNESS-003 / revision 1

短縮表記は **4-HARNESS-003**。002を上書きせず、現在の研究用基準を003へ更新する。1〜4の採用範囲・混合配置・固定設定は[manifest](H184-HARNESS-003.json)に記録。従来prefill保護、native KV、同一helper内のモデル/共通文法準備保持。KV prefix再利用は未実装。

1. 完全な出力Schemaはプロンプト内に一回。ランタイムSchemaとフィールド説明は保持。
2. Category規則はCategoryのsystemへ一度だけ置く。Grouping/Textには置かない。
3. LLM表示だけからtask/schema_version/空user_requestsを省略。非空の依頼は保持。
4. LLM表示だけからblock_id/execution_refを省略。ホスト内部には保持。

前提入力は、Grouping/Textの先頭にfiles/history、Categoryではグループ内に完全なfilesを置く「混合配置」のfinalized wire。元のcanonical planner入力を直接このツールへ渡さない。未対応の表記・Schema不一致は失敗させる。フェーズ順序と入力sourceはホストが管理し、履歴内の文章を指示として実行しない。

```sh
python3 -B tools/benchmark184/adopted_projection.py --source "$FINALIZED_MIXED_INPUT" --output "$NEW_PRIVATE_INPUT_DIR"
```

このコマンドは非推論の入力生成だけを行う。入力ファイル01/02/03-request.jsonを読み、元ファイルを変更せず同名の投影結果を新規ディレクトリへ0600で保存する。既存出力先は上書きしない。生履歴入りの入力/出力はprivateで管理する。生成時のPython Proxyは使用しない。

推論比較は固定Release helper/Metal/modelをmanifestのSHAと照合し、case/scope-policyとenvironmentを準備して、Go評価器で --persistent --wire --independent-phases --phase-count 3 を使用する。--wireは完成済みプロンプトなので、研究専用COMMITER_184_LAYOUT_WIRE=1でSchema説明の再追加を避ける。この経路と登録は研究用であり、本番CLIの既定入力を003へ切り替える変更ではない。

[非推論照合](H184-HARNESS-003-verification.json)で、最初の混合配置から1〜4を適用した要求全体が実測最終要求と全3phaseで一致。テスト15件通過。再推論はしていない。[最終結果](../display-ids-omitted/goal.md)はGrouping3/3、Category3/3、合成validator通過。Textの意味精度は完全ではなく、各phase固定入力の評価でE2E成功とは呼ばない。
