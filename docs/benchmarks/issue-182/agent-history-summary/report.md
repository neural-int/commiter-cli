# agent_history導入後の共通fixture検証一覧（2026-10-27）

対象はH182-011〜024の14実験。13種類のモデル、MiniCPMのint4/int8を分けて14種類の重みを試した。登録113条件のうち実推論を開始したのは100条件で、99条件の結果が確定、1条件は生成中に中断した。残る13条件は未実行／skip。H182-024の新指示2条件はGrouping・Categoryの実応答を再利用し、Textのみ新しく生成している。

今回の単一fixtureで、Grouping完全一致・Category全正解・最終validator受理を同時に満たしたのは、H182-015のGemma ON両条件とOFF文法ありの3条件だった。MiniCPM int8の2条件は最終validatorに受理されたがCategoryは1/3で、内容まで全正解ではない。Qwen3.5-4BはGroupingで3/3を繰り返し記録した一方、Category、scope、日本語summary、出力上限が後段の障害となった。

これは各条件1回・1fixtureの探索結果であり、モデルの一般的な正解率や安定性能の順位ではない。プロンプト、KV、ランタイム、履歴原本が変わった実験は、その違いを下表に明記する。

## 1. 共通fixture・採点・数値の読み方

| 目的 | 現在のGit変更 | 期待するGrouping | Category正解 |
| --- | --- | --- | --- |
| A | Message-ID正規化実装＋対応する回帰テスト | 2ファイルを一群 | fix、breaking=false |
| B | 期限日tooltip実装＋対応テスト | 2ファイルを一群、Aと分離 | feat、breaking=false |
| C | READMEの品質チェック手順 | 独立した一群 | docs、breaking=false |

- Groupingの`n/3`は、正解3群と**ファイル集合が完全一致**した群数。Gラベルの名前は採点に使わない。3/3が正解partition。最終JSONのないケースは判定不能とし0点の正常回答にしない。
- Categoryは**正解集合と一致した群だけ**を分母にする。`1/1`は3目的すべてを分類できた意味ではない。`—`は対象なし／採点不能。
- validatorはJSON・ID・scope・日本語summaryなどホストの形式／安全性判定。受理だけではGrouping・分類・文章の意味正解を保証しない。日本語判定は日本語文字の存在を使うため、自然な日本語の保証にもならない。
- decode速度は実出力tokens総数÷`runtimeGenerationSeconds`総和。Thinking込み、入力tokens・load・prefillを除く。フェーズ速度の単純平均ではない。Pythonは公式APIの時間換算とemitted/EOS除外を使用し、Swiftとの完全同定義は主張しない。
- TAT（turnaround）はセル開始からホスト結果確定までのwall秒。到達したphaseのみで、各phaseのhelper起動・model load・prefill・decode等を含む。履歴取得の固定化ETLは別計測。途中停止の短いTATを成功処理の高速化と扱わない。
- H182-024新指示のTATとtokensは**Textのみ**。旧指示の3phase TATとは比較せず、対応する旧Textのhelper wallで対比較する。再利用G/Cは採点上の条件として残し、生成回数・速度・tokensに二重計上しない。
- RSSは約2秒ごとのOS観測最大、MLX peakはライブラリの確保量の最大。KV・weight・MLX peak・RSSを足し合わせない。GiB=2^30 bytes、MiB=2^20 bytes。native KV未exportの`0`は実容量0とは解釈せず未取得とする。
- 表は秒・token/s・MiBを原則小数3桁へ丸める。元の精度と追加の診断値・KV層情報は隣接する`measurements.json`に保持。`—`は未取得で0ではない。既存の失敗・中断を再試行して埋めていない。

### 履歴入力の世代

| 入力ID | 対象実験 | 固定したものと制約 |
| --- | --- | --- |
| F0 | H182-011 | 元の5ファイル／3目的。取得3会話・14コマンドだが16K自動投影で実Groupingのコマンドは0、H3依頼／最終応答も予算省略 |
| F1 | H182-012〜022 | 元の同じ固定fixtureを32Kへ投影。Groupingに3会話本文・14コマンドを保持。履歴partial/history_incomplete、9ブロックambiguous（候補あり4・なし5）の制約は維持 |
| F2 | H182-023〜024 | 旧private原本消失後、ユーザー承認で同じ5ファイル／3目的から履歴を再取得して固定。3会話・3編集操作。F1との履歴本文byte同一性は未保証。H024 old/new内では同一入力 |

元fixtureの固定ETLは2.422584042秒。Git収集0.349367125秒、構文・内容読取0.067673833秒、関連抽出0.021924625秒、履歴取得・照合・snapshot確認1.982554083秒。履歴本文取得内wallは1.334306秒、内容照合約1.933ms、コマンド投影約4.040ms、受信129,717 bytes／12 RPC。これらには包含関係があり全項目を再加算しない。F2固定化は約2.67秒。各セルで履歴取得を繰り返した測定ではない。

## 2. モデル・重み量子化・sampling

下の`case prefix`と実験表を結び付けることで、各セルの実行条件を特定できる。`05`は0.5B、`15`は1.5B。Qwen2.5-CoderもInstruct版。重み量子化とKV量子化は別設定。

| case prefix | モデル | 重み量子化 | 固定revision | temperature / top_p / top_k | Thinking |
| --- | --- | --- | --- | --- | --- |
| qwen25-05 | Qwen2.5-0.5B-Instruct | MLX公開4bit | a5339a4131f135d0fdc6a5c8b5bbed2753bbe0f3 | 0.7 / 0.8 / 20 | なし |
| qwen25-15 | Qwen2.5-1.5B-Instruct | MLX公開4bit | 8b403126fc14f14cfc99bb4cfa72ecbc129ea677 | 0.7 / 0.8 / 20 | なし |
| qwen25-3 | Qwen2.5-3B-Instruct | MLX公開4bit | 4f83f8f146fdf28b512a06562b671d7af4fab457 | 0.7 / 0.8 / 20 | なし |
| coder25-05 | Qwen2.5-Coder-0.5B-Instruct | MLX公開4bit | 6b16732e5af5cd9bd600186ad59fa618867ef7a4 | 0.7 / 0.8 / 20 | なし |
| coder25-15 | Qwen2.5-Coder-1.5B-Instruct | MLX公開4bit | b3252a2f97102b1fb1571fec2c9b27219a8536be | 0.7 / 0.8 / 20 | なし |
| coder25-3 | Qwen2.5-Coder-3B-Instruct | MLX公開4bit | 3dd939c621c08e5753d5b89f35a2642cd83b98ca | 0.7 / 0.8 / 20 | なし |
| qwen2-off | Qwen3.5-2B | MLX公開4bit | 674aaa7240b91e8012fcad5d791b7dfe5ba90207 | 1.0 / 1.0 / 20 | OFFのみ |
| qwen4-on/off | Qwen3.5-4B | MLX Affine4/group64 | 0e7ffd5c629ef7719d4cbc04069232580bfa9d9c | 0.6 / 0.95 / 20 | ON / OFF |
| gemma-on/off | Gemma 4 E4B-it | MLX公開4bit | 475b9088d29754a3379866cf5aeb6b41acd313c2 | 1.0 / 0.95 / 64 | ON / OFF |
| ministral-instruct | Ministral-3-3B-Instruct-2512 | MLX公開4bit | a962dcb09eee4169c890e544c9eb938f1113fdee | 0.1 / 0.95 / 20 | なし |
| ministral-reasoning | Ministral-3-3B-Reasoning-2512 | MLX公開4bit | 2cd2087aad40c28747f8ede17851de6035f12b16 | 0.7 / 0.95 / 20 | ON |
| spark | Spark-X2.6-4B | MLX Affine4/group64 | 9aa68e67d64f5728b42f15661524fb7dca8d984d | 1.0 / 0.95 / 0（無効） | OFF / Low / High |
| minicpm5-int4 | MiniCPM5-2B / openbmb | MLX Affine4/group64 | 8a9ad7539ac86281d0ac2b017ba04a5de53fe9a3 | 1.0 / 0.95 / 0（無効） | ON |
| minicpm5-int8 | MiniCPM5-2B / mlx-community | MLX Affine8/group64 | 2d20e8e672ce892d50f7265bfd3fc9b59b718f2a | 1.0 / 0.95 / 0（無効） | ON |

Qwen/Gemma/Ministralの公開元はmlx-community。Sparkは0x36/Spark-X2.6-4B-mlx-4bit。MiniCPM int4はopenbmb/MiniCPM5-2B-MLX、int8はmlx-community/MiniCPM5-2B-8bitで、公開元が異なるため量子化だけの因果比較ではない。旧モデルで4bit以外の方式・group値を現存の集計から再確認できないものは「MLX公開4bit」と記載し、KV group64から重みgroup64を推定していない。Qwen2.5のsamplingは本研究の固定値であり公式既定値との一致を主張しない。

### プロンプトのID

| ID | 実際の変更内容 |
| --- | --- |
| Q0 | 従来Qwen詳細指示＋現行agent_historyフィールド＋構造のフィールド名／型／必須／Schema／構造例。Textはrequested languageとuser内summary_language=ja |
| G0 | 採用済みGemma compactを履歴スキーマへ適応。構造説明・自動構造例あり |
| M0 | Ministral公式mistral-common/tekken経路＋Schema説明。Reasoning公式systemを保持し独自Thinking指示を重ねない |
| G1 | Gemma compactへdocs=文書のみ、Text scopeのtype語禁止、文字数ルール。偏った自動構造例を除外しフィールド／型／必須／Schemaを保持。日本語summaryを明示 |
| Q1 | 従来Qwen詳細指示を保持し、G1と同方針のdocs／scope／文字数説明を追加。Gemma compactに置き換えず、自動構造例だけ除外 |
| SP1 | G1系のSpark OFFでCategory・Text指示のみ改善。Groupingは元と同じ。意味判断と短い文章の説明を追加 |
| SP2 | 元のCategory・Textへ戻し、Thinkingを簡潔にして最終回答へ移る指示のみ変更。OFF再測定にも同じbrief文面を使用 |
| SP3 | SP2＋Groupingのラベルは無意味なID／全label使用不要／目的が整理できたら確定と明示 |
| G1-R | G1保存仕様からMiniCPM用に再構成。旧Gemma wireとのbyte同一性は未検証。Sparkのbrief／label追記は混ぜない |
| Q1-R | 現行Go system＋保存されたQ1方針から再構成。失われた旧Qwen wireとのbyte同一性は未検証 |
| Q1-R+JA | Q1-RのText systemだけへ、日本語一文、技術用語は原表記可、言語指定はsummaryのみ、JSONキー・ID・type・scope形式は維持、と3文追記 |

全実験でThinking本文を日本語に指定していない。Ministral Reasoningの公式指示は入力と同じ言語で思考・回答する指示で、混在入力のため日本語固定ではない。`summary_language=ja`はcommiter独自userフィールドで、MLXの言語強制機能ではない。

## 3. 実行環境とハーネス設定

| 設定 | 値／条件 |
| --- | --- |
| ハードウェア | 同じMacBook Air M3／統合メモリ16GB。バックグラウンド負荷・OS file cache・熱状態を完全固定していない |
| S | Go Three-phase Planner → phaseごとにSwift **Release** helperを別起動。MLX Swift 0.31.6（0bb916c67f4b9e5c682cbe02a42c701c93ab5021）、mlx-swift-lm c6446cf7bfb7cea76408013b614d4b2c530eaa03の依存固定を作業ツリーで確認。各実験のhelper/source SHA監査あり |
| S/M | Sに加えMinistralの公式mistral-common1.8.6・tekken・公式system／EOS処理。公式経路にもSchema説明後のmessagesを供給 |
| P | Go Planner → 専用Pythonアダプター → mlx-lm0.32.2、llguidance1.9.1。Swiftのモデル実装を使わない。Spark/MiniCPMとも専用venv・依存lock、MLX Python版はMiniCPM lockで0.32.4 |
| 計測時のOS／compiler版 | 旧private原本の環境ファイル消失等により、現存の公開集計から各実験の正確なmacOS／Swift compiler版を再確認できない。現在のOS版を過去値へ転記しない |
| seed / 試行 | seed144、各条件1回。warmup0、retry0、repair0。保存baselineを後続実験で再測定した件数には加えない |
| phaseとプロセス | 実Grouping結果をCategory→Textへ継承。Gold注入なし。各phase別load、model保持／prefix KV再利用なし、セル直列・最大helper1。H024新側だけG/C実応答を再利用 |
| pipeline / 診断 | 従来eager。Swiftはtoken読取・mask・状態更新の診断と進捗を保持。各phaseでCPU user/systemとwall、host投影・serialization・検証を取得可能範囲で保存 |
| 文法ON | output_constraint=json_schema。Thinking中は自由、境界後の最終JSONだけ制約。OFF/非Thinkingは開始からJSON制約 |
| 文法OFF | output_constraint=none。Schema・JSON構造指示・境界追跡・事後validatorは残す。自由文を回答として受理する設定ではない |
| 出力上限 | 下表の2048/4096はThinking＋最終回答の合計。Thinking専用上限・強制closeなし。全実験timeoutなし。H012は生成中の総context上限が未適用で中断 |
| 通常penalty | repetition=1.0。Sparkはpresence=0、frequency=0。MiniCPMはmin_p=0。旧Swiftで独立のpresence/frequency/min_pの実効値を保存集計から確認できないものは未確認 |
| 反復比較 | H020はLow Thinkingのみ、H021はその1.1を保持。H022はON=Thinkingのみ／OFF=JSON回答。生成済直近20tokens、入力prompt除外、EOS・stop・Thinking境界除外、ON最終JSONはpenaltyなし |
| Thinking本文保存 | H019〜021 SparkとH022 Qwen、H023 MiniCPMはprivate保存。H024既存Swift経路は本文を保存せず数のみ。公開版に本文・token IDs・会話内容を含めない |
| 監視 | 約2秒間隔、system CPU/GPU、プロセスRSS、MLXメモリ、swap、free%、thermalState。helper RSS12GiBや監視異常時は安全停止。物理温度・Watts・process別GPUは未取得 |

### 実験ごとの設定と結果の総括

| 実験 | 登録 | 結果確定／未実行 | Grouping全正解/確定条件 | Grouping採点可能数 | validator受理 | G/C/validator全正解 | 実験wall秒 | セルdecode範囲 tok/s | 公開元 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| H182-011 | 18 | 18確定 | 2/18 | 18 | 6/18 | 0/18 | 1137.655 | 13.82–113.29 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6412409939) |
| H182-012 | 18 | 6確定＋1中断／11未実行 | 0/6 | 5 | 2/6 | 未確定 | 1611.482 | — | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013) |
| H182-013 | 18 | 18確定 | 4/18 | 17 | 6/18 | 0/18 | 1155.651 | 16.89–112.96 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013) |
| H182-014 | 8 | 8確定 | 3/8 | 6 | 4/8 | 0/8 | 1142.918 | 15.94–24.38 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013) |
| H182-015 | 4 | 4確定 | 4/4 | 4 | 3/4 | 3/4 | 536.589 | 17.13–24.99 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741220) |
| H182-016 | 18 | 18確定 | 4/18 | 17 | 5/18 | 0/18 | 1194.874 | 17.85–121.01 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741220) |
| H182-017 | 6 | 6確定 | 2/6 | 2 | 1/6 | 0/6 | 632.842 | 22.37–30.65 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420) |
| H182-018 | 2 | 2確定 | 1/2 | 1 | 1/2 | 0/2 | 265.383 | 13.83–29.94 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420) |
| H182-019 | 2 | 2確定 | 0/2 | 1 | 0/2 | 0/2 | 198.412 | 14.88–30.08 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420) |
| H182-020 | 2 | 2確定 | 0/2 | 0 | 0/2 | 0/2 | 214.497 | 25.25–27.65 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677) |
| H182-021 | 1 | 1確定 | 0/1 | 0 | 0/1 | 0/1 | 104.566 | 27.36–27.36 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677) |
| H182-022 | 4 | 4確定 | 4/4 | 4 | 0/4 | 0/4 | 745.979 | 17.52–23.38 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677) |
| H182-023 | 4 | 4確定 | 2/4 | 4 | 2/4 | 0/4 | 973.767 | 21.10–43.50 | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6431804563) |
| H182-024 | 8 | 6確定／2skip | 4/4実生成＋2/2再利用 | 6 | 0/6 | 0/6 | 639.058 | 20.32–27.62 | 本集約でIssue初掲載 |

Grouping全正解の分母は確定条件数で、採点不能の条件も含む。「採点可能数」を併読する。たとえばH017は全条件で2/6だが、回答を得て採点できたものに限れば2/2。H018も全体1/2、採点可能1/1。H012の6確定は後続11条件と中断セルを含まない。H024の6確定にはG/Cを再利用した新Text2条件を含むため、独立した6回の全工程推論という意味ではない。

| 実験 | 入力 | 出力合計上限 | KV | prompt | 環境 | 追加条件 |
| --- | --- | --- | --- | --- | --- | --- |
| H182-011 | F0 / 16K自動選択 | 無制限 | KV8/group64 | Q0 | S | prefix別Thinking×文法あり/なし |
| H182-012 | F1 / 32768固定 | 無制限 | KV8/group64 | Q0 | S | prefix別Thinking×文法あり/なし |
| H182-013 | F1 / 32768固定 | 4096 | KV8/group64 | Q0 | S | prefix別Thinking×文法あり/なし |
| H182-014 | F1 / 32768固定 | 4096 | KV8/group64 | G0 / M0 | S/M | prefix別Thinking×文法あり/なし |
| H182-015 | F1 / 32768固定 | 4096 | native | G1 | S | prefix別Thinking×文法あり/なし |
| H182-016 | F1 / 32768固定 | 4096 | native | Q1 | S | prefix別Thinking×文法あり/なし |
| H182-017 | F1 / 32768固定 | 2048 | native | G1 | P | prefix別Thinking×文法あり/なし |
| H182-018 | F1 / 32768固定 | 2048 | native | SP1 / SP2 | P | stage1 OFF→stage2 Low、各1回 |
| H182-019 | F1 / 32768固定 | 2048 | native | SP2 | P | Low/OFF、本文保存 |
| H182-020 | F1 / 32768固定 | 2048 | native | SP2 | P | Low・文法あり・penalty1.0/1.1 |
| H182-021 | F1 / 32768固定 | 2048 | native | SP3 | P | Low・文法あり・penalty1.1 |
| H182-022 | F1 / 32768固定 | 4096 | native | Q1 | S | ON/OFF・文法あり・penalty1.0/1.1 |
| H182-023 | F2 / 32768固定 | 4096 | native | G1-R | P | prefix別Thinking×文法あり/なし |
| H182-024 | F2 / 32768固定 | 4096 | native | Q1-R / Q1-R+JA | S | ON/OFF×文法×old/new、newはTextのみ |

## 4. 全ケースの品質・速度・TAT・メモリ

表中G=Grouping、C=Category、T=Text。phase到達と正常回答は別。`grammar_failure`や`max_tokens`のTも「Text到達」に含む。pair P/Rは同じ群に属すべきファイル対のprecision/recall（未定義は—）。

### H182-011 — kimdel-inference

条件: F0、prompt=Q0、16K自動選択／出力無制限、KV8/group64、環境S。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6412409939)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | 0/3 | — | 0.20/1.00 | 拒否 | 到達 | 否 | invalid_summary_language |
| qwen25-05-none | 0/3 | — | 0.17/0.50 | 拒否 | 未実行 | 否 | invalid_assignment |
| qwen25-15-none | 0/3 | — | 0.20/1.00 | 拒否 | 到達 | 否 | invalid_schema |
| qwen25-15-json_schema | 0/3 | — | 0.00/0.00 | 受理 | 到達 | 否 | 完走 |
| qwen25-3-json_schema | 1/3 | 0/1 | —/0.00 | 受理 | 到達 | 否 | 完走 |
| qwen25-3-none | 1/3 | 0/1 | —/0.00 | 拒否 | 到達 | 否 | invalid_schema |
| coder25-05-none | 1/3 | 0/1 | —/0.00 | 拒否 | 到達 | 否 | grammar_failure |
| coder25-05-json_schema | 1/3 | 0/1 | —/0.00 | 受理 | 到達 | 否 | 完走 |
| coder25-15-json_schema | 0/3 | — | 0.00/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| coder25-15-none | 0/3 | — | 0.00/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| coder25-3-none | 0/3 | — | 0.20/1.00 | 拒否 | 到達 | 否 | invalid_schema |
| coder25-3-json_schema | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| qwen2-off-json_schema | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| qwen2-off-none | 0/3 | — | 0.00/0.00 | 拒否 | 到達 | 否 | invalid_schema |
| qwen4-off-none | 1/3 | 1/1 | 0.33/0.50 | 拒否 | 到達 | 否 | invalid_schema |
| qwen4-off-json_schema | 0/3 | — | 0.25/0.50 | 受理 | 到達 | 否 | 完走 |
| qwen4-on-json_schema | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope, invalid_summary_language |
| qwen4-on-none | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope, invalid_summary_language |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | GCT | 94 | 102.250 | 2222.190 | 5.871 | 10.744 | 419.062 | 895.625 | 到達phase計 |
| qwen25-05-none | G | 58 | 113.295 | 2148.837 | 2.168 | 4.067 | 405.719 | 868.977 | 到達phase計 |
| qwen25-15-none | GCT | 161 | 55.857 | 831.103 | 15.698 | 21.655 | 971.484 | 1452.933 | 到達phase計 |
| qwen25-15-json_schema | GCT | 238 | 52.253 | 781.531 | 18.299 | 26.320 | 987.984 | 1452.908 | 到達phase計 |
| qwen25-3-json_schema | GCT | 285 | 31.320 | 356.758 | 41.171 | 54.506 | 1818.812 | 2402.649 | 到達phase計 |
| qwen25-3-none | GCT | 316 | 27.719 | 334.074 | 43.966 | 59.635 | 1802.734 | 2402.649 | 到達phase計 |
| coder25-05-none | GCT | 342 | 105.665 | 1599.381 | 9.184 | 17.364 | 408.703 | 868.977 | 到達phase計 |
| coder25-05-json_schema | GCT | 357 | 73.690 | 1530.617 | 9.596 | 19.960 | 421.266 | 868.977 | 到達phase計 |
| coder25-15-json_schema | GC | 93 | 48.253 | 563.554 | 16.605 | 22.621 | 972.703 | 1452.908 | 到達phase計 |
| coder25-15-none | GC | 93 | 57.419 | 543.345 | 17.223 | 22.267 | 972.078 | 1452.908 | 到達phase計 |
| coder25-3-none | GCT | 150 | 27.212 | 294.450 | 44.310 | 54.715 | 1805.281 | 2391.774 | 到達phase計 |
| coder25-3-json_schema | GCT | 123 | 23.429 | 293.605 | 44.437 | 54.889 | 1821.250 | 2391.774 | 到達phase計 |
| qwen2-off-json_schema | GCT | 111 | 53.723 | 474.571 | 26.097 | 35.671 | 1918.547 | 1803.431 | 到達phase計 |
| qwen2-off-none | GCT | 530 | 37.985 | 524.935 | 25.186 | 46.737 | 1890.250 | 1803.431 | 到達phase計 |
| qwen4-off-none | GCT | 198 | 16.458 | 215.399 | 61.384 | 82.295 | 3115.344 | 3203.655 | 到達phase計 |
| qwen4-off-json_schema | GCT | 154 | 13.818 | 192.429 | 66.534 | 87.859 | 3099.406 | 3202.554 | 到達phase計 |
| qwen4-on-json_schema | GCT | 3251 | 17.093 | 215.742 | 61.254 | 260.438 | 3164.656 | 3202.025 | 到達phase計 |
| qwen4-on-none | GCT | 3297 | 17.721 | 229.797 | 57.507 | 251.650 | 3145.859 | 3202.025 | 到達phase計 |

### H182-012 — kimdel-inference-32k

条件: F1、prompt=Q0、32768固定／出力無制限、KV8/group64、環境S。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | 0/3 | — | — | 拒否 | 詳細未収録（calls=3） | 未確定 | 詳細不明 |
| qwen25-05-none | 判定不能 | — | — | 拒否 | 詳細未収録（calls=1） | 未確定 | 詳細不明 |
| qwen25-15-none | 0/3 | — | — | 拒否 | 詳細未収録（calls=3） | 未確定 | 詳細不明 |
| qwen25-15-json_schema | 0/3 | — | — | 受理 | 詳細未収録（calls=3） | 未確定 | 完走 |
| qwen25-3-json_schema | 1/3 | — | — | 受理 | 詳細未収録（calls=3） | 未確定 | 完走 |
| qwen25-3-none | 1/3 | — | — | 拒否 | 詳細未収録（calls=3） | 未確定 | 詳細不明 |
| coder25-05-none | 判定不能 | — | — | 中断 | 生成中に中断 | 未確定 | 生成33,856 tokens超・総context制御未適用 |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | 3phase | — | — | — | — | 13.640 | — | — | 到達phase計 |
| qwen25-05-none | 1phase | — | — | — | — | 5.610 | — | — | 到達phase計 |
| qwen25-15-none | 3phase | — | — | — | — | 30.220 | — | — | 到達phase計 |
| qwen25-15-json_schema | 3phase | — | — | — | — | 27.800 | — | — | 到達phase計 |
| qwen25-3-json_schema | 3phase | — | — | — | — | 87.460 | — | — | 到達phase計 |
| qwen25-3-none | 3phase | — | — | — | — | 81.760 | — | — | 到達phase計 |
| coder25-05-none | Text中断 | 33,856超（Textのみ） | — | — | — | — | — | — | 中断値・確定なし |

7番目`coder25-05-none`はText生成33,856 tokens超で中断。実行を開始したのは7条件、確定は6条件。セルTAT・decode・メモリ細目は現存集計にない。後続11条件は未実行。6セルのwallは元報告の小数2桁値のため、それ以上の精度を主張しない。

| 未実行case | 状態 |
| --- | --- |
| coder25-05-json_schema | 全phase未実行・スコア／速度／時間／メモリなし |
| coder25-15-json_schema | 全phase未実行・スコア／速度／時間／メモリなし |
| coder25-15-none | 全phase未実行・スコア／速度／時間／メモリなし |
| coder25-3-none | 全phase未実行・スコア／速度／時間／メモリなし |
| coder25-3-json_schema | 全phase未実行・スコア／速度／時間／メモリなし |
| qwen2-off-json_schema | 全phase未実行・スコア／速度／時間／メモリなし |
| qwen2-off-none | 全phase未実行・スコア／速度／時間／メモリなし |
| qwen4-off-none | 全phase未実行・スコア／速度／時間／メモリなし |
| qwen4-off-json_schema | 全phase未実行・スコア／速度／時間／メモリなし |
| qwen4-on-json_schema | 全phase未実行・スコア／速度／時間／メモリなし |
| qwen4-on-none | 全phase未実行・スコア／速度／時間／メモリなし |

### H182-013 — kimdel-inference-32k-4096

条件: F1、prompt=Q0、32768固定／出力4096、KV8/group64、環境S。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | 0/3 | — | 0.20/1.00 | 拒否 | 到達 | 否 | invalid_summary_language |
| qwen25-05-none | 判定不能 | — | — | 拒否 | 未実行 | 否 | invalid_schema |
| qwen25-15-none | 0/3 | — | 0.00/0.00 | 拒否 | 到達 | 否 | grammar_failure |
| qwen25-15-json_schema | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| qwen25-3-json_schema | 1/3 | 0/1 | 0.00/0.00 | 受理 | 到達 | 否 | 完走 |
| qwen25-3-none | 1/3 | 0/1 | 0.00/0.00 | 拒否 | 到達 | 否 | invalid_schema |
| coder25-05-none | 1/3 | 0/1 | —/0.00 | 拒否 | 到達 | 否 | max_tokens |
| coder25-05-json_schema | 1/3 | 0/1 | —/0.00 | 受理 | 到達 | 否 | 完走 |
| coder25-15-json_schema | 0/3 | — | 0.00/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| coder25-15-none | 0/3 | — | 0.00/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| coder25-3-none | 0/3 | — | 0.20/1.00 | 拒否 | 到達 | 否 | invalid_schema |
| coder25-3-json_schema | 0/3 | — | 0.00/0.00 | 受理 | 到達 | 否 | 完走 |
| qwen2-off-json_schema | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| qwen2-off-none | 0/3 | — | 0.00/0.00 | 拒否 | 到達 | 否 | invalid_schema |
| qwen4-off-none | 3/3 | 2/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_schema |
| qwen4-off-json_schema | 3/3 | 2/3 | 1.00/1.00 | 受理 | 到達 | 否 | 完走 |
| qwen4-on-json_schema | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| qwen4-on-none | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | GCT | 98 | 104.600 | 1864.379 | 8.580 | 14.361 | 418.328 | 895.820 | 到達phase計 |
| qwen25-05-none | G | 57 | 112.958 | 1869.049 | 3.422 | 5.292 | 403.750 | 867.285 | 到達phase計 |
| qwen25-15-none | GCT | 299 | 58.197 | 782.542 | 21.511 | 29.588 | 970.719 | 1547.141 | 到達phase計 |
| qwen25-15-json_schema | GCT | 90 | 54.867 | 761.668 | 21.003 | 26.052 | 984.062 | 1547.141 | 到達phase計 |
| qwen25-3-json_schema | GCT | 196 | 32.448 | 351.532 | 47.885 | 57.790 | 1818.172 | 2394.617 | 到達phase計 |
| qwen25-3-none | GCT | 224 | 20.321 | 256.009 | 65.752 | 80.420 | 1801.594 | 2394.617 | 到達phase計 |
| coder25-05-none | GCT | 4218 | 74.412 | 1487.772 | 11.876 | 71.884 | 406.125 | 898.421 | 到達phase計 |
| coder25-05-json_schema | GCT | 369 | 85.651 | 1630.113 | 10.839 | 18.843 | 422.969 | 898.421 | 到達phase計 |
| coder25-15-json_schema | GC | 93 | 55.642 | 559.927 | 20.899 | 25.036 | 972.844 | 1547.141 | 到達phase計 |
| coder25-15-none | GC | 127 | 52.277 | 522.501 | 22.396 | 27.409 | 970.906 | 1547.141 | 到達phase計 |
| coder25-3-none | GCT | 150 | 21.657 | 271.240 | 58.977 | 70.670 | 1801.000 | 2395.496 | 到達phase計 |
| coder25-3-json_schema | GCT | 161 | 24.062 | 277.681 | 59.115 | 71.272 | 1816.672 | 2408.888 | 到達phase計 |
| qwen2-off-json_schema | GCT | 118 | 52.035 | 617.495 | 24.424 | 33.822 | 1897.219 | 1797.515 | 到達phase計 |
| qwen2-off-none | GCT | 379 | 44.249 | 634.424 | 25.090 | 41.664 | 1868.578 | 1805.622 | 到達phase計 |
| qwen4-off-none | GCT | 184 | 17.155 | 229.804 | 69.272 | 87.370 | 3126.438 | 3187.793 | 到達phase計 |
| qwen4-off-json_schema | GCT | 196 | 16.885 | 233.288 | 68.233 | 87.851 | 3131.016 | 3187.793 | 到達phase計 |
| qwen4-on-json_schema | GCT | 2576 | 19.573 | 250.536 | 63.512 | 202.685 | 3142.656 | 3187.793 | 到達phase計 |
| qwen4-on-none | GCT | 2609 | 19.312 | 261.269 | 60.903 | 202.822 | 3144.031 | 3187.793 | 到達phase計 |

### H182-014 — kimdel-inference-gemma-ministral

条件: F1、prompt=G0 / M0、32768固定／出力4096、KV8/group64、環境S/M。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| gemma-on-none | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| gemma-off-none | 3/3 | 1/3 | 1.00/1.00 | 受理 | 到達 | 否 | 完走 |
| gemma-off-json_schema | 1/3 | 0/1 | 0.33/1.00 | 受理 | 到達 | 否 | 完走 |
| ministral-instruct-json_schema | 3/3 | 1/3 | 1.00/1.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| ministral-instruct-none | 3/3 | — | 1.00/1.00 | 拒否 | 未実行 | 否 | grammar_failure |
| ministral-reasoning-none | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |
| ministral-reasoning-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema | GCT | 3246 | 24.375 | 484.263 | 33.812 | 179.006 | 3212.656 | 4910.684 | 到達phase計 |
| gemma-on-none | GCT | 3246 | 21.916 | 468.313 | 34.964 | 191.616 | 3494.047 | 4910.684 | 到達phase計 |
| gemma-off-none | GCT | 233 | 15.937 | 401.434 | 42.229 | 66.269 | 2521.969 | 4910.684 | 到達phase計 |
| gemma-off-json_schema | GCT | 138 | 16.677 | 371.768 | 44.813 | 63.610 | 2783.906 | 4910.684 | 到達phase計 |
| ministral-instruct-json_schema | GC | 129 | 20.404 | 233.285 | 54.174 | 65.584 | 2809.656 | 2971.444 | 到達phase計 |
| ministral-instruct-none | GC | 319 | 18.600 | 226.739 | 55.738 | 78.046 | 2826.172 | 2971.444 | 到達phase計 |
| ministral-reasoning-none | G | 4096 | 21.045 | 223.827 | 31.319 | 228.718 | 1550.078 | 3021.807 | 到達phase計 |
| ministral-reasoning-json_schema | G | 4096 | 21.042 | 248.345 | 28.227 | 225.793 | 1109.516 | 3021.807 | 到達phase計 |

### H182-015 — kimdel-inference-gemma-native

条件: F1、prompt=G1、32768固定／出力4096、native、環境S。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741220)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema | 3/3 | 3/3 | 1.00/1.00 | 受理 | 到達 | 可 | 完走 |
| gemma-on-none | 3/3 | 3/3 | 1.00/1.00 | 受理 | 到達 | 可 | 完走 |
| gemma-off-none | 3/3 | 2/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_schema |
| gemma-off-json_schema | 3/3 | 3/3 | 1.00/1.00 | 受理 | 到達 | 可 | 完走 |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema | GCT | 3709 | 24.993 | 503.425 | 33.413 | 191.669 | 3656.250 | 4910.684 | 到達phase計 |
| gemma-on-none | GCT | 3745 | 22.506 | 481.222 | 34.955 | 209.374 | 3106.766 | 4910.684 | 到達phase計 |
| gemma-off-none | GCT | 300 | 18.771 | 445.367 | 37.755 | 62.629 | 3588.781 | 4910.684 | 到達phase計 |
| gemma-off-json_schema | GCT | 201 | 17.133 | 397.334 | 42.320 | 62.858 | 3652.938 | 4910.684 | 到達phase計 |

### H182-016 — kimdel-inference-qwen-native

条件: F1、prompt=Q1、32768固定／出力4096、native、環境S。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741220)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | 1/3 | 0/1 | —/0.00 | 受理 | 到達 | 否 | 完走 |
| qwen25-05-none | 判定不能 | — | — | 拒否 | 未実行 | 否 | invalid_schema |
| qwen25-15-none | 1/3 | — | —/0.00 | 拒否 | 未実行 | 否 | invalid_schema |
| qwen25-15-json_schema | 0/3 | — | 0.00/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| qwen25-3-json_schema | 0/3 | — | 0.00/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| qwen25-3-none | 1/3 | 1/1 | 0.50/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| coder25-05-none | 0/3 | — | 0.20/1.00 | 拒否 | 未実行 | 否 | grammar_failure |
| coder25-05-json_schema | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| coder25-15-json_schema | 0/3 | — | 0.20/1.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| coder25-15-none | 1/3 | 0/1 | —/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| coder25-3-none | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| coder25-3-json_schema | 0/3 | — | 0.20/1.00 | 受理 | 到達 | 否 | 完走 |
| qwen2-off-json_schema | 0/3 | — | 0.17/0.50 | 受理 | 到達 | 否 | 完走 |
| qwen2-off-none | 0/3 | — | 0.00/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |
| qwen4-off-none | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| qwen4-off-json_schema | 3/3 | 1/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| qwen4-on-json_schema | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_summary_language |
| qwen4-on-none | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_summary_language |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema | GCT | 427 | 99.380 | 2422.894 | 7.221 | 15.641 | 420.469 | 651.764 | 到達phase計 |
| qwen25-05-none | G | 331 | 121.006 | 2512.471 | 2.525 | 6.745 | 403.938 | 651.764 | 到達phase計 |
| qwen25-15-none | GC | 73 | 60.849 | 861.710 | 13.954 | 17.524 | 969.594 | 1512.873 | 到達phase計 |
| qwen25-15-json_schema | GC | 91 | 60.654 | 853.442 | 13.340 | 17.074 | 970.141 | 1512.873 | 到達phase計 |
| qwen25-3-json_schema | GC | 93 | 35.731 | 390.264 | 29.173 | 34.335 | 1801.094 | 2351.856 | 到達phase計 |
| qwen25-3-none | GCT | 166 | 35.805 | 386.969 | 42.195 | 50.109 | 1800.688 | 2351.856 | 到達phase計 |
| coder25-05-none | GC | 1599 | 84.689 | 2516.450 | 4.440 | 25.587 | 400.969 | 651.764 | 到達phase計 |
| coder25-05-json_schema | GCT | 109 | 97.425 | 2545.896 | 6.261 | 10.869 | 413.922 | 651.764 | 到達phase計 |
| coder25-15-json_schema | GC | 65 | 56.065 | 743.940 | 15.017 | 18.625 | 970.781 | 1512.873 | 到達phase計 |
| coder25-15-none | GC | 199 | 38.852 | 511.389 | 23.512 | 31.630 | 969.359 | 1512.873 | 到達phase計 |
| coder25-3-none | GCT | 128 | 22.491 | 277.781 | 57.380 | 70.094 | 1799.969 | 2351.856 | 到達phase計 |
| coder25-3-json_schema | GCT | 113 | 32.545 | 320.902 | 49.669 | 59.847 | 1817.609 | 2351.856 | 到達phase計 |
| qwen2-off-json_schema | GCT | 193 | 52.936 | 637.330 | 24.179 | 36.247 | 1885.281 | 1870.239 | 到達phase計 |
| qwen2-off-none | GC | 194 | 60.224 | 682.300 | 16.414 | 25.397 | 1865.547 | 1870.239 | 到達phase計 |
| qwen4-off-none | GCT | 179 | 17.845 | 249.330 | 63.366 | 80.357 | 3120.578 | 3217.433 | 到達phase計 |
| qwen4-off-json_schema | GCT | 204 | 18.289 | 246.222 | 64.166 | 82.846 | 3124.375 | 3217.433 | 到達phase計 |
| qwen4-on-json_schema | GCT | 5348 | 23.148 | 253.728 | 62.244 | 301.399 | 3122.031 | 3217.433 | 到達phase計 |
| qwen4-on-none | GCT | 5424 | 22.493 | 258.078 | 61.195 | 309.607 | 2837.109 | 3217.433 | 到達phase計 |

### H182-017 — spark

条件: F1、prompt=G1、32768固定／出力2048、native、環境P。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| spark-off-json_schema | 3/3 | 2/3 | 1.00/1.00 | 受理 | 到達 | 否 | 完走 |
| spark-off-none | 3/3 | 1/3 | 1.00/1.00 | 拒否 | 到達 | 否 | runtime_error |
| spark-low-none | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |
| spark-low-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |
| spark-high-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |
| spark-high-none | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-off-json_schema | GCT | 248 | 29.992 | 287.966 | 61.466 | 77.820 | — | 3918.663 | 到達phase計 |
| spark-off-none | GCT | 257 | 30.655 | 275.485 | 64.254 | 78.604 | — | 3918.663 | 到達phase計 |
| spark-low-none | G | 2048 | 23.164 | 255.792 | 26.092 | 116.847 | — | 3918.663 | 到達phase計 |
| spark-low-json_schema | G | 2048 | 23.118 | 273.697 | 24.385 | 115.661 | — | 3918.663 | 到達phase計 |
| spark-high-json_schema | G | 2048 | 22.689 | 268.425 | 24.864 | 117.882 | — | 3918.663 | 到達phase計 |
| spark-high-none | G | 2048 | 22.372 | 264.152 | 25.266 | 119.252 | — | 3918.663 | 到達phase計 |

受理されたSpark OFFにもCategoryのメールfix→feat誤りが残り、Textには不自然／不正確なUI説明がある。形式受理を内容全正解へ加算しない。

### H182-018 — spark

条件: F1、prompt=SP1 / SP2、32768固定／出力2048、native、環境P。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| stage1-spark-off-json_schema | 3/3 | 2/3 | 1.00/1.00 | 受理 | 到達 | 否 | 完走 |
| stage2-spark-low-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| stage1-spark-off-json_schema | GCT | 223 | 13.828 | 291.369 | 61.252 | 91.676 | 2656.328 | 3918.663 | 到達phase計 |
| stage2-spark-low-json_schema | G | 2048 | 29.945 | 73.857 | 90.838 | 166.224 | 2126.594 | 3918.663 | 到達phase計 |

受理されたSpark OFFにもCategoryのメールfix→feat誤りが残り、Textには不自然／不正確なUI説明がある。形式受理を内容全正解へ加算しない。

### H182-019 — spark

条件: F1、prompt=SP2、32768固定／出力2048、native、環境P。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |
| spark-off-json_schema | 1/3 | 1/1 | —/0.00 | 拒否 | 未実行 | 否 | unresolved_breaking_evidence |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-json_schema | G | 2048 | 30.082 | 297.684 | 22.537 | 94.856 | — | 3918.663 | 到達phase計 |
| spark-off-json_schema | GC | 153 | 14.876 | 147.563 | 85.367 | 101.325 | — | 3918.663 | 到達phase計 |

### H182-020 — spark

条件: F1、prompt=SP2、32768固定／出力2048、native、環境P。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp10-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |
| spark-low-rp11-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp10-json_schema | G | 2048 | 27.654 | 285.086 | 23.533 | 102.339 | — | 3918.663 | 到達phase計 |
| spark-low-rp11-json_schema | G | 2048 | 25.253 | 263.649 | 25.447 | 109.814 | — | 3918.663 | 到達phase計 |

### H182-021 — spark

条件: F1、prompt=SP3、32768固定／出力2048、native、環境P。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp11-json_schema | 判定不能 | — | — | 拒否 | 未実行 | 否 | max_tokens |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp11-json_schema | G | 2048 | 27.362 | 284.338 | 24.028 | 103.442 | 966.891 | 3918.663 | 到達phase計 |

### H182-022 — qwen-repetition

条件: F1、prompt=Q1、32768固定／出力4096、native、環境S。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| qwen4-on-rp10-json_schema | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_summary_language |
| qwen4-on-rp11-json_schema | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_summary_language |
| qwen4-off-rp10-json_schema | 3/3 | 1/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| qwen4-off-rp11-json_schema | 3/3 | 1/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen4-on-rp10-json_schema | GCT | 5348 | 23.380 | 285.519 | 55.313 | 293.353 | 884.766 | 3217.433 | 到達phase計 |
| qwen4-on-rp11-json_schema | GCT | 4634 | 22.025 | 265.969 | 59.379 | 277.709 | 2096.859 | 3217.433 | 到達phase計 |
| qwen4-off-rp10-json_schema | GCT | 204 | 17.520 | 232.941 | 67.824 | 87.774 | 3136.641 | 3217.433 | 到達phase計 |
| qwen4-off-rp11-json_schema | GCT | 204 | 17.624 | 235.745 | 67.017 | 86.923 | 2969.938 | 3217.433 | 到達phase計 |

### H182-023 — minicpm5

条件: F2、prompt=G1-R、32768固定／出力4096、native、環境P。[元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6431804563)。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| minicpm5-int4-json_schema | 1/3 | 1/1 | —/0.00 | 拒否 | 到達 | 否 | max_tokens |
| minicpm5-int4-none | 1/3 | — | —/0.00 | 拒否 | 未実行 | 否 | invalid_schema |
| minicpm5-int8-none | 3/3 | 1/3 | 1.00/1.00 | 受理 | 到達 | 否 | 完走 |
| minicpm5-int8-json_schema | 3/3 | 1/3 | 1.00/1.00 | 受理 | 到達 | 否 | 完走 |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| minicpm5-int4-json_schema | GCT | 8705 | 43.496 | 501.040 | 32.089 | 239.114 | 1784.750 | 2240.238 | 到達phase計 |
| minicpm5-int4-none | GC | 4589 | 35.828 | 446.675 | 24.689 | 157.794 | 1626.844 | 2240.238 | 到達phase計 |
| minicpm5-int8-none | GCT | 5172 | 22.118 | 405.828 | 38.524 | 282.118 | 2826.328 | 3363.020 | 到達phase計 |
| minicpm5-int8-json_schema | GCT | 5164 | 21.097 | 395.375 | 39.542 | 294.342 | 2792.016 | 3363.020 | 到達phase計 |

### H182-024 — qwen-japanese

条件: F2、prompt=Q1-R / Q1-R+JA、32768固定／出力4096、native、環境S。本集約でIssue初掲載。

| case | Grouping | Category | pair P/R | validator | Text | G/C/V全正解 | 停止／拒否理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| on-json_schema-old | 3/3 | — | 1.00/1.00 | 拒否 | 未実行 | 否 | max_tokens |
| on-none-old | 3/3 | — | 1.00/1.00 | 拒否 | 未実行 | 否 | max_tokens |
| off-none-old | 3/3 | 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| off-none-new | 再利用 3/3 | 再利用 3/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| off-json_schema-old | 3/3 | 2/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| off-json_schema-new | 再利用 3/3 | 再利用 2/3 | 1.00/1.00 | 拒否 | 到達 | 否 | invalid_scope |
| on-json_schema-new | — | — | — | skip | 未実行 | 未評価 | 対応oldのCategory max_tokensでText未到達 |
| on-none-new | — | — | — | skip | 未実行 | 未評価 | 対応oldのCategory max_tokensでText未到達 |

| case | 実推論phase | 生成tokens | decode tok/s | prefill tok/s | prefill計秒 | TAT秒 | RSS観測最大MiB | MLX peak MiB | 集計範囲 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| on-json_schema-old | GC | 4623 | 26.472 | 314.141 | 34.965 | 215.574 | — | 3217.433 | 到達phase計 |
| on-none-old | GC | 4625 | 26.320 | 319.826 | 34.344 | 216.345 | — | 3217.433 | 到達phase計 |
| off-none-old | GCT | 190 | 27.617 | 296.973 | 53.264 | 68.889 | — | 3217.794 | 到達phase計 |
| off-none-new | T | 82 | 27.452 | 295.503 | 16.592 | 22.076 | — | 3204.810 | Textのみ |
| off-json_schema-old | GCT | 191 | 20.319 | 248.063 | 63.766 | 81.177 | — | 3217.794 | 到達phase計 |
| off-json_schema-new | T | 82 | 20.385 | 259.994 | 18.858 | 25.836 | — | 3204.810 | Textのみ |
| on-json_schema-new | 0 | — | — | — | — | — | — | — | skip・推論なし |
| on-none-new | 0 | — | — | — | — | — | — | — | skip・推論なし |

`on-json_schema-new`と`on-none-new`は対応oldのCategoryが4096上限でTextへ到達しなかったためskip。新旧Textの2組ではoldも既に日本語3/3であり、英語→日本語への改善効果は未評価。

Textのみの対比較（G/C再利用を含まない）。

| 組 | system | 日本語summary | scope違反数 | Text実出力tokens | prefill秒 | decode秒 | decode tok/s | Text helper wall秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| off-none | old | 3/3 | 3 | 83 | 16.758 | 3.054 | 27.179 | 21.975 |
| off-none | new | 3/3 | 1 | 82 | 16.592 | 2.987 | 27.452 | 21.348 |
| off-json_schema | old | 3/3 | 1 | 68 | 20.065 | 3.507 | 19.387 | 26.820 |
| off-json_schema | new | 3/3 | 1 | 82 | 18.858 | 4.023 | 20.385 | 25.056 |

## 5. CPU・GPU・メモリ・swap・熱状態

各セルのRSSがない場合に、run全体のRSSをそのセル固有値として埋めていない。以下は計測区間ごとの集計。system CPU/GPU/swapは他アプリの影響を含む。thermalStateのfairを実クロック低下・スロットリングの確証としない。

| 実験 | 集計区間 | CPU平均% | GPU平均% | GPU min/max% | RSS最大MiB | MLX peak MiB | swap開始/終了/最大MiB | free min/max% | thermal件数 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| H182-011 | run全体 | 21.40 | — | 0.00/100.00 | 3164.160 | 3204.096 | 2066.19/3588.69/— | — | nominal:57, fair:511 |
| H182-012 | run全体 | — | — | — | — | — | — | — | — |
| H182-013 | run全体 | 8.13 | — | 0.00/100.00 | 3144.031 | 3187.793 | 2657.31/2641.31/2657.31 | 43.00/71.00 | nominal:46, fair:530 |
| H182-014 | run全体 | 14.77 | — | 0.00/100.00 | 3494.047 | 4910.684 | 2599.12/2694.94/2942.94 | 30.00/70.00 | nominal:91, fair:480 |
| H182-015 | run全体 | 9.99 | — | 0.00/100.00 | 3656.250 | 4910.684 | 2662.94/2654.94/2662.94 | 29.00/69.00 | nominal:82, fair:186 |
| H182-016 | run全体 | 12.11 | — | 0.00/100.00 | 3124.375 | 3217.433 | 2582.94/2558.94/2582.94 | 42.00/72.00 | nominal:56, fair:540 |
| H182-017 | run全体 | 10.43 | — | 0.00/100.00 | 2615.328 | 3918.663 | 2542.94/2542.94/2542.94 | 28.00/67.00 | nominal:58, fair:258 |
| H182-018 | stage1-spark-off-json_schema | 15.51 | — | 4.00/100.00 | 2656.328 | 3918.663 | 2542.94/2542.94/2542.94 | 28.00/68.00 | nominal:44 |
| H182-018 | stage2-spark-low-json_schema | 11.85 | — | 0.00/100.00 | 2126.594 | 3918.663 | 2542.94/2534.94/2542.94 | 32.00/58.00 | nominal:74, fair:9 |
| H182-019 | run全体 | 6.81 | — | 0.00/100.00 | 2642.906 | 3918.663 | 2494.94/2494.94/2494.94 | 31.00/70.00 | nominal:35, fair:64 |
| H182-020 | run全体 | 43.05 | — | 0.00/100.00 | 2427.141 | 3918.663 | 2382.94/2382.94/2382.94 | 21.00/62.00 | nominal:53, fair:54 |
| H182-021 | run全体 | 48.46 | — | 17.00/100.00 | 966.891 | 3918.663 | 2382.94/2480.81/2480.81 | 19.00/63.00 | nominal:53 |
| H182-022 | qwen4-on-rp10-json_schema | 31.05 | 89.48 | —/100.00 | 884.766 | — | 2472.81/2472.81/2472.81 | 31.00/— | nominal:37, fair:110 |
| H182-022 | qwen4-on-rp11-json_schema | 11.17 | 91.46 | —/100.00 | 2096.859 | — | 2472.81/2472.81/2472.81 | 35.00/— | fair:138 |
| H182-022 | qwen4-off-rp10-json_schema | 9.97 | 92.18 | —/100.00 | 3136.641 | — | 2472.81/2472.81/2472.81 | 35.00/— | fair:44 |
| H182-022 | qwen4-off-rp11-json_schema | 7.90 | 93.88 | —/100.00 | 2969.938 | — | 2472.81/2472.81/2472.81 | 40.00/— | fair:43 |
| H182-023 | minicpm5-int4-json_schema | 17.05 | 94.56 | 0.00/100.00 | 1784.750 | 2240.238 | 0.00/0.00/0.00 | 45.00/— | nominal:40, fair:65 |
| H182-023 | minicpm5-int4-none | 18.88 | 93.62 | 5.00/100.00 | 1626.844 | 2240.238 | 0.00/0.00/0.00 | 42.00/— | fair:77 |
| H182-023 | minicpm5-int8-none | 40.96 | 89.41 | 0.00/100.00 | 2826.328 | 3363.020 | 0.00/0.00/0.00 | 39.00/— | fair:136 |
| H182-023 | minicpm5-int8-json_schema | 51.19 | 88.75 | 0.00/100.00 | 2792.016 | 3363.020 | 0.00/0.00/0.00 | 39.00/— | fair:142 |
| H182-024 | execution | 12.34 | 90.39 | 0.00/100.00 | 1464.562 | — | 0.00/377.88/377.88 | — | nominal:109 |
| H182-024 | continuation | 12.76 | 89.11 | 0.00/100.00 | 3123.047 | — | 361.88/551.62/551.62 | — | nominal:128, fair:83 |

H011のrun資源値は元goalの丸め値。H012は監視803件・fatalなしだが詳細の数値集計は現存資料にない。H019〜020等でrun資源しかないものは、その値を各セルへコピーしていない。H023は開始直後RSSが53.565秒、thermalが27.125秒欠測しており、初期peakを保証しない。H024はexecution／continuationの別区間であり、人の操作待ちは実験wall合計639.058秒に含めない。

### native KVと計測上の制約

| 経路 | 実配列／容量 |
| --- | --- |
| KV8 | 要求8bit/group64。各phaseの取得済KV bytesは付録に掲載。MLX cacheMemoryBytes（allocator cache）とは別 |
| Swift native | workingCache=nilの内部生成に委譲し、配列dtype／bytesを未export。0bytesという結果ではなく未取得 |
| Spark native | bfloat16。27層RotatingKVCache（window512）＋9層KVCache。正常OFFの終端容量G297／C261／T252MiB、初回Low/High上限時369MiB。JSON parse失敗Text等の終端snapshot欠測を0としない |
| MiniCPM native | bfloat16、42層KVCache。phase別の終端容量を付録・JSONに保持 |

## 6. 結果から確認できた範囲

- GemmaはH014のThinking ONでGrouping0/3だったが、改訂compact＋nativeのH015では4条件すべて3/3、うち3条件が分類とvalidatorも満たした。promptとKVを同時に変更しているため、どちらだけの効果とは断定できない。
- Qwen4Bは32K投影でGrouping3/3を得られる条件が増えた。native＋改訂promptや反復penaltyでも、Categoryのfix/feat判断、禁止scope、英語summaryが残った。小型Qwenは全統合／実装・テスト分離が多く、Textまで進むことと目的別Groupingの正解は一致しない。
- Ministral InstructはGrouping3/3でもCategoryで止まり、Reasoningは4096以内でGrouping最終回答に到達しなかった。
- SparkはOFFでGrouping3/3を得たが分類・文章に問題が残った。Low/Highは2048の全量をThinkingに消費。brief、penalty1.1、label明示でも最終回答には到達しなかった。penalty1.1で同一token長反復が1631から最大2へ減っても、回答成功には結び付いていない。
- MiniCPM int8は両文法条件でGrouping3/3・validator受理だがCategory1/3。int4は5singletonへ分離し正解群1/3。重み公開元が異なるため、int8化だけが原因とは確定できない。
- 日本語のsystem明示比較H024はOFF2組のみ成立し、旧指示でも日本語だったため主仮説を確かめられていない。ONのCategory上限到達を、新Text指示の効果なしと解釈しない。

モデル採用の一般的判断・安定性・複数fixture／複数seedの精度は未確定。今回の作業は既存計測の集約であり、追加推論・条件変更・欠測を補う再測定はしていない。

## 7. 全フェーズの計測値

phaseはG=Grouping、C=Category、T=Text。すべて実推論phaseのみを掲載し、H024新側の再利用G/Cは除外。H012の詳細は現存しないため、セル結果以上の値を補っていない。

※Thinking数はH019〜021/Spark・H022/Qwen・H023/MiniCPMでemitted本文の区間計測があるものを採用。従来SwiftのnativeThoughtTokensはsampling側のrawカウンタで、境界・lookaheadとの差を含み得る。4096上限時の4097を実出力4097と扱わない。旧Qwenの一部summaryにはThinking数が未収録で、その場合は—。最終tokensも厳密区間値がないものは総数から推測せず—にする。

grammar準備はsetupに内包され、generation wallはprefill/decodeを含み、helper wallはそれら全体を含む。CPU診断のtoken.item読取wallはGPU待ちを含み得るためCPU実行時間と同一視しない。追加のread/mask/update診断、prefix値、KV各層dtype/shape等はJSONに原値を保持。

### H182-011 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 4658 | 48 | — | — | 2.176 | 2140.778 | 0.486 | 98.693 | — | completed |
| qwen25-05-json_schema/C | 4242 | 24 | — | — | 1.851 | 2291.375 | 0.215 | 111.629 | — | completed |
| qwen25-05-json_schema/T | 4147 | 22 | — | — | 1.844 | 2248.792 | 0.218 | 100.936 | — | completed |
| qwen25-05-none/G | 4658 | 58 | — | — | 2.168 | 2148.837 | 0.512 | 113.295 | — | completed |
| qwen25-15-none/G | 4658 | 57 | — | — | 5.660 | 823.011 | 1.104 | 51.652 | — | completed |
| qwen25-15-none/C | 4242 | 29 | — | — | 5.076 | 835.656 | 0.502 | 57.739 | — | completed |
| qwen25-15-none/T | 4147 | 75 | — | — | 4.962 | 835.674 | 1.277 | 58.752 | — | completed |
| qwen25-15-json_schema/G | 4658 | 42 | — | — | 5.802 | 802.870 | 0.716 | 58.688 | — | completed |
| qwen25-15-json_schema/C | 4929 | 65 | — | — | 6.396 | 770.632 | 1.137 | 57.152 | — | completed |
| qwen25-15-json_schema/T | 4714 | 131 | — | — | 6.101 | 772.666 | 2.702 | 48.486 | — | completed |
| qwen25-3-json_schema/G | 4658 | 41 | — | — | 12.514 | 372.221 | 1.219 | 33.632 | — | completed |
| qwen25-3-json_schema/C | 5127 | 81 | — | — | 14.517 | 353.166 | 2.471 | 32.779 | — | completed |
| qwen25-3-json_schema/T | 4903 | 163 | — | — | 14.139 | 346.761 | 5.409 | 30.133 | — | completed |
| qwen25-3-none/G | 4658 | 41 | — | — | 13.296 | 350.318 | 1.209 | 33.911 | — | completed |
| qwen25-3-none/C | 5127 | 81 | — | — | 14.979 | 342.279 | 2.425 | 33.400 | — | completed |
| qwen25-3-none/T | 4903 | 194 | — | — | 15.691 | 312.477 | 7.766 | 24.981 | — | completed |
| coder25-05-none/G | 4658 | 41 | — | — | 2.402 | 1939.088 | 0.371 | 110.661 | — | completed |
| coder25-05-none/C | 5127 | 137 | — | — | 3.694 | 1388.110 | 1.275 | 107.452 | — | completed |
| coder25-05-none/T | 4903 | 164 | — | — | 3.088 | 1587.820 | 1.591 | 103.070 | — | grammar_failure |
| coder25-05-json_schema/G | 4658 | 41 | — | — | 2.990 | 1557.908 | 1.029 | 39.827 | — | completed |
| coder25-05-json_schema/C | 5127 | 81 | — | — | 3.197 | 1603.940 | 0.758 | 106.827 | — | completed |
| coder25-05-json_schema/T | 4903 | 235 | — | — | 3.410 | 1437.948 | 3.057 | 76.875 | — | completed |
| coder25-15-json_schema/G | 4658 | 41 | — | — | 7.686 | 606.001 | 0.958 | 42.809 | — | completed |
| coder25-15-json_schema/C | 4700 | 52 | — | — | 8.919 | 526.972 | 0.970 | 53.630 | — | completed |
| coder25-15-none/G | 4658 | 41 | — | — | 8.540 | 545.430 | 0.733 | 55.960 | — | completed |
| coder25-15-none/C | 4700 | 52 | — | — | 8.683 | 541.294 | 0.887 | 58.624 | — | completed |
| coder25-3-none/G | 4658 | 57 | — | — | 15.774 | 295.299 | 1.730 | 32.942 | — | completed |
| coder25-3-none/C | 4242 | 29 | — | — | 14.537 | 291.816 | 0.825 | 35.172 | — | completed |
| coder25-3-none/T | 4147 | 64 | — | — | 13.999 | 296.227 | 2.957 | 21.641 | — | completed |
| coder25-3-json_schema/G | 4658 | 47 | — | — | 15.656 | 297.531 | 1.378 | 34.117 | — | completed |
| coder25-3-json_schema/C | 4242 | 24 | — | — | 14.165 | 299.461 | 0.703 | 34.121 | — | completed |
| coder25-3-json_schema/T | 4147 | 52 | — | — | 14.616 | 283.726 | 3.169 | 16.409 | — | completed |
| qwen2-off-json_schema/G | 4443 | 52 | — | — | 10.458 | 424.833 | 0.961 | 54.111 | — | completed |
| qwen2-off-json_schema/C | 4017 | 25 | — | — | 9.134 | 439.804 | 0.422 | 59.283 | — | completed |
| qwen2-off-json_schema/T | 3925 | 34 | — | — | 6.505 | 603.342 | 0.683 | 49.747 | — | completed |
| qwen2-off-none/G | 4443 | 63 | — | — | 9.071 | 489.806 | 1.086 | 58.016 | — | completed |
| qwen2-off-none/C | 4475 | 96 | — | — | 8.884 | 503.719 | 2.953 | 32.513 | — | completed |
| qwen2-off-none/T | 4303 | 371 | — | — | 7.231 | 595.070 | 9.914 | 37.421 | — | completed |
| qwen4-off-none/G | 4443 | 46 | — | — | 21.835 | 203.476 | 2.936 | 15.665 | — | completed |
| qwen4-off-none/C | 4475 | 55 | — | — | 21.287 | 210.224 | 3.278 | 16.779 | — | completed |
| qwen4-off-none/T | 4304 | 97 | — | — | 18.262 | 235.687 | 5.816 | 16.678 | — | completed |
| qwen4-off-json_schema/G | 4443 | 41 | — | — | 20.289 | 218.981 | 3.170 | 12.936 | — | completed |
| qwen4-off-json_schema/C | 4246 | 48 | — | — | 24.221 | 175.300 | 3.562 | 13.475 | — | completed |
| qwen4-off-json_schema/T | 4114 | 65 | — | — | 22.023 | 186.807 | 4.413 | 14.729 | — | completed |
| qwen4-on-json_schema/G | 4441 | 1914 | 1861 | — | 24.016 | 184.916 | 111.406 | 17.180 | — | completed |
| qwen4-on-json_schema/C | 4473 | 859 | 809 | — | 18.168 | 246.197 | 51.099 | 16.810 | — | completed |
| qwen4-on-json_schema/T | 4301 | 478 | 421 | — | 19.069 | 225.549 | 27.692 | 17.261 | — | completed |
| qwen4-on-none/G | 4441 | 1911 | 1861 | — | 20.052 | 221.471 | 104.252 | 18.331 | — | completed |
| qwen4-on-none/C | 4473 | 907 | 809 | — | 18.982 | 235.650 | 54.091 | 16.768 | — | completed |
| qwen4-on-none/T | 4301 | 479 | 421 | — | 18.474 | 232.819 | 27.713 | 17.285 | — | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 0.403 | 0.030 | 0.328 | 0.329 | 2.663 | — | — | 300.420 | 60.228 | 868.977 | 30.891 |
| qwen25-05-json_schema/C | 0.390 | 0.026 | 0.346 | 0.347 | 2.067 | — | — | 296.670 | 21.623 | 895.625 | 26.695 |
| qwen25-05-json_schema/T | 0.337 | 0.025 | 0.332 | 0.674 | 2.062 | — | — | 295.920 | 21.627 | 854.673 | 26.147 |
| qwen25-05-none/G | 0.331 | 0.028 | 0.327 | 0.327 | 2.680 | — | — | 300.420 | 57.260 | 868.977 | 30.891 |
| qwen25-15-none/G | 0.359 | 0.028 | 0.331 | 0.331 | 6.763 | — | — | 904.717 | 81.082 | 1452.908 | 72.080 |
| qwen25-15-none/C | 0.410 | 0.027 | 0.377 | 0.378 | 5.579 | — | — | 895.233 | 13.290 | 1429.081 | 62.289 |
| qwen25-15-none/T | 0.362 | 0.025 | 0.330 | 0.331 | 6.239 | — | — | 897.717 | 74.085 | 1452.933 | 64.729 |
| qwen25-15-json_schema/G | 0.352 | 0.028 | 0.328 | 0.329 | 6.518 | — | — | 901.217 | 16.259 | 1452.908 | 68.361 |
| qwen25-15-json_schema/C | 0.372 | 0.030 | 0.349 | 0.352 | 7.534 | — | — | 908.217 | 87.847 | 1447.291 | 75.610 |
| qwen25-15-json_schema/T | 0.388 | 0.030 | 0.366 | 0.773 | 8.803 | — | — | 905.592 | 86.801 | 1429.081 | 72.864 |
| qwen25-3-json_schema/G | 0.606 | 0.028 | 0.331 | 0.332 | 13.733 | — | — | 1745.809 | 16.895 | 2391.774 | 87.893 |
| qwen25-3-json_schema/C | 0.604 | 0.032 | 0.359 | 0.363 | 16.989 | — | — | 1758.184 | 111.364 | 2402.649 | 101.377 |
| qwen25-3-json_schema/T | 0.612 | 0.031 | 0.366 | 0.753 | 19.549 | — | — | 1754.809 | 107.755 | 2401.217 | 96.876 |
| qwen25-3-none/G | 0.608 | 0.031 | 0.385 | 0.385 | 14.506 | — | — | 1745.809 | 13.926 | 2391.774 | 87.893 |
| qwen25-3-none/C | 0.611 | 0.034 | 0.360 | 0.360 | 17.404 | — | — | 1758.184 | 108.396 | 2402.649 | 101.377 |
| qwen25-3-none/T | 0.606 | 0.032 | 0.365 | 0.366 | 23.457 | — | — | 1754.809 | 104.786 | 2401.217 | 96.876 |
| coder25-05-none/G | 0.406 | 0.029 | 0.342 | 0.342 | 2.773 | — | — | 298.920 | 27.811 | 868.977 | 29.298 |
| coder25-05-none/C | 0.417 | 0.040 | 0.483 | 0.484 | 4.969 | — | — | 303.420 | 52.121 | 868.977 | 33.792 |
| coder25-05-none/T | 0.682 | 0.072 | 0.728 | 0.729 | 4.679 | — | — | 0.003 | 352.537 | 865.781 | 32.292 |
| coder25-05-json_schema/G | 0.444 | 0.033 | 0.481 | 0.482 | 4.020 | — | — | 298.920 | 26.811 | 868.977 | 29.298 |
| coder25-05-json_schema/C | 0.674 | 0.045 | 0.437 | 0.441 | 3.955 | — | — | 303.420 | 55.091 | 868.977 | 33.792 |
| coder25-05-json_schema/T | 0.438 | 0.038 | 0.429 | 0.974 | 6.467 | — | — | 301.920 | 53.595 | 865.781 | 32.292 |
| coder25-15-json_schema/G | 0.465 | 0.038 | 0.468 | 0.470 | 8.647 | — | — | 901.217 | 16.259 | 1452.908 | 68.361 |
| coder25-15-json_schema/C | 1.057 | 0.051 | 0.639 | 0.642 | 9.889 | — | — | 905.592 | 84.926 | 1425.986 | 72.603 |
| coder25-15-none/G | 0.531 | 0.048 | 0.400 | 0.400 | 9.273 | — | — | 901.217 | 13.290 | 1452.908 | 68.361 |
| coder25-15-none/C | 0.706 | 0.049 | 0.623 | 0.624 | 9.570 | — | — | 905.592 | 81.957 | 1425.986 | 72.603 |
| coder25-3-none/G | 0.623 | 0.032 | 0.370 | 0.370 | 17.505 | — | — | 1750.309 | 100.286 | 2391.774 | 92.674 |
| coder25-3-none/C | 1.126 | 0.074 | 0.588 | 0.589 | 15.361 | — | — | 1738.027 | 13.848 | 2358.866 | 80.086 |
| coder25-3-none/T | 0.615 | 0.029 | 0.393 | 0.394 | 16.957 | — | — | 1741.309 | 91.208 | 2339.973 | 83.224 |
| coder25-3-json_schema/G | 0.479 | 0.030 | 0.349 | 0.351 | 17.033 | — | — | 1745.809 | 16.895 | 2391.774 | 87.893 |
| coder25-3-json_schema/C | 0.642 | 0.056 | 0.768 | 0.770 | 14.869 | — | — | 1738.027 | 16.817 | 2358.866 | 80.086 |
| coder25-3-json_schema/T | 0.547 | 0.031 | 0.424 | 1.046 | 17.786 | — | — | 1736.809 | 16.817 | 2339.973 | 78.442 |
| qwen2-off-json_schema/G | 0.947 | 0.051 | 0.679 | 0.680 | 11.420 | — | — | 1059.238 | 57.595 | 1803.431 | 48.018 |
| qwen2-off-json_schema/C | 0.995 | 0.059 | 1.518 | 1.520 | 9.556 | — | — | 1056.644 | 55.173 | 1792.144 | 45.291 |
| qwen2-off-json_schema/T | 0.915 | 0.048 | 0.643 | 1.314 | 7.189 | — | — | 1056.238 | 54.798 | 1775.955 | 44.811 |
| qwen2-off-none/G | 0.933 | 0.052 | 0.641 | 0.642 | 10.157 | — | — | 1059.238 | 56.623 | 1803.431 | 48.018 |
| qwen2-off-none/C | 1.171 | 0.076 | 1.062 | 1.063 | 11.838 | — | — | 1059.425 | 71.048 | 1785.456 | 48.217 |
| qwen2-off-none/T | 1.627 | 0.061 | 0.773 | 0.773 | 17.146 | — | — | 1059.988 | 57.357 | 1786.667 | 48.814 |
| qwen4-off-none/G | 1.615 | 0.063 | 0.952 | 0.954 | 24.777 | — | — | 2385.933 | 131.000 | 3203.655 | 127.484 |
| qwen4-off-none/C | 1.551 | 0.059 | 0.806 | 0.807 | 24.565 | — | — | 2386.418 | 161.875 | 3183.702 | 128.016 |
| qwen4-off-none/T | 1.302 | 0.064 | 0.741 | 0.742 | 24.080 | — | — | 2383.668 | 154.234 | 3180.056 | 125.492 |
| qwen4-off-json_schema/G | 1.155 | 0.055 | 0.722 | 0.724 | 23.460 | — | — | 2385.918 | 162.141 | 3199.748 | 127.484 |
| qwen4-off-json_schema/C | 1.518 | 0.073 | 0.994 | 0.996 | 27.784 | — | — | 2382.668 | 152.608 | 3189.539 | 124.562 |
| qwen4-off-json_schema/T | 2.078 | 0.059 | 0.833 | 1.600 | 26.436 | — | — | 2380.933 | 149.984 | 3202.554 | 122.570 |
| qwen4-on-json_schema/G | 1.295 | 0.056 | 0.779 | 0.780 | 135.422 | — | — | 2415.433 | 158.718 | 3186.107 | 157.234 |
| qwen4-on-json_schema/C | 1.357 | 0.060 | 0.762 | 0.764 | 69.267 | — | — | 2398.965 | 142.607 | 3190.264 | 140.633 |
| qwen4-on-json_schema/T | 1.569 | 0.058 | 0.779 | 1.536 | 46.761 | — | — | 2387.980 | 134.588 | 3202.025 | 129.609 |
| qwen4-on-none/G | 1.210 | 0.053 | 0.739 | 0.740 | 124.304 | — | — | 2415.418 | 157.778 | 3186.107 | 157.234 |
| qwen4-on-none/C | 1.562 | 0.066 | 0.909 | 0.909 | 73.073 | — | — | 2398.918 | 174.575 | 3190.264 | 140.633 |
| qwen4-on-none/T | 1.324 | 0.059 | 0.788 | 0.789 | 46.186 | — | — | 2387.949 | 133.812 | 3202.025 | 129.609 |


### H182-013 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 6396 | 52 | — | — | 3.725 | 1717.231 | 0.482 | 107.947 | 5.890 | completed |
| qwen25-05-json_schema/C | 4848 | 24 | — | — | 2.468 | 1964.053 | 0.234 | 102.660 | 3.466 | completed |
| qwen25-05-json_schema/T | 4753 | 22 | — | — | 2.387 | 1990.890 | 0.221 | 99.365 | 3.745 | completed |
| qwen25-05-none/G | 6396 | 57 | — | — | 3.422 | 1869.049 | 0.505 | 112.958 | 4.659 | completed |
| qwen25-15-none/G | 6396 | 57 | — | — | 8.365 | 764.656 | 1.012 | 56.326 | 10.143 | completed |
| qwen25-15-none/C | 5306 | 83 | — | — | 6.718 | 789.863 | 1.422 | 58.356 | 8.899 | completed |
| qwen25-15-none/T | 5131 | 159 | — | — | 6.428 | 798.165 | 2.703 | 58.814 | 9.894 | grammar_failure |
| qwen25-15-json_schema/G | 6396 | 42 | — | — | 8.577 | 745.730 | 0.758 | 55.432 | 10.096 | completed |
| qwen25-15-json_schema/C | 4848 | 17 | — | — | 6.356 | 762.801 | 0.297 | 57.210 | 7.471 | completed |
| qwen25-15-json_schema/T | 4753 | 31 | — | — | 6.070 | 783.000 | 0.585 | 52.947 | 7.826 | completed |
| qwen25-3-json_schema/G | 6396 | 41 | — | — | 18.322 | 349.085 | 1.257 | 32.617 | 20.625 | completed |
| qwen25-3-json_schema/C | 5306 | 49 | — | — | 15.126 | 350.786 | 1.438 | 34.070 | 17.437 | completed |
| qwen25-3-json_schema/T | 5131 | 106 | — | — | 14.437 | 355.418 | 3.345 | 31.687 | 19.064 | completed |
| qwen25-3-none/G | 6396 | 41 | — | — | 24.728 | 258.656 | 2.995 | 13.688 | 28.581 | completed |
| qwen25-3-none/C | 5306 | 49 | — | — | 23.238 | 228.333 | 2.691 | 18.206 | 27.002 | completed |
| qwen25-3-none/T | 5131 | 134 | — | — | 17.786 | 288.490 | 5.336 | 25.110 | 24.148 | completed |
| coder25-05-none/G | 6396 | 41 | — | — | 4.308 | 1484.776 | 0.399 | 102.734 | 5.629 | completed |
| coder25-05-none/C | 5764 | 81 | — | — | 3.901 | 1477.645 | 0.732 | 110.595 | 5.405 | completed |
| coder25-05-none/T | 5509 | 4096 | — | — | 3.668 | 1502.062 | 55.553 | 73.732 | 60.072 | max_tokens |
| coder25-05-json_schema/G | 6396 | 41 | — | — | 3.874 | 1650.822 | 0.386 | 106.121 | 5.048 | completed |
| coder25-05-json_schema/C | 5764 | 81 | — | — | 3.649 | 1579.782 | 0.727 | 111.384 | 5.128 | completed |
| coder25-05-json_schema/T | 5509 | 247 | — | — | 3.316 | 1661.294 | 3.195 | 77.317 | 7.861 | completed |
| coder25-15-json_schema/G | 6396 | 41 | — | — | 11.297 | 566.173 | 0.752 | 54.535 | 12.859 | completed |
| coder25-15-json_schema/C | 5306 | 52 | — | — | 9.602 | 552.579 | 0.920 | 56.548 | 11.423 | completed |
| coder25-15-none/G | 6396 | 41 | — | — | 12.075 | 529.708 | 0.727 | 56.406 | 13.747 | completed |
| coder25-15-none/C | 5306 | 86 | — | — | 10.322 | 514.070 | 1.703 | 50.514 | 12.918 | completed |
| coder25-3-none/G | 6396 | 57 | — | — | 24.946 | 256.391 | 2.965 | 19.223 | 29.285 | completed |
| coder25-3-none/C | 4848 | 29 | — | — | 17.419 | 278.311 | 0.858 | 33.803 | 19.389 | completed |
| coder25-3-none/T | 4753 | 64 | — | — | 16.612 | 286.126 | 3.103 | 20.626 | 20.857 | completed |
| coder25-3-json_schema/G | 6396 | 47 | — | — | 23.765 | 269.131 | 1.843 | 25.500 | 26.483 | completed |
| coder25-3-json_schema/C | 5077 | 45 | — | — | 17.836 | 284.643 | 1.546 | 29.114 | 20.931 | completed |
| coder25-3-json_schema/T | 4942 | 69 | — | — | 17.513 | 282.191 | 3.302 | 20.895 | 23.013 | completed |
| qwen2-off-json_schema/G | 6094 | 52 | — | — | 9.912 | 614.824 | 0.927 | 56.072 | 12.627 | completed |
| qwen2-off-json_schema/C | 4540 | 25 | — | — | 7.238 | 627.268 | 0.450 | 55.569 | 9.790 | completed |
| qwen2-off-json_schema/T | 4448 | 41 | — | — | 7.275 | 611.411 | 0.890 | 46.045 | 10.696 | completed |
| qwen2-off-none/G | 6094 | 63 | — | — | 9.625 | 633.163 | 1.124 | 56.048 | 12.620 | completed |
| qwen2-off-none/C | 4998 | 96 | — | — | 7.934 | 629.938 | 1.721 | 55.797 | 11.867 | completed |
| qwen2-off-none/T | 4826 | 220 | — | — | 7.532 | 640.761 | 5.721 | 38.458 | 16.204 | completed |
| qwen4-off-none/G | 6094 | 41 | — | — | 25.414 | 239.785 | 2.469 | 16.605 | 29.956 | completed |
| qwen4-off-none/C | 4998 | 55 | — | — | 22.857 | 218.666 | 3.322 | 16.558 | 28.498 | completed |
| qwen4-off-none/T | 4827 | 88 | — | — | 21.001 | 229.848 | 4.935 | 17.833 | 28.059 | completed |
| qwen4-off-json_schema/G | 6094 | 41 | — | — | 25.734 | 236.808 | 2.528 | 16.217 | 30.196 | completed |
| qwen4-off-json_schema/C | 4998 | 71 | — | — | 21.383 | 233.740 | 4.228 | 16.792 | 27.772 | completed |
| qwen4-off-json_schema/T | 4826 | 84 | — | — | 21.117 | 228.542 | 4.851 | 17.315 | 28.907 | completed |
| qwen4-on-json_schema/G | 6092 | 561 | 519 | — | 25.689 | 237.143 | 32.032 | 17.514 | 59.689 | completed |
| qwen4-on-json_schema/C | 4996 | 1370 | 1298 | — | 19.089 | 261.721 | 66.904 | 20.477 | 88.076 | completed |
| qwen4-on-json_schema/T | 4824 | 645 | 576 | — | 18.734 | 257.504 | 32.675 | 19.740 | 53.995 | completed |
| qwen4-on-none/G | 6092 | 562 | 519 | — | 23.851 | 255.419 | 32.149 | 17.481 | 57.907 | completed |
| qwen4-on-none/C | 4996 | 1396 | 1298 | — | 19.496 | 256.263 | 69.217 | 20.168 | 90.847 | completed |
| qwen4-on-none/T | 4824 | 651 | 576 | — | 17.556 | 274.777 | 33.729 | 19.301 | 53.189 | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 0.365 | 0.036 | 0.332 | 0.333 | 4.207 | 1.465 | 0.488 | 310.920 | 76.657 | 867.316 | 41.537 |
| qwen25-05-json_schema/C | 0.350 | 0.028 | 0.354 | 0.356 | 2.702 | 1.114 | 0.319 | 300.420 | 22.639 | 895.820 | 30.362 |
| qwen25-05-json_schema/T | 0.354 | 0.029 | 0.360 | 0.716 | 2.609 | 3.662 | 0.351 | 299.670 | 22.659 | 891.505 | 29.858 |
| qwen25-05-none/G | 0.336 | 0.036 | 0.331 | 0.332 | 3.927 | 1.234 | 0.380 | 310.920 | 73.684 | 867.285 | 41.537 |
| qwen25-15-none/G | 0.360 | 0.036 | 0.331 | 0.331 | 9.377 | 1.477 | 0.561 | 930.092 | 105.847 | 1547.141 | 96.920 |
| qwen25-15-none/C | 0.355 | 0.031 | 0.331 | 0.331 | 8.140 | 1.543 | 0.540 | 913.467 | 90.067 | 1449.881 | 81.319 |
| qwen25-15-none/T | 0.359 | 0.029 | 0.331 | 0.331 | 9.132 | 2.003 | 0.627 | 0.003 | 998.280 | 1454.685 | 78.994 |
| qwen25-15-json_schema/G | 0.341 | 0.036 | 0.341 | 0.342 | 9.335 | 1.393 | 0.560 | 930.092 | 108.816 | 1547.141 | 96.920 |
| qwen25-15-json_schema/C | 0.390 | 0.029 | 0.355 | 0.356 | 6.653 | 1.184 | 0.473 | 903.842 | 16.493 | 1452.975 | 70.845 |
| qwen25-15-json_schema/T | 0.379 | 0.030 | 0.364 | 0.727 | 6.650 | 3.900 | 0.495 | 902.092 | 18.133 | 1436.892 | 69.668 |
| qwen25-3-json_schema/G | 0.626 | 0.038 | 0.352 | 0.353 | 19.565 | 1.816 | 1.241 | 1782.934 | 135.083 | 2360.717 | 124.611 |
| qwen25-3-json_schema/C | 0.414 | 0.034 | 0.367 | 0.370 | 16.561 | 1.723 | 0.954 | 1761.559 | 114.739 | 2394.153 | 104.552 |
| qwen25-3-json_schema/T | 0.439 | 0.034 | 0.382 | 0.760 | 17.781 | 5.165 | 1.005 | 1758.184 | 111.364 | 2394.617 | 101.564 |
| qwen25-3-none/G | 0.399 | 0.041 | 0.378 | 0.379 | 27.723 | 1.840 | 1.107 | 1782.934 | 132.114 | 2360.717 | 124.611 |
| qwen25-3-none/C | 0.522 | 0.044 | 0.459 | 0.459 | 25.930 | 2.037 | 1.280 | 1761.559 | 111.771 | 2394.153 | 104.552 |
| qwen25-3-none/T | 0.499 | 0.040 | 0.419 | 0.419 | 23.122 | 2.872 | 1.249 | 1758.184 | 108.396 | 2394.617 | 101.564 |
| coder25-05-none/G | 0.460 | 0.046 | 0.379 | 0.380 | 4.707 | 1.417 | 0.374 | 310.920 | 73.685 | 867.347 | 41.537 |
| coder25-05-none/C | 0.342 | 0.036 | 0.363 | 0.364 | 4.633 | 1.363 | 0.379 | 307.170 | 56.732 | 898.421 | 37.721 |
| coder25-05-none/T | 0.374 | 0.037 | 0.409 | 0.410 | 0.000 | 26.291 | 3.572 | 0.003 | 432.219 | 886.084 | — |
| coder25-05-json_schema/G | 0.375 | 0.037 | 0.344 | 0.345 | 4.261 | 1.257 | 0.368 | 310.920 | 76.652 | 867.347 | 41.537 |
| coder25-05-json_schema/C | 0.335 | 0.034 | 0.350 | 0.353 | 4.376 | 1.353 | 0.371 | 307.170 | 59.700 | 898.421 | 37.721 |
| coder25-05-json_schema/T | 0.334 | 0.033 | 0.376 | 0.947 | 6.511 | 6.819 | 0.505 | 305.670 | 58.198 | 886.084 | 35.971 |
| coder25-15-json_schema/G | 0.399 | 0.039 | 0.336 | 0.337 | 12.049 | 1.580 | 0.679 | 930.092 | 108.816 | 1547.141 | 96.920 |
| coder25-15-json_schema/C | 0.396 | 0.035 | 0.427 | 0.430 | 10.522 | 1.543 | 0.535 | 913.467 | 93.036 | 1449.881 | 81.319 |
| coder25-15-none/G | 0.452 | 0.046 | 0.404 | 0.404 | 12.802 | 1.556 | 0.627 | 930.092 | 105.848 | 1547.141 | 96.920 |
| coder25-15-none/C | 0.387 | 0.035 | 0.408 | 0.408 | 12.024 | 1.744 | 0.592 | 913.467 | 90.066 | 1449.881 | 81.319 |
| coder25-3-none/G | 0.776 | 0.059 | 0.433 | 0.433 | 27.912 | 2.620 | 1.761 | 1782.934 | 132.114 | 2360.717 | 124.611 |
| coder25-3-none/C | 0.607 | 0.037 | 0.395 | 0.395 | 18.278 | 1.790 | 1.282 | 1753.684 | 103.661 | 2395.496 | 95.868 |
| coder25-3-none/T | 0.457 | 0.035 | 0.566 | 0.566 | 19.715 | 2.211 | 1.096 | 1751.434 | 101.411 | 2366.525 | 94.355 |
| coder25-3-json_schema/G | 0.428 | 0.038 | 0.327 | 0.328 | 25.609 | 1.912 | 1.312 | 1782.934 | 135.083 | 2360.717 | 124.611 |
| coder25-3-json_schema/C | 0.858 | 0.068 | 0.554 | 0.556 | 19.382 | 2.505 | 1.282 | 1757.059 | 112.583 | 2395.704 | 99.734 |
| coder25-3-json_schema/T | 0.903 | 0.076 | 0.594 | 1.161 | 20.815 | 7.103 | 1.321 | 1754.809 | 107.755 | 2408.888 | 97.549 |
| qwen2-off-json_schema/G | 1.003 | 0.065 | 0.646 | 0.648 | 10.839 | 2.262 | 0.820 | 1069.613 | 68.560 | 1790.119 | 58.203 |
| qwen2-off-json_schema/C | 1.065 | 0.079 | 0.888 | 0.889 | 7.688 | 2.426 | 0.778 | 1059.863 | 58.154 | 1797.515 | 48.565 |
| qwen2-off-json_schema/T | 0.933 | 0.059 | 0.699 | 1.476 | 8.166 | 7.932 | 0.752 | 1059.285 | 57.623 | 1791.994 | 48.067 |
| qwen2-off-none/G | 1.027 | 0.075 | 0.690 | 0.691 | 10.749 | 2.462 | 0.882 | 1069.550 | 67.685 | 1790.119 | 58.203 |
| qwen2-off-none/C | 1.170 | 0.084 | 0.836 | 0.837 | 9.655 | 2.926 | 0.891 | 1062.878 | 60.533 | 1805.622 | 51.429 |
| qwen2-off-none/T | 2.029 | 0.063 | 0.705 | 0.705 | 13.253 | 4.710 | 1.562 | 1061.503 | 79.548 | 1803.268 | 50.477 |
| qwen4-off-none/G | 1.202 | 0.068 | 0.642 | 0.643 | 27.884 | 2.856 | 2.354 | 2413.371 | 158.469 | 3186.813 | 154.645 |
| qwen4-off-none/C | 1.489 | 0.061 | 0.646 | 0.647 | 26.179 | 3.188 | 2.594 | 2395.308 | 140.296 | 3171.779 | 136.582 |
| qwen4-off-none/T | 1.267 | 0.066 | 0.678 | 0.679 | 25.936 | 3.561 | 2.000 | 2392.168 | 165.813 | 3187.793 | 134.042 |
| qwen4-off-json_schema/G | 1.141 | 0.068 | 0.648 | 0.649 | 28.263 | 2.775 | 2.137 | 2413.308 | 159.398 | 3186.813 | 154.645 |
| qwen4-off-json_schema/C | 1.261 | 0.069 | 0.722 | 0.724 | 25.612 | 3.196 | 1.935 | 2395.324 | 141.250 | 3171.779 | 136.582 |
| qwen4-off-json_schema/T | 1.254 | 0.067 | 0.718 | 1.531 | 25.968 | 9.187 | 1.851 | 2392.277 | 166.528 | 3187.793 | 134.042 |
| qwen4-on-json_schema/G | 1.160 | 0.068 | 0.644 | 0.645 | 57.722 | 9.687 | 3.581 | 2421.824 | 168.029 | 3186.813 | 163.145 |
| qwen4-on-json_schema/C | 1.218 | 0.069 | 0.692 | 0.695 | 85.993 | 20.499 | 5.940 | 2416.168 | 159.561 | 3171.779 | 157.832 |
| qwen4-on-json_schema/T | 1.099 | 0.059 | 0.696 | 1.341 | 51.409 | 15.706 | 3.916 | 2400.855 | 144.562 | 3187.793 | 142.542 |
| qwen4-on-none/G | 1.088 | 0.066 | 0.649 | 0.650 | 56.001 | 9.845 | 3.607 | 2421.715 | 167.313 | 3186.813 | 163.145 |
| qwen4-on-none/C | 1.216 | 0.071 | 0.741 | 0.742 | 88.713 | 20.764 | 5.986 | 2416.262 | 158.594 | 3171.779 | 157.832 |
| qwen4-on-none/T | 1.109 | 0.061 | 0.646 | 0.647 | 51.286 | 10.510 | 3.622 | 2400.793 | 143.427 | 3187.793 | 142.542 |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 0.6901 | 0.0883 | 0.0162 | 0.0442 | 0.1277 | 0.0767 | 12065 | 0.9273 | 5.892 |
| qwen25-05-json_schema/C | 0.3971 | 0.0652 | 0.0267 | 0.0319 | 0.0988 | 0.0578 | 12042 | 0.7165 | 3.467 |
| qwen25-05-json_schema/T | 0.4187 | 0.0635 | 0.0290 | 0.0312 | 0.0674 | 0.0615 | 12028 | 0.7307 | 3.746 |
| qwen25-05-none/G | 0.6860 | 0.0902 | 0.0160 | 0.0481 | 0.1001 | 0.0836 | 12067 | 0.8976 | 4.660 |
| qwen25-15-none/G | 0.6784 | 0.1209 | 0.0216 | 0.0460 | 0.1530 | 0.1066 | 13895 | 0.9225 | 10.145 |
| qwen25-15-none/C | 0.3890 | 0.0810 | 0.0402 | 0.0331 | 0.0752 | 0.0832 | 14049 | 0.7668 | 8.900 |
| qwen25-15-none/T | 0.4884 | 0.0727 | — | 0.0388 | 0.1009 | 0.0746 | 13766 | 0.8458 | 9.896 |
| qwen25-15-json_schema/G | 0.6739 | 0.1138 | 0.0270 | 0.0553 | 0.1200 | 0.1033 | 13882 | 0.9090 | 10.097 |
| qwen25-15-json_schema/C | 0.4575 | 0.0831 | 0.0288 | 0.0350 | 0.0954 | 0.0771 | 13858 | 0.8730 | 7.473 |
| qwen25-15-json_schema/T | 0.4847 | 0.0912 | 0.0321 | 0.0473 | 0.0800 | 0.0727 | 13917 | 0.8615 | 7.827 |
| qwen25-3-json_schema/G | 0.6715 | 0.1053 | 0.0143 | 0.0473 | 0.1155 | 0.1082 | 17430 | 0.8962 | 20.627 |
| qwen25-3-json_schema/C | 0.4135 | 0.0746 | 0.0328 | 0.0330 | 0.0846 | 0.1017 | 17543 | 0.7597 | 17.439 |
| qwen25-3-json_schema/T | 0.4600 | 0.0722 | 0.0483 | 0.0423 | 0.0836 | 0.0915 | 17733 | 0.8505 | 19.066 |
| qwen25-3-none/G | 0.6697 | 0.0950 | 0.0237 | 0.0452 | 0.1052 | 0.1065 | 17426 | 0.8720 | 28.583 |
| qwen25-3-none/C | 0.4190 | 0.0768 | 0.0400 | 0.0334 | 0.0673 | 0.0849 | 17539 | 0.7839 | 27.003 |
| qwen25-3-none/T | 0.4392 | 0.0656 | 0.0384 | 0.0358 | 0.0977 | 0.0920 | 17916 | 0.9445 | 24.150 |
| coder25-05-none/G | 0.8963 | 0.1554 | 0.0138 | 0.0700 | 0.1530 | 0.0855 | 12046 | 1.2236 | 5.631 |
| coder25-05-none/C | 0.3853 | 0.0818 | 0.0277 | 0.0323 | 0.0684 | 0.0587 | 12282 | 0.7263 | 5.407 |
| coder25-05-none/T | 0.3932 | 0.0621 | — | 0.0302 | 0.0738 | 0.0245 | 1296 | 0.6877 | 60.073 |
| coder25-05-json_schema/G | 0.7140 | 0.0875 | 0.0196 | 0.0507 | 0.1532 | 0.0792 | 12052 | 0.9312 | 5.049 |
| coder25-05-json_schema/C | 0.3860 | 0.0903 | 0.0306 | 0.0333 | 0.0662 | 0.0569 | 12290 | 0.7070 | 5.129 |
| coder25-05-json_schema/T | 0.4120 | 0.0612 | 0.0577 | 0.0294 | 0.0634 | 0.0904 | 12778 | 0.6984 | 7.862 |
| coder25-15-json_schema/G | 0.8030 | 0.1194 | 0.0225 | 0.0518 | 0.1620 | 0.0927 | 13878 | 1.0705 | 12.861 |
| coder25-15-json_schema/C | 0.4067 | 0.0822 | 0.0504 | 0.0376 | 0.1193 | 0.0742 | 14004 | 0.7856 | 11.424 |
| coder25-15-none/G | 0.7736 | 0.1022 | 0.0157 | 0.0611 | 0.1428 | 0.1107 | 13877 | 1.0120 | 13.749 |
| coder25-15-none/C | 0.7638 | 0.2283 | 0.0533 | 0.0481 | 0.1552 | 0.1225 | 14072 | 1.3685 | 12.921 |
| coder25-3-none/G | 1.5842 | 0.2456 | 0.0372 | 0.1235 | 0.2750 | 0.1710 | 17460 | 2.1090 | 29.289 |
| coder25-3-none/C | 0.7016 | 0.1207 | 0.0295 | 0.0498 | 0.1647 | 0.0852 | 17445 | 1.3351 | 19.390 |
| coder25-3-none/T | 0.4508 | 0.0704 | 0.0413 | 0.0345 | 0.0964 | 0.1293 | 17594 | 0.8335 | 20.859 |
| coder25-3-json_schema/G | 0.7715 | 0.1232 | 0.0481 | 0.0520 | 0.1252 | 0.1667 | 17455 | 1.0328 | 26.485 |
| coder25-3-json_schema/C | 0.5950 | 0.1123 | 0.0488 | 0.0462 | 0.4739 | 0.1080 | 17508 | 1.1574 | 20.933 |
| coder25-3-json_schema/T | 0.5890 | 0.1150 | 0.0384 | 0.0537 | 0.0953 | 0.0975 | 17587 | 1.0526 | 23.015 |
| qwen2-off-json_schema/G | 0.7726 | 0.1336 | 0.0220 | 0.0515 | 0.1398 | 0.0738 | 8017 | 1.0505 | 12.629 |
| qwen2-off-json_schema/C | 0.4026 | 0.0722 | 0.0398 | 0.0301 | 0.0770 | 0.0443 | 7977 | 0.7431 | 9.791 |
| qwen2-off-json_schema/T | 0.3939 | 0.0676 | 0.0292 | 0.0299 | 0.0790 | 0.0470 | 8076 | 0.7302 | 10.697 |
| qwen2-off-none/G | 0.7453 | 0.1171 | 0.0182 | 0.0542 | 0.1404 | 0.0780 | 8024 | 0.9984 | 12.622 |
| qwen2-off-none/C | 0.4110 | 0.0948 | 0.0700 | 0.0340 | 0.0788 | 0.1210 | 8159 | 0.9320 | 11.869 |
| qwen2-off-none/T | 1.1258 | 0.1568 | 0.0809 | 0.0990 | 0.1727 | 0.1077 | 8722 | 1.9180 | 16.206 |
| qwen4-off-none/G | 0.7788 | 0.1276 | 0.0713 | 0.0615 | 0.1400 | 0.2041 | 10209 | 1.0440 | 29.959 |
| qwen4-off-none/C | 0.9680 | 0.2309 | 0.0393 | 0.0782 | 0.2715 | 0.0685 | 10325 | 1.9400 | 28.500 |
| qwen4-off-none/T | 0.5131 | 0.0762 | 0.0639 | 0.0391 | 0.0976 | 0.0667 | 10542 | 0.9088 | 28.062 |
| qwen4-off-json_schema/G | 0.9589 | 0.1314 | 0.0228 | 0.0662 | 0.1325 | 0.0874 | 10215 | 1.2578 | 30.199 |
| qwen4-off-json_schema/C | 0.4192 | 0.0917 | 0.0311 | 0.0335 | 0.0975 | 0.0560 | 10352 | 0.8109 | 27.773 |
| qwen4-off-json_schema/T | 0.4335 | 0.0934 | 0.0381 | 0.0314 | 0.0905 | 0.0680 | 10541 | 0.8268 | 28.908 |
| qwen4-on-json_schema/G | 0.9553 | 0.1476 | 0.0219 | 0.0690 | 0.1326 | 0.0801 | 10530 | 1.2639 | 59.691 |
| qwen4-on-json_schema/C | 0.3939 | 0.0850 | 0.0343 | 0.0323 | 0.0917 | 0.0610 | 10664 | 0.8470 | 88.077 |
| qwen4-on-json_schema/T | 0.4713 | 0.1173 | 0.0371 | 0.0383 | 0.1271 | 0.0759 | 10753 | 0.8785 | 53.996 |
| qwen4-on-none/G | 1.0749 | 0.0933 | 0.0185 | 0.1020 | 0.1778 | 0.1152 | 10523 | 1.3087 | 57.909 |
| qwen4-on-none/C | 0.4124 | 0.1230 | 0.0298 | 0.0389 | 0.0885 | 0.0615 | 10708 | 0.8353 | 90.849 |
| qwen4-on-none/T | 0.4301 | 0.1007 | 0.0510 | 0.0400 | 0.0804 | 0.0778 | 10745 | 0.8105 | 53.190 |


### H182-014 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema/G | 6488 | 974 | 932 | — | 13.580 | 477.748 | 40.330 | 24.150 | 58.025 | completed |
| gemma-on-json_schema/C | 4985 | 1073 | 1054 | — | 9.897 | 503.689 | 44.506 | 24.109 | 57.127 | completed |
| gemma-on-json_schema/T | 4901 | 1199 | 1174 | — | 10.335 | 474.221 | 48.331 | 24.808 | 62.303 | completed |
| gemma-on-none/G | 6488 | 974 | 932 | — | 13.389 | 484.574 | 40.070 | 24.308 | 55.988 | completed |
| gemma-on-none/C | 4985 | 1073 | 1054 | — | 10.418 | 478.491 | 45.857 | 23.399 | 58.621 | completed |
| gemma-on-none/T | 4901 | 1199 | 1174 | — | 11.157 | 439.294 | 62.186 | 19.281 | 76.092 | completed |
| gemma-off-none/G | 6486 | 46 | — | — | 13.462 | 481.805 | 1.820 | 25.271 | 18.160 | completed |
| gemma-off-none/C | 5299 | 104 | — | — | 11.792 | 449.386 | 6.268 | 16.593 | 20.606 | completed |
| gemma-off-none/T | 5167 | 83 | — | — | 16.975 | 304.387 | 6.532 | 12.706 | 26.425 | completed |
| gemma-off-json_schema/G | 6486 | 41 | — | — | 17.119 | 378.866 | 2.868 | 14.296 | 22.716 | completed |
| gemma-off-json_schema/C | 5141 | 36 | — | — | 13.278 | 387.190 | 1.482 | 24.296 | 17.823 | completed |
| gemma-off-json_schema/T | 5033 | 61 | — | — | 14.416 | 349.133 | 3.925 | 15.540 | 22.139 | completed |
| ministral-instruct-json_schema/G | 6865 | 52 | — | — | 29.426 | 233.298 | 2.517 | 20.657 | 34.100 | completed |
| ministral-instruct-json_schema/C | 5773 | 77 | — | — | 24.748 | 233.269 | 3.805 | 20.237 | 30.589 | completed |
| ministral-instruct-none/G | 6865 | 61 | — | — | 30.691 | 223.685 | 3.161 | 19.297 | 35.839 | completed |
| ministral-instruct-none/C | 5773 | 258 | — | — | 25.048 | 230.481 | 13.989 | 18.443 | 41.260 | grammar_failure |
| ministral-reasoning-none/G | 7010 | 4096 | 4097 | — | 31.319 | 223.827 | 194.632 | 21.045 | 227.798 | max_tokens |
| ministral-reasoning-json_schema/G | 7010 | 4096 | 4097 | — | 28.227 | 248.345 | 194.658 | 21.042 | 224.957 | max_tokens |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema/G | 2.544 | 0.049 | 0.507 | 0.508 | 53.911 | 13.708 | 4.990 | 4089.513 | 83.711 | 4910.684 | 82.455 |
| gemma-on-json_schema/C | 1.965 | 0.046 | 0.548 | 0.550 | 54.404 | 14.858 | 4.770 | 4079.263 | 74.991 | 4910.684 | 72.154 |
| gemma-on-json_schema/T | 2.323 | 0.047 | 0.553 | 1.155 | 58.668 | 18.524 | 5.003 | 4078.638 | 74.370 | 4910.684 | 71.481 |
| gemma-on-none/G | 1.826 | 0.047 | 0.497 | 0.498 | 53.459 | 14.449 | 4.754 | 4089.513 | 82.710 | 4910.684 | 82.455 |
| gemma-on-none/C | 1.547 | 0.043 | 0.507 | 0.508 | 56.276 | 16.711 | 4.887 | 4079.263 | 73.994 | 4910.684 | 72.154 |
| gemma-on-none/T | 1.976 | 0.044 | 0.498 | 0.499 | 73.343 | 22.844 | 7.270 | 4078.638 | 73.372 | 4910.684 | 71.481 |
| gemma-off-none/G | 1.984 | 0.046 | 0.575 | 0.576 | 15.283 | 3.033 | 2.706 | 4083.263 | 78.038 | 4910.684 | 76.080 |
| gemma-off-none/C | 1.692 | 0.044 | 0.532 | 0.533 | 18.060 | 4.439 | 2.600 | 4073.388 | 66.991 | 4910.684 | 66.385 |
| gemma-off-none/T | 2.150 | 0.044 | 0.500 | 0.501 | 23.508 | 4.967 | 3.920 | 4072.388 | 66.000 | 4910.684 | 65.389 |
| gemma-off-json_schema/G | 1.950 | 0.047 | 0.507 | 0.508 | 19.988 | 3.038 | 2.936 | 4083.263 | 79.038 | 4910.684 | 76.080 |
| gemma-off-json_schema/C | 2.327 | 0.050 | 0.553 | 0.555 | 14.760 | 3.711 | 3.020 | 4070.263 | 24.109 | 4910.684 | 63.098 |
| gemma-off-json_schema/T | 2.388 | 0.045 | 0.521 | 1.194 | 18.342 | 9.687 | 3.518 | 4071.138 | 66.820 | 4910.684 | 64.027 |
| ministral-instruct-json_schema/G | 0.934 | 0.589 | 0.374 | 0.375 | 31.944 | 2.915 | 2.573 | 2225.663 | 391.016 | 2971.444 | 385.833 |
| ministral-instruct-json_schema/C | 1.171 | 0.457 | 0.309 | 0.311 | 28.553 | 3.449 | 2.106 | 2167.163 | 326.541 | 2809.678 | 326.914 |
| ministral-instruct-none/G | 1.004 | 0.571 | 0.313 | 0.314 | 33.852 | 2.917 | 2.539 | 2225.663 | 390.516 | 2971.444 | 385.833 |
| ministral-instruct-none/C | 1.300 | 0.477 | 0.311 | 0.311 | 39.037 | 5.324 | 2.672 | 0.003 | 2186.238 | 2809.678 | 326.914 |
| ministral-reasoning-none/G | 0.895 | 0.541 | 0.326 | 0.327 | 0.000 | 48.063 | 11.617 | 0.003 | 3288.895 | 3021.807 | — |
| ministral-reasoning-json_schema/G | 1.074 | 0.583 | 0.327 | 0.328 | 0.000 | 53.460 | 13.528 | 2440.945 | 606.984 | 3021.807 | — |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema/G | 0.6880 | 0.0767 | 0.0203 | 0.0450 | 0.0971 | 0.1271 | 8119 | 0.9009 | 58.027 |
| gemma-on-json_schema/C | 0.4799 | 0.0954 | 0.0349 | 0.0312 | 0.0731 | 0.0529 | 8107 | 0.9493 | 57.128 |
| gemma-on-json_schema/T | 0.4205 | 0.0716 | 1.3260 | 0.0321 | 0.0777 | 1.8082 | 8160 | 0.8208 | 62.311 |
| gemma-on-none/G | 0.8707 | 0.6337 | 0.0191 | 0.0481 | 0.1039 | 0.1227 | 8115 | 1.8765 | 55.990 |
| gemma-on-none/C | 0.4049 | 0.0795 | 0.0335 | 0.0278 | 0.2475 | 0.0520 | 8108 | 0.7552 | 58.622 |
| gemma-on-none/T | 0.4379 | 0.0851 | 0.0653 | 0.0340 | 0.0995 | 0.1201 | 8156 | 0.8779 | 76.093 |
| gemma-off-none/G | 0.6819 | 0.0815 | 0.0255 | 0.0461 | 0.0984 | 0.0907 | 7805 | 0.8874 | 18.162 |
| gemma-off-none/C | 0.4500 | 0.1061 | 0.0936 | 0.0325 | 0.1057 | 0.1237 | 7989 | 0.8420 | 20.608 |
| gemma-off-none/T | 1.0568 | 0.1590 | 0.1247 | 0.0768 | 0.2202 | 0.1689 | 8061 | 1.8885 | 26.429 |
| gemma-off-json_schema/G | 0.6742 | 0.0900 | 0.0380 | 0.0458 | 0.1106 | 0.1020 | 7809 | 0.8795 | 22.718 |
| gemma-off-json_schema/C | 0.5977 | 0.1314 | 0.0468 | 0.0447 | 0.1348 | 0.0496 | 7856 | 1.1925 | 17.825 |
| gemma-off-json_schema/T | 0.4431 | 0.0694 | 0.1155 | 0.0343 | 0.0983 | 0.1392 | 7987 | 0.8732 | 22.141 |
| ministral-instruct-json_schema/G | 1.4994 | 0.4183 | 0.0377 | 0.0544 | 0.2857 | 0.2262 | 13239 | 2.2044 | 34.103 |
| ministral-instruct-json_schema/C | 0.8295 | 0.2246 | 0.0348 | 0.0711 | 0.1707 | 0.0762 | 13377 | 1.6323 | 30.591 |
| ministral-instruct-none/G | 0.6976 | 0.1063 | 0.0683 | 0.0503 | 0.1101 | 0.1870 | 13241 | 0.9295 | 35.841 |
| ministral-instruct-none/C | 1.0402 | 0.1893 | — | 0.0718 | 0.1474 | 0.1313 | 13115 | 1.8654 | 41.263 |
| ministral-reasoning-none/G | 0.7502 | 0.0780 | — | 0.0499 | 0.1151 | 0.0532 | 1359 | 0.9400 | 227.800 |
| ministral-reasoning-json_schema/G | 0.7764 | 0.1048 | — | 0.1878 | 0.1346 | 0.0559 | 1368 | 0.9984 | 224.958 |


### H182-015 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema/G | 6437 | 1497 | 1444 | — | 13.093 | 491.636 | 60.536 | 24.729 | 76.952 | completed |
| gemma-on-json_schema/C | 5244 | 1075 | 1020 | — | 10.123 | 518.005 | 42.651 | 25.205 | 55.389 | completed |
| gemma-on-json_schema/T | 5140 | 1137 | 1042 | — | 10.197 | 504.089 | 45.213 | 25.148 | 58.105 | completed |
| gemma-on-none/G | 6437 | 1508 | 1444 | — | 13.135 | 490.057 | 63.876 | 23.608 | 79.106 | completed |
| gemma-on-none/C | 5244 | 1075 | 1020 | — | 10.595 | 494.945 | 50.253 | 21.392 | 63.359 | completed |
| gemma-on-none/T | 5140 | 1162 | 1042 | — | 11.224 | 457.930 | 52.273 | 22.230 | 66.143 | completed |
| gemma-off-none/G | 6435 | 68 | — | — | 13.174 | 488.457 | 2.741 | 24.808 | 18.149 | completed |
| gemma-off-none/C | 5242 | 104 | — | — | 11.546 | 453.991 | 5.286 | 19.674 | 19.709 | completed |
| gemma-off-none/T | 5138 | 128 | — | — | 13.035 | 394.177 | 7.955 | 16.090 | 23.971 | completed |
| gemma-off-json_schema/G | 6435 | 50 | — | — | 16.836 | 382.209 | 3.158 | 15.833 | 22.096 | completed |
| gemma-off-json_schema/C | 5242 | 66 | — | — | 12.770 | 410.480 | 3.787 | 17.430 | 19.170 | completed |
| gemma-off-json_schema/T | 5138 | 85 | — | — | 12.713 | 404.159 | 4.787 | 17.756 | 20.787 | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema/G | 1.824 | 0.049 | 0.527 | 0.528 | 73.630 | 21.382 | 6.557 | 4150.388 | 149.025 | 4910.684 | — |
| gemma-on-json_schema/C | 1.875 | 0.044 | 0.516 | 0.518 | 52.775 | 15.651 | 4.875 | 4128.513 | 126.108 | 4910.684 | — |
| gemma-on-json_schema/T | 1.565 | 0.044 | 0.493 | 0.968 | 55.410 | 19.409 | 4.677 | 4097.232 | 154.396 | 4910.684 | — |
| gemma-on-none/G | 1.422 | 0.046 | 0.506 | 0.507 | 77.012 | 23.198 | 7.281 | 4148.888 | 149.540 | 4910.684 | — |
| gemma-on-none/C | 1.869 | 0.045 | 0.496 | 0.497 | 60.848 | 17.669 | 5.663 | 4098.857 | 154.769 | 4910.684 | — |
| gemma-on-none/T | 1.991 | 0.046 | 0.499 | 0.500 | 63.497 | 19.630 | 6.478 | 4110.372 | 140.254 | 4910.684 | — |
| gemma-off-none/G | 1.586 | 0.047 | 0.488 | 0.489 | 15.918 | 3.099 | 2.393 | 4129.763 | 128.363 | 4910.684 | — |
| gemma-off-none/C | 2.180 | 0.044 | 0.517 | 0.518 | 16.833 | 4.422 | 2.881 | 4112.513 | 109.066 | 4910.684 | — |
| gemma-off-none/T | 2.295 | 0.044 | 0.511 | 0.512 | 20.990 | 4.810 | 3.267 | 4108.388 | 109.941 | 4910.684 | — |
| gemma-off-json_schema/G | 1.469 | 0.045 | 0.490 | 0.493 | 19.995 | 2.881 | 2.393 | 4098.544 | 160.587 | 4910.684 | — |
| gemma-off-json_schema/C | 1.933 | 0.046 | 0.488 | 0.491 | 16.557 | 3.834 | 2.469 | 4110.513 | 112.066 | 4910.684 | — |
| gemma-off-json_schema/T | 1.938 | 0.045 | 0.529 | 1.176 | 17.500 | 8.583 | 2.563 | 4085.638 | 133.691 | 4910.684 | — |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| gemma-on-json_schema/G | 0.9439 | 0.3754 | 0.0208 | 0.0571 | 0.1091 | 0.1340 | 1727 | 1.4527 | 76.954 |
| gemma-on-json_schema/C | 0.4085 | 0.1068 | 0.0312 | 0.0514 | 0.0871 | 0.0335 | 1832 | 0.9271 | 55.390 |
| gemma-on-json_schema/T | 0.3980 | 0.1104 | 0.0350 | 0.0500 | 0.0887 | 0.0384 | 2002 | 0.8413 | 58.106 |
| gemma-on-none/G | 0.6866 | 0.1065 | 0.0407 | 0.0440 | 0.1087 | 0.0545 | 1736 | 0.9070 | 79.107 |
| gemma-on-none/C | 0.3985 | 0.0807 | 0.0297 | 0.0288 | 0.0691 | 0.0318 | 1825 | 0.7752 | 63.360 |
| gemma-on-none/T | 0.4453 | 0.0902 | 0.0450 | 0.0355 | 0.0852 | 0.0636 | 2051 | 0.8321 | 66.145 |
| gemma-off-none/G | 0.6897 | 0.0960 | 0.3666 | 0.0448 | 0.1018 | 1.1322 | 1421 | 0.8985 | 18.155 |
| gemma-off-none/C | 3.8591 | 0.2905 | 0.0509 | 0.0397 | 0.0877 | 0.0435 | 1578 | 6.6773 | 19.714 |
| gemma-off-none/T | 0.5194 | 0.0925 | 0.0419 | 0.0398 | 0.1102 | 0.0287 | 1725 | 1.2669 | 23.973 |
| gemma-off-json_schema/G | 0.6838 | 0.1381 | 0.0258 | 0.0435 | 0.1115 | 0.0542 | 1409 | 0.9377 | 22.097 |
| gemma-off-json_schema/C | 0.4248 | 0.0849 | 0.0451 | 0.0450 | 0.0703 | 0.0375 | 1526 | 0.8080 | 19.171 |
| gemma-off-json_schema/T | 0.6002 | 0.1237 | 0.0419 | 0.0817 | 0.1297 | 0.0357 | 1661 | 1.1303 | 20.789 |


### H182-016 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 6345 | 52 | — | — | 2.921 | 2172.348 | 0.429 | 121.106 | 4.100 | completed |
| qwen25-05-json_schema/C | 5679 | 112 | — | — | 2.191 | 2591.662 | 0.917 | 122.087 | 3.862 | completed |
| qwen25-05-json_schema/T | 5471 | 263 | — | — | 2.109 | 2594.559 | 2.950 | 89.156 | 6.186 | completed |
| qwen25-05-none/G | 6345 | 331 | — | — | 2.525 | 2512.471 | 2.735 | 121.006 | 6.017 | completed |
| qwen25-15-none/G | 6345 | 61 | — | — | 7.658 | 828.532 | 1.002 | 60.855 | 9.547 | completed |
| qwen25-15-none/C | 5679 | 12 | — | — | 6.296 | 902.069 | 0.197 | 60.819 | 7.273 | completed |
| qwen25-15-json_schema/G | 6345 | 47 | — | — | 7.600 | 834.815 | 0.801 | 58.672 | 9.173 | completed |
| qwen25-15-json_schema/C | 5040 | 44 | — | — | 5.740 | 878.108 | 0.699 | 62.924 | 7.235 | completed |
| qwen25-3-json_schema/G | 6345 | 47 | — | — | 16.714 | 379.633 | 1.340 | 35.071 | 19.099 | completed |
| qwen25-3-json_schema/C | 5040 | 46 | — | — | 12.459 | 404.526 | 1.263 | 36.432 | 14.584 | completed |
| qwen25-3-none/G | 6345 | 57 | — | — | 16.738 | 379.074 | 1.634 | 34.891 | 19.243 | completed |
| qwen25-3-none/C | 5040 | 56 | — | — | 12.720 | 396.223 | 1.545 | 36.235 | 15.130 | completed |
| qwen25-3-none/T | 4943 | 53 | — | — | 12.736 | 388.103 | 1.457 | 36.376 | 15.081 | completed |
| coder25-05-none/G | 6345 | 46 | — | — | 2.596 | 2444.054 | 0.353 | 130.305 | 3.753 | completed |
| coder25-05-none/C | 4827 | 1553 | — | — | 1.843 | 2618.402 | 18.528 | 83.820 | 21.182 | grammar_failure |
| coder25-05-json_schema/G | 6345 | 46 | — | — | 2.700 | 2349.751 | 0.504 | 91.319 | 3.951 | completed |
| coder25-05-json_schema/C | 4827 | 17 | — | — | 1.837 | 2627.890 | 0.131 | 129.821 | 2.884 | completed |
| coder25-05-json_schema/T | 4767 | 46 | — | — | 1.724 | 2765.816 | 0.484 | 95.016 | 3.368 | completed |
| coder25-15-json_schema/G | 6345 | 47 | — | — | 8.386 | 756.661 | 0.832 | 56.490 | 10.021 | completed |
| coder25-15-json_schema/C | 4827 | 18 | — | — | 6.632 | 727.855 | 0.327 | 54.985 | 7.837 | completed |
| coder25-15-none/G | 6345 | 57 | — | — | 12.170 | 521.376 | 0.953 | 59.837 | 14.260 | completed |
| coder25-15-none/C | 5679 | 142 | — | — | 11.343 | 500.674 | 4.169 | 34.058 | 16.508 | completed |
| coder25-3-none/G | 6345 | 57 | — | — | 23.066 | 275.085 | 2.702 | 21.093 | 26.921 | completed |
| coder25-3-none/C | 4827 | 29 | — | — | 17.789 | 271.341 | 1.716 | 16.904 | 20.786 | completed |
| coder25-3-none/T | 4767 | 42 | — | — | 16.525 | 288.478 | 1.273 | 32.989 | 21.104 | completed |
| coder25-3-json_schema/G | 6345 | 47 | — | — | 18.847 | 336.651 | 1.353 | 34.736 | 21.161 | completed |
| coder25-3-json_schema/C | 4827 | 24 | — | — | 15.173 | 318.124 | 0.814 | 29.480 | 17.201 | completed |
| coder25-3-json_schema/T | 4767 | 42 | — | — | 15.649 | 304.628 | 1.305 | 32.185 | 19.852 | completed |
| qwen2-off-json_schema/G | 6042 | 52 | — | — | 9.837 | 614.204 | 0.896 | 58.029 | 12.913 | completed |
| qwen2-off-json_schema/C | 4731 | 48 | — | — | 7.387 | 640.409 | 0.865 | 55.516 | 10.060 | completed |
| qwen2-off-json_schema/T | 4637 | 93 | — | — | 6.954 | 666.771 | 1.885 | 49.332 | 12.018 | completed |
| qwen2-off-none/G | 6042 | 63 | — | — | 8.749 | 690.628 | 1.051 | 59.925 | 11.806 | completed |
| qwen2-off-none/C | 5157 | 131 | — | — | 7.665 | 672.795 | 2.170 | 60.369 | 11.765 | completed |
| qwen4-off-none/G | 6042 | 53 | — | — | 23.441 | 257.757 | 3.005 | 17.636 | 28.372 | completed |
| qwen4-off-none/C | 4944 | 54 | — | — | 20.031 | 246.813 | 3.165 | 17.063 | 25.304 | completed |
| qwen4-off-none/T | 4813 | 72 | — | — | 19.894 | 241.934 | 3.861 | 18.649 | 25.805 | completed |
| qwen4-off-json_schema/G | 6042 | 52 | — | — | 24.349 | 248.140 | 3.029 | 17.168 | 29.268 | completed |
| qwen4-off-json_schema/C | 4944 | 71 | — | — | 19.906 | 248.368 | 3.844 | 18.470 | 25.812 | completed |
| qwen4-off-json_schema/T | 4813 | 81 | — | — | 19.911 | 241.731 | 4.281 | 18.919 | 26.955 | completed |
| qwen4-on-json_schema/G | 6040 | 527 | 474 | — | 25.037 | 241.242 | 27.127 | 19.427 | 54.123 | completed |
| qwen4-on-json_schema/C | 4942 | 3985 | 3913 | — | 20.030 | 246.729 | 163.173 | 24.422 | 186.005 | completed |
| qwen4-on-json_schema/T | 4811 | 836 | 780 | — | 17.177 | 280.092 | 40.731 | 20.525 | 60.385 | completed |
| qwen4-on-none/G | 6040 | 529 | 474 | — | 22.919 | 263.539 | 26.756 | 19.771 | 51.530 | completed |
| qwen4-on-none/C | 4942 | 4011 | 3913 | — | 20.806 | 237.528 | 170.474 | 23.528 | 193.940 | completed |
| qwen4-on-none/T | 4811 | 884 | 780 | — | 17.470 | 275.386 | 43.909 | 20.133 | 63.355 | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 0.346 | 0.039 | 0.328 | 0.329 | 3.351 | 1.034 | 0.389 | 346.170 | 92.986 | 651.764 | — |
| qwen25-05-json_schema/C | 0.345 | 0.033 | 0.336 | 0.339 | 3.109 | 1.230 | 0.419 | 321.045 | 103.111 | 626.999 | — |
| qwen25-05-json_schema/T | 0.341 | 0.032 | 0.340 | 0.716 | 5.059 | 4.929 | 0.471 | 325.827 | 25.704 | 635.358 | — |
| qwen25-05-none/G | 0.347 | 0.036 | 0.340 | 0.340 | 5.261 | 1.729 | 0.506 | 349.170 | 93.017 | 651.764 | — |
| qwen25-15-none/G | 0.437 | 0.037 | 0.367 | 0.367 | 8.661 | 1.240 | 0.639 | 979.998 | 226.435 | 1512.873 | — |
| qwen25-15-none/C | 0.357 | 0.034 | 0.335 | 0.336 | 6.493 | 0.944 | 0.482 | 987.842 | 15.716 | 1492.764 | — |
| qwen25-15-json_schema/G | 0.351 | 0.036 | 0.336 | 0.337 | 8.402 | 1.157 | 0.547 | 1002.639 | 206.763 | 1512.873 | — |
| qwen25-15-json_schema/C | 0.359 | 0.029 | 0.353 | 0.355 | 6.439 | 1.069 | 0.494 | 963.498 | 174.154 | 1499.264 | — |
| qwen25-3-json_schema/G | 0.606 | 0.037 | 0.345 | 0.346 | 18.054 | 1.413 | 1.003 | 1872.512 | 264.460 | 2351.856 | — |
| qwen25-3-json_schema/C | 0.400 | 0.033 | 0.359 | 0.361 | 13.722 | 1.348 | 0.734 | 1842.559 | 202.163 | 2301.684 | — |
| qwen25-3-none/G | 0.402 | 0.038 | 0.368 | 0.369 | 18.372 | 1.535 | 0.825 | 1888.684 | 245.319 | 2351.856 | — |
| qwen25-3-none/C | 0.405 | 0.032 | 0.361 | 0.361 | 14.266 | 1.498 | 0.786 | 1842.559 | 199.194 | 2301.684 | — |
| qwen25-3-none/T | 0.424 | 0.032 | 0.366 | 0.366 | 14.194 | 1.457 | 0.772 | 1840.309 | 196.944 | 2296.106 | — |
| coder25-05-none/G | 0.367 | 0.040 | 0.368 | 0.368 | 2.949 | 0.998 | 0.256 | 333.170 | 103.017 | 651.764 | — |
| coder25-05-none/C | 0.367 | 0.031 | 0.383 | 0.384 | 20.372 | 6.573 | 1.180 | 0.003 | 356.060 | 617.874 | — |
| coder25-05-json_schema/G | 0.334 | 0.036 | 0.342 | 0.343 | 3.204 | 1.050 | 0.293 | 346.170 | 92.986 | 651.764 | — |
| coder25-05-json_schema/C | 0.422 | 0.032 | 0.426 | 0.427 | 1.968 | 1.023 | 0.241 | 321.608 | 19.423 | 617.874 | — |
| coder25-05-json_schema/T | 0.369 | 0.030 | 0.364 | 0.731 | 2.208 | 3.778 | 0.236 | 328.170 | 74.986 | 613.389 | — |
| coder25-15-json_schema/G | 0.367 | 0.040 | 0.355 | 0.356 | 9.218 | 1.170 | 0.542 | 999.404 | 209.998 | 1512.873 | — |
| coder25-15-json_schema/C | 0.416 | 0.032 | 0.377 | 0.379 | 6.960 | 1.100 | 0.533 | 964.217 | 18.685 | 1468.623 | — |
| coder25-15-none/G | 0.643 | 0.041 | 0.403 | 0.403 | 13.123 | 1.351 | 0.629 | 1002.639 | 203.795 | 1512.873 | — |
| coder25-15-none/C | 0.415 | 0.041 | 0.469 | 0.470 | 15.513 | 1.814 | 0.680 | 994.842 | 176.467 | 1492.764 | — |
| coder25-3-none/G | 0.669 | 0.044 | 0.388 | 0.388 | 25.769 | 1.712 | 1.306 | 1885.449 | 248.553 | 2351.856 | — |
| coder25-3-none/C | 0.516 | 0.043 | 0.516 | 0.516 | 19.506 | 1.864 | 1.240 | 1826.809 | 17.194 | 2283.715 | — |
| coder25-3-none/T | 2.281 | 0.084 | 0.787 | 0.788 | 17.801 | 2.120 | 1.891 | 1834.684 | 191.319 | 2277.903 | — |
| coder25-3-json_schema/G | 0.499 | 0.042 | 0.350 | 0.351 | 20.201 | 1.529 | 0.981 | 1888.684 | 248.288 | 2351.856 | — |
| coder25-3-json_schema/C | 0.531 | 0.048 | 0.571 | 0.573 | 15.988 | 1.570 | 0.839 | 1826.809 | 54.006 | 2283.715 | — |
| coder25-3-json_schema/T | 1.553 | 0.079 | 0.670 | 1.174 | 16.954 | 6.188 | 1.271 | 1832.199 | 196.772 | 2277.903 | — |
| qwen2-off-json_schema/G | 1.340 | 0.066 | 0.669 | 0.670 | 10.734 | 2.402 | 1.018 | 1100.534 | 106.634 | 1870.239 | — |
| qwen2-off-json_schema/C | 0.964 | 0.064 | 0.704 | 0.706 | 8.253 | 2.223 | 0.787 | 1085.534 | 91.631 | 1825.530 | — |
| qwen2-off-json_schema/T | 1.554 | 0.074 | 0.780 | 1.481 | 8.840 | 8.321 | 0.922 | 1084.409 | 90.509 | 1801.386 | — |
| qwen2-off-none/G | 1.190 | 0.071 | 0.670 | 0.671 | 9.800 | 2.532 | 1.026 | 1100.534 | 105.681 | 1870.239 | — |
| qwen2-off-none/C | 0.987 | 0.068 | 0.745 | 0.746 | 9.836 | 2.694 | 0.921 | 1090.597 | 95.740 | 1831.638 | — |
| qwen4-off-none/G | 1.126 | 0.065 | 0.649 | 0.650 | 26.447 | 2.686 | 2.306 | 2498.371 | 256.939 | 3217.433 | — |
| qwen4-off-none/C | 1.255 | 0.064 | 0.670 | 0.671 | 23.197 | 2.813 | 2.035 | 2464.402 | 222.861 | 3214.404 | — |
| qwen4-off-none/T | 1.239 | 0.060 | 0.640 | 0.641 | 23.756 | 3.058 | 2.017 | 2460.808 | 219.392 | 3203.924 | — |
| qwen4-off-json_schema/G | 1.042 | 0.065 | 0.635 | 0.636 | 27.379 | 2.738 | 1.976 | 2498.293 | 257.970 | 3217.433 | — |
| qwen4-off-json_schema/C | 1.234 | 0.059 | 0.650 | 0.652 | 23.751 | 3.061 | 1.946 | 2466.277 | 221.970 | 3214.404 | — |
| qwen4-off-json_schema/T | 1.175 | 0.069 | 0.687 | 1.437 | 24.193 | 8.581 | 1.812 | 2460.824 | 280.642 | 3203.924 | — |
| qwen4-on-json_schema/G | 1.051 | 0.066 | 0.650 | 0.652 | 52.165 | 8.126 | 3.809 | 2510.464 | 59.261 | 3217.433 | — |
| qwen4-on-json_schema/C | 1.932 | 0.060 | 0.687 | 0.689 | 183.207 | 48.816 | 16.717 | 2584.277 | 339.765 | 3211.357 | — |
| qwen4-on-json_schema/T | 1.161 | 0.063 | 0.650 | 1.165 | 57.909 | 15.907 | 4.569 | 2484.527 | 243.814 | 3202.643 | — |
| qwen4-on-none/G | 1.053 | 0.066 | 0.660 | 0.662 | 49.676 | 8.323 | 3.892 | 2514.293 | 272.826 | 3217.433 | — |
| qwen4-on-none/C | 1.379 | 0.138 | 1.039 | 1.040 | 191.285 | 48.454 | 15.916 | 2586.324 | 336.843 | 3211.357 | — |
| qwen4-on-none/T | 1.165 | 0.065 | 0.668 | 0.669 | 61.380 | 12.752 | 4.965 | 2484.527 | 242.817 | 3202.643 | — |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen25-05-json_schema/G | 0.6458 | 0.1410 | 0.0161 | 0.1817 | 0.1328 | 0.0652 | 1423 | 0.9212 | 4.102 |
| qwen25-05-json_schema/C | 0.4948 | 0.0986 | 0.0431 | 0.0411 | 0.1110 | 0.0341 | 1709 | 0.8766 | 3.864 |
| qwen25-05-json_schema/T | 0.4547 | 0.0921 | 0.0565 | 0.0369 | 0.3304 | 0.0722 | 2227 | 0.8436 | 6.188 |
| qwen25-05-none/G | 0.6738 | 0.1105 | 0.0333 | 0.0440 | 0.1181 | 0.0944 | 2167 | 0.9196 | 6.018 |
| qwen25-15-none/G | 0.7020 | 0.1080 | 0.0186 | 0.0530 | 0.1302 | 0.0612 | 1433 | 0.9262 | 9.549 |
| qwen25-15-none/C | 0.4586 | 0.1029 | 0.0325 | 0.0390 | 0.0767 | 0.0382 | 1379 | 0.8709 | 7.274 |
| qwen25-15-json_schema/G | 0.6698 | 0.0883 | 0.0334 | 0.0453 | 0.0984 | 0.1010 | 1425 | 0.8754 | 9.175 |
| qwen25-15-json_schema/C | 0.5492 | 0.1091 | 0.0480 | 0.0415 | 0.1195 | 0.0453 | 1478 | 1.0817 | 7.237 |
| qwen25-3-json_schema/G | 0.6594 | 0.0975 | 0.0278 | 0.0459 | 0.1146 | 0.0662 | 1425 | 0.8861 | 19.100 |
| qwen25-3-json_schema/C | 0.5202 | 0.0909 | 0.0371 | 0.0347 | 0.0896 | 0.0376 | 1490 | 0.9325 | 14.585 |
| qwen25-3-none/G | 0.6725 | 0.0805 | 0.0397 | 0.0515 | 0.0940 | 0.0586 | 1443 | 0.8643 | 19.245 |
| qwen25-3-none/C | 0.4941 | 0.1142 | 0.0562 | 0.0384 | 0.1140 | 0.0505 | 1498 | 0.9672 | 15.131 |
| qwen25-3-none/T | 0.5171 | 0.0980 | 0.0529 | 0.0426 | 0.0956 | 0.0368 | 1502 | 1.0357 | 15.083 |
| coder25-05-none/G | 0.6694 | 0.1028 | 0.0153 | 0.0485 | 0.1256 | 0.0417 | 1417 | 0.8888 | 3.754 |
| coder25-05-none/C | 0.3822 | 0.0661 | — | 0.0291 | 0.0821 | 0.0224 | 1310 | 0.6607 | 21.183 |
| coder25-05-json_schema/G | 0.6726 | 0.0794 | 0.0246 | 0.0456 | 0.0976 | 0.0565 | 1419 | 0.8628 | 3.952 |
| coder25-05-json_schema/C | 0.4925 | 0.0876 | 0.0268 | 0.0383 | 0.1258 | 0.0235 | 1399 | 0.8791 | 2.885 |
| coder25-05-json_schema/T | 0.4050 | 0.0774 | 0.0282 | 0.0303 | 0.0660 | 0.0276 | 1501 | 0.9115 | 3.369 |
| coder25-15-json_schema/G | 0.6640 | 0.1253 | 0.0302 | 0.0458 | 0.1190 | 0.0714 | 1431 | 0.9252 | 10.023 |
| coder25-15-json_schema/C | 0.4021 | 0.0919 | 0.0501 | 0.0339 | 0.0861 | 0.0319 | 1403 | 0.7805 | 7.838 |
| coder25-15-none/G | 0.6773 | 0.0828 | 0.0304 | 0.0451 | 0.1196 | 0.0698 | 1439 | 0.8818 | 14.262 |
| coder25-15-none/C | 0.5410 | 0.1340 | 0.0577 | 0.0456 | 0.1414 | 0.0437 | 1795 | 1.0884 | 16.509 |
| coder25-3-none/G | 0.9130 | 0.1187 | 0.0172 | 0.0615 | 0.3574 | 0.0495 | 1428 | 1.1880 | 26.923 |
| coder25-3-none/C | 0.4311 | 0.0731 | 4.0802 | 0.0318 | 0.2857 | 0.0558 | 1414 | 0.7993 | 20.792 |
| coder25-3-none/T | 6.1551 | 0.1890 | 0.9171 | 0.1383 | 2.7119 | 0.7499 | 1471 | 11.7459 | 21.120 |
| coder25-3-json_schema/G | 0.9557 | 0.2001 | 0.0375 | 0.0687 | 0.1785 | 0.0803 | 1431 | 1.3363 | 21.163 |
| coder25-3-json_schema/C | 0.5028 | 0.1025 | 0.0333 | 0.0431 | 0.1085 | 0.0309 | 1409 | 1.0275 | 17.202 |
| coder25-3-json_schema/T | 0.7060 | 0.0939 | 0.0696 | 0.0594 | 0.1089 | 0.0421 | 1483 | 1.1650 | 19.854 |
| qwen2-off-json_schema/G | 1.7848 | 0.1912 | 0.0235 | 0.1021 | 0.2475 | 0.0573 | 1405 | 2.3527 | 12.916 |
| qwen2-off-json_schema/C | 0.3978 | 0.0819 | 0.0336 | 0.0301 | 0.0898 | 0.0317 | 1459 | 0.7474 | 10.061 |
| qwen2-off-json_schema/T | 0.4647 | 0.0768 | 0.0365 | 0.0368 | 0.1114 | 0.0320 | 1675 | 0.8590 | 12.019 |
| qwen2-off-none/G | 1.3440 | 0.1795 | 0.0237 | 0.1003 | 0.2388 | 0.0510 | 1418 | 1.7364 | 11.808 |
| qwen2-off-none/C | 0.3959 | 0.0813 | 0.0708 | 0.0343 | 0.0777 | 0.0662 | 1689 | 0.7648 | 11.767 |
| qwen4-off-none/G | 0.8069 | 0.1298 | 0.0405 | 0.0605 | 0.1331 | 0.1017 | 1407 | 1.0791 | 28.374 |
| qwen4-off-none/C | 0.5775 | 0.1750 | 0.0666 | 0.0495 | 0.1198 | 0.0691 | 1504 | 1.4667 | 25.306 |
| qwen4-off-none/T | 0.7064 | 0.1478 | 0.0498 | 0.0731 | 0.1617 | 0.0314 | 1647 | 1.4066 | 25.806 |
| qwen4-off-json_schema/G | 0.7652 | 0.0980 | 0.0478 | 0.0548 | 0.1243 | 0.1087 | 1410 | 1.0022 | 29.270 |
| qwen4-off-json_schema/C | 0.9400 | 0.2511 | 0.0377 | 0.0721 | 0.1636 | 0.0365 | 1531 | 1.8416 | 25.814 |
| qwen4-off-json_schema/T | 0.4441 | 0.0844 | 0.2675 | 0.0383 | 0.0802 | 0.0654 | 1681 | 0.8249 | 26.957 |
| qwen4-on-json_schema/G | 0.9025 | 0.3110 | 0.0605 | 0.1922 | 0.1420 | 0.1343 | 1721 | 1.3583 | 54.125 |
| qwen4-on-json_schema/C | 1.0024 | 0.3331 | 0.0298 | 0.0748 | 0.2825 | 0.0372 | 1846 | 2.0332 | 186.008 |
| qwen4-on-json_schema/T | 0.4077 | 0.1195 | 0.0339 | 0.0405 | 0.0935 | 0.0481 | 1877 | 0.7905 | 60.386 |
| qwen4-on-none/G | 1.0808 | 0.2983 | 0.0239 | 0.0762 | 0.1546 | 0.0960 | 1721 | 1.5140 | 51.533 |
| qwen4-on-none/C | 0.4462 | 0.1275 | 0.0327 | 0.0349 | 0.0765 | 0.0370 | 1891 | 0.8791 | 193.941 |
| qwen4-on-none/T | 0.4167 | 0.0685 | 0.0314 | 0.0337 | 0.0972 | 0.0291 | 1945 | 0.9417 | 63.356 |


### H182-017 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-off-json_schema/G | 6623 | 50 | 0 | 50 | 22.674 | 292.093 | 1.671 | 29.919 | 26.113 | completed |
| spark-off-json_schema/C | 5594 | 69 | 0 | 69 | 19.817 | 282.289 | 2.308 | 29.900 | 24.488 | completed |
| spark-off-json_schema/T | 5483 | 129 | 0 | 129 | 18.975 | 288.963 | 4.290 | 30.070 | 25.920 | completed |
| spark-off-none/G | 6623 | 50 | 0 | 50 | 23.845 | 277.756 | 1.646 | 30.373 | 27.125 | completed |
| spark-off-none/C | 5594 | 70 | 0 | 70 | 20.270 | 275.981 | 2.283 | 30.656 | 24.425 | completed |
| spark-off-none/T | 5484 | 137 | 0 | 137 | 20.140 | 272.298 | 4.454 | 30.758 | 26.350 | runtime_error |
| spark-low-none/G | 6674 | 2048 | 2048 | 0 | 26.092 | 255.792 | 88.414 | 23.164 | 116.166 | max_tokens |
| spark-low-json_schema/G | 6674 | 2048 | 2048 | 0 | 24.385 | 273.697 | 88.589 | 23.118 | 114.981 | max_tokens |
| spark-high-json_schema/G | 6674 | 2048 | 2048 | 0 | 24.864 | 268.425 | 90.265 | 22.689 | 117.151 | max_tokens |
| spark-high-none/G | 6674 | 2048 | 2048 | 0 | 25.266 | 264.152 | 91.542 | 22.372 | 118.490 | max_tokens |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-off-json_schema/G | 0.607 | 0.041 | 0.344 | 0.344 | 24.391 | 2.018 | 1.287 | 2503.819 | 248.334 | 3918.663 | 297.000 |
| spark-off-json_schema/C | 0.894 | 0.047 | 0.358 | 0.358 | 22.157 | 2.102 | 1.092 | 2467.819 | 212.866 | 3918.663 | 261.000 |
| spark-off-json_schema/T | 1.066 | 0.129 | 0.352 | 0.352 | 23.313 | 2.863 | 1.281 | 2458.819 | 10.866 | 3918.663 | 252.000 |
| spark-off-none/G | 0.562 | 0.037 | 0.000 | 0.000 | 25.526 | 1.622 | 1.032 | 2503.819 | 248.069 | 3918.663 | 297.000 |
| spark-off-none/C | 0.757 | 0.046 | 0.000 | 0.000 | 22.588 | 1.908 | 1.058 | 2467.819 | 212.597 | 3918.663 | 261.000 |
| spark-off-none/T | 0.660 | 0.040 | 0.000 | 0.000 | 24.629 | 2.626 | 1.184 | — | — | — | — |
| spark-low-none/G | 0.561 | 0.040 | 0.000 | 0.000 | 114.545 | 25.319 | 6.626 | 2575.819 | 319.951 | 3918.663 | 369.000 |
| spark-low-json_schema/G | 0.558 | 0.038 | 0.350 | 0.350 | 113.013 | 25.050 | 6.585 | 2575.819 | 319.953 | 3918.663 | 369.000 |
| spark-high-json_schema/G | 0.560 | 0.039 | 0.343 | 0.343 | 115.166 | 25.813 | 6.735 | 2575.819 | 319.953 | 3918.663 | 369.000 |
| spark-high-none/G | 0.563 | 0.040 | 0.000 | 0.000 | 116.847 | 24.531 | 6.508 | 2575.819 | 319.954 | 3918.663 | 369.000 |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-off-json_schema/G | 0.6421 | 0.1776 | 0.0239 | 0.0423 | 0.1237 | 0.0850 | 11012 | 0.9731 | 26.115 |
| spark-off-json_schema/C | 0.4584 | 0.0965 | 0.0740 | 0.0523 | 0.0794 | 0.0819 | 11132 | 0.8540 | 24.490 |
| spark-off-json_schema/T | 0.4423 | 0.0962 | 0.0450 | 0.0876 | 0.0833 | 0.0556 | 11376 | 0.9143 | 25.921 |
| spark-off-none/G | 0.6998 | 0.0817 | 0.0282 | 0.0481 | 0.0917 | 0.0883 | 10997 | 0.8987 | 27.127 |
| spark-off-none/C | 0.4010 | 0.0911 | 0.0553 | 0.0315 | 0.0689 | 0.0750 | 11111 | 0.8027 | 24.426 |
| spark-off-none/T | 0.4038 | 0.0696 | — | 0.0312 | 0.0684 | 0.0222 | 1919 | 0.8080 | 26.352 |
| spark-low-none/G | 0.6878 | 0.1068 | — | 0.0450 | 0.0980 | 0.0957 | 10916 | 0.9087 | 116.168 |
| spark-low-json_schema/G | 0.6940 | 0.0780 | — | 0.0437 | 0.0909 | 0.0813 | 10916 | 0.8865 | 114.983 |
| spark-high-json_schema/G | 0.6857 | 0.1011 | — | 0.0482 | 0.0986 | 0.0976 | 10914 | 0.8967 | 117.153 |
| spark-high-none/G | 0.6742 | 0.0985 | — | 0.0433 | 0.1146 | 0.0954 | 10909 | 0.8774 | 118.493 |


### H182-018 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| stage1-spark-off-json_schema/G | 6623 | 50 | 0 | 50 | 22.504 | 294.300 | 1.675 | 29.842 | 26.506 | completed |
| stage1-spark-off-json_schema/C | 5676 | 69 | 0 | 69 | 19.259 | 294.715 | 2.266 | 30.451 | 24.043 | completed |
| stage1-spark-off-json_schema/T | 5548 | 104 | 0 | 104 | 19.489 | 284.678 | 12.186 | 8.535 | 35.332 | completed |
| stage2-spark-low-json_schema/G | 6709 | 2048 | 2048 | 0 | 90.838 | 73.857 | 68.393 | 29.945 | 163.478 | max_tokens |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| stage1-spark-off-json_schema/G | 0.704 | 0.050 | 0.345 | 0.345 | 24.298 | 1.966 | 1.329 | 2503.819 | 248.330 | 3918.663 | 297.000 |
| stage1-spark-off-json_schema/C | 1.053 | 0.039 | 0.343 | 0.343 | 21.556 | 2.203 | 1.211 | 2467.819 | 10.864 | 3918.663 | 261.000 |
| stage1-spark-off-json_schema/T | 0.782 | 0.040 | 0.345 | 0.345 | 31.845 | 3.731 | 1.751 | 2467.819 | 212.863 | 3918.663 | 261.000 |
| stage2-spark-low-json_schema/G | 1.298 | 0.195 | 0.762 | 0.762 | 159.414 | 24.902 | 6.833 | 2575.819 | 319.951 | 3918.663 | 369.000 |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| stage1-spark-off-json_schema/G | 0.6529 | 0.0747 | 0.0180 | 0.0410 | 0.0991 | 0.1024 | 11015 | 0.8513 | 26.508 |
| stage1-spark-off-json_schema/C | 0.4193 | 0.1175 | 0.0669 | 0.0344 | 0.1001 | 0.0729 | 11133 | 0.8786 | 24.044 |
| stage1-spark-off-json_schema/T | 0.3995 | 0.0723 | 0.5051 | 0.0354 | 0.0820 | 0.3431 | 11317 | 0.8988 | 35.381 |
| stage2-spark-low-json_schema/G | 1.8702 | 0.2050 | — | 0.1217 | 0.3364 | 1.3777 | 10912 | 2.4491 | 163.485 |


### H182-019 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-json_schema/G | 6709 | 2048 | 2048 | 0 | 22.537 | 297.684 | 68.080 | 30.082 | 93.424 | max_tokens |
| spark-off-json_schema/G | 6658 | 50 | 0 | 50 | 24.633 | 270.286 | 3.568 | 14.013 | 30.233 | completed |
| spark-off-json_schema/C | 5939 | 103 | 0 | 103 | 60.734 | 97.788 | 6.717 | 15.335 | 70.318 | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-json_schema/G | 0.772 | 0.074 | 0.355 | 0.355 | 90.720 | 23.232 | 5.917 | 2575.819 | 319.953 | 3918.663 | 369.000 |
| spark-off-json_schema/G | 0.569 | 0.037 | 0.347 | 0.347 | 28.250 | 1.960 | 1.083 | 2503.819 | 10.830 | 3918.663 | 297.000 |
| spark-off-json_schema/C | 0.919 | 0.063 | 0.445 | 0.445 | 67.519 | 3.219 | 1.419 | 2476.819 | 10.863 | 3918.663 | 270.000 |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-json_schema/G | 0.6302 | 0.1140 | — | 0.0457 | 0.0876 | 0.0928 | 10913 | 0.8742 | 93.426 |
| spark-off-json_schema/G | 0.6715 | 0.1333 | 0.0323 | 0.0463 | 0.1130 | 0.1295 | 11014 | 0.9258 | 30.235 |
| spark-off-json_schema/C | 0.5178 | 0.1301 | 0.0677 | 0.0376 | 0.0852 | 0.0668 | 11258 | 0.9844 | 70.319 |


### H182-020 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp10-json_schema/G | 6709 | 2048 | 2048 | 0 | 23.533 | 285.086 | 74.059 | 27.654 | 100.965 | max_tokens |
| spark-low-rp11-json_schema/G | 6709 | 2048 | 2048 | 0 | 25.447 | 263.649 | 81.099 | 25.253 | 109.076 | max_tokens |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp10-json_schema/G | 1.049 | 0.090 | 0.387 | 0.387 | 97.822 | 25.995 | 8.319 | 2575.819 | 319.953 | 3918.663 | 369.000 |
| spark-low-rp11-json_schema/G | 0.692 | 0.097 | 0.360 | 0.360 | 106.617 | 27.210 | 7.606 | 2575.819 | 321.327 | 3918.663 | 369.000 |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp10-json_schema/G | 0.7185 | 0.0977 | — | 0.0446 | 0.1363 | 1.5177 | 11027 | 0.9760 | 100.970 |
| spark-low-rp11-json_schema/G | 0.6650 | 0.0816 | — | 0.0431 | 0.1175 | 0.1356 | 11084 | 0.8759 | 109.078 |


### H182-021 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp11-json_schema/G | 6832 | 2048 | 2048 | 0 | 24.028 | 284.338 | 74.848 | 27.362 | 102.044 | max_tokens |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp11-json_schema/G | 1.020 | 0.043 | 0.368 | 0.368 | 99.078 | 25.936 | 8.043 | 2575.819 | 321.326 | 3918.663 | 369.000 |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| spark-low-rp11-json_schema/G | 0.6625 | 0.1216 | — | 0.0464 | 0.1195 | 0.1554 | 11082 | 0.9398 | 102.046 |


### H182-022 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen4-on-rp10-json_schema/G | 6040 | 527 | 474 | 52 | 20.923 | 288.672 | 21.705 | 24.280 | 45.057 | completed |
| qwen4-on-rp10-json_schema/C | 4942 | 3985 | 3913 | 71 | 16.974 | 291.152 | 166.267 | 23.968 | 185.401 | completed |
| qwen4-on-rp10-json_schema/T | 4811 | 836 | 780 | 55 | 17.416 | 276.242 | 40.774 | 20.503 | 61.164 | completed |
| qwen4-on-rp11-json_schema/G | 6040 | 598 | 545 | 52 | 23.189 | 260.467 | 34.239 | 17.466 | 59.454 | completed |
| qwen4-on-rp11-json_schema/C | 4942 | 3331 | 3259 | 71 | 18.838 | 262.344 | 141.534 | 23.535 | 162.640 | completed |
| qwen4-on-rp11-json_schema/T | 4811 | 705 | 646 | 58 | 17.352 | 277.257 | 34.621 | 20.364 | 54.776 | completed |
| qwen4-off-rp10-json_schema/G | 6042 | 52 | 0 | 52 | 23.811 | 253.749 | 2.968 | 17.518 | 28.824 | completed |
| qwen4-off-rp10-json_schema/C | 4944 | 71 | 0 | 71 | 21.292 | 232.199 | 4.208 | 16.871 | 27.636 | completed |
| qwen4-off-rp10-json_schema/T | 4813 | 81 | 0 | 81 | 22.721 | 211.830 | 4.467 | 18.132 | 30.452 | completed |
| qwen4-off-rp11-json_schema/G | 6042 | 52 | 0 | 52 | 25.202 | 239.738 | 3.203 | 16.235 | 30.542 | completed |
| qwen4-off-rp11-json_schema/C | 4944 | 71 | 0 | 71 | 20.864 | 236.965 | 3.946 | 17.992 | 26.937 | completed |
| qwen4-off-rp11-json_schema/T | 4813 | 81 | 0 | 81 | 20.951 | 229.726 | 4.426 | 18.301 | 28.545 | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen4-on-rp10-json_schema/G | 1.559 | 0.073 | 0.689 | 0.692 | 42.631 | — | — | 2512.449 | 160.308 | 3217.433 | — |
| qwen4-on-rp10-json_schema/C | 1.244 | 0.061 | 0.718 | 0.721 | 183.246 | — | — | 2258.043 | 666.046 | 3211.357 | — |
| qwen4-on-rp10-json_schema/T | 1.549 | 0.062 | 0.650 | 1.193 | 58.190 | — | — | 2488.652 | 239.671 | 3202.643 | — |
| qwen4-on-rp11-json_schema/G | 1.234 | 0.066 | 0.640 | 0.642 | 57.428 | — | — | 2487.574 | 300.999 | 3217.433 | — |
| qwen4-on-rp11-json_schema/C | 1.411 | 0.065 | 0.697 | 0.700 | 160.371 | — | — | 2527.824 | 93.499 | 3211.357 | — |
| qwen4-on-rp11-json_schema/T | 1.430 | 0.064 | 0.664 | 1.192 | 51.975 | — | — | 2450.496 | 261.986 | 3202.643 | — |
| qwen4-off-rp10-json_schema/G | 1.205 | 0.068 | 0.640 | 0.644 | 26.782 | — | — | 2486.027 | 319.551 | 3217.433 | — |
| qwen4-off-rp10-json_schema/C | 1.245 | 0.065 | 0.700 | 0.703 | 25.501 | — | — | 2307.043 | 442.767 | 3214.404 | — |
| qwen4-off-rp10-json_schema/T | 1.549 | 0.075 | 0.858 | 1.544 | 27.189 | — | — | 2438.808 | 242.379 | 3203.924 | — |
| qwen4-off-rp11-json_schema/G | 1.307 | 0.071 | 0.645 | 0.646 | 28.406 | — | — | 2473.621 | 357.049 | 3217.433 | — |
| qwen4-off-rp11-json_schema/C | 1.282 | 0.061 | 0.642 | 0.645 | 24.810 | — | — | 2258.043 | 491.767 | 3214.404 | — |
| qwen4-off-rp11-json_schema/T | 1.500 | 0.064 | 0.689 | 1.514 | 25.378 | — | — | 2450.824 | 230.361 | 3203.924 | — |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen4-on-rp10-json_schema/G | 0.6942 | 0.0851 | 0.3065 | — | — | — | — | — | 45.058 |
| qwen4-on-rp10-json_schema/C | 1.3060 | 0.3156 | 0.0478 | — | — | — | — | — | 185.402 |
| qwen4-on-rp10-json_schema/T | 0.3992 | 0.1198 | 0.0519 | — | — | — | — | — | 61.164 |
| qwen4-on-rp11-json_schema/G | 0.7924 | 0.0884 | 0.0382 | — | — | — | — | — | 59.455 |
| qwen4-on-rp11-json_schema/C | 0.6236 | 0.1227 | 0.0505 | — | — | — | — | — | 162.641 |
| qwen4-on-rp11-json_schema/T | 0.4545 | 0.1049 | 1.1669 | — | — | — | — | — | 54.778 |
| qwen4-off-rp10-json_schema/G | 0.8234 | 0.1691 | 0.3130 | — | — | — | — | — | 28.826 |
| qwen4-off-rp10-json_schema/C | 0.9527 | 0.2990 | 0.0318 | — | — | — | — | — | 27.637 |
| qwen4-off-rp10-json_schema/T | 0.4651 | 0.1125 | 0.0478 | — | — | — | — | — | 30.453 |
| qwen4-off-rp11-json_schema/G | 0.9814 | 0.2054 | 0.0579 | — | — | — | — | — | 30.543 |
| qwen4-off-rp11-json_schema/C | 0.6906 | 0.1402 | 0.0473 | — | — | — | — | — | 26.938 |
| qwen4-off-rp11-json_schema/T | 0.6771 | 0.0983 | 0.0617 | — | — | — | — | — | 28.546 |


### H182-023 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| minicpm5-int4-json_schema/G | 5880 | 1809 | 1761 | 47 | 12.894 | 456.020 | 37.213 | 48.612 | 52.641 | completed |
| minicpm5-int4-json_schema/C | 5148 | 2800 | 2697 | 102 | 9.103 | 565.521 | 57.996 | 48.279 | 68.835 | completed |
| minicpm5-int4-json_schema/T | 5050 | 4096 | 4053 | 42 | 10.092 | 500.398 | 104.925 | 39.038 | 116.808 | max_tokens |
| minicpm5-int4-none/G | 5880 | 1794 | 1761 | 32 | 12.141 | 484.329 | 53.656 | 33.435 | 67.480 | completed |
| minicpm5-int4-none/C | 5148 | 2795 | 2697 | 97 | 12.549 | 410.245 | 74.430 | 37.552 | 89.139 | completed |
| minicpm5-int8-none/G | 5880 | 1558 | 1509 | 48 | 13.866 | 424.056 | 66.893 | 23.291 | 83.707 | completed |
| minicpm5-int8-none/C | 4910 | 1203 | 1139 | 63 | 12.413 | 395.560 | 54.484 | 22.080 | 69.895 | completed |
| minicpm5-int8-none/T | 4844 | 2411 | 2332 | 78 | 12.245 | 395.595 | 112.460 | 21.439 | 127.230 | completed |
| minicpm5-int8-json_schema/G | 5880 | 1557 | 1509 | 47 | 14.658 | 401.150 | 75.822 | 20.535 | 93.102 | completed |
| minicpm5-int8-json_schema/C | 4910 | 1197 | 1139 | 57 | 12.399 | 395.994 | 56.299 | 21.261 | 71.912 | completed |
| minicpm5-int8-json_schema/T | 4844 | 2410 | 2332 | 77 | 12.485 | 387.982 | 112.650 | 21.394 | 128.102 | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| minicpm5-int4-json_schema/G | 0.552 | 0.130 | 0.359 | 0.359 | 50.162 | 12.420 | 3.236 | 1676.082 | 332.109 | 2174.082 | 325.500 |
| minicpm5-int4-json_schema/C | 0.453 | 0.045 | 0.367 | 0.367 | 67.119 | 19.416 | 4.428 | 1686.582 | 342.609 | 2240.238 | 336.000 |
| minicpm5-int4-json_schema/T | 0.461 | 0.044 | 0.372 | 0.372 | 115.037 | 32.615 | 7.486 | 1728.582 | 384.640 | 2213.332 | 378.000 |
| minicpm5-int4-none/G | 0.448 | 0.043 | 0.000 | 0.000 | 65.819 | 17.779 | 4.640 | 1665.582 | 12.206 | 2174.082 | 315.000 |
| minicpm5-int4-none/C | 0.577 | 0.049 | 0.000 | 0.000 | 87.001 | 30.706 | 7.916 | 1686.582 | 339.097 | 2240.238 | 336.000 |
| minicpm5-int8-none/G | 1.251 | 0.088 | 0.000 | 0.000 | 80.968 | 21.826 | 5.849 | 2865.582 | 318.223 | 3363.020 | 315.000 |
| minicpm5-int8-none/C | 0.933 | 0.056 | 0.000 | 0.000 | 66.951 | 32.500 | 8.826 | 2802.582 | 12.188 | 3358.363 | 252.000 |
| minicpm5-int8-none/T | 0.621 | 0.049 | 0.000 | 0.000 | 124.757 | 62.149 | 16.372 | 2855.082 | 307.095 | 3358.363 | 304.500 |
| minicpm5-int8-json_schema/G | 0.716 | 0.069 | 0.390 | 0.390 | 90.547 | 43.050 | 11.227 | 2865.582 | 319.610 | 3363.020 | 315.000 |
| minicpm5-int8-json_schema/C | 0.730 | 0.055 | 0.374 | 0.374 | 68.747 | 33.447 | 8.909 | 2802.582 | 12.454 | 3358.363 | 252.000 |
| minicpm5-int8-json_schema/T | 0.640 | 0.047 | 0.374 | 0.374 | 125.185 | 63.077 | 16.392 | 2855.082 | 307.360 | 3358.363 | 304.500 |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| minicpm5-int4-json_schema/G | 0.6296 | 0.0701 | 0.0327 | 0.0425 | 0.1022 | 0.0865 | 10107 | 0.8734 | 52.642 |
| minicpm5-int4-json_schema/C | 0.3950 | 0.0996 | 0.0320 | 0.0390 | 0.0811 | 0.0475 | 10363 | 0.8312 | 68.836 |
| minicpm5-int4-json_schema/T | 0.4044 | 0.1163 | — | 0.0399 | 0.0784 | 0.0526 | 9996 | 0.8188 | 116.809 |
| minicpm5-int4-none/G | 0.7938 | 0.0902 | 0.0368 | 0.0540 | 0.1172 | 0.1425 | 10070 | 1.0185 | 67.482 |
| minicpm5-int4-none/C | 1.0565 | 0.2085 | 0.0475 | 0.0884 | 0.1761 | 0.2105 | 10299 | 1.8736 | 89.141 |
| minicpm5-int8-none/G | 0.7969 | 0.1278 | 0.1365 | 0.0537 | 0.1478 | 0.2229 | 10103 | 1.0729 | 83.711 |
| minicpm5-int8-none/C | 1.1400 | 0.3888 | 0.1497 | 0.1029 | 0.2107 | 0.1791 | 10205 | 3.7848 | 69.909 |
| minicpm5-int8-none/T | 1.0874 | 0.4748 | 0.0752 | 0.1191 | 0.6402 | 0.1333 | 10293 | 3.9420 | 127.234 |
| minicpm5-int8-json_schema/G | 0.7828 | 0.1768 | 0.0540 | 0.0578 | 0.1371 | 0.3403 | 10114 | 1.1124 | 93.106 |
| minicpm5-int8-json_schema/C | 1.1627 | 0.3343 | 0.0655 | 0.0905 | 0.2057 | 0.1323 | 10212 | 2.7001 | 71.915 |
| minicpm5-int8-json_schema/T | 1.0560 | 0.1820 | 0.0911 | 0.0917 | 0.2457 | 0.1453 | 10305 | 1.9083 | 128.106 |


### H182-024 フェーズ詳細

| case/phase | 入力tokens | 実出力tokens | Thinking数※ | 最終tokens | prefill秒 | prefill tok/s | decode秒 | decode tok/s | helper wall秒 | stop |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| on-json_schema-old/G | 6040 | 527 | 474 | — | 19.140 | 315.574 | 19.689 | 26.766 | 41.311 | completed |
| on-json_schema-old/C | 4944 | 4096 | 4097 | — | 15.825 | 312.408 | 154.945 | 26.435 | 172.849 | max_tokens |
| on-none-old/G | 6040 | 529 | 474 | — | 18.870 | 320.079 | 19.754 | 26.780 | 41.065 | completed |
| on-none-old/C | 4944 | 4096 | 4097 | — | 15.473 | 319.517 | 155.970 | 26.261 | 173.908 | max_tokens |
| off-none-old/G | 6042 | 53 | — | — | 19.803 | 305.109 | 1.872 | 28.316 | 23.663 | completed |
| off-none-old/C | 4946 | 54 | — | — | 16.703 | 296.112 | 1.954 | 27.634 | 22.501 | completed |
| off-none-old/T | 4830 | 83 | — | — | 16.758 | 288.217 | 3.054 | 27.179 | 21.975 | completed |
| off-none-new/T | 4903 | 82 | — | — | 16.592 | 295.503 | 2.987 | 27.452 | 21.348 | completed |
| off-json_schema-old/G | 6042 | 52 | — | — | 20.598 | 293.329 | 1.920 | 27.079 | 24.320 | completed |
| off-json_schema-old/C | 4946 | 71 | — | — | 23.103 | 214.085 | 3.972 | 17.874 | 29.349 | completed |
| off-json_schema-old/T | 4830 | 68 | — | — | 20.065 | 240.717 | 3.507 | 19.387 | 26.820 | completed |
| off-json_schema-new/T | 4903 | 82 | — | — | 18.858 | 259.994 | 4.023 | 20.385 | 25.056 | completed |

| case/phase | load秒 | prepare秒 | grammar準備秒 | grammar setup秒 | generation wall秒 | CPU user秒 | CPU system秒 | MLX active MiB | allocator cache MiB | MLX peak MiB | KV終端MiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| on-json_schema-old/G | 1.211 | 0.080 | 0.674 | 0.677 | 38.832 | 6.786 | 3.911 | 2510.386 | 59.559 | 3217.433 | — |
| on-json_schema-old/C | 1.125 | 0.057 | 0.673 | 0.675 | 0.000 | 40.354 | 11.075 | 254.770 | 2668.392 | 3214.404 | — |
| on-none-old/G | 1.245 | 0.080 | 0.813 | 0.815 | 38.624 | 7.541 | 3.759 | 2514.355 | 272.870 | 3217.433 | — |
| on-none-old/C | 1.356 | 0.062 | 0.724 | 0.726 | 0.000 | 47.144 | 13.504 | 2590.371 | 332.749 | 3214.404 | — |
| off-none-old/G | 1.094 | 0.065 | 0.635 | 0.636 | 21.675 | 2.229 | 2.199 | 2498.402 | 256.908 | 3217.433 | — |
| off-none-old/C | 2.742 | 0.072 | 0.668 | 0.669 | 18.657 | 2.523 | 2.151 | 2464.371 | 222.892 | 3217.794 | — |
| off-none-old/T | 1.205 | 0.061 | 0.703 | 0.705 | 19.839 | 3.065 | 2.228 | 2461.043 | 219.689 | 3205.169 | — |
| off-none-new/T | 0.923 | 0.057 | 0.645 | 0.646 | 19.579 | 2.796 | 1.799 | 2465.324 | 219.908 | 3204.810 | — |
| off-json_schema-old/G | 0.933 | 0.068 | 0.664 | 0.668 | 22.537 | 2.487 | 2.118 | 2500.371 | 255.895 | 3217.433 | — |
| off-json_schema-old/C | 1.218 | 0.072 | 0.805 | 0.807 | 27.075 | 3.277 | 2.552 | 2466.402 | 283.470 | 3217.794 | — |
| off-json_schema-old/T | 1.765 | 0.065 | 0.671 | 1.258 | 23.573 | 7.787 | 2.729 | 2461.105 | 220.551 | 3205.169 | — |
| off-json_schema-new/T | 0.885 | 0.058 | 0.639 | 1.113 | 22.881 | 6.043 | 1.923 | 2463.277 | 222.876 | 3204.810 | — |

ホスト側内訳。ms列はすべてミリ秒。重なる区間を全列加算しない。

| case/phase | 履歴投影ms | serialize等ms | 応答安全性ms | encode ms | write ms | decode JSON ms | 応答bytes | phase間gap ms | host phase計秒 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| on-json_schema-old/G | — | — | — | 0.0461 | 0.0945 | 0.0618 | 1722 | 0.9016 | — |
| on-json_schema-old/C | — | — | — | 0.0321 | 0.1008 | 0.0320 | 1335 | 0.8020 | — |
| on-none-old/G | — | — | — | 0.0552 | 0.1292 | 0.1213 | 1724 | 0.8769 | — |
| on-none-old/C | — | — | — | 0.0365 | 0.0936 | 0.0653 | 1330 | 0.8214 | — |
| off-none-old/G | — | — | — | 0.0444 | 0.1114 | 5.7646 | 1409 | 0.8892 | — |
| off-none-old/C | — | — | — | 22.1954 | 0.1598 | 0.0471 | 1503 | 23.8904 | — |
| off-none-old/T | — | — | — | 0.2231 | 0.0861 | 0.0658 | 1696 | 1.3189 | — |
| off-none-new/T | — | — | — | 0.0535 | 0.0661 | 0.0369 | 1685 | 0.8033 | — |
| off-json_schema-old/G | — | — | — | 0.0550 | 0.1266 | 0.0499 | 1405 | 0.8737 | — |
| off-json_schema-old/C | — | — | — | 0.0434 | 0.1948 | 0.0712 | 1525 | 1.5938 | — |
| off-json_schema-old/T | — | — | — | 0.0634 | 0.2005 | 0.0460 | 1634 | 1.8137 | — |
| off-json_schema-new/T | — | — | — | 0.0295 | 0.0755 | 0.0374 | 1688 | 0.6484 | — |


## 8. 出典と保存状態

| 実験 | 保存済み報告 | 計測元 | Issue上の元報告 |
| --- | --- | --- | --- |
| H182-011 | `docs/benchmarks/issue-182/kimdel-inference/goal.md` | `docs/benchmarks/issue-182/kimdel-inference/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6412409939) |
| H182-012 | `docs/benchmarks/issue-182/kimdel-inference-32k/goal.md` | goal内の表のみ | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013) |
| H182-013 | `docs/benchmarks/issue-182/kimdel-inference-32k-4096/goal.md` | `docs/benchmarks/issue-182/kimdel-inference-32k-4096/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013) |
| H182-014 | `docs/benchmarks/issue-182/kimdel-inference-gemma-ministral/goal.md` | `docs/benchmarks/issue-182/kimdel-inference-gemma-ministral/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741013) |
| H182-015 | `docs/benchmarks/issue-182/kimdel-inference-gemma-native/goal.md` | `docs/benchmarks/issue-182/kimdel-inference-gemma-native/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741220) |
| H182-016 | `docs/benchmarks/issue-182/kimdel-inference-qwen-native/goal.md` | `docs/benchmarks/issue-182/kimdel-inference-qwen-native/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741220) |
| H182-017 | `docs/benchmarks/issue-182/spark/goal.md` | `docs/benchmarks/issue-182/spark/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420) |
| H182-018 | `docs/benchmarks/issue-182/spark/prompt-stages-goal.md` | `docs/benchmarks/issue-182/spark/prompt-stages-measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420) |
| H182-019 | `docs/benchmarks/issue-182/spark/thinking-log-goal.md` | `docs/benchmarks/issue-182/spark/thinking-log-measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741420) |
| H182-020 | `docs/benchmarks/issue-182/spark/repetition-goal.md` | `docs/benchmarks/issue-182/spark/repetition-measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677) |
| H182-021 | `docs/benchmarks/issue-182/spark/label-clarity-goal.md` | `docs/benchmarks/issue-182/spark/label-clarity-measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677) |
| H182-022 | `docs/benchmarks/issue-182/qwen-repetition/goal.md` | `docs/benchmarks/issue-182/qwen-repetition/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6430741677) |
| H182-023 | `docs/benchmarks/issue-182/minicpm5/goal.md` | `docs/benchmarks/issue-182/minicpm5/measurements.json` | [元報告](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6431804563) |
| H182-024 | `docs/benchmarks/issue-182/qwen-japanese/goal.md` | `docs/benchmarks/issue-182/qwen-japanese/measurements.json` | 本集約でIssue初掲載 |

旧private原本がない実験は保存済み集計からの再整理であり、旧wireの再監査ではない。本Markdown・隣接JSONを専用作業ツリーへ保存。生の会話、Thinking本文、実token IDs、個人絶対パス、秘密情報は公開版から除外。元報告・失敗履歴は変更していない。

## GitHub掲載

[第1部](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6435831833) / [第2部](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6435832208) / [第3部](https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6435832558)

第1部は条件・ケース結果・資源・要点、第2部と第3部は全フェーズの詳細計測表。投稿後に本文とIssue番号を再取得して照合済み。
