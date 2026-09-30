# Issue #143: モデル交換比較の事前登録条件

## 状態

2026-09-30 に人間が主比較、全5モデル、数値条件を確定した。比較推論は未実施。
主比較は #142 の既存条件をそのまま固定する方針で人間が確定済み。
以下を事前固定し、実測結果により調整しない。

## 引き継ぎと解釈

- 起点は `codex/issue-142-candidate-partitions` の `aefc2e6c9db376ccecb40af337f2b4b336211092`。
- 作業ブランチは `codex/issue-143-model-comparison`。
- #141 の局所/cluster contract は結合と分離の両立・安全性・call条件を満たさず、#142 では gold を含む固定二択でも非gold選択が残った。この残存失敗を別モデルへの交換で比較する。
- 重み・architecture・学習・tokenizer・chat template を含むモデル一式の交換効果として扱う。系列や容量の単独効果を推定しない。
- 複数モデルの共通失敗だけでは入力情報不足を特定できない。共通 contract・grammar・予算・template/thinking・合成goldの曖昧さ・モデル能力不足も残る。
- 未使用合成fixtureへの転移と、実開発者判断/productionの精度・採用判断は分ける。

## 確定した主比較

- `issue142GenericForcedInput` による complete-partition / fixed two-candidate forced-choice。noneを許さない。pair-localやdifference-onlyを混ぜない。
- 各fixture・候補順で同じ system/user bytes、candidate IDs/grouping、schema bytes を全モデルへ渡し、hashで一致を検証する。
- モデル固有の tokenizer/chat template は正規のものを使う。同一token列を仮定しない。第一段階ではthinkingやsamplingのモデル別最適化をしない。
- helperは `/private/tmp/issue142-bidirectional-helper/commiter-mlx-helper`。確認したSHA-256は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。
- `mlx-helper/Package.resolved` の mlx-swift-lm pinは `c6446cf7bfb7cea76408013b614d4b2c530eaa03`。同revisionのGuidedGenerationLoopはgrammar拘束の上でargMaxを使う。JSON拘束は生成開始から適用する。Graniteの公式推奨sampling/thinkingでの能力評価とは異なる。
- 同じhelperを使ったロード/推論互換性は未検証。失敗はbackend/runtime failureとして残し、別backendへ暗黙に差し替えない。

## 公開配布の確認値

公開Hugging Face APIのrevision/config/全配布ファイルsizeを2026-09-30に確認。下記bytesは全配布ファイルの合計であり、必要ファイルだけを取得した際の実取得量ではない。

| モデル | 配布repository | revision | 全配布bytes | model_type | 量子化 |
| --- | --- | --- | ---: | --- | --- |
| Ministral baseline | mlx-community/Ministral-3-3B-Instruct-2512-4bit | a962dcb09eee4169c890e544c9eb938f1113fdee | 既存cache 2779150244 | mistral3 | 4bit/group64/affine |
| Granite 4.2 3B | ibm-granite/granite-4.2-3b-q4-mlx | 0c6f39b1827afd5eb2c1c3b13751929857434953 | 2066182986 | granite | 4bit/group64/affine |
| Phi-4-mini-instruct 3.8B | mlx-community/Phi-4-mini-instruct-4bit | ac1c269cb4222a4e136a3d09edad301056c1f36a | 2180067259 | phi3 | 4bit/group64 |
| Nemotron 3 Nano 3.97B | mlx-community/NVIDIA-Nemotron-3-Nano-4B-4bit | c4d79ba1901d99806ef757642a552acebb851a35 | 2254295077 | nemotron_h | 4bit/group64/affine |
| Gemma E4B | mlx-community/gemma-4-E4B-it-4bit | 475b9088d29754a3379866cf5aeb6b41acd313c2 | 5179241512 | gemma4 / gemma4_text | 4bit/group64/affine |

Gemmaはeffective 4.5B / embeddingsを含むtotal 8B。追加4モデル全配布合計は11,679,786,834bytes。ダウンロード自体は人間が明示許可済み。全5モデルを結果による途中除外なしで比較する。

## 確定した数値条件

- 既知4fixtureと、条件確定後に作成する新規holdout8fixtureを分ける。
- 既知例: `verify_misleading_relation` weighted top-2、`lexical_crossdir_pairs` weighted top-2、`new_stem_doc_diverged` capped top-2、`new_paraphrase` enumerated capped top-2。それぞれ#142の固定候補と入力hashとの一致を確認する。
- 新規8例はsplit側4例（うちmisleading structural relationのguardrail2例）、正しいjoinを要する4例。gold/許容集合と2候補を推論前に固定する。goldはprompt/候補生成の根拠へ渡さず、評価側だけが持つ。候補生成精度は主比較の対象にしない。holdoutは評価側で事前に2候補を手動固定し、gold包含を設計上保証したselector対照実験である。generatorを評価した結果ではない。
- 各fixtureは正順・逆順・逆順・正順の4call。各順序を2回ずつ測る。全5モデルで240評価call。
- 2048 output tokens/call、120秒/call、repair/retryなし。隠れたretryや成功するまでの再実行はしない。追加warm-upはしない。wallはhelper起動/モデルロードを含むbackend call全体として計測し、環境を別記する。
- baselineも同時期に再測定する。実行順はモデル間で固定ローテーションし、順序をmanifestに保存する。GPU推論は同時実行しない。
- 次段候補はholdout gold成功28/32以上、かつ同時期baselineより4call以上改善、guardrail false merge 0、正しいjoin・false split・完了率にbaseline比悪化なし、構造的不正0、holdout wall中央値がbaselineの2倍以内を全て満たすこと。
- gold成功率は全予定callを分母とし、未完了/不正出力も失敗に含める。確定partitionがない試行のfalse merge/splitは算出不能とする。valid試行のpair集計と全callの失敗率を両方記録する。
- fixtureごとの4/4一致・方向差・反復差も記録する。反復callを独立した未使用fixtureとして扱わない。少数合成fixtureの記述的比較であり統計的一般化を主張しない。
- 閾値未達のモデルはproduction不可一般とは扱わず、この固定contract/予算で次段候補条件を満たさなかったと記録する。

## 記録とquality gate

- JSONLは数値・file/candidate IDs・hash・failure codeのみ。raw prompt/response/生成summaryは保存しない。output token数を取得できない場合はunavailable。
- 事前fixtureと条件manifestをモデル実行前に固定し、必要なbenchmark専用実装を追加した場合はgo test ./...、go vet ./...、git diff --checkを通す。
- production planner、SRS、既定model/backend、通常CLIは変更しない。
- 検証終了後のIssueコメントは観測値・実行条件・失敗・未測定事項に限定する。第二段階の推論は今回の条件へ混ぜない。

## 公式情報

- https://huggingface.co/ibm-granite/granite-4.2-3b
- https://huggingface.co/ibm-granite/granite-4.2-3b-q4-mlx
- https://huggingface.co/microsoft/Phi-4-mini-instruct
- https://huggingface.co/google/gemma-4-E4B-it
- https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Nano-4B-BF16

## 固定fixtureと実行順

- 新規8例の入力・2候補・gold・gold根拠は `issue-143-holdout-2026-09-30.json`。path/diffは合成fixtureであり実repository内容ではない。
- `-issue143 -describe` の出力を `issue-143-manifest-2026-09-30.json` に固定する。known/holdoutの全12例、5model pin、helper hash、2方向のprompt/schema hash、4巡の候補順を含む。実行時は再構築manifestが一致しない場合に停止する。
- 実行順はrun→fixture→modelの順。model位置を `(position + fixture_index + run_index) % 5` で回転する。run候補順は正順・逆順・逆順・正順。
- holdout gold C001/C002は各4例、join/splitは各4例、guardrailはsource/testとcode/docs basenameの2例。
- 既知例のhistorical hashと候補の一致、schema/gold非漏洩、固定holdoutの構造を推論前のtestで検証する。
- この段階の採否は後続end-to-end候補への進行条件でありproduction採用条件ではない。

実行環境はApple M3 / unified memory 16GiB / macOS 26.6.2 arm64 / Go 1.27.1。全12fixtureのcontext設定は8192tokens。取得したmodel cacheとhelperを使い、モデルのダウンロード時間はinference wallに含めない。
