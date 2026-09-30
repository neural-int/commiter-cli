# #143 候補表現比較の観測結果（2026-09-30）

384callを実行し、completed・有効候補378件、max_tokens6件だった。事前進行基準を全て満たした所属表armは0件。従来arm192callの選択ID・停止理由は前回比較の対応する192callと全件一致した。

## 固定条件と検査

- 条件・manifest・コードを4aab5cbに固定した後に実行。source commit: 4aab5cb60d082d32ae05af4586f8e60c0f4cf9b6
- 既使用の既知4例＋元holdout8例、4モデル×2arm×12例×正逆各2回=384call。新規holdoutではない。Nemotronは前回internal_errorの別互換性問題として今回の品質比較には実行していない。
- 前回4モデルの同じrevision・4bit、helper hash da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48。Apple M3 / unified memory 16GiB。詳細は環境JSON。
- partition-listは従来入力。file-membershipはgroup列挙をファイル別所属表へ可逆変換し、systemに読み方を追加。候補ID列順とgroup_idsが対応し、同一候補内で同group IDなら同commit。正逆で列順を反転、schema enum順は固定。
- task・repository_input bytes・差分本文・relation context・候補IDとpartition集合・gold・output schemaを固定。round tripと非漏洩を検査。
- 新表現と説明文を含むcontract一式の比較。説明文・長さ・token化の単独効果は未分離。helper内の実token列・token count・context取り込みは未確認。
- 2048 output tokens、120秒/call、context設定8192、warm-up/repair/retryなし。backend wallはhelper起動・モデルロード込み。GPU同時推論なし。model/arm順を固定ローテーション。
- 全384行の一意性・sequence・入力hash・model pin・helper hash・予算検査成功。前回240callとgold/fixture/基準は変更していない。

## 選択と完了

goldの分母は予定call全件。未完了を成功に含めない。

| モデル | arm | 既知gold | 元holdout gold | 全有効call | 元holdout有効 | max_tokens |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| Ministral | partition-list | 0/16 | 14/32 | 48/48 | 32/32 | 0 |
| Ministral | file-membership | 4/16 | 16/32 | 46/48 | 30/32 | 2 |
| Granite | partition-list | 8/16 | 16/32 | 48/48 | 32/32 | 0 |
| Granite | file-membership | 4/16 | 16/32 | 48/48 | 32/32 | 0 |
| Phi | partition-list | 8/16 | 16/32 | 48/48 | 32/32 | 0 |
| Phi | file-membership | 4/16 | 14/32 | 44/48 | 32/32 | 4 |
| Gemma | partition-list | 10/16 | 12/32 | 48/48 | 32/32 | 0 |
| Gemma | file-membership | 8/16 | 18/32 | 48/48 | 32/32 | 0 |

Phi所属表は既知new_stem_doc_divergedの4/4でmax_tokens、Ministral所属表は元holdout h143_two_features_interleavedの逆順2/2でmax_tokens。未完了のpartitionはなく、false merge/splitは算出不能。停止原因となる生成内容を結果として記録しておらず、内部原因は特定していない。max_tokens以外の未完了・timeoutは0件。

## 元holdoutのpair・join・方向

| モデル | arm | false merge | false split | join成功 | guardrail false merge | 方向差/8 | 反復差/8 | 不完全/8 | stableだがgold不一致/8 | wall中央値 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Ministral | partition-list | 6 | 24 | 4/16 | 2 | 3 | 0 | 0 | 3 | 4.911s |
| Ministral | file-membership | 8 | 10 | 10/16 | 4 | 1 | 0 | 1 | 3 | 5.341s |
| Granite | partition-list | 14 | 16 | 8/16 | 4 | 8 | 0 | 0 | 0 | 4.030s |
| Granite | file-membership | 14 | 16 | 8/16 | 4 | 8 | 0 | 0 | 0 | 4.276s |
| Phi | partition-list | 14 | 16 | 8/16 | 4 | 6 | 0 | 0 | 1 | 4.390s |
| Phi | file-membership | 10 | 16 | 8/16 | 6 | 1 | 0 | 0 | 4 | 4.175s |
| Gemma | partition-list | 12 | 20 | 8/16 | 8 | 2 | 0 | 0 | 4 | 6.223s |
| Gemma | file-membership | 6 | 16 | 8/16 | 4 | 7 | 0 | 0 | 0 | 6.652s |

方向差は各方向2反復内で同じID、方向間で異なる例。反復差は同方向2回でIDが変わる例。不完全は4callのどれかが未完了/不正。無回答を方向依存の解消とは扱わない。pair件数は有効partitionだけのunordered file pair合計。Ministral所属表は30有効call、他は32であり、未完了2callを誤り0として比較しない。

- Ministral: gold14→16、join4→10、方向差3→1だが不完全例1、完了32→30、guardrail false merge2→4。
- Granite: gold16→16、方向差8→8、先頭列選択32/32→32/32。元holdoutのpairとguardrail値も同じ。
- Phi: 方向差6→1、gold16→14、guardrail false merge4→6。所属表でsource_test_independent/shared_directory_independent/atomic_validation/paraphrased_featureの4例は正逆4call全てgold不一致の同じID。
- Gemma: gold12→18、false merge12→6、false split20→16、guardrail false merge8→4。一方、方向差2→7。

この既使用例・helper・model/template・生成条件での観測であり、semantic理解・系列/容量の原因・入力不足を特定した結果ではない。

## 全12例の方向・提示位置

| モデル | arm | 方向差/12 | 反復差/12 | 不完全/12 | 先頭候補/列選択 / 有効call |
| --- | --- | ---: | ---: | ---: | ---: |
| Ministral | partition-list | 3 | 0 | 0 | 18/48 |
| Ministral | file-membership | 3 | 0 | 1 | 30/46 |
| Granite | partition-list | 12 | 0 | 0 | 48/48 |
| Granite | file-membership | 10 | 0 | 0 | 44/48 |
| Phi | partition-list | 8 | 0 | 0 | 12/48 |
| Phi | file-membership | 1 | 0 | 1 | 20/44 |
| Gemma | partition-list | 3 | 0 | 0 | 30/48 |
| Gemma | file-membership | 11 | 0 | 0 | 46/48 |

## Fixture別gold成功

各セルは従来 / 所属表。各armの分母は4。

| fixture | Ministral | Granite | Phi | Gemma |
| --- | ---: | ---: | ---: | ---: |
| verify_misleading_relation | 0/4 / 0/4 | 2/4 / 2/4 | 2/4 / 0/4 | 0/4 / 2/4 |
| lexical_crossdir_pairs | 0/4 / 2/4 | 2/4 / 2/4 | 2/4 / 4/4 | 4/4 / 2/4 |
| new_stem_doc_diverged | 0/4 / 0/4 | 2/4 / 0/4 | 0/4 / 0/4 | 2/4 / 2/4 |
| new_paraphrase | 0/4 / 2/4 | 2/4 / 0/4 | 4/4 / 0/4 | 4/4 / 2/4 |
| h143_source_test_independent | 2/4 / 0/4 | 2/4 / 2/4 | 2/4 / 0/4 | 0/4 / 2/4 |
| h143_stem_docs_independent | 4/4 / 4/4 | 2/4 / 2/4 | 2/4 / 2/4 | 0/4 / 2/4 |
| h143_shared_directory_independent | 0/4 / 0/4 | 2/4 / 2/4 | 2/4 / 0/4 | 0/4 / 2/4 |
| h143_two_features_interleaved | 4/4 / 2/4 | 2/4 / 2/4 | 2/4 / 4/4 | 4/4 / 4/4 |
| h143_atomic_validation | 0/4 / 2/4 | 2/4 / 2/4 | 0/4 / 0/4 | 0/4 / 2/4 |
| h143_crossdir_documentation | 2/4 / 4/4 | 2/4 / 2/4 | 2/4 / 4/4 | 2/4 / 2/4 |
| h143_paraphrased_feature | 0/4 / 0/4 | 2/4 / 2/4 | 2/4 / 0/4 | 4/4 / 2/4 |
| h143_crosscomponent_feature | 2/4 / 4/4 | 2/4 / 2/4 | 4/4 / 4/4 | 2/4 / 2/4 |

## 事前基準との照合

条件は元holdout gold28/32以上・同時期Ministral従来arm比+4call以上、同モデル従来arm比の方向差2例以上減、guardrail false merge0、join/false split/完了率のMinistral従来arm比悪化なし、構造不正0、wall中央値2倍以内、同モデル比反復差増加なし。

| 所属表モデル | 未達条件キー | 全条件 |
| --- | --- | --- |
| Ministral | gold_at_least_28, gain_at_least_4_over_ministral_control, guardrail_false_merge_zero, completion_no_worse | 未達 |
| Granite | gold_at_least_28, gain_at_least_4_over_ministral_control, direction_decrease_at_least_2, guardrail_false_merge_zero | 未達 |
| Phi | gold_at_least_28, gain_at_least_4_over_ministral_control, guardrail_false_merge_zero | 未達 |
| Gemma | gold_at_least_28, direction_decrease_at_least_2, guardrail_false_merge_zero | 未達 |

基準は元holdoutに適用。Phiの既知4call未完了は全call完了率・総wallに記録し、元holdout32/32完了から全call完了とは主張しない。進行条件であってproduction採用基準ではない。

## 全callの費用

| モデル | arm | calls | backend wall合計 | system+user bytes | 完了output bytes |
| --- | --- | ---: | ---: | ---: | ---: |
| Ministral | partition-list | 48 | 235.359s | 111,420 | 1,226 |
| Ministral | file-membership | 48 | 441.329s | 131,324 | 1,226 |
| Granite | partition-list | 48 | 205.818s | 111,420 | 1,344 |
| Granite | file-membership | 48 | 223.206s | 131,324 | 1,344 |
| Phi | partition-list | 48 | 206.789s | 111,420 | 1,104 |
| Phi | file-membership | 48 | 584.868s | 131,324 | 1,030 |
| Gemma | partition-list | 48 | 298.139s | 111,420 | 1,200 |
| Gemma | file-membership | 48 | 318.560s | 131,324 | 1,200 |
| 計 | | 384 | 2514.069s | 970,976 | 9,674 |

全384callでoutput token countはunavailable。input実token count/列は未取得。完了output bytesは未完了生成量を含まず、総wallには未完了の費用を含む。構造不正・unknown ID・不正JSON・複数選択0。候補の完全割当は固定partitionの構造であり、Pass2や最終plan成功率ではない。

## 再現と未測定事項

```sh
go run ./tools/benchmark110 -issue143-contract -helper /private/tmp/issue142-bidirectional-helper/commiter-mlx-helper > docs/benchmarks/issue-143-contract-comparison-2026-09-30.jsonl
python3 tools/benchmark110/report_issue143_contract.py .
```

1行目は新たな384callを実行して結果を上書きする。既存helper/model cacheが必要。再集計のみなら2行目。
go test ./...、最終serializer修正後のgo test ./tools/benchmark110、go vet ./...、hash/partition同値性/非漏洩/384行一意性検査成功。再集計scriptのsyntax検査成功。
人間の例ごとの独立gold判断は未記録。入力十分性は未確定。追加情報・Nemotron診断・公式推奨モデル別生成条件・Pass2+planning.Validate end-to-end・production精度は未測定。依存追加・model download・production/SRS/既定model/backend変更なし。結果JSONLにはraw prompt/response/生成summaryを保存していない。
