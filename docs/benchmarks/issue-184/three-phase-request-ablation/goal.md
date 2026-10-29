# H184-004 3フェーズ全体での依頼本文有無の比較

## Goal / Conditions

日本語C184-02、補完済みH184-KIMDEL-REPAIRED-002を用い、依頼ありと最終応答のみの2条件をGrouping→Category→Textで新規測定。最終応答のみは各フェーズでuser_requestsを空配列にし、他の入力bytesを保持。Grouping実応答とCategory実応答を各条件内の後続入力へ渡し、ホストの最終validatorまで評価。前回H184-003の固定Grouping・Category単独の結果と区別する。

Qwen3.5-4B Affine4bit/group64固定revision、Swift Release、Thinking OFF、json_schema、native KV、32768/4096、temperature0.6/top_p0.95/top_k20、penalty1、seed144、eager、時間制限なし、warmup/retry/repair0。日本語の最終応答、補完済み候補、raw_diff、既存system指示は共通。動的グループ／typeは各群の実出力を反映する。partial/ambiguous/history_incompleteは維持。

## Results

| 条件 | 正解Grouping | 正解Category（完全一致群） | Text | 最終validator受理 | 3phase host秒 |
| --- | --- | --- | --- | --- | --- |
| with-request | 3/3 | 2/3 | 到達 | False | 68.399 |
| final-only | 3/3 | 3/3 | 到達 | False | 60.914 |

生成JSONの成立、意味上の分類、Textの表現、最終validator受理は別評価。詳細な各応答と失敗理由はmeasurements.jsonに収録。

## Measurements

| 条件 | phase | 入力tokens | 生成tokens | prefill秒 | decode tok/s | helper wall秒 | MLX peak GiB |
| --- | --- | --- | --- | --- | --- | --- | --- |
| with-request | grouping | 6092 | 52 | 19.265 | 26.48 | 24.494 | 3.149 |
| with-request | category | 4996 | 71 | 15.917 | 24.54 | 20.744 | 3.153 |
| with-request | text | 4880 | 74 | 16.372 | 24.95 | 21.860 | 3.140 |
| final-only | grouping | 5148 | 52 | 17.192 | 24.66 | 22.211 | 3.123 |
| final-only | category | 4052 | 71 | 13.904 | 28.03 | 18.992 | 3.148 |
| final-only | text | 3936 | 74 | 14.110 | 24.75 | 19.631 | 3.120 |

## Validation / Limits

実入力の依頼本文除去を各phaseで監査し、逆置換による非対象bytes一致、最終応答保持、実wireのThinking OFF/json_schema、固定ソースハッシュ維持を確認。監視正常、同時helper最大1。CPU・各host/runtime時間・メモリ・熱は隣接JSONとprivate原本へ保存。単一fixture/seed・各1回、順序は依頼あり→なし。背景負荷・OSキャッシュは固定せず、速度の安定性や一般的精度を断定しない。最終validator受理は意味的な完全正解の保証ではない。raw会話や思考は公開せず、原本保持。App Server再取得と製品コード変更なし。

## Manual Text review

両群とも3件に日本語を含むが、意味的な完全正解を表さない。README要約は実変更と整合。UI要約もtooltipの変更と整合するが、最終応答のみでは表示追加という点が明示されていない。メール要約は依頼ありで「山括括角」という誤記があり、最終応答のみでも「空文字列にも正規化」という不明瞭な表現になっている。実変更は空の山括弧をNone/欠落として扱う修正であり、後者を意味的な改善と判定しない。両群ともREADMEのscope=docsが禁止語に該当し、最終validatorはinvalid_scope。Category改善とText/全体成功を区別する。
