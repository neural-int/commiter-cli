# H184-005 Textのscope生成制約A/B

ハーネス番号: **H184-HARNESS-001 / revision 1**（[登録仕様](../harnesses/H184-HARNESS-001.json)）。Bが改善版、Aが同じ実装の対照。

## Goal / Conditions

H184-004の依頼あり／最終応答のみ、各保存済みText入力を固定し、A=既存JSON文法、B=同じJSON文法とscope値制約の積で比較した。Grouping/Categoryは各元条件の実応答を再生、Textのみ各1回計4生成。A/Bで同じ新Swift Releaseバイナリを使用し、モデル可視messagesとSchemaが保存済みwireと一致することを検証。過去のバイナリとの速度比較ではない。

C184-02、Qwen3.5-4B 4bit固定revision、Thinking OFF、native KV、32768/4096、temperature0.6/top_p0.95/top_k20、penalty1、seed144、eager、時間制限なし、warmup/retry/repair0。順序は依頼ありA→B、最終応答のみB→A。補完fixtureのpartial/ambiguousは維持。

## Implementation / Validation

固定xgrammarはJSON Schema notと否定先読みが未対応で、pattern使用時には文字数制約を無視する。このためSchemaを変更せず、既存grammar maskにscope値のprefix検査を重ね、禁止値を完成させるトークンをサンプリング前に除外した。出力後の置換ではない。JSON文字列を解釈し、type/summaryは対象外、空白除去・小文字化はGo validatorに合わせた。禁止一覧はGo allowedTypes定義から生成して固定。研究用フラグのみで有効化し、本番既定動作は変更しない。

境界・回帰の非推論テスト13件成功。大文字・空白・Unicode escape、同名type、summary中の文字列、別名docs-api、文字数・停止状態を確認。モデル・helper・入力・policyのハッシュ維持と正常監視、同時helper最大1を確認。mask除外数は候補トークンを除外した延べイベント数で、生成トークン数ではない。

## Results

| 元Text入力 | 文法条件 | 生成scope | 最終validator | mask除外数 |
| --- | --- | --- | --- | --- |
| with-request | A | G001=docs, G002=ui, G003=mail | False | なし |
| with-request | B | G001=docs/README.md, G002=app/ui/widget, G003=mail/store/ | True | 6 |
| final-only | B | G001=docs/README.md, G002=app/ui, G003=mail/store | True | 6 |
| final-only | A | G001=docs, G002=ui, G003=mail | False | なし |

## Measurements

| 条件 | 入力tokens | 生成tokens | prefill秒 | decode tok/s | grammar setup秒 | Text helper wall秒 | MLX peak GiB |
| --- | --- | --- | --- | --- | --- | --- | --- |
| with-request-A | 4880 | 74 | 18.842 | 24.24 | 1.159 | 27.610 | 3.140 |
| with-request-B | 4880 | 83 | 16.459 | 17.08 | 1.743 | 24.908 | 3.140 |
| final-only-B | 3936 | 85 | 13.250 | 16.54 | 1.862 | 21.886 | 3.120 |
| final-only-A | 3936 | 74 | 13.466 | 24.34 | 1.107 | 19.563 | 3.120 |

## Limits

固定入力ごとに各1回であり、一般化や安定した速度差を主張しない。追加prefix検査のCPU負荷とquote-token準備も測定時間に含む。最終validator受理はCategoryやsummaryの意味的正解を保証しない。依頼あり元Categoryは2/3、最終応答のみ元Categoryは3/3で固定。新たなGrouping/Category精度として集計しない。原本保持、追加推論・再試行なし。

## Manual Text review

Bの両条件はscope=docs/README.mdとなり最終validatorを通過した。これは禁止語docsから別名への生成であり、事後置換ではない。ただし依頼ありBのメールsummaryには「山括括値」の誤記、scopeにはmail/store/という末尾slashが残る。最終応答のみBも「角括弧」「空文字列」という説明で、空の山括弧値を欠落扱いにする変更を正確に限定できていない。文章意味品質の全正解とは判定しない。BでCategoryの誤分類を補正したわけでもなく、依頼ありのCategoryは2/3のまま。

Bのdecode速度低下が両入力で観測されており、追加prefix検査のCPU負荷は今後の改善対象。単発測定でprefillやwallの変動もあるため、wall全体の優劣とdecodeの負荷を分けて扱う。
