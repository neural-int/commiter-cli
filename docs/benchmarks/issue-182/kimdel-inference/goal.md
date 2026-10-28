# H182-011 kimdel履歴付きQwen推論比較

## Goal

実Git差分とCodex履歴の取得・照合から開始し、固定した5ファイル・3目的を8モデル・9Thinking設定×文法制約2条件で比較する。実GroupingをCategory/Textへ引き継ぎ、形式、目的別判断、文章品質、時間・資源を分けて評価する。

## Result / Decision

全18セル・50フェーズ要求が正常に測定終了した。最終planのvalidator受理は6/18だが、受理された6セルは全て正解partitionと不一致。GroupingとCategoryが共に正解だったのはQwen3.5-4B Thinking ONの文法制約あり／なし2セル。両方ともTextが英語かつscopeにfeat/fixを使用し、最終validatorが拒否した。したがって今回の正解3群を保った最終plan成功は0/18。モデル採用・通常設定の変更は行わない。

## Conditions

8モデルの公開MLX 4bit固定revision、同一Release helper、seed144、eager、KV8/group64、無制限の出力・Thinking予算、履歴取得・cycleの時間制限なし。各条件1回、warmup/retry/repairなし。各モデル×制約条件を交互の順序で直列実行。Qwen2.5のsamplingは0.7/0.8/20、Qwen3.5-2Bは1.0/1.0/20、Qwen3.5-4Bは0.6/0.95/20。Qwen2.5のsamplingは本測定で固定した研究設定で、モデル既定値との完全一致を主張しない。
文法制約ON/OFFでsystemの構造説明・Schema・例を同じにした。Thinking ONは思考中自由生成、境界後にJSON制約。noneは文法マスクを外し、JSON指示・境界追跡・事後validatorを維持する。最終Schemaのscope16文字・summary48文字と日本語条件は保持する。
各phaseはGoクライアントから別のhelper子プロセスを起動する実経路。モデルをphase間で保持する旧ベンチとは異なる。モデルload/prefillは各phaseに含まれ、OSファイルcacheのcold状態を保証しない。
正解はA=メール正規化実装＋回帰テスト（fix）、B=期限日tooltip実装＋テスト（feat）、C=README（docs）。breakingは全てfalseを期待。正解は採点側のみへ置き、モデル入力に注入しない。

## Fixed history / ETL

前処理全体2.422584秒。Git収集0.349367秒、構文・内容取得0.067674秒、関連付け0.021925秒、履歴取得・照合・snapshot確認1.982554秒。履歴本文取得の内部wallは1.334306秒、内容照合1.933ms、コマンド投影4.040ms。129,717 bytes、12 RPC、3 fileChange操作、3会話コンテキスト、14コマンド。
取得状態はpartial / history_incomplete。9ブロックすべてambiguousで、候補あり4・候補なし5。完全取得・全ブロックmatchedの条件ではない。全18 Grouping要求のmessagesとSchemaは同一hash。履歴をセルごとに再取得していない。

## Cell measurements

正解群はファイル集合で照合し、G番号には依存しない。Categoryは正解と完全一致した群だけを分母にする。—は対象群なし。decode token/sは各セルの生成token総数÷runtimeGenerationSeconds総和。停止phase数が異なるセルの全体秒を同じ仕事量の速度として比較しない。

| ケース | calls | validator | 正解群/3 | Category正解/対象 | 全体秒 | 生成tokens | decode token/s |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: |
| qwen25-05-json_schema | 3 | 拒否 | 0/3 | — | 10.74 | 94 | 102.25 |
| qwen25-05-none | 1 | 拒否 | 0/3 | — | 4.07 | 58 | 113.29 |
| qwen25-15-none | 3 | 拒否 | 0/3 | — | 21.65 | 161 | 55.86 |
| qwen25-15-json_schema | 3 | 受理 | 0/3 | — | 26.32 | 238 | 52.25 |
| qwen25-3-json_schema | 3 | 受理 | 1/3 | 0/1 | 54.51 | 285 | 31.32 |
| qwen25-3-none | 3 | 拒否 | 1/3 | 0/1 | 59.64 | 316 | 27.72 |
| coder25-05-none | 3 | 拒否 | 1/3 | 0/1 | 17.36 | 342 | 105.67 |
| coder25-05-json_schema | 3 | 受理 | 1/3 | 0/1 | 19.96 | 357 | 73.69 |
| coder25-15-json_schema | 2 | 拒否 | 0/3 | — | 22.62 | 93 | 48.25 |
| coder25-15-none | 2 | 拒否 | 0/3 | — | 22.27 | 93 | 57.42 |
| coder25-3-none | 3 | 拒否 | 0/3 | — | 54.72 | 150 | 27.21 |
| coder25-3-json_schema | 3 | 受理 | 0/3 | — | 54.89 | 123 | 23.43 |
| qwen2-off-json_schema | 3 | 受理 | 0/3 | — | 35.67 | 111 | 53.72 |
| qwen2-off-none | 3 | 拒否 | 0/3 | — | 46.74 | 530 | 37.98 |
| qwen4-off-none | 3 | 拒否 | 1/3 | 1/1 | 82.29 | 198 | 16.46 |
| qwen4-off-json_schema | 3 | 受理 | 0/3 | — | 87.86 | 154 | 13.82 |
| qwen4-on-json_schema | 3 | 拒否 | 3/3 | 3/3 | 260.44 | 3251 | 17.09 |
| qwen4-on-none | 3 | 拒否 | 3/3 | 3/3 | 251.65 | 3297 | 17.72 |

## Format / semantic quality

文法制約ありは26/26 phaseでJSON解析・Schema適合。なしは24 phase中helper受理23、重複キーを拒否するJSON解析22、Schema適合17。文法制約が型を守ってもGroupingやtypeの意味、日本語、scopeの用途、事実性が正しくなるとは限らない。
Qwen4B Thinking ONは両条件で3群完全一致、Category 3/3、breaking evidence none。Textは3群それぞれの目的を表したが、英語summaryとfeat/fix scopeで拒否。JSONとCategoryの成功を最終E2E成功に数えない。
小型モデルでは全変更の過剰統合、実装と対応テストの分離、README説明の反復、実差分にない.dev.py修正の記述などを観測した。制約ありの複数応答は48文字上限に達して文末・語が途中で終わっている。生成後に文章を切り詰めた結果ではない。

| ケース | Textの定性確認 |
| --- | --- |
| qwen25-05-json_schema | scope/summaryがExampleのみで現在変更の目的を説明していない |
| qwen25-05-none | Groupingの形式違反で後段へ進まない |
| qwen25-15-none | READMEとメール正規化を混在して記述し、ツールチップ目的を落とす。summary上限違反 |
| qwen25-15-json_schema | 変更されていない.dev.pyの修正を記述。実ファイル対応と説明がずれる |
| qwen25-3-json_schema | READMEとMessage-IDの要旨はあるがUI説明が不明瞭でscopeに実在しない表記が混じる |
| qwen25-3-none | 個別目的の要旨はあるがUIの名称が不明瞭。scopeに長いパスを使い形式違反 |
| coder25-05-none | Text helperがgrammar_failureで終了し、有効な最終JSONを得ていない |
| coder25-05-json_schema | 全5グループに同じ品質チェック説明を反復し、メール修正・UI機能の説明になっていない。文末も不完全 |
| coder25-15-json_schema | CategoryのunresolvedでTextを実行しない |
| coder25-15-none | CategoryのunresolvedでTextを実行しない |
| coder25-3-none | 全変更を一群にし、READMEの説明だけを記述。scope上限違反 |
| coder25-3-json_schema | 全変更を一群にし、READMEの説明に偏る。末尾が「個別テストの実行例と」で終わる |
| qwen2-off-json_schema | 全変更を一群にしREADMEの説明に偏る。typecheckの語が途中で終わる |
| qwen2-off-none | Textに重複G003があり、実差分にないNone防止処理をUI変更として記述 |
| qwen4-off-none | 目的の要旨はあるが、実Groupingの所属にない対応テストまで記述。scope上限違反 |
| qwen4-off-json_schema | README・UIを一群にしながらREADME中心に記述。typecheck対象の実処理変更と誤解し得る表現 |
| qwen4-on-json_schema | 3群の目的説明は差分に沿うが全summaryが英語で、scopeにfeat/fixを使用 |
| qwen4-on-none | 3群の目的説明は差分に沿うが全summaryが英語で、scopeにfeat/fixを使用 |

## Performance / resource observations

全体1137.66秒（約18分58秒）。2秒間隔の監視568件は全件有効。samplerの平均wall 0.210秒、最大1.325秒。これは監視作業の経過時間であり、そのまま推論減速率を意味しない。
helperの観測RSS最大3.090GiB、MLX peak最大3.129GiB。RSSは2秒観測の最大で真の瞬間peakではない。MLX allocation、KV bytes、RSS、Activity Monitor footprintは別指標で足し合わせない。
GPU system utilizationは0–100%。Mach CPU busyの観測平均は21.4%（システム全体）。thermalStateは{'nominal': 57, 'fair': 511}。fairは熱状態の観測であり、CPU/GPUクロック低下やスロットリングの確証とはしない。
system swap使用は2066.19MiB→3588.69MiB。system memory free表示は開始48%→終了53%。どちらもシステム全体で、helper単独の変化へ帰属させない。前景操作や他アプリの負荷を制御しておらず、条件順序と熱・cacheの交絡を排除しない。
CPUはsystem tick、process psの累積/区間値、helper user/system CPU秒。GPUはAGX利用率とPerformanceStatistics、メモリはvm_stat/pressure/swap/RSS/VSZ/MLX/KV、diskはiostat、電源はpmsetと低電力モードを記録した。摂氏温度、Watts、プロセス別GPU使用率、GPU kernel別時間、実クロック周波数は未取得。powermetricsはsudoパスワードが必要で取得できなかった。
token Read/Mask/Updateの診断wallも保存した。特にtoken.itemのRead時間はGPU待ちを含み、CPU実行時間とは同一でない。prefill/decode/grammar/全体の計測区間にも包含関係があり、全列の単純合算で全体時間を再構成しない。

## Helper process observation

ユーザーのActivity Monitorでは2件が表示された。PID71976/72321、79005/79663をOSで照会し、指摘時点では既に終了していた。79663のUI詳細は実メモリ0・CPU0だった。568監視サンプルと0.5秒×10回の追加確認でhelper同時存在は観測せず、後にActivity Monitor一覧も1件となった。表示残存の詳細原因は特定していない。強制終了やセル再実行はしていない。

## Verification / limits

18/18セル確定、50/54上限以内の実要求、monitor_fatalなし、固定fixture・runtime/source hash一致、全セル前後のGit snapshot一致、Grouping入力hash一致を監査した。件数とcycle/history deadlineを外したのは研究用Go overlayだけ。全テスト7件とビルド成功。
一つの5ファイルfixture・各条件1回。一般的な精度、安定性、モデル間の因果的速度優位性は未検証。Text確認はagentによる定性評価で、独立した人間の採点ではない。今回の履歴取得不足とshell補助情報を含む条件を明記して解釈する。
measurements.jsonにセル・phase・RPC集計と定性確認を保存。原本request/response、進捗、OS時系列、実行契約、凍結Git/履歴はローカル保持。会話・source ID・個人絶対pathを公開資料へ収録しない。コード・成果物のcommit/pushは未実施。結果はIssue #182コメント6412409939へ投稿済み。


## 追加分析で確認した入力投影条件

凍結時に14件あったコマンドは、実際のGrouping入力では全18セルとも0件だった。H3の依頼・最終応答も入力予算で省略されていた。16k投影の非推論再現で実送信payloadと完全一致し、コマンド14件すべての省略理由がinput_budget_exceededであることを確認した。32k/64kの投影ではコマンドと3会話本文を保持できたが、モデル推論は追加していない。取得件数を全件LLMへ渡った件数と解釈しない。

詳しくはerror-analysis.mdとprojection-replay.json。補足はIssue #182コメント6412593558へ投稿済み。元の18セルの結果と原本は変更していない。
