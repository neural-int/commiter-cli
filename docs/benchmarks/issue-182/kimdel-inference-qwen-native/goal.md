# H182-016 Qwen改訂prompt＋native KV比較

## Goal

Qwen既存promptの基本ルールを保持し、docs定義・scope禁止語・文字数を明示、自動構造例を除外する。native KVへ同時変更し、前回同様18条件を固定fixture・32K/4096で測る。

## Result / Decision

全18セル・45phase測定完了。1194.874秒（約19分55秒）。monitor596件valid、fatalなし。Grouping完全正解4/18は前回と同じQwen3.5-4B全4条件。validator受理は5/18（前回6/18）。正解Grouping+Category+validatorは0/18。Qwen全体の改善・採用を支持する結果ではない。
Qwen4 ONはGrouping/Category正解でscope禁止語を回避したが、Textが英語となりinvalid_summary_language。OFF noneはCategoryも正解へ改善したがscope=docs違反。OFF grammarありはCategoryが1/3へ悪化しscopeも禁止語を使う。小型モデルは目的混在・実装とテストの分離が残る。今回結果を元に追加推論・再試行・条件変更は行わない。

## Conditions / verification

前回と同じ8モデル9Thinking設定×grammarあり/なし。固定revision・sampling・seed144・eager・各phase別Release helper・同じGold。32K入力/Thinking含む4096出力、時間制限なし。warmup/retry/repairなし。実Groupingを後段へ渡す。全要求値と入力+生成が32768以内であることを確認。上限到達0。全18Groupingは同一入力、コマンド14件・会話3件の依頼/最終本文あり。Git snapshotとfixtureは不変。履歴取得partial/history_incompleteとブロックambiguousの制約は残る。
promptは事前提示したQwen改訂版。Groupingの詳細ルールを維持し、Gemma用compactへの置換はしない。構造化出力のフィールド/型/必須/完全Schema指示を維持し、例の意味値のみ除外。Textのja指定は今回も全要求で確認。
KV8を外しnativeを要求。SwiftのkvBits=nil/workingCache=nilでMLX内部native cache生成へ委譲し、requestedKVBitsは未設定。実native KV dtype/bytesは未exportのため容量の直接比較不可。プロンプトとKVを同時に変更しており各効果を単独断定しない。各条件1回、実行時の熱・OS cacheは固定していない。

## Cell measurements

Categoryは正解群のみ採点。—は採点不能。validator受理と意味の正しさを分離。decode速度は生成tokens合計/runtimeGenerationSeconds合計、Thinking込み、load/prefill除外。

| 条件 | calls | 正解群/3 | Category | validator | 秒 | tokens | decode token/s |
| --- | ---: | --- | --- | --- | ---: | ---: | ---: |
| qwen25-05-json_schema | 3 | 1/3 | 0/1 | 受理 | 15.64 | 427 | 99.38 |
| qwen25-05-none | 1 | — | — | 拒否 | 6.75 | 331 | 121.01 |
| qwen25-15-none | 2 | 1/3 | — | 拒否 | 17.52 | 73 | 60.85 |
| qwen25-15-json_schema | 2 | 0/3 | — | 拒否 | 17.07 | 91 | 60.65 |
| qwen25-3-json_schema | 2 | 0/3 | — | 拒否 | 34.33 | 93 | 35.73 |
| qwen25-3-none | 3 | 1/3 | 1/1 | 拒否 | 50.11 | 166 | 35.81 |
| coder25-05-none | 2 | 0/3 | — | 拒否 | 25.59 | 1599 | 84.69 |
| coder25-05-json_schema | 3 | 0/3 | — | 受理 | 10.87 | 109 | 97.42 |
| coder25-15-json_schema | 2 | 0/3 | — | 拒否 | 18.62 | 65 | 56.06 |
| coder25-15-none | 2 | 1/3 | 0/1 | 拒否 | 31.63 | 199 | 38.85 |
| coder25-3-none | 3 | 0/3 | — | 受理 | 70.09 | 128 | 22.49 |
| coder25-3-json_schema | 3 | 0/3 | — | 受理 | 59.85 | 113 | 32.54 |
| qwen2-off-json_schema | 3 | 0/3 | — | 受理 | 36.25 | 193 | 52.94 |
| qwen2-off-none | 2 | 0/3 | — | 拒否 | 25.40 | 194 | 60.22 |
| qwen4-off-none | 3 | 3/3 | 3/3 | 拒否 | 80.36 | 179 | 17.85 |
| qwen4-off-json_schema | 3 | 3/3 | 1/3 | 拒否 | 82.85 | 204 | 18.29 |
| qwen4-on-json_schema | 3 | 3/3 | 3/3 | 拒否 | 301.40 | 5348 | 23.15 |
| qwen4-on-none | 3 | 3/3 | 3/3 | 拒否 | 309.61 | 5424 | 22.49 |

## Comparison

| 指標 | H182-013 Qwen | H182-016 Qwen | H182-015 Gemma |
| --- | ---: | ---: | ---: |
| 条件数 | 18 | 18 | 4 |
| Grouping完全正解 | 4 | 4 | 4 |
| validator受理 | 6 | 5 | 3 |
| Grouping/Category/validator全正解 | 0 | 0 | 3 |

Gemmaは改訂compact、Qwenは従来の詳しいpromptに同じ修正方針を適用。モデル固有のsampling/templateが異なり、同一prompt・単一要因のモデル優劣比較ではない。Gemmaでの改善がQwenへそのまま移るという仮説は今回支持されない。

| Qwen4条件 | 前回秒 → 今回秒 | 前回decode → 今回decode |
| --- | --- | --- |
| qwen4-off-none | 87.37 → 80.36 | 17.16 → 17.85 |
| qwen4-off-json_schema | 87.85 → 82.85 | 16.89 → 18.29 |
| qwen4-on-json_schema | 202.69 → 301.40 | 19.57 → 23.15 |
| qwen4-on-none | 202.82 → 309.61 | 19.31 → 22.49 |

ONのCategoryが3985/4011生成tokens（Thinking raw3913）まで増え、前回1370/1396（Thinking1298）より長い。全体時間増加をnativeの低速化と判断しない。ON Textも日本語→英語へ変化。OFF grammarありはUI=fix、メール=featと逆に分類した。

## Output / Thinking boundary

json_schema: 24phase、helper成功24、JSON解析成功24、Schema適合24。
none: 21phase、helper成功20、JSON解析成功20、Schema適合18。
qwen4-on-json_schema: bounded-grouping output=527/Thinking raw=474/forcedClose=False, bounded-category output=3985/Thinking raw=3913/forcedClose=False, bounded-text output=836/Thinking raw=780/forcedClose=False
qwen4-on-none: bounded-grouping output=529/Thinking raw=474/forcedClose=False, bounded-category output=4011/Thinking raw=3913/forcedClose=False, bounded-text output=884/Thinking raw=780/forcedClose=False

出力上限到達0。Categoryでunresolved_breaking_evidence停止は5条件。実行失敗や停止に対し既存評価器のevaluation_note=KeyErrorがある場合、有効出力がない採点不能として扱い、下位stop_reasonを優先する。

## Performance / resources

helper RSS最大3.051GiB、MLX peak最大3.142GiB（別指標、加算不可）。CPU busy平均12.11%、system GPU範囲[0, 100]%。thermal {'nominal': 56, 'fair': 540}。swap MiB開始/終了/最大[2582.94, 2558.94, 2582.94]。free%範囲[42, 72]。同時helper最大1。
load/prepare/prefill/grammar/decode/host/CPU user-system/prefixを保存。2秒間隔監視。温度実数・消費電力・プロセス別GPU・native配列dtype/bytes未取得。背景処理を含むため単純因果を主張しない。

## Limits / Next

単一fixture各条件1回。一般精度・再現性・prompt/KV単独効果は未確定。次に切り分けるならQwen4のText日本語指定とscope制約の提示方法を対象にする。今回の成果として採用確定は行わない。原本・前回結果保持。生履歴/要求/応答は公開しない。
