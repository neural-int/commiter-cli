# Issue 184 ハーネス登録

| ID | revision | 用途 | 採用状態 |
| --- | --- | --- | --- |
| [H184-HARNESS-001](H184-HARNESS-001.json) | 1 | Text scopeの追加マスク比較。profile BがON | 親ハーネスの保存仕様 |
| [H184-HARNESS-002](H184-HARNESS-002.json) | 1 | 3phase共通でraw_diffを保持し、LLM表示だけ実改行化 | 今回採用する研究用基準 |

H184-HARNESS-002の短縮表記は **4-HARNESS-002 / revision 1**。実験番号H184-005、ケースC184-02とは別の識別子である。旧H184-HARNESS-001は上書きしない。製品mainの既定経路を変更した登録ではない。

## 固定する仕様

- モデル、量子化、Swift Release helper、Metalライブラリは001と同じ。バイナリのSHA-256と依存の固定revisionは002のmanifestへ記録する。
- Thinking OFF、JSON文法制約ON、Text scope追加マスクON、native KV、入力32768・出力4096、temperature=0.6、top_p=0.95、top_k=20、repetition_penalty=1、seed=144、eager。
- Grouping・Category・Textのuser本文を、同じフィールド・配列・値のまま表示変換する。ホストのcanonical JSONと外側IPCは有効なJSONを保持する。
- LLM表示は文字列内LFを実改行にしたJSON風テキストであり、厳密なJSONとして再利用しない。`restore_view`で元のcanonical文字列へbyte単位で復元できる。
- `raw_diff`のファイルヘッダー、`@@`、行頭`+/-`、コード、diff_rangeはすべて保持する。added_code/deleted_code化、ヘッダー削除は採用しない。
- user_requestsは両群共通条件どおり空配列、final_responsesは日本語のまま保持。Categoryのdocs定義、Textのscope禁止語・文字数指示もH184-005で両群共通だったものを維持する。
- 出力JSONのSchema、検証、先行phaseの実結果の後続への継承は変更しない。モデル入力への命令追加や翻訳は行わない。

## 使用方法

既存の測定bundleへ`paths.json`を次の形式で用意する。helperの値だけ実資産のパスへ置換する。`display_format`を指定した場合は実改行・raw_diff保持に固定され、残存した`STUDY_ARM=A`などでは切り替わらない。

```json
{
  "helper": "CALLER_SUPPLIED_RELEASE_HELPER",
  "representation": "raw",
  "display_format": "literal_newlines"
}
```

bundleには測定時と同じ`case.json`と`scope-policy.json`も必要。固定されたhelper/Metalのハッシュ、case設定とmanifest、モデルrevisionを実行前に確認する。親ハーネスのsource hashは歴史的な測定バイナリの出典であり、このチェックアウトのソースから同じバイナリを再ビルドしたという意味ではない。

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
