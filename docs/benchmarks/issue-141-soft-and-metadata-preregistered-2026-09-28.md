# Issue #141: 解除可能な relation 候補と Pass 2 診断の事前条件

この文書と benchmark code を推論前にコミットする。[hybrid の結果](issue-141-hybrid-2026-09-28.md)を受け、Pass 1 の解除可能な候補と、Pass 2 の固定 group label 診断を**別々の試験**として実施する。両試験の成功数は合算しない。

## A. Pass 1: file ID を縮約しない soft candidate

| 条件 | 出力 schema と割当対象 | 追加する候補 |
| --- | --- | --- |
| file-centric | 前回と同じ、全 file ID → group label | なし |
| soft-source-test | 同じ file ID schema | `source_test`、`Soft`、`matching_test_path` |
| soft-source-test-import | 同じ file ID schema | 上記と `direct_import`、`Soft`、`observed_import_path` |

候補は prepared Document の edge と file ID からのみ選び、正解 group は入力にも選択にも使わない。候補を prompt に列挙して同じ group の可能性を知らせるが、異なる変更目的なら分離できると明示する。Go は全 file ID の完全割当、未知・重複 key、空 label を前回と同じ方法で検証する。Pass 1 の semantic grouping failure は修復せず、structural failure は最大1回だけ修復する。**この試験では Pass 2 を呼ばず**、Pass 1 の誤結合・誤分割と call 数だけを比較する。

前回と同じ主比較6 fixture に加え、[hybrid 事前登録](issue-141-hybrid-preregistered-2026-09-28.md)の2 guardrail を LLM 推論まで含めて測る。guardrail は同名 source/test が別目的である例と、別目的の変更 file 間に直接 import がある例。どちらも正解は2つの singleton group。guardrail の outcome は主比較6 fixture の分母に混ぜず、別表で示す。候補 edge の true / false pair 数はモデル出力前に正解と照合するが、その値は prompt に含めない。

8 fixture × 3条件 × 2反復の48試行。2巡目は条件順を逆にする。主指標は主比較6 fixture の完全割当、exact grouping、false merge / false split pair、per-file accuracyと、guardrail それぞれの false merge。補助指標は候補 edge 件数・true / false pair、repair・total calls、wall、prompt/output bytes、stop reason。構造的に不正な結果の grouping は欠測とする。

比較するモデルは既存ローカル MLX の `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。各 request の output 上限1024 token、plan cycle timeout 2分、同じ repository input / relation context、2反復。追加モデル・依存・クラウド API を使わない。

事前の候補 edge census は主比較で `multi_commit` 1、`cross_directory` は import 追加時1、`same_directory_independent` 0、`mixed_24` 0、`holdout_split` 2、`holdout_join` 1。guardrail は source/test 側で1 false edge、import 側で import 追加時1 false edge。`mixed_24` の初回 prompt は file-centric 15,281 bytes、両 soft 条件15,412 bytes。soft 条件では候補 list と指示が同時に変わるため、差を edge 情報だけの因果効果とは断定しない。

採用候補となるには、主比較6 fixture で完全割当12/12、false merge 0、false split 6以下、exact 8/12以上を維持し、guardrail 2 fixture でも誤結合0を確認する必要がある。guardrail は小さな合成反例であり、これに通っても一般的安全性や production 採用は確定しない。失敗した場合は現在の soft 候補 prompt が不十分と判定し、別の表現・モデル一般まで否定しない。

## B. Pass 2: 正解 group label を固定した診断

[先行する oracle 比較](issue-141-grouping-first-2026-09-28.md)と同じ `multi_commit`、`mixed_24`、`holdout_split`、`japanese` の正解 group と `G1`、`G2` 形式の固定 label を使う。Pass 1 は実行せず、batch metadata と per-group metadata を比較する。4 fixture × 2方式 × 2反復の16試行。2巡目は方式の順序を逆にする。1024 output token/request、2分/plan、同じ model/backend/input と Go の `planning.Validate()` を使う。

各応答はメモリ上で検証し、JSON/schema 失敗、期待 group 数、観測 metadata 件数、欠落・未知・重複 group の件数、`breaking` 欠落件数、最終 validation failure を**件数・コードだけ**記録する。prompt、raw response、生成された group label・summary は成果物へ保存しない。これで `invalid_metadata_groups` の内訳を調べるが、今回の固定 label と過去の Pass 1 生成 label は異なるため、失敗の全因果を固定 label 試験だけで特定したとは言わない。

主指標は validation 成功率と failure 内訳。補助指標は request・total calls、wall、prompt/output bytes、stop reason、取得できれば output tokens、取得できなければ `unavailable`。per-group が成功しても group 数比例の backend calls を production で許可する判断にはしない。Pass 2 の診断結果は A の Pass 1 exact 件数へ加算しない。

production planner、通常 CLI、SRS は変更しない。
