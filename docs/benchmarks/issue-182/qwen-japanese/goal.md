# H182-024 Qwen3.5-4B Text日本語指示の対比較

## Final Result / Decision

承認した8条件の処理を終了。旧4条件＋新2条件の6条件を実行、新2条件は対応するON旧指示がTextへ到達しなかったため未成立としてskip。Textの有効な対比較はOFF grammarあり/なしの2組のみ。全8条件成功ではない。初回停止216.871秒＋続行422.187秒、計測区間合計639.058秒（人間への確認待ち時間を除く）。実モデル呼び出し12回、再利用G/Cは生成回数・メトリクスへ二重計上しない。

OFFの両組で旧/新とも3summary中3件が日本語（仮名/漢字を含む）だった。英語summaryが日本語へ改善した例は今回得られていない。日本語指示の効果なしと断定するのでもなく、違反していたON条件のText比較が欠けており主仮説は未評価。新指示の採用は保留。追加推論/再試行/予算変更なし。

| Thinking | grammar | 旧指示 | 新指示 | 判定 |
| --- | --- | --- | --- | --- |
| ON | あり | Grouping3/3、Category4096上限、Textなし | skip | 対比較不可 |
| ON | なし | Grouping3/3、Category4096上限、Textなし | skip | 対比較不可 |
| OFF | なし | 日本語3/3、scope禁止語3件、validator拒否 | 日本語3/3、scope禁止語1件、validator拒否 | 日本語維持、scope部分変化 |
| OFF | あり | 日本語3/3、scope禁止語1件、validator拒否 | 日本語3/3、scope禁止語1件、validator拒否 | 日本語維持、最終受理変化なし |

OFF grammarなしのscopeは旧docs/feat/fix→新docs/ui/mailで、UI/メールは型名からcomponentへ変わったがREADMEのdocs違反が残る。grammarありは旧新ともdocs/ui/mail。新指示側の最終JSONは両grammar条件で同じ内容となった。新summaryは変更の主旨を記述するが「山括括」など重複した表記が残る。旧にも同種の重複表記があり、自然さを全正解とは扱わない。CategoryはOFFなし3/3、あり2/3の実測をそのまま固定し、新Textで修正しない。

## Text pair timing

全4Textは自然完了、JSON/Schema適合。Thinking OFFのためThinking数0。scope禁止語はSchemaの長さ/型では表せずhost validatorで拒否。各1回で背景負荷も異なるため速度改善の因果は未確定。旧全pipeline wallと新Textのみwallは比較しない。

| 条件 | arm | tokens | prefill秒 | decode秒 | token/s | helper wall秒 |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| off-none | old | 83 | 16.758 | 3.054 | 27.18 | 21.975 |
| off-none | new | 82 | 16.592 | 2.987 | 27.45 | 21.348 |
| off-json_schema | old | 68 | 20.065 | 3.507 | 19.39 | 26.820 |
| off-json_schema | new | 82 | 18.858 | 4.023 | 20.38 | 25.056 |

## Final audit / resources

2組ともraw Text request全体、effective user、Schemaは完全一致。system差は日本語明示の3文追記だけであることを保存wireから再確認。全固定source/model/fixture hash不変、Git不変。原本・初回の失敗は保持。既存SwiftではThinking本文を保存していないため、ONについて本文ベースの反復分析はできない。nativeThoughtTokens=4097はsampling/先読みカウンタであり、emitted4096より多い実出力とは解釈しない。native KV実配列dtype/bytesは内部cache未exportのため未取得。

OS monitor初回109＋続行211=320件すべてvalid、fatalなし。終了後runner/helper残留なし。CPU/GPUはsystem全体、RSSとMLX peakは別指標で加算しない。温度実数・電力・プロセス別GPUは未取得。約2秒監視で瞬間peakを保証しない。

| 区間 | CPU平均% | GPU平均% | helper RSS最大MiB | swap開始/終了/最大MiB | thermal |
| --- | ---: | ---: | ---: | --- | --- |
| execution | 12.34 | 90.39 | 1464.56 | [0.0, 377.88, 377.88] | {'nominal': 109} |
| continuation | 12.76 | 89.11 | 3123.05 | [361.88, 551.62, 551.62] | {'nominal': 128, 'fair': 83} |

同時helper最大1。swapはsystem全体の増加であり本測定だけの寄与は未分離。thermalはnominal/fair、serious/critical観測なし。CLI既定・本番採用・commit/push/Issue投稿は変更なし。一時監視は最終報告後に削除する。

---

以下は準備・初回停止・続行時点の履歴。最終状態は上記の6実行/2skipを優先する。


## Goal / Contract

ユーザー承認により旧指示4＋新指示4の8条件。Thinking ON/OFF×JSON grammarあり/なしの4組。同じ再取得固定kimdel5ファイル・3目的・3履歴contextを使い、Textへの日本語明示の効果を検証。旧private原本が失われたため、H182-016の過去保存結果を対照に流用せず、新しい対照群を実行する。

Qwen3.5-4B-4bit revision0e7ffd5c629ef7719d4cbc04069232580bfa9d9c、同一Swift Release helper、native KV、入力32768/出力4096（Thinking込み）、temperature.6/top_p.95/top_k20、repetition1、seed144。時間制限・Thinking専用予算・強制終了なし。warmup/retry/repairなし。

## Pair isolation

各組で旧指示の実Grouping→Category→Textを生成する。新指示側は、その組のGrouping/Category応答を再利用し、Textのみ実推論。正解partition/Categoryは注入しない。最大16実モデル呼び出し（旧12＋新Text4）、host上は8条件×最大3phase。新側のGrouping/Categoryのメトリクスは対照応答の再利用なので二重計上しない。日本語指示の時間比較はTextに限定し、旧側全pipeline時間と新側再利用込み時間を同等仕事として比較しない。

対になるraw Text request全体の同一性、effective user messagesとSchemaの同一性、system差が追記だけであることをassert。基準群がText前に止まった場合や入力/ソース/監視不一致は全体停止し、条件を成功させるための再試行・修復をしない。

## System delta

旧指示側は現行Go systemへ保存仕様にあるdocs定義・scope禁止語・文字数説明を補い、共通Schema説明を付け、自動構造例を除く再構成。失われたH182-016 wireとのbyte一致は未検証。今回の旧/新間は同一基準を使用する。

新Text systemだけに以下を追記する。summary_language=ja、JSON schema、その他指示、Thinking制御は両群で不変。

```text
各グループのsummaryは、必ず自然な日本語の1文で記述してください。
技術用語・識別子・ファイル名は原表記を使用できますが、summary全体を英語で記述しないでください。
この言語指定はsummaryに適用します。JSONのキー、ID、type、scopeは、それぞれの指定形式を維持してください。
```

## Verification / Measurement

非推論テストでText systemのみの差、Grouping/Category無変更、user/schema/予算/入力オブジェクト保持を検証し成功。モデルは固定revisionを復元。既存の32K/4096/ファイル数制限撤廃/時間制限なしGo overlayとRelease helperを再利用。約2秒間隔でCPU/GPU/memory/swap/thermalを監視。Swift helper RSS12GiB・監視失敗で停止。温度実数・電力・プロセス別GPUは未取得。

Text生成量/Thinking数/prefill/decode/phase時間、JSON/Schema/validator、日本語指示遵守、scope/内容の維持を比較する。日本語validatorは仮名/漢字の存在を調べる簡易検査なので、その受理だけで自然な日本語と判定しない。各条件1回・固定順であり速度差の一般化はしない。

## Status

測定停止。8条件のうち最初の旧指示1条件のみ実行され、2phaseで停止した。残り7条件は未実行。日本語system指示の効果比較は未成立。

Groupingはcompleted、生成527/Thinking raw474、正解3/3。Categoryは生成4096上限（max_tokens）で最終JSONが得られずText未到達。nativeThoughtTokens=4097はsampling側の先読みを含むカウンタであり、4097を実出力Thinking数とは扱わない。Category decode154.945秒。基準群がTextへ到達しなければ全体停止する事前ゲートを適用し、分類結果の捏造・正解注入・上限変更・再試行はしなかった。

全体216.871秒、監視109件すべてvalid、fatalなし。Git不変、終了後runner/生成helper残留なし。completion.state=stoppedであり、8条件完了ではない。既存採点器のKeyErrorはCategory応答がないための二次的採点不能。根本の停止理由はCategory生成の上限到達。

一時監視追加の承認を得た後の確認時には停止していたため、監視を新規作成しなかった。原本保持。生の履歴・入力・出力はprivate保存。通常CLI設定変更、commit/push、Issueへの投稿は未実施。


## Independent-pair continuation

初回の全体停止はランナーの過剰な停止設計であり、独立する残り3組まで停止する必要はなかった。ユーザー承認済み8条件の未実行範囲から、ON/grammarなし、OFF/grammarなし、OFF/grammarありの3組6条件を同じ固定入力・設定で続行した。初回のON/grammarあり対照は再試行せず、対応する新指示側は比較不能として保持。

原runner・失敗ログ・completionは上書きせず、別continuation保存先を使用。元contractの全source/model/fixture hash不変を確認。変更は未実行3組の選択と、基準がTextに到達しない場合に対応する新armだけをskipして独立組を続行する制御。予算・sampling・system・Schema・KVは変更なし。監視/実行基盤異常・snapshot/source/input不一致の全体停止は維持する。全8条件の成功完了とせず、初回失敗・対応不可1条件・続行6条件を合わせて報告する。
