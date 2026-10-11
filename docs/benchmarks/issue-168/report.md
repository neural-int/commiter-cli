# #168 Purpose Discoveryの判定

固定projection・Gemma/helper・purpose説明契約ではGate 1はNO-GO。研究をここで終了し、Phase 2/3・直接GroupingとのFM/FS比較・mixed fallback・Metadata・production適用は未評価として残す。既存#167 NO-GO・default4file上限・製品コードは変更していない。

最終判定は具体的な変更目的と入力根拠の固定rubricに基づくagentのオフライン照合であり、独立人手評定は実施していない。各説明の全文、対象/意味gold対応、grounding、理由をevaluation JSONに保存してレビューできるようにした。モデルの全体的な能力限界やPurpose-first全方式の不可能性を示す結果ではない。

| 集合 | 受理 / 生成 | 具体的目的coverage | exact | over-merge / over-split | grounded説明 | wall p50 / p95秒 |
|---|---:|---:|---:|---:|---:|---:|
| 既知K1/K2 | 2/2 | 0/3 | 0/2 | 0/1 | 0/4 | 30.711/31.192 |
| 既知再利用H4 | 2/2 | 0/4 | 0/2 | 0/0 | 3/4 | 37.904/39.033 |
| 未使用H1/H2/H3/H5 | 8/8 | 2/12 | 2/8 | 0/2 | 10/14 | 30.666/41.149 |
| 当初登録holdout集合（H4含む） | 10/10 | 2/16 | 2/10 | 0/2 | 13/18 | 33.911/41.149 |

coverageはgold目的、exactは各入力の全purpose list（必要な統合・独立性・具体性・根拠）の一致。Phase 2のfile partition exact/FM/FSとは異なる。normal/reverse各1回を独立fixture数として一般化しない。未使用4fixtureのexactは1/4、通常/逆順を合わせた生成観測では2/8。p95はnearest rankで小標本では最大値。

H6は既存baselineの16K予算超過で通常/逆順とも推論未到達。意味誤答や実測時間0として数えない。Gateの当初計画分母（H4含む）ではcoverage2/18、exact2/12であり、未到達の処置と受入条件未達を別記した。H4除外後の未使用計画分母ではcoverage2/14、exact2/10。どの集合でもGateを通過しない。

## 入力・既知/未使用の限界

- #167の9 Git fieldsと4 Evidence位置fieldsだけを除くprojectionを共通適用。baselineとdiscoveryのfiles値/順序は14 ordered pairで一致。purpose list・gold・fixture名・commit messageのリークはない。モデルに送ったtask/system/schemaはarchitecture差として固定した。
- H4を新規値の未使用入力とした登録記述は誤りだった。3fileのpath/before/afterが#167のunused-common-dependency-independentと完全一致し、同fixtureは4call評価済み。元の登録・cohort labels・receiptは保持し、独立holdoutから除外して別集計した。置換fixture・追加callなし。
- H1/H2/H3は今回新規の合成入力、H5は公開commiter-cliのCRLF修正履歴。H6も公開履歴だが予算超過。実履歴は同一作者由来であり、別作者/広いdistributionの独立採用証明ではない。
- 対応言語の入力はsyntax kind/name/enclosing_declaration/roleを含むが、今回の数式/定数/変更前の意味は含まない。projectionで除いた位置/同一性情報による情報不足だとは証明していない。元から存在しない意味をPurpose Discoveryが復元できるとは仮定しない。
- raw_diffのあるH5はCRLF/CR→LFを具体的に説明した。構文名だけのH1/H2/H3は一般的な役割説明か、実装・consumerの目的分断にとどまった。これは別fixture間の観測であり、raw_diff追加の因果効果を測ったA/Bではない。

## 実行契約と費用

- v1の出力1024は固定helperの768 guardに不適合。known4回は生成前失敗として保持。runnerが元stop/error_classをprotocol_mismatchへ上書きしたため、v1の個々の実停止classは未取得。source上の契約不適合と、取得できた失敗観測を区別する。
- v2は768へ適合し、prompt/schema/gold/品質閾値/モデルは維持。known通常順2call、当初holdout通常/逆順10call。invalid4+generation12=16process calls、retry/追加LLM verification0。total wallはv2のhelper attemptだけを合計し、準備・評価・品質テストを含めない。
- v2 input tokens 10595 / output tokens 7028、helper attempt wall合計 406.155秒、max RSS 3.338GiB。16GiB arm64で測定。すべてcompleted/schema valid、原Git状態保持12/12。v1もGit保持4/4。
- v2はswap前後増分の最大247.88MiB、pressure最大2で資源閾値内。v1は3/4で256MiBを超えた。全processを含む契約資源Gateは未達だが、無効requestのloadをPurpose Discoveryの実用費用と混同しない。前後値だけで短いpeakは観測できず、共存アプリとの因果分離もない。
- source/helper/binary/fixture/requestのv2固定hashは生成終了後も一致。ローカルのfixture root/cache/helperパスは公開成果物から除き、端末内のruntime receiptにだけ保存。

## Gateと停止判断

- all_planned_inputs_admitted: FAIL
- v2_schema_valid_rate_1: PASS
- coverage_at_least_90_percent: FAIL
- exact_at_least_80_percent: FAIL
- over_merge_zero: PASS
- over_split_zero: FAIL
- grounded_at_least_95_percent: FAIL
- hallucinations_zero: FAIL
- git_unchanged_all: PASS
- wall_within_120: PASS
- rss_within_6gib: PASS
- swap_delta_within_256mib: FAIL
- pressure_below_4: PASS
- process_calls_at_most_16: PASS

Gate 1のcoverage90%・exact80%・grounded95%・oversplit0・hallucination0・全入力受入に達しない。H4を分けても判定は変わらない。追加prompt調整・model追加・purpose verification・入力evidence変更へ自動拡張せず、この候補の研究を停止する。

## Next Steps

- 本結果と8達成条件の照合をIssueに保存し、チェックの意味を「検証/停止判断の記録完了」と明記する。未評価Phaseを実施済みとして扱わないため。
- Verificationの達成後にrepositoryのtest/vet/build、release notes、研究用回帰テストとdiff整合を実行・記録する。意味NO-GOとソフトウェア品質を別に確認するため。
- 成果物を専用branchへcommit/pushし、今回の研究を終了する。入力意味を増やす別研究やproduction接続は本Issueのscopeへ追加しないため。
