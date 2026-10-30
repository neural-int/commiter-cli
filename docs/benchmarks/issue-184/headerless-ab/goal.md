# H184-005 / headerless-ab

## Conditions

登録H184-HARNESS-001 revision1（指定名4-HARNESS-001）、profile B scope制約ON、日本語・最終応答のみ・C184-02。元fixtureと元リクエスト保持。A=元raw_diff、B=raw_diffからdiff/index/パス/hunkヘッダーだけを除去。一行JSONとraw_diffフィールド、行頭+/-、新旧コード、フィールド配置は維持。diff_range数値は新しい行位置へ付け直す。これはヘッダー除去と必要な位置数値更新の組であり、両者は独立していない。

Qwen3.5-4B 4bit固定revision、Swift Release、Thinking OFF、native KV、32768/4096、temperature0.6/top_p0.95/top_k20、penalty1、seed144、eager、時間制限なし、warmup/retry/repair0。各群Grouping/Category/Text新規計6生成。

## Results

AはGrouping3/3、Category3/3、最終validator通過。BはGrouping3/3、Category2/3、最終validator通過。両群ともTextまで完了。Categoryの誤りをvalidator通過と混同しない。単一fixture・seed・各1回。

| 条件 | host秒 |
| --- | --- |
| A | 67.754 |
| B | 56.792 |

| 条件 | phase | 入力tokens | 出力tokens | decode tok/s |
| --- | --- | --- | --- | --- |
| A | 01 | 5148 | 52 | 26.33 |
| A | 02 | 4052 | 71 | 25.96 |
| A | 03 | 3936 | 85 | 17.25 |
| B | 01 | 4507 | 52 | 27.34 |
| B | 02 | 3411 | 71 | 27.29 |
| B | 03 | 3295 | 87 | 17.85 |

## Interpretation

構造化フィールドを使わずヘッダーを除いた条件でもCategoryの悪化が観測された。したがってフィールド分解やadded_codeという名前だけを主因とする証拠は得られず、ヘッダー除去を優先して調べる必要がある。特定のヘッダー文字列、行位置数値、入力長・配置変化のどれが効いたかは未分離。前の「ヘッダー削除の直接根拠は弱い」という見立てを更新する。監視異常なし、原本保持、追加試行なし。
