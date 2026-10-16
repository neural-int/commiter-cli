# Issue #175: 2ケースの初回比較

## 要約

ケース1では、補助情報なしのCategoryは `refactor`、ありでは参照どおりの `fix` となった。ただし補助情報ありはTextで共有120秒の期限に達し、最終planを得られなかった。ケース2は両条件とも入力準備後の圧縮によりモデル呼び出し前に拒否された。今回の観測は分類への追加証拠を示す一例であり、最終planの意味品質改善、feat識別、一般化、production採用を立証しない。

## 実行条件

- 指定の `neural-int/goals/goal.md` とIssue #175を取得し、既存専用worktreeで実行した。mainは変更していない。agent skillsは使用していない。
- 2ケース×A（補助情報なし）/B（あり）×1回。最大12呼び出しの枠内で、実際の生成要求は6回。再試行・repair・追加の採点モデル呼び出しは0回。
- Gemma 4 E4B 4bit、revision `475b9088d29754a3379866cf5aeb6b41acd313c2`、既存MLX helper。temperature=0、seed=144、出力枠768/512/768、共有120秒/cycle。helperの起動はproduction同様に各呼び出しで行った。
- 既存のGit収集、構文解析、relation抽出、入力準備、Grouping projection、ThreePhaseGenerator、最終Validatorを使用した。AにもBと同じ補助情報の扱いの指示を追加する評価用adapterを通した。productionの分類・生成ロジックは変更していない。
- `execution-contract.json` に入力・コード・Gold参照・runtimeのハッシュを推論前に固定した。Goldはユーザー指定の2ケースに対するagent作成の探索的参照であり、独立した人によるGold評価としては扱わない。
- ケース1は過去の最終blobにログの編集を逆適用してbaseを復元した。周辺の未変更内容は保持し、変更行はログに基づく。ケース2は実コミットの選択4パスの前後状態。完全な歴史上のCLI起動状態の再現ではない。
- 初回のfixture準備、既存tokenizer helperのMetal resource欠落、16kのみの受付という事前確認時の失敗は保持した。別runtimeに既存resourceを配置して修復し、同じtext/profileを16k envelopeでtokenizeし、実際のgeneration ceilingと比較した。tokenizer helperの当時のソースは現存せず、binary hashと同一dependency lockを記録した。実生成helperでも各要求のtoken budgetは再検証される。

## 検証結果

| ケース | 条件 | 生成要求 | Grouping | Category | Text | 最終Validator |
| --- | --- | ---: | --- | --- | --- | --- |
| C01 画像404修正 | A: 補助情報なし | 3 | 3ファイル1グループ、参照一致 | refactor、参照fixと不一致 | 「画像表示のためのプログレッシブ画像コンポーネントを実装」。URL正規化と既存不具合修正の意味を満たさない | 受理 |
| C01 画像404修正 | B: 補助情報あり | 3 | 3ファイル1グループ、参照一致 | fix、参照一致 | timeout。意味は未評価 | 未受理 |
| C02 一時index構造検証 | A | 0 | 未実行 | 未実行 | 未実行 | 未到達 |
| C02 一時index構造検証 | B | 0 | 未実行 | 未実行 | 未実行 | 未到達 |

C01-Aの各phaseは約31.01 / 36.97 / 22.56秒、C01-Bは約48.17 / 65.72 / 6.35秒であった。正確な値は `runs/*/*-response.json` にある。BのTextは新たに120秒を与えられるのではなく、共有cycleの残時間で停止した。モデル出力は取得できていない。

C01の事前token数はGroupingでA=1,590/B=2,047、Categoryで7,951/8,408、Textで7,873/8,330。capture時の全phaseでsystem・schema・context枠は一致し、ユーザーpayloadはcodex_contextだけが異なることを確認した。後続phaseのcapture用応答は機械的な1グループ/fixであり、モデル結果やGoldの成功件数へ含めていない。実際のText入力にはモデルが確定したA=refactor/B=fixが入る。

C02の元のpromptは既存byte予算上の1,047,993 bytesであった。大きな新規Goファイルから3,375件・3,337件の構文evidenceが出ていた。既存Prepareはsummaryを5回、strong compressionとevidence reductionを1回行い、64kに収めた。その後ThreePhaseGeneratorが `provisional three-phase planner does not support compressed input` として両条件を拒否した。これはログ追加前の入力受理の問題であり、意味評価の誤答として採点しない。差分やGoldを縮小して通す操作はしていない。

完了した5つのnative helper processの最大RSSは最大3,007,201,280 bytes（約2.80 GiB）。ホスト物理メモリは `resource-environment.json` に記録した。これはhelper processのresident量で、GPU/Metal peakやシステム全体のmemory pressureではない。timeoutしたTextのnative計測は取得できず、backendのtimeout記録を根拠とする。終了後にhelper/wrapperの残存processは検出されなかった。

両fixtureのHEAD・index canonical identity・working-tree bytesは実行前後で一致し、推論前に固定したコード/入力のハッシュも一致した。`integrity-audit.json`に確認結果を保存した。

## 考察

C01ではGroupingが同じままtypeが参照へ変わったため、依頼・最終応答が既存表示の修正という目的を解釈する追加情報となった可能性がある。ただし各条件1回の観測であり、安定性や改善率は推定しない。

最終Validatorは所属やschema等の契約を確認するが、修正の目的を十分に説明していることまでは保証しない。C01-Aの受理と意味不一致を分けて保存した。BはCategoryまでの成功を最終planの成功へ数えない。

Bの時間増加には入力増加と生成内容の違いの双方が関係し得る。phaseごとの追加反復やprefill/generation別の計測をしていないため、原因を断定しない。C02もnative token数による上限超過を立証したものではなく、既存の保守的byte予算と圧縮非対応というhost側の制約を観測した。

## Next Steps

- C02の圧縮非対応とC01-Bの共有期限を、ログの取得・関連付けとは別の入力受理/実行制約として扱う。現行条件では2ケースの最終意味評価を完了できなかった根拠があるため。
- この初回結果と入力を保持し、追加呼び出しを行う場合は新しい比較条件と停止枝を先に固定する。残りの呼び出し枠を自動的に再試行へ振り替えないため。
- C02を同じfixtureのまま実行可能にするにはPlanner側の制約変更が必要となる。Issue #175の全面再設計の対象外条件と照合し、別Issueで改善するか、次のログ評価用に小さな別ケースを追加するかを決める。今回のケースを結果の後で差し替えて成功扱いにしないため。
- Groupingの誤結合・途中取り消しの誤帰属・複数目的に対する実モデルの意味評価は未完了として残す。この2ケースだけでは網羅できないため。

## 開発用チェックとGoal監査

`go test ./internal/planning ./internal/cli`、同パッケージの `go vet`、`go build ./cmd/commiter`（出力は一時ディレクトリ）、Python4スクリプトのcompile、`git diff --check` は成功した。opt-in runnerのGo test成功は処理の完了を示すもので、各ケースの意味品質成功を示さない。

`completion-audit.json` のとおり、Issue全体のVerificationは未完了であり、Goalの最終品質ゲート達成やIssue完了は宣言しない。既存のチェックボックスを根拠なく完了へ変更しない。
