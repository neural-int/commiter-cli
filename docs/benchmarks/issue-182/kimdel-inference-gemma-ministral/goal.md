# H182-014 Gemma / Ministral kimdel履歴付きE2E推論比較

## Goal

H182-013と同一の5ファイル・3目的fixtureを使用し、Gemma E4B ON/OFF、Ministral3-3B Instruct/Reasoningの各文法制約あり/なし計8条件を、32K入力枠・4096出力上限で比較する。

## Result / Decision

全8条件・18phaseを測定終了。1142.918秒（約19分3秒）。監視571件すべてvalid、fatalなし。実行終了後helper/evaluationプロセス不在を確認。最終validator受理4/8はGemmaの4条件。Grouping完全正解は3/8（Gemma OFF none、Ministral Instruct両条件）。ただしCategoryの誤り・未完了・安全停止があり、Grouping/Category/validatorすべて正解は0/8。モデル採用や再試行は行わない。

## Conditions / Reuse

同一固定fixture、seed144、eager、KV8/group64、各phase別helper起動、時間制限なし、warmup/retry/repairなし。実Groupingを後段へ渡し正解partitionは注入しない。全18要求context_tokens=32768/output_tokens=4096。Thinkingと最終JSONの合計に上限を適用し、上限到達は未完了。全phaseの実測input+generatedも32768以内。
Gemma sampling=1/.95/64、Ministral Instruct=.1/.95/20、Reasoning=.7/.95/20。前回の各モデル固定revisionを再利用。Gemmaは採用済compactを新しい履歴フィールド名とブロック曖昧状態に合わせて適応。Qwenとはsystemが異なるためモデル単体差の比較とはしない。全モデルに構造化出力のSchema・フィールド・型・例の説明を付与。
旧ハーネスのモデル固有処理を再利用し、旧C01/C02や固定後段partitionは使わない。Ministralはmistral-common1.8.6の公式tekken serializer、Reasoning公式systemを保持。公式経路にSchema説明後のmessagesを渡すようSwiftを修正し、公式経路では独自THINK指示を重ねない。Swift Release再ビルド。旧tmp環境の欠損を以前のrequirements.lockから別venvへ復元し、公式資産を固定revisionと保存SHAで復元。語彙監査はInstructの既知予約slot34/35だけの差、Reasoningは差0を確認。BOS/EOS/終端を3phaseで確認。prefix cache非使用のため旧prefix長の条件は今回の前提にしない。
公式assets revision: Instruct b35d4dfe56c142746f54dbd64f579faab2744308 / Reasoning 4a36357c811bf511a7b625d132e12f22408aac91。モデルconfigはGemma131072、Ministral両モデル262144のposition上限を持ち、今回の32768は範囲内。
全8 Groupingにコマンド14件・会話3件と依頼/最終応答本文あり。履歴予算による省略0。fixtureはH182-013とbyte一致。Git差分・ソース・weight SHA不変。今回履歴は再取得せず、前回のpartial/history_incomplete・9ブロックambiguousという制約を保持。

## Cell measurements

正解はREADME(docs)、UI実装+テスト(feat)、メール正規化実装+テスト(fix)、breaking=false。Categoryは正解ファイル集合の群のみ採点。—は有効出力がなく採点不能。validator受理と意味の正解を分離する。decode=生成tokens総数/runtimeGenerationSeconds総和。Thinkingを含みload/prefillは含まない。

| 条件 | calls | 正解群/3 | Category正解/対象 | validator | 秒 | 生成tokens | decode token/s |
| --- | ---: | --- | --- | --- | ---: | ---: | ---: |
| gemma-on-json_schema | 3 | 0/3 | — | 受理 | 179.01 | 3246 | 24.38 |
| gemma-on-none | 3 | 0/3 | — | 受理 | 191.62 | 3246 | 21.92 |
| gemma-off-none | 3 | 3/3 | 1/3 | 受理 | 66.27 | 233 | 15.94 |
| gemma-off-json_schema | 3 | 1/3 | 0/1 | 受理 | 63.61 | 138 | 16.68 |
| ministral-instruct-json_schema | 2 | 3/3 | 1/3 | 拒否 | 65.58 | 129 | 20.40 |
| ministral-instruct-none | 2 | 3/3 | — | 拒否 | 78.05 | 319 | 18.60 |
| ministral-reasoning-none | 1 | — | — | 拒否 | 228.72 | 4096 | 21.04 |
| ministral-reasoning-json_schema | 1 | — | — | 拒否 | 225.79 | 4096 | 21.04 |

## Comparison with H182-013 Qwen

| 指標 | Qwen 18条件 | Gemma/Ministral 8条件 |
| --- | ---: | ---: |
| Grouping完全正解 | 4/18 | 3/8 |
| 最終validator受理 | 6/18 | 4/8 |
| Grouping/Category全正解かつvalidator受理 | 0/18 | 0/8 |

Qwen3.5-4Bは4条件ともGrouping正解、Thinking ON両条件はCategory3/3でscope=docsによる拒否が残る。Gemma OFF noneはGrouping正解・日本語Text・validator受理だがCategory1/3。Ministral Instructは両条件Grouping正解だがCategoryを通過せず。Reasoningは今回4096以内でGroupingを終えられず比較可能な最終Groupingを得られなかった。条件数とsystemが異なり上記率をモデル全体順位として扱わない。

decode速度の目安はQwen3.5-2B OFF44.25–52.03、Qwen3.5-4B OFF16.89–17.16、ON19.31–19.57 token/s。Gemma ON21.92–24.38、OFF15.94–16.68、Ministral Instruct18.60–20.40、Reasoning上限まで21.04 token/s。出力長・達したphaseが異なり、同じ成功仕事量の時間ではない。

## Thinking / output limit

Gemma ONは両条件で同じ回答・同じ生成量。G974 tokens(Thought raw932)、C1073(1054)、T1199(1174)、いずれもforcedClose=falseで自然に最終JSONへ移行。Thinking OFFは思考区間なし。
Ministral Reasoning両条件はGで4096 emitted tokens、max_tokens、forcedClose=false、最終JSONなし。思考を強制終了して回答へ誘導する処理は追加していない。生のnativeThoughtTokensは4097で、emitted generatedTokens=4096と1ずれる。これはsampling側カウンタとiterator出力数の集計境界差として保存し、4097を実出力Thinking数として用いない。開始markerを含むカウンタであり、厳密な本文token数も未確定。
評価JSONの一部にevaluation_note=KeyErrorがある。下位phaseが上限/grammar_failureで有効JSONを返さず既存評価器が参照できなかった箇所で、実モデルエラーは上記stop_reasonで判断する。採点不能を0点の正常出力と混同しない。

## Text / error patterns

- gemma-on-json_schema: 5ファイルを1群に統合し、summaryはUI追加だけを記述。メール修正とREADMEを落としている
- gemma-on-none: 文法制約ありと同一のGrouping/Category/Text。全統合とUI偏重が残る
- gemma-off-none: Groupingは3群完全正解。Textも日本語で3目的を説明。Categoryが全群featでdocs/fixを誤る
- gemma-off-json_schema: READMEと他4ファイルの2群。全群feat、TextはREADMEとメール修正だけでUI機能を落とす
- ministral-instruct-json_schema: Groupingは完全正解。CategoryでREADME=feat、UI=refactorと誤分類。メール群のbreaking=unresolvedで安全停止しText未実行
- ministral-instruct-none: Groupingは完全正解。Categoryはgrammar_failureで有効JSONなし、Text未実行
- ministral-reasoning-none: GroupingのThinking中に4096上限、最終JSONなし。Grouping精度は判定不能
- ministral-reasoning-json_schema: GroupingのThinking中に4096上限、最終JSONなし。Grouping精度は判定不能

## Performance / resources

helper RSS最大3.412GiB、MLX peak最大4.796GiB。別指標のため加算しない。system CPU busy平均14.77%、GPU利用率範囲[0, 100]%。thermalStateサンプル{'nominal': 91, 'fair': 480}。swap MiB（開始/終了/最大）[2599.12, 2694.94, 2942.94]、system free%範囲[30, 70]。同時helper最大1。
時間細目はphase-timing.mdとmeasurements.json。load・prepare・文法準備/setup・prefill・decode・generation wall・helper wall・user/system CPU・phase間gap・Go serialization/history投影/JSON検証・prefix処理を保存。2秒間隔監視自身の負荷を含む。温度実数・消費電力・プロセス別GPUは未取得。system資源は背景処理の影響を含み、thermal状態のみから実throttleを断定しない。

## Limits / Next

単一fixture・各条件1回。モデルごとの公式template・sampling・採用systemを使用するため最適化済み経路の探索比較。一般精度・統計的優位性・Thinking ONの速度優位は未確定。Ministral Reasoningは上限を増やせば成功するという保証もない。次はGrouping/Category/文章の不備を独立に扱い、4096上限や他条件を今回途中変更しない。原本と旧結果を保持し、生履歴・要求・応答は公開しない。
