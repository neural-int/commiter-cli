# R6/R7のSystem Prompt明示：追加診断結果

指定コメント: [issuecomment-6139165124](https://github.com/neural-int/commiter-cli/issues/171#issuecomment-6139165124)。生成前の事前登録commit: `0d3a89691e06a39de237fc16e8972fe8a4d8edad`、[事前登録報告](https://github.com/neural-int/commiter-cli/issues/171#issuecomment-6139331801)。

## 要約

48/48callを完了した。指定されたR6/R7の6行をSystem Promptに追加する処置Bは、Exactが10/22→11/22、正しいFallback隔離が0/8→1/8に改善した一方、FSが0→1、通常/逆順一致が11/11→7/11となった。事前固定した非回帰条件は未達（`criteria_not_met`）。診断上の一部改善はあるが、回帰を伴うため採用を支持する結果にはならない。採用GOは今回未評価、Production変更0、旧run3の意味品質NO-GOは維持する。

全48出力のSchema/完全割当、資源・Git保持は成功した。対照Aの24Partitionは旧run3 Aとすべて一致した。既知ケース・自己作成合成fixtureの診断であり、新規holdoutや実repositoryの一般化結果ではない。

## 検証条件

既知F01〜F12 × A/B × 通常/逆順、各1回、48call、retry0。Aは旧run3 Arm Aのrequest全文bytesをコピーした新規再測定、BはAのSystem Promptの安全文直前に指定段落のみを挿入。今回のBは旧run3のRaw Diff arm Bではない。user message・既存AST入力・file ID・schema・task/safety・Gold v1.0・partition採点器を維持した。新しい入力方式、Planner、LLM stage、依存追加はない。

MLX Gemma E4B 4bit revision `475b9088d29754a3379866cf5aeb6b41acd313c2`、同じMLX helper、seed144、temperature0、top_p1/top_k0、context16384/output768/native thought512、bounded-grouping、timeout120秒。48requestは静的検査・native tokenize-only受理とも48/48、Bの追加は各81input tokens。推論前にrequest/hash・資源上限・診断条件をcommit/pushした。

採点対象はconfirmed11fixture、通常/逆順を含む22cell/arm。F12は従来のundeterminedとして事前除外（1/12、8.33%）し、4出力は構造・資源集計に含めた。通常/逆順はpaired診断であり、22独立標本ではない。複数許容解のfixtureは0件。F05/F06/F07は前回の生成前監査で既存入力がinsufficientのまま。

## 検証結果

| 指標 | A：既存promptの再測定 | B：指定R6/R7を追加 | B−A |
|---|---:|---:|---:|
| Exact Partition | 10/22（45.5%） | 11/22（50.0%） | +1cell |
| 通常順Exact | 5/11 | 4/11 | −1 |
| 逆順Exact | 5/11 | 7/11 | +2 |
| FMペア | 26/38 | 24/38 | −2 |
| FSペア | 0/24 | 1/24 | +1 |
| FM発生cell | 12/22 | 10/22 | −2 |
| FM発生unique fixture | 6/11 | 7/11 | +1 |
| FS発生cell / unique fixture | 0/22・0/11 | 1/22・1/11 | +1・+1 |
| 通常/逆順Partition一致 | 11/11 | 7/11 | −4 |
| 正しいR6/R7 Fallback隔離 | 0/8 | 1/8 | +1 |
| 全件merge | 18/22 | 16/22 | −2 |
| 全件singleton | 0/22 | 3/22 | +3 |
| Schema適合・完全割当 | 24/24 | 24/24 | 同じ |

FMの分母38はGoldで別groupのペア数、FSの分母24はGoldで同一groupのペア数（各2順の合計）。Fallbackの分母8は4対象ファイル×2順。上の割合をfixtureの独立成功率と混同しない。FM総数が減っても、F03で新しい誤統合が生じたため発生fixture数は増加した。

| fixture | A通常：Exact/FM/FS | A逆順 | B通常 | B逆順 |
|---|---:|---:|---:|---:|
| F01 | 1/0/0 | 1/0/0 | 1/0/0 | 1/0/0 |
| F02 | 0/2/0 | 0/2/0 | 0/2/0 | 0/0/1 |
| F03 | 1/0/0 | 1/0/0 | 0/4/0 | 1/0/0 |
| F04 | 1/0/0 | 1/0/0 | 1/0/0 | 1/0/0 |
| F05 | 0/3/0 | 0/3/0 | 0/3/0 | 0/3/0 |
| F06 | 0/3/0 | 0/3/0 | 0/3/0 | 1/0/0 |
| F07 | 0/3/0 | 0/3/0 | 0/3/0 | 0/3/0 |
| F08 | 0/1/0 | 0/1/0 | 0/1/0 | 1/0/0 |
| F09 | 1/0/0 | 1/0/0 | 1/0/0 | 1/0/0 |
| F10 | 0/1/0 | 0/1/0 | 0/1/0 | 0/1/0 |
| F11 | 1/0/0 | 1/0/0 | 1/0/0 | 1/0/0 |
| F12 | Gold未確定 | Gold未確定 | Gold未確定 | Gold未確定 |

- F05：複数目的実装 `display/settings.go` の単独隔離は両arm・両順0/2。全件mergeを維持。
- F06：B逆順のみ、共有テスト `service/policies_test.go` と独立実装をすべてsingletonにし、Goldと一致。Fallback隔離はA0/2→B1/2。B通常順は全件mergeで失敗した。
- F07：2つのFallback対象ファイルの単独隔離は両arm・両順0/4。指定規則を加えても全件mergeのまま。
- F02：B逆順が全件singletonとなり、同一契約のproducer/consumer `encoder/rate.go` と `decoder/rate.go` を誤分割（FS1）。FM低下だけを改善とは扱えない。
- F03：B通常順が全件mergeとなり、独立したwidth/badge目的を誤統合（FM4）。B逆順は正解で、順序依存が生じた。
- F08：B逆順のみ独立目的を正しく分離（Exact1）。通常順は誤統合のまま。Fallbackの改善件数には加えていない。
- F01/F04/F09：必要統合の対照は両arm・両順でExact、FS0。F10は独立テスト目的の誤統合を維持。F11の正解はホストだけの仕様がモデル入力から識別できた証明にはならない。

Bの全件singleton3cellはF02逆順・F06逆順・F08逆順。F06/F08ではGoldと一致するが、F02では必要統合を壊している。全件singletonを一般的な成功として扱わない。順序不一致はF02/F03/F06/F08の4fixture。

## 資源・構造・Git

| 指標（各24call、F12含む） | A | B |
|---|---:|---:|
| wall p50 / p95（秒、nearest-rank） | 37.117 / 45.626 | 40.196 / 47.272 |
| input tokens p50 / 最大 | 3154 / 4533 | 3235 / 4614 |
| output tokens p50 / 最大 | 488 / 577 | 543 / 577 |
| helper RSS最大（GiB） | 4.181 | 4.149 |
| MLX allocation最大（GiB） | 4.796 | 4.796 |

call wall合計1852.256秒（約30分52秒）。モデルはcallごとにcold helper processで呼び出す。壁時計時間は環境の影響を含み、Generalな速度因果効果やCLI全段階の性能とは主張しない。RSSとMLX allocationは異なる観測量なので合算しない。今回測定したのは第1段階のMLX helperであり、Production CLI全体のメモリ評価ではない。

helper RSS・MLX allocationは各8GiB上限以内、全callのpressure levelは1。timeout・不正Schema/割当・Git driftは0。全体swapはユーザー指示どおりraw記録のみで停止・合否に使用しない。request・採点器・helper/model hashを開始/終了時に確認し、各callでfixture Gitと署名を確認した。

事後の一時index/object directoryによるbyte/tree監査12/12 PASS。元mainのHEAD/index/worktreeと旧成果物374ファイルを保持し、run1/run2/run3の記録を上書きしていない。新規48call、旧98callを含む総146call、retry0、終了後の追加生成0。

最終品質ゲートは `go test ./...`、`go vet ./...`、`go build ./...`、Python unittest11件、対象diff checkがすべてPASS。Go suiteは既存キャッシュを使用。新規2テストは既存24リクエストについてSystem段落以外の変更を拒む契約、および曖昧な安全文挿入位置の拒否を検証する。旧9テストを維持し、無効化・不要なProductionテスト追加はない。

## 考察

指定R6/R7の明示により、F06逆順の正しい隔離は1cell観測された。しかしF05/F07は両順で失敗し、処置Bは新しい誤統合F03と誤分割F02、4fixtureの順序依存を生じた。Exact +1、FM −2という集計値だけでは、この処置を支持するには不十分である。事前条件のFS非悪化・順序非悪化が未達のため、診断の改善信号は成立しない。

今回の既存入力は変更式・リテラル等を欠き、主診断F05〜F07の事前分類はinsufficientである。したがって未改善をR6/R7指示一般の無効性と断定できない。またF06の改善は全singleton出力の一例なので、モデルが複数目的を正しく識別したことの独立証明にはならない。自己作成の既知ケース、同一モデル、各順1call、未使用holdout0件という限界を維持する。

## Next Steps

- 固定48callの有限追加診断をこの記録で終了し、Issueへ報告・commit/pushする。指定仮説の局所結果と回帰を取得できたため。
- Production採用や指定文言の反復調整は行わない。非回帰条件が未達で、採用GOも今回の評価対象外であるため。
- 今後採用を評価する場合は、別の明示された有限計画で独立未使用fixture・Gold・Gate数値を生成前に固定する必要がある。今回の既知ケースで事後調整した成功率を採用根拠にしないため。今回のタスクでは追加生成しない。

証跡: `preregistration.json`、`frozen-inputs/`、`tokenization-preflight.jsonl`、`attempts.jsonl`、`measurements.jsonl`、`evaluation.json`、`termination.json`、`final-audit.json`、`final-quality.json`、`quality-*.log`。
