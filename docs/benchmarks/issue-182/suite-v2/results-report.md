# Issue #182: モデル別ハーネス比較実測（H182-002）

## 要約

指定された7設定をH180-004と同じC01/C02 fixtureで各1回実行した。GemmaのON/OFFは再測定済みで、7設定×2 fixture、14セルの実行記録を得た。plannerが最終planを返したのは9/14セル。GoldとのGrouping完全一致は、モデル応答を得た12セル中11セルで、Ministral Reasoning C02だけが不一致だった。

Category/Textは実Groupingを検証した後に固定参照partitionを注入しているため、後段の成功をend-to-endのplan成功とは扱わない。Goldは探索用に作成されたもので、人間が独立承認した正解ではない。Issue #182のモデル採用判断・完了条件は未達のまま。

## 実行条件

- PR #181のmerge commit `69bfc0775d23e62481cd45bcb4bf73608a355757` を基点にした専用worktreeで実行した。モデルrevision・helper・overlay・fixture等の実行契約を推論前に固定した。
- 4bitモデル、seed=144。Gemma ON/OFFは `temperature=1.0, top_p=0.95, top_k=64`、Qwen3.5-4B ON/OFFは `0.6, 0.95, 20`、Qwen3.5-2B OFFは `1.0, 1.0, 20`、Ministral Instructは `0.1, 0.95, 20`、Ministral Reasoningは `0.7, 0.95, 20`。revisionの完全な値は同ディレクトリの各JSON設定に記録した。
- 出力token予算・thinking予算は無制限（helper内で `maxTokens=nil`、native thought limitなし）。120秒timeoutなし、warmupなし、各fixture一回、retry/repair/judgeなし。14セルの上限42 phase要求に対し36要求を実行した。
- 引き継いだH180-004のC01（3ファイル）/C02（4ファイル）fixtureを使用した。16ファイルfixtureには置き換えていない。Grouping後にCategory/Textへ固定参照partitionを注入する方式も維持した。
- 64K input context ceilingとIPCの要求/応答サイズ制限は保持した。したがって「無制限」は生成予算・thought予算・120秒timeoutについての設定であり、入力・通信容量まで無制限ではない。
- 実行契約、個別request/response、cell結果、OS監視ログは `/private/tmp/issue182-suite-v3/` に保存した。テキスト形式の実行ログと再現補助ソースを `docs/benchmarks/issue-182/suite-v2/raw/` に収録し、公開用コピーではメール形式文字列を `redacted@example.invalid`、絶対ホームディレクトリパスを `$HOME` に置換した。計測値・モデル応答はそのまま残し、環境依存の `evaluation.test` 実行ファイルだけを除外した。

## 検証結果

`G/C/T`はGrouping/Category/Text。各値は「生成token数（thinking token数）/ decode tokens/s」。thinking token数は出力token数の内数。tokens/sはhelperが返したモデル生成時間で計算し、prefillやphase全体の経過時間は含まない。`—`は未実行、`不明`は応答がなく計測不能を示す。

| 条件 | Fixture | planner結果・Grouping | G | C | T | cycle秒 |
| --- | --- | --- | ---: | ---: | ---: | ---: |
| Gemma4 ON | C01 | 受理・一致 | 828 (766) / 15.21 | 1,062 (1,035) / 14.43 | 1,023 (980) / 14.41 | 226.74 |
| Gemma4 ON | C02 | 受理・一致 | 1,027 (947) / 9.27 | 844 (817) / 8.81 | 853 (812) / 11.18 | 393.32 |
| Gemma4 OFF | C01 | 受理・一致 | 66 / 13.49 | 26 / 13.48 | 42 / 10.57 | 41.99 |
| Gemma4 OFF | C02 | 受理・一致 | 88 / 7.48 | 26 / 8.17 | 39 / 8.09 | 175.71 |
| Qwen3.5-4B ON | C01 | 受理・一致 | 575 (515) / 14.20 | 1,683 (1,657) / 13.38 | 824 (781) / 17.35 | 246.83 |
| Qwen3.5-4B ON | C02 | 失敗・Grouping応答なし | 不明 | — | — | 2,472.58 |
| Qwen3.5-4B OFF | C01 | 受理・一致 | 59 / 13.27 | 25 / 7.97 | 34 / 9.96 | 57.70 |
| Qwen3.5-4B OFF | C02 | 受理・一致 | 65 / 6.66 | 25 / 6.48 | 35 / 7.11 | 249.69 |
| Qwen3.5-2B OFF | C01 | 拒否: `invalid_summary_language`・一致 | 59 / 18.71 | 25 / 21.14 | 32 / 11.01 | 34.11 |
| Qwen3.5-2B OFF | C02 | 受理・一致 | 74 / 12.47 | 25 / 11.08 | 49 / 11.63 | 108.92 |
| Ministral Instruct | C01 | 停止: `unresolved_breaking_evidence`・一致 | 58 / 16.89 | 28 / 19.56 | — | 24.92 |
| Ministral Instruct | C02 | 停止: `unresolved_breaking_evidence`・一致 | 74 / 9.96 | 27 / 9.80 | — | 276.20 |
| Ministral Reasoning | C01 | 失敗: `native_eos_before_close`・Grouping不成立 | 613 (613) / 17.06 | — | — | 46.11 |
| Ministral Reasoning | C02 | 受理*・不一致（予測3群、Gold1群） | 2,193 (2,120) / 7.92 | 2,311 (2,283) / 7.94 | 1,192 (1,130) / 10.56 | 929.53 |

Qwen3.5-4B ON C02はGrouping要求を1回送った後、2,472.58秒で`benchmark helper ended without bounded response`となった。完了応答・生成token数がなく、token/sは不明。これは「token生成が進んだ」と確認できる証拠ではない。同じ条件のC01 Groupingは575 tokens、うちthinking 515、生成40.50秒、約14.20 tokens/sだったが、C02の代替値にはならない。

planner結果がエラーなしだった9セルのうち、Reasoning C02はGroupingがGoldと異なる。それでも後段Category/Textは固定参照partitionで実行されたため、`受理*`を実Grouping込みのend-to-end成功と解釈しない。Qwen3.5-2B C01は3 phase応答後にsummary language違反で最終planが拒否された。Ministral Instructは両fixtureともCategoryでbreaking evidenceが未解決となり、Textへ進まなかった。

Grouping応答を保存できた12セルを探索Goldと比較すると11セルでpartitionが一致した。C01のGoldは3ファイル1群、C02は4ファイル1群で、Reasoning C02はC02を3群（1/1/2ファイル）に分割した。Grouping応答がないQwen4 ON C02と、native thought文法で失敗したReasoning C01はこの一致率の分母に含めていない。

実行は2026-10-19 10:25:30–11:53:39 UTCの88分08秒で終了した。OS監視は5秒間隔で1,057サンプルを取得し、1,057件すべて有効。観測中に熱・性能警告はなかった。最低memory free表示は27%。これは当該環境・単一実行の観測であり、他の負荷条件での性能や温度を示さない。

## 考察

- モデル別ハーネスは7設定すべて用意でき、tokenizer・template・thinking制御・KV構成をモデルごとに監査してから実行した。QwenのMamba cacheは `offset` をprefill長として更新しないため、attention cacheはoffset、Mamba cacheは2つの再帰state配列でprefix状態を判定するよう対応した。Ministralにはrole並びと `[THINK]` token形式を合わせた。
- 生成速度は同じモデル内でもfixtureとphaseで差が大きい。各セル1回のみで、C02はより大きなfixture contextを含む。prompt prefill、背景負荷、decodeを分離した因果比較や安定速度の推定には使わない。
- 16,009 tokensは応答メトリクスを取得できた35 phaseの合計。Qwen3.5-4B ON C02の失敗応答はtoken数不明で合計に含めていない。モデル間の総合tokens/sに集約せず、上表のphase別値に留める。
- Groupingの11/12一致は2つの小規模fixtureと探索Goldに限られる。後段を固定partitionに置き換えており、全工程の精度や人間承認Goldに対する正解率を示さない。今回の結果だけではデフォルトモデルを変更する根拠にならない。

## Next Steps

- この一巡は終了し、失敗セルの再試行はしない。Qwen4 ON C02はtoken進捗をライブ表示しないまま応答なしで終了したため、次の研究実行を行う場合は生成途中のtoken数とdecode速度を別ログへ出す計測を先に追加する。
- モデル採用判断を行う場合は、独立承認されたGold、要求された16ファイルfixture、十分な反復数、Grouping end-to-end採点とCategory/Textの人手ルーブリックを事前固定する。
- 現行のC01/C02はGemma再測定を含む探索比較として保存し、production default・通常CLI・Issue #182の完了条件へ外挿しない。

## 開発用チェックとGoal監査

Swift helperのDebug buildとMetal smoke（`metal_ok`）、Swiftテスト（29件とprotocol test 10件）、非推論prepare/capture/audit、14セルのtokenizer/cache audit、Python compile、`git diff --check`は成功済み。監査時に新規依存は追加していない。

今回のH182-002比較は1回のC01/C02探索fixtureで、16ファイル条件および独立Gold評価を満たさない。Issue #182のGoalは未完了として残す。報告書・設定・ログは専用worktreeに保存し、報告書は本Issueのコメントで共有した。
