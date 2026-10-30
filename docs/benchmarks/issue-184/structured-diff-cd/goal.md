# H184-005 / structured-diff-cd 準備

## Goal / State

ホストの原データを維持し、LLM表示からraw_diffを外して各change_blocksへdeleted_code/added_codeを追加する。C=一行JSON表示、D=同じ内容の文字列内改行だけを実改行にした表示。prepared_not_executed、モデル生成0回。

ハーネス・実行環境は指定表記4-HARNESS-001 / revision 1（登録名H184-HARNESS-001）。登録済みprofile Bのscope制約をC/Dとも有効化。実験番号H184-005、study_keyで過去A/B・scope比較と区別。元実験を上書きしない。

## Projection

各ファイルのraw_diffと各ブロックのdiff_rangeをLLM表示からだけ外し、ブロック単位でdeleted_codeとadded_codeを必須文字列として格納する。追加のみはdeleted_codeを空、削除のみはadded_codeを空にする。file_id、パス、変更状態、言語等、block_id、宣言名が存在する場合の宣言名、履歴参照と照合状態は維持する。

diff --git/index/---/+++/@@といったdiffヘッダーや、変更行の先頭1文字のdiff記号をコード表示から外す。コード内の記号、インデント、空行、行順、末尾改行の有無、日本語、文字列内バックスラッシュは維持する。READMEの本文も同じ2フィールドで扱う。

ホストのraw_diff、diff_range、元リクエスト、抽出コードのSHA-256対応表を非公開バンドルで保持。各diff変更行が重複なくすべていずれかのブロックへ収録されたことを検証。元のヘッダー表現までLLM表示へ復元することは目的とせず、元の監査証拠は別に保持する。今回のU0 fixtureには文脈行がなく、文脈行を含む新たなケースは黙って削除せず準備エラーにする。

Cは有効なJSON文字列、DはJSON風の表示用テキストで文字列内に実改行を含むため厳密なJSONではない。ホストJSONとIPCは有効なJSONのまま。D→Cの復元はbyte一致する。C/Dの論理フィールド内容は完全一致。

## Fixed conditions / Execution plan

補完済みH184-KIMDEL-REPAIRED-002、日本語、最終応答のみ、5ファイル／9ブロック／3contexts。partial/ambiguous/history_incompleteを維持。Qwen3.5-4B Affine4bit/group64固定revision、Swift Release、Thinking OFF、json_schema、Text scope制約ON、native KV、32768/4096、temperature0.6/top_p0.95/top_k20、penalty1、seed144、eager、warmup/retry/repair0、時間制限なし。systemと出力Schemaは前回から変更しない。

実行承認後はC→D、Grouping→Category→Textを各1回新規生成し、各条件の実出力を後続へ継承、最大6生成。Grouping精度、Category、Textの意味・自然さ、最終validator、全フェーズ時間、tokens、CPU/GPU/memory/swap/thermalを分けて評価する。

## Preparation validation

単体テスト9件成功。コード中の複数+/-、正規表現、引用符、バックスラッシュn、Unicode、日本語、インデントと空行、末尾改行なし、範囲重複・欠落・文脈行拒否を検証。元fixture34ファイル検証成功。全3フェーズで5ファイル／9ブロックと履歴を保持。native contract probe6件成功、generation_calls=0。ハーネスのhelper/shader SHA-256一致。

| phase | C 実改行 | D 実改行 | C user bytes | D user bytes |
| --- | --- | --- | --- | --- |
| Grouping | 0 | 100 | 12844 | 12744 |
| Category | 0 | 100 | 9212 | 9112 |
| Text | 0 | 100 | 9294 | 9194 |

上記は準備preview値であり、動的な推論結果を継承した後の測定値ではない。入力tokens・精度・速度は未測定。既存A/Bとの比較にはデータ構造と表示の変更範囲を区別し、C/D内では表示のみを比較する。新たな依存・製品経路の変更なし。非公開の実履歴原文は公開資料へ転載しない。
