# Iteration26: architecture探索残余の監査

2026-10-06時点。追加推論0、gold/既存測定/production変更なし。Iteration25 Next Stepsに従い、現worktreeのIteration1〜25レポートとGitHubの既存Issueを確認した。これは候補選定または普遍的探索消尽の証明ではない。

## 既知evidence

| family / source | 観測上の棄却根拠 | 次試験で繰り返さないもの |
|---|---|---|
| #139/#140 intent discoveryとsingle-pass input | 6件中5件のdiscovery未完了、独立目的のover-merge、atomicity指示でも改善なし | raw自由形式のintent生成、指示文だけの追加 |
| #141 grouping-first/file-centric/local cluster | over-merge/over-splitの両立未達、small cluster guardrail exact4/4→0/4 | 局所判定の単純追加、soft edge hard縮約 |
| #142 bounded complete partition candidate selection | 候補欠落/順位でgold脱落/正解候補がある二択でも非gold選択、提示順依存 | graph重み/lexical規則の反復調整、正解候補手動注入、forced choiceによる回復 |
| #146 bounded local windows/bridge/all-pairs | 6files整合したFM12、8/12files矛盾、費用増 | bounded-window reconciliation再実装、監査回数を増やせば正解になるという前提 |
| #143とcurrent baseline | Iteration10 serialized4filesは現行・H5ともexact/Validate成功。#143最新コメントはtype目的の可視根拠に関する別評価で未完了 | typeの不可観測性を本Issue groupingの入力不足へ転用、4file成功を拡大証明へ転用 |

参照: [#140 initial](https://github.com/neural-int/commiter-cli/issues/140#issuecomment-5853482243)、[#140 final](https://github.com/neural-int/commiter-cli/issues/140#issuecomment-5856979961)、[#139](https://github.com/neural-int/commiter-cli/issues/139#issuecomment-5866764811)、[#141](https://github.com/neural-int/commiter-cli/issues/141#issuecomment-5866759525)、[#142 final](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5901137417)、[#146 final](https://github.com/neural-int/commiter-cli/issues/146#issuecomment-5991726449)、[#143 live latest](https://github.com/neural-int/commiter-cli/issues/143#issuecomment-5966794785)。Issue143の最新stateを他threadで更新/実装しない。

## H1〜H18の棄却根拠

| hypothesis / iteration | 改善観測 | 未達 / 棄却根拠 |
|---|---|---|
| H1 per-file semantic IR / 1〜2 | 4/6/holdout6一致 | cross12 FS24、13calls |
| H2 batch IR/raw global / 3 | cross12 FS24→8 | batch6 empty IR停止、batch/raw12ともFS8 |
| H3 host facts / 4 | 観測をhostへ保持 | cross12 FS20 |
| H4 AST calls / 5 | independent6分離 | cross12 FS16 |
| H5 test assertions / 6〜7 | forward6/12一致、7/9 observations一致 | weak16 FM120、reverse12 FS24 |
| H6 canonical IR / 8 | 非意味的入力順を決定化 | cross12 FM36/weak16 FM120 |
| H7 global generation + existing routes / 9〜10 | Ministral Reasoning weak16一致、baseline4最終plan成功 | Gemma native1024も不一致、Qwen3.5/Granite FM120、Phi JSON不正、Nemotron internal_error、Instruct cutoff、Reasoning cross12 FS30 |
| H8 observed contract output / 11・13 | weak16一致 | cross12根拠coverage停止、draft自体FS24 |
| H9 observed anchor assignment / 12・14 | weak16一致、output費用減 | Gemma cross12 FS24/Qwen8 FS30 |
| H10 contract records / 15 | weak16一致 | cross12 FS24 |
| H11 per-record semantic delta / 16 | 抽出と割当責務分離 | cross12 FS24、費用増 |
| H12 snapshot observer / 17〜19 | 24実行値一致、unsupported unknown、weak16/guardrail一致 | cross12 FS24、有限inputはpurpose証明ではない |
| H13 host contrasts / 20 | 有限値比較をdeterministic化 | cross12 FS24 |
| H14 contrast + Qwen8 / 21 | weak16/guardrail一致 | cross12 FS30 |
| H15 abstract purpose G-ID / 22 | 観測IDと出力group identity分離 | weak16 FM120/cross12 FM36/guardrail FM12 |
| H16 purpose + bounded generation / 23 | backend生成完成 | weak16 unresolved、cross12 FM36/guardrail FM12。nativeとgrammarは交絡 |
| H17 purpose + Qwen8 / 24 | weak16/guardrail一致 | cross12 FS30 |
| H18 bounded global discovery + assignment / 25 | 2stage全完成、guardrail一致 | weak16 FM120/cross12 FS30、費用増 |

失敗は異なるcheckpoint/入力/taskの有限観測。内部model理由や任意入力精度、全architectureの不可能性は不明。unknown/invalidはFM0成功へ変換しない。goldはuser指定のdiff/code観測根拠を維持。fresh protocol8は採用holdoutとして未推論。

## 残余と判断

| 残余案 | failureへ作用する根拠 | 判断 |
|---|---|---|
| prompt/seed/native budget/graph weight sweep | 新しい観測や責務差がない | 行わない |
| 任意組合せのmodel×task総当たり | 未測定だけでは改善根拠にならない。既存routeでsplit/merge失敗あり | 行わない |
| hard runtime/call union、test-greenで境界確定 | shared-callee guardrail、Iteration10のgreen中間partitionで既に反証 | 対象外 |
| trusted author intent追加、gold修正 | userはdiff/code観測のみを指定 | 対象外 |
| global候補selectionにhost contract/test/snapshot/contrastを全件保持 | #142はraw中心・旧モデル。今回のdirect assignmentは広いmergeとfine splitを繰り返す。候補比較なら自由なpartition生成をhostへ戻し、目的区別だけをglobal modelへ分離できる | H19として最小の独立比較を残す。成功保証なし |
| 新checkpoint/finetuning等 | 今回のfamilyを成功させるcapabilityの独立根拠は未確認。新取得/学習は容量/依存/許可判断が必要 | H19結果前には拡げない |

H19はcandidate familyそのものの一般的再発明ではなく、新しい観測representationと比較taskの組合せ評価。#142で正解候補があっても選択失敗した反例を保持し、既存失敗を無視しない。候補品質とselector品質を分離して評価する。

## H19で先に固定する契約

- 候補はsingleton、観測source/test対応のcomponent、同対応+観測source caller/callee component、all-filesの最大4 complete partitions。重複はlabel/orderに依存せず除去。元file集合全件ちょうど1回とunknown/duplicate/missing拒否。graph weight/fixture名/gold/期待group数は生成に使用しない。
- source/test対応とcallsは候補生成用soft evidence。候補採用はglobal selectorの明示選択後のみ。どの候補もmust-link boundaryにしない。これは少数候補generatorで、任意gold包含を保証しない。
- weak16/cross12/guardrailを先にLLM0で候補包含診断。どれかgold欠落なら選択測定へ進まずgeneratorの不足として記録。goldを候補へ追加せず、ルールをそのcaseへ調整しない。
- 全包含時のみ固定Qwen3-8B/native0で各1call。元contract/test/calls/snapshot/contrast全件と候補を提示、candidate IDまたはunresolvedを要求。unknown/invalid/unresolved/timeoutを拒否し代替候補へ回復しない。
- 候補列挙順はcanonical partition lexicographic順で固定。全資格通過時のみfresh protocol8、range/baseline/order/metadata/Validateへ進む。出力call/whole/context契約は既存1536/120秒/600秒/16K、retry/repair0。追加取得/依存/production変更なし。

## 完了条件との照合

既試験案に採用可能なcandidateはない。しかしH19の作用根拠と検証可能な残余があり、合理的追加探索消尽は現時点で未立証。candidate-found責務/child Issueの条件も未達。4未チェックの完了条件を変更しない。最終quality gateを先行実行して完了扱いにしない。
