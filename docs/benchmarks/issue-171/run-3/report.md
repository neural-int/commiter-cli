# Issue #171 最終比較報告

96/96cellの4条件paired比較を完了した。固定品質Gateは**NO-GO**。構造・実行資源・元Git/byte/tree安全のGateはPASSだが、Raw Diff＋暫定BのDはExact50%、FSと順序不一致が悪化し、意味品質の受理条件を満たさない。有限研究として終了し、追加runやProduction変更を行わない。

## 比較条件とGold

既存ThreePhaseGeneratorの実Grouping第1requestを捕捉したA、Raw Diffだけを変更したB、#170暫定Bの意味規則だけをGrouping出力へ適応したC、両方を変更したD。Schema、ID、task/安全指示、共通metadata、fixture/Git snapshot、モデル/decoder/budgetを共有した。通常順は既存パスソート、逆順はfiles配列だけ反転。スキーマ要因やソート導入効果は評価していない。

合成fixture12件、変更ファイル33件。Gold規則v1.0によるconfirmed11件、undetermined1件(F12)、multiple_valid0件。F12は事前除外（1/12、8.33%、8/96cell）。モデル入力不足のF11はconfirmed分母に残す。F01は旧2runでA通常順を見た既知診断ケースとして明記する。未生成confirmed F02〜F11の成績を別記し、全12件を未使用holdoutとは呼ばない。独立作者・実repository・checkpoint学習データとの非重複は未検証。

R6/R7 Fallbackは4ファイル、Gold由来の目的分断は7件/fixture集合。Goldと一致する単独コミットはExact正解であり、その目的分断をFSに加算しない。Fallback実例・複数許容解集合・未確定を区別し、複数許容解の実fixtureは0件、採点処理だけ契約テストで検証した。

## 主指標・順序

分母22cell/armはconfirmed11件×2順序。通常/逆順はpairedであり独立標本ではない。FM/FSは2順序合算のペア件数。発生case件数は通常/逆順いずれかで発生したunique fixtureを数える（評価JSONのlegacy fm_fixtures/fs_fixturesフィールドはcell件数なので区別）。

| Arm | 入力 / prompt | Exact | 通常 /11 | 逆順 /11 | FMペア | FSペア | FM発生case /11 | FS発生case /11 | 順序一致 /11 |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| A | 既存 / 既存 | 10/22 (45.5%) | 5 | 5 | 26 | 0 | 6 | 0 | 11 |
| B | Raw / 既存 | 10/22 (45.5%) | 5 | 5 | 26 | 0 | 6 | 0 | 11 |
| C | 既存 / 暫定B | 12/22 (54.5%) | 5 | 7 | 17 | 2 | 5 | 2 | 7 |
| D | Raw / 暫定B | 11/22 (50.0%) | 5 | 6 | 20 | 1 | 5 | 1 | 9 |

全96cellでbackend completed、Schema/完全割当24/24/arm。confirmedの構造不正・unscorableは0。F12の意味採点はnull/undeterminedであり、成功やFM/FS0と偽らない。

未生成confirmed10件(F02〜F11、20cell/arm)のExactはA8/20、B8/20、C10/20、D9/20。既知F01は全arm2/2。元の固定分母・Gateは再定義していない。

## 要因比較

| 比較 | Exact差 | FM差 | FS差 | 順序一致case差 |
|---|---:|---:|---:|---:|
| A→B：既存promptで入力変更 | +0 | +0 | +0 | +0 |
| A→C：既存入力でprompt変更 | +2 | -9 | +2 | -4 |
| B→D：Raw入力でprompt変更 | +1 | -6 | +1 | -2 |
| C→D：暫定Bで入力変更 | -1 | +3 | -1 | +2 |
| D−B−C+A：交互作用差分 | -1 | +3 | -1 | +2 |

既存promptではA/Bのconfirmed22分割もすべて一致し、Rawによる品質差は観測されなかった。暫定Bは既存入力でExact+2/FM−9だがFS+2/順序一致−4。Rawと組み合わせたDはB比Exact+1/FM−6、FS+1/順序一致−2。C→DではExact−1/FM+3/FS−1/順序一致+2であり、入力とpromptの効果をD対Aだけから説明しない。差分はこの固定小標本の観測であり、統計的有意性や母集団因果効果を主張しない。

## ケース別の監査

| Case | A Exact /2 | B /2 | C /2 | D /2 | 根拠・観測 |
|---|---:|---:|---:|---:|---|
| F01 | 2 | 2 | 2 | 2 | 実装/対応テストと独立prefixを全条件で分離。既知診断。 |
| F02 | 0 | 0 | 0 | 1 | encoder/decoderとretryをA/Bは全統合。C逆順は全singletonでFS1。D通常はFS1、逆順は正解。 |
| F03 | 2 | 2 | 2 | 2 | same-packageの2実装/各対応テストを全条件で正しく分離。 |
| F04 | 2 | 2 | 1 | 2 | 同一wire契約の実装とテスト。C通常だけFS1。 |
| F05 | 0 | 0 | 0 | 0 | 複数目的実装のR6/R7 singletonに全条件不一致。C通常はFM1、他はFM3。 |
| F06 | 0 | 0 | 1 | 0 | 共有テストのR6/R7 singletonはC逆順のみ正解。D通常FM1、他の不一致はFM3。 |
| F07 | 0 | 0 | 0 | 0 | 同じ1行に依存する単位変更＋独立label/retry、共有policyテスト。全条件全統合でFM3/cell。 |
| F08 | 0 | 0 | 2 | 0 | host仕様はAPI機能と利用側機能を独立目的として定義。Cだけ分離、A/B/Dは依存を含む2ファイルを統合しFM1。 |
| F09 | 2 | 2 | 2 | 2 | config/producer/consumer/docsの共通separatorを全条件で統合。 |
| F10 | 0 | 0 | 0 | 0 | host仕様はzero/negativeテスト拡張を独立coverage目標とする。全条件で共有helperの2testを統合しFM1。 |
| F11 | 2 | 2 | 2 | 2 | hostだけが持つ共通batch仕様。全条件Exactだが入力の識別可能性を証明する結果ではない。 |
| F12 | 対象外 | 対象外 | 対象外 | 対象外 | opaque option変更。undeterminedのため意味指標を採点せず、出力/資源を保持。 |

A/CのGo入力はAST種類・名前・位置中心で変更式やリテラル、具体的call対象を欠く。RawのB/Dは本体差分を維持するが、関係や独立性を正しく分類できる保証ではない。入力充足性は生成前の監査をそのまま保持し、失敗を理由にGold/除外/識別可能性を事後変更していない。F08/F10を含め、採点は事前に定義した具体的fixture目標に対する一致であり、コード参照だけから作者の唯一意図を立証するものではない。

## Fallbackと偏り

| Arm | 全統合cell /22 | 全singleton /22 | Gold Fallbackの正しい隔離 /8 |
|---|---:|---:|---:|
| A | 18 | 0 | 0 |
| B | 18 | 0 | 0 |
| C | 12 | 5 | 1 |
| D | 15 | 1 | 0 |

全統合にはF04/F09/F11の正解も含むため件数だけで誤りとしない。全singletonにもR6/R7や独立目的の正解がある。Cの誤統合減少とFS増加を同時に評価する。モデルSchemaに意味的unresolved欄はなく強制的に全fileへG labelを割り当てるため、完全割当を意味解決とみなさない。

## 費用・プロセスメモリ

| Arm | Input中央値 | Output中央値 | wall p50秒 | p95秒 | 最大RSS GiB | 最大MLX allocation GiB |
|---|---:|---:|---:|---:|---:|---:|
| A | 3172.5 | 501.0 | 47.547 | 62.115 | 3.785 | 4.796 |
| B | 1401.0 | 541.5 | 41.384 | 54.663 | 3.802 | 4.796 |
| C | 3250.5 | 559.0 | 49.881 | 83.431 | 3.771 | 4.796 |
| D | 1479.0 | 558.0 | 46.896 | 57.551 | 3.923 | 4.796 |

観測対象はGrouping第1段階のhelperプロセス。wallには毎回のモデルロードを含み、CLI全体/後続Category/Textや常駐helperの費用を示さない。RSSとMLX allocationは異なる量であり加算しない。背景負荷・cold load・OS状態の変動があるため時間差を一般的な因果効果にしない。Rawは入力tokensを減らしたが、A→Bの品質改善は観測されなかった。

ユーザー指示によりシステム全体swapはraw contextとしてのみ保存し、停止・受理に用いない。旧run1は+3722.31MiB、run2は+4183.50MiBの旧条件で各1call終了し、その結果を改変していない。新run3のRSS/MLX上限8GiB、critical pressure、timeout120秒、Git/hash条件はすべて満たした。旧2callと新96callで全体98call、各run内retry0、追加run4なし。

## 固定Gateの判定

- complete_execution: PASS
- all_arms_structural_valid: PASS
- D_exact_minimum: FAIL
- D_exact_improvement: FAIL
- D_FM_no_regression: PASS
- D_FS_no_regression: FAIL
- D_order_minimum: FAIL
- D_nontrivial_mixed_cases: FAIL
- resources_and_Git: PASS

DはExact11/22（18以上必要）、A比+1（+4以上必要）、FS1（A0以下必要）、順序一致9/11（10以上必要）で未達。必須境界F02/F08も両順序Exactでない。FM20≤A26はPASS、全実行/構造/資源安全もPASSだが、品質の同時受理条件を満たさずNO-GOとする。

## 安全と完了条件

全入力/Gold/prompt/evaluator/helper/modelの凍結hash、12fixtureの元Git digest、temp index/object directoryによるbyte/tree一致、main HEAD/index/既存.gitignore diffの事後一致を確認。Production default・4file上限・#169/#170 NO-GOを維持した。

- [x] 実Direct Grouping第1段階・契約・有限Gateを生成前に記録
- [x] Baseline/Treatmentの全文input/prompt/schema・同一model/Gold/orderを固定
- [x] 誤統合/誤分割/識別不能のケース別根拠監査
- [x] 4条件の通常/逆順paired比較を実行し品質/構造/資源を評価
- [x] 意味品質・安全・費用を区別してNO-GO/限界/次の条件を記録
- [x] Production/元Git状態を維持し有限研究として終了

Issue本文の6完了条件と報告目次を反映した後、指定goal.mdに従って最終品質ゲートを実施した。go test ./...、go vet ./...、go build ./...、研究用契約テスト9件、対象diff checkはすべてPASS。final-quality.jsonにexit codeと実行時間を保存した。Productionの機能削除/更新やテスト削除はなく、実回帰・採点契約以外の自明なテストを追加していない。

## Next Steps

- Production採用を進めない。DはExact/FS/順序の同時Gate未達で、Cも54.5%とFS悪化/順序不安定が残るため。
- 次の研究を行う場合は、同一目的統合とR6/R7 file fallbackのtask契約がmodelに十分伝わるかを、人間の新しい有限計画で検討する。F05/F06/F07でGold fallbackを実現できず、Raw保持だけでは改善しなかったため。現run内でpromptを修正しない。
- producer/consumerと独立利用側機能の境界を根拠監査し、input不足と判断不一致を区別する。F02/F08で順序・prompt・依存解釈による違いが観測されたため。新モデル・新stage・複雑な依存解析を今回追加しない。
- 実repository/別作者の未使用holdoutとmultiple_valid実例は、収集方針が承認される将来の検証へ残す。合成小標本と既知F01の結果を16file約80%やProductionへ一般化できないため。
