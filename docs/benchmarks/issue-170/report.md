# System Prompt比較の最終判定

Gate 1は **NO-GO**。BはAよりpurpose過剰分割を8→5に減らしたが、具体的coverageは11/12→10/12に低下し、目的リストexactは4/8のまま。Cは過剰分割8、exact3/8、根拠のない説明2件で、Aに対する実質改善を示さなかった。Phase 2の直接Groupingは条件不達により未評価。製品採用は行わず有限研究を終了する。

## 観測結果

| 指標 | A | B | C |
|---|---:|---:|---:|
| 具体的coverage | 11/12 (91.7%) | 10/12 (83.3%) | 11/12 (91.7%) |
| 目的リストexact | 4/8 (50.0%) | 4/8 (50.0%) | 3/8 (37.5%) |
| purpose過剰統合 | 0 | 0 | 0 |
| purpose過剰分割 | 8 | 5 | 8 |
| grounding | 20/20 | 17/17 | 18/20 |
| 根拠のない説明 | 0 | 0 | 2 |
| 順序で品質/粒度が変わるfixture | 2/4 | 3/4 | 3/4 |
| schema / admission | 8/8 / 8/8 | 8/8 / 8/8 | 8/8 / 8/8 |
| Input / output tokens合計 | 7580 / 5046 | 7652 / 4773 | 7316 / 4876 |
| wall p50 / p95 (秒) | 50.12 / 53.95 | 46.35 / 53.02 | 48.46 / 61.60 |
| helper peak RSS (GiB) | 3.227 | 3.369 | 3.180 |
| swap増分最大 (MiB) | 151.62 | 81.75 | 350.44 |
| pressure最大 | 1 | 1 | 1 |
| 資源基準 | PASS | PASS | FAIL |

新規holdoutは自作合成4fixture、normal/reverseの8paired cell。coverage分母12は計6goldを両順序で照合した観測数であり、独立12目的標本ではない。過剰分割/統合はpurpose説明のgold対応数であり、file partitionのFM/FSではない。Groupingは0callなのでFM/FS、complete assignment、fallback、singleton逃げは未評価。

| Fixture | A normal/reverse | B normal/reverse | C normal/reverse |
|---|---|---|---|
| N01 | exact (1説明) / 不達 (3説明) | 不達 (2説明) / 不達 (3説明) | 不達 (3説明) / exact (1説明) |
| N02 | 不達 (3説明) / 不達 (3説明) | exact (1説明) / 不達 (3説明) | 不達 (3説明) / 不達 (3説明) |
| N03 | exact (2説明) / 不達 (4説明) | exact (2説明) / exact (2説明) | 不達 (4説明) / exact (2説明) |
| N04 | exact (2説明) / exact (2説明) | exact (2説明) / 不達 (2説明) | exact (2説明) / 不達 (2説明) |

N01はsearch capとUI defaultを同じpage-size契約にするgoldを事前登録した。N02はsender/receiver/testの同じJSON wire-key契約。N03は共通normalize helperを持つslug/visibilityの独立目的、N04は同fileのtimeout/batch-size独立目的。N04はpurpose gold2個だがwhole-file grouping gold1groupであり、同一尺度として扱わない。

Known H01-H04 normal順の回帰12callは採用根拠から分離した。各arm coverage6/7、exact A1/4・B2/4・C2/4、過剰分割A3・B2・C3、根拠のない説明A1・B0・C1。詳細はknown-evaluation.json、holdout-evaluation.jsonに、全文とsemantic_gold/target_gold/grounding/理由を保存している。

## 資源・安全の観測

全36生成はcompleted/schema valid、実input tokensはnative preflightと一致。全fixtureのsnapshot/index/worktreeとmainのHEAD/index/既存.gitignore変更を保持した。全runtime/source/request pinsが一致。失われた試行0、retry/repair0、Grouping0。helper/modelはローカル既存cacheを使い、モデル取得・クラウド推論は0。変更は#169最終headからtools/benchmark170とdocs/benchmarks/issue-170のみ。

hardwareは17179869184 bytes（16GiB）。RSSはhelper process peakでありホスト全体の実メモリではない。swapは実行開始前から約5.5GiBを使用し、C N02 normalで350.44MiB増加して256MiB基準を超えた。pressureは全観測1。前後観測だけなので短時間peak、他appとの因果、長時間共存の安定性は未確認。Cの資源GateはFAILだが、意味品質だけでもGOに達しない。

S16は16file×150変更行のtokenization-only stress。A47146・B47155・C47113 input tokensで、context16384を全条件で超過。生成せず、意味誤答やquality分母に換算しない。この小標本から「16file約80%正解」を達成したとは主張しない。

## 最終判定の問いへの回答

1. Bはこのholdoutで過剰分割を減らしたがcoverageを落とし、exact/順序安定性を改善しなかった。過剰統合0・根拠のない説明0は維持した。Cは過剰分割を減らさず、根拠のない説明を増やした。どちらも採用根拠には不足する。
2. BはN02 normalとN03両順で同目的を統合できたが、N02 reverseでは3分割し、N01でも分割が残る。CはN03のnormal/reverseで4/2目的となる。Bの一部観測が有利でも、同目的統合と独立目的分離を安定して両立した候補とは判定できない。
3. Bのwall p95はAより約0.92秒短く、Cは約7.65秒長い。独立同一条件下の因果的latency差ではなく、当該ホストの順次観測である。品質Gate未達、Cのswap基準超過、S16未受理により、Mac16GiBとの十分な共存・16file適用は立証されていない。
4. Phase 1がNO-GOなので直接Groupingへの有効性は未評価。目的説明のcoverageからGrouping精度を推定しない。

## 解釈上の限界

差は固定Prompt全体のpaired効果であり、independently reversible/behavioral goal、重複禁止、否定文のそれぞれの単独因果効果を識別しない。BとCは否定文以外にも文言差がある。評価は同じagentによるoffline input/gold照合であり、独立人手評定ではない。未使用holdoutは今回の初回生成という意味で、別作者/実repositoryの無作為標本ではない。N01の共通page-size契約も設計者が固定したgoldであり、直接caller参照はfixtureに含まれない。適用範囲をこの合成goldに限定する。

Cの根拠のない説明2件はN01 normalのn>n比較、N04 reverseのbatch単位pages。値を一般化しただけの他の説明はtarget-onlyに留めた。これら2件を仮に許容してもCはexactが最大4/8、過剰分割8で絶対Gateに達しないため、NO-GOはこの境界的な評定だけに依存しない。正式scoreやrubricは変更していない。

## Next Steps

- Phase 2・製品採用を行わず終了する。絶対品質と実質改善の事前登録条件に達しなかったため。
- 再研究する場合は別Issueで、より多くの別作者/実repository holdoutと独立評定を先に定義する。小さい自作合成fixtureの局所改善を一般化しないため。本Issueでは実施しない。
- #169 Closed / Gate1 NO-GOを維持する。今回のPrompt比較は過去の不達を覆す結果ではないため。
