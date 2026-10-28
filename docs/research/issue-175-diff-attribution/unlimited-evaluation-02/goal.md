# 理由分類・rename対応と無制限履歴取得量の実測

## Goal

エラーと曖昧な内容対応を分け、renameを独立した照合ケースにする。以前4MiBで停止した実履歴を、研究用に取得時間・累積受信量の上限を外して測り、受信量と照合用の保持量を区別する。既存の研究用Go PoCだけを変更し、Collector/Planner/LLMには接続しない。

## Result / Decision

分類とrename照合を実装し、偽App Server・一時Gitで回帰検証を通した。実データでは現在worktree1件と、前回選定した10ファイルの共有1回取得を実施。旧予算を超える履歴を読み切ることは確認できたが、一般的な入力分布・精度改善・本番採用の根拠にはしない。

- 現在worktreeの`.gitignore`: 42,209,601 bytes（42.21MB）受信、443Turn / 15,072item。取得はcomplete、対象操作0件。対応根拠なしのunmatched。Git入力の前後不変確認は通過。
- 過去10ファイル: 7,006,978 bytes（7.01MB）受信、80Turn / 3,088item。対象操作22件・保持Turn5件、照合用の操作/依頼/応答JSONは56,603 bytes（56.60KB）。安全フィルタによる除外1件が残りpartial。
- ファイル別保持JSONは2,959〜21,968 bytes、中央値6,786.5 bytes。10ファイルは共有取得の内訳で、独立した10回の受信量測定ではない。
- 既存10件のGit差分に更新履歴を組み合わせた診断比較は9件ambiguous、1件入力拒否。partialをmatchedへ昇格させない。独立Goldがないため正確率は算出しない。

## Implemented contract

取得失敗の`StageError`は`category`, `reason`, `stage`, `code`を保持する。比較結果は`status`, 主理由`reason`, 全理由`reasons[]`を持つ。例えば部分一致と履歴不足が同時にある場合、両方を残す。history API失敗はトップレベル`errors[]`にも保存し、ブロックの`history_incomplete`と混同しない。

| 分類 | reason | 実装した判定 |
| --- | --- | --- |
| error | `git_snapshot_unavailable` | Git差分・HEAD/index取得や検証失敗 |
| error | `snapshot_changed` | 固定前後でHEAD/index/選択ファイルの状態が変化 |
| error | `history_unavailable` | API・通信・通常モードの取得予算超過 |
| error | `history_parse_failed` | 履歴JSON/schema/diffの形式不正・欠落 |
| error | `unsupported_change_type` | Gitバイナリ等の未対応形式 |
| ambiguous | `multiple_history_candidates` | 内容完全一致の履歴操作が複数 |
| ambiguous | `partial_match` | 方向付き行の部分対応 |
| ambiguous | `overlapping_candidates` | 同一ブロックの複数位置対応 |
| ambiguous | `history_superseded_or_rewritten` | 同一Turnのitem順付きA→B→Cと現在A→Cの対応根拠 |
| ambiguous | `history_incomplete` | 未読領域や対象情報のフィルタ除外 |
| ambiguous | `split_merge_mapping_uncertain` | 複数pathにまたがる非重複の完全包含断片があり、対応範囲を一意化できない |
| ambiguous | `rename_multiple_candidates` | 明示した旧新パスまたは同一内容のmove操作が競合 |
| ambiguous | `rename_unresolved` | renameの内容/対応先を一意化できない |

分割・統合の検出は選択したpath内の連続した追加/削除断片だけに限定する。copy等でも同じ根拠は成立するので分割・統合自体を断定しない。上書きは同一Turnの順序根拠がある場合に限定し、Turnをまたぐ編集順・非連続な断片・単なる不一致から原因を推測しない。観測根拠がなければunmatchedやpartialのまま残す。

renameはGitの旧新パスとApp Serverの`move_path`を保存する。純粋移動と編集付き移動を照合し、一意ならmatched。言語構文のパースを必須にしないため、ソースが構文エラーでも内容ベースで続行できる。壊れたpatchを無検証で完全一致にはしない。Gitがrenameとして検出した両端の明示選択が必要で、rename連鎖や未追跡宛先・低類似度の全ケースを網羅しない。

## Acquisition conditions

通常PoCの共有5秒・受信4MiB・固定JSON8MiBは維持した。研究用`--unlimited-history`で履歴deadline・累積受信量・固定JSON上限を外す。RPCの1行サイズにもScanner上限を残さない。Git初期取得/不変確認の5秒、Git出力/個別ファイル4MiB、安全path・機密除外は維持する。今回は履歴側の予算の診断であり、すべてのI/O制約を撤去した測定ではない。

現在worktreeは通常と同じHEAD時刻からのTurn時間窓。10ファイルは前回R01–R10のpathを固定し、それらのGit親commitの最古時刻から取得開始時刻までの約76.6時間を対象にした。両者は時間窓も対象worktreeも異なるため、受信量や速度を同条件のA/Bとは扱わない。候補cwd集合はいずれも同一リポジトリの10worktree、走査threadは87。

## Data volume

ファイル別保持JSONには当該pathの操作、対応Turnの依頼・最終応答、historyメタデータを含む。同じTurnの依頼/応答は複数行に重複するため、この列を合計して共有保持量にはしない。受信量には対象外itemの本文も含まれ、LLMへの入力サイズやトークン数ではない。

| ファイル | 操作 | 対応Turn | ファイル別履歴JSON KB（10進） |
| --- | ---: | ---: | ---: |
| docs/benchmarks/issue-182/follow-up-plan.md | 1 | 1 | 6.89 |
| docs/benchmarks/issue-182/ministral-official/goal.md | 6 | 1 | 17.73 |
| docs/benchmarks/issue-182/ministral-official/raw/README.md | 1 | 1 | 6.68 |
| docs/benchmarks/issue-182/prompt-candidate.md | 1 | 1 | 7.85 |
| docs/benchmarks/issue-182/prompt-comparison-goal.md | 1 | 1 | 4.37 |
| docs/benchmarks/issue-182/prompt-comparison-preflight.md | 1 | 1 | 5.27 |
| docs/benchmarks/issue-182/suite-v2/README.md | 3 | 2 | 8.48 |
| docs/benchmarks/issue-182/suite-v2/raw/README.md | 1 | 1 | 2.96 |
| docs/benchmarks/issue-182/suite-v2/results-report.md | 6 | 3 | 21.97 |
| tools/benchmark182/mistral_audit.py | 1 | 1 | 5.83 |

## Time / memory observations

- 現在worktree: 履歴取得22,572.500ms、RPC1,142回、最大1応答1,156,308 bytes、外部timeの全体23.61秒、maximum RSS151,568,384 bytes。
- 10ファイル共有: 履歴取得10,908.729ms、RPC332回、最大1応答213,728 bytes、外部timeの全体12.43秒、maximum RSS163,397,632 bytes。

二つの診断取得は実行時間が一部重なる。さらに後者の外部timeは`go test`ドライバを含む。経過時間・RSSは観測値として残すが、単独実行の性能比較やPoC/App Server個別のpeak、Mac全体のメモリ圧迫を表さない。次の取得予算をこの時間だけから決めない。

## Real-data classification check

前回R01–R10の保存Git差分を維持し、10ファイル共有取得からpath別履歴を切り出した。取得状態partialは全件へ伝搬する。入力hash・最終比較バイナリhash・source hashを実行前に私有planへ固定。

- R01/R02/R05/R07/R08/R09: `partial_match`と`history_incomplete`を併記。
- R03/R04/R06: 完全一致候補はあるが`history_incomplete`としてambiguousを保持。
- R10: `input.schema/sensitive_content`として入力拒否。以前の`unsafe_or_duplicate_file`から、拒否箇所が機密検出であることを明確化した。真の秘密か検出false positiveかは未判定で、値は出力しない。
- 現在worktree: 履歴取得completeでも対応操作0件のため`unmatched: no_content_evidence`。完全取得は対応根拠の発見を保証しない。

実rename・実分割統合の独立Goldによる精度は未評価。今回の該当分岐は一時Git/人工履歴による回帰テスト。実データ比較も同一リポジトリで前回選定済みの標本であり、一般的な精度・保持量の分布へ外挿しない。

## Correction record

初回無制限取得は42,178,274 bytes・約19.83秒で完走したが、対象外の別worktreeのpathを欠落として数える不具合でpartial/除外297件とした。コードを修正し、対象外pathがcoverageを汚さない回帰テストを追加。別名のv3入力で再取得しcomplete/除外0件を確認した。最初の入力・バイナリ・timeログは削除や上書きをしていない。生履歴は取得中に増え得るため受信bytesが完全一致するとはしない。

## Validation

`go test ./tools/attribution175go ./internal/codexcontext ./internal/planning`、`go vet ./tools/attribution175go`、研究用コマンドbuildが成功。新規検証では4MiB超の正当なRPC1応答を無制限モードで読めること、通常モードでは拒否すること、8MiB超の固定JSONの保存/読取、分類併記、Git変更検出、バイナリ、rename純移動/編集/競合/未解決、ソース構文不正時の内容照合、順序根拠の有無、分割統合の非重複断片、対象外pathの除外を確認した。

## Artifacts / Next gate

公開可能な集計は本goalとsummary.json。私有原本は`<PRIVATE_TMP>/issue182-attribution-unlimited-02/`に保存し、capture入力、10path固定plan、history、ファイル別集計、timeログ、比較入力/結果/source hashを保持した。生diff・発話・応答・内部ID・ホームpath・機密候補値は公開用成果物へ含めない。旧検証の原本も保持。外部送信・LLM呼び出しは行っていない。

今回の標本では受信7.01〜42.21MBに対し、保持は最大でも共有56.60KBだった。今後の取得改善は受信量と照合保持量を別に設計する根拠が得られた。一般的な入力サイズの上限を確定したものではなく、本番予算の採用とLLM投影は未実施。
