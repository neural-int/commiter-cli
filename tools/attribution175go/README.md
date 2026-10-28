# Go製 Attribution PoC

Issue #182の計画に基づく研究用コマンド。Issue #183で取得・照合エンジンを`internal/historyattribution`へ共通化し、通常CLIの3フェーズ経路にも接続した。このコマンドのcapture/compare入口は維持する。既存Python人工照合PoCとは別実装。追加依存はなし（標準ライブラリと既存の機密値検出を利用）。

## 採用した取得方式

研究用captureコマンドの既定は`pipeline100x4`（最大4本文worker・一覧中本文取得・接続再利用）。100件ページ、queue16、重複/競合検出、段階的メモリ保護、1応答300MB上限を維持する。配布版App Serverを使い、診断用ビルドや詳細プロファイルは既定で使わない。低レベルのfixture用capture helperは逐次baselineを維持する。

逐次方式を明示する場合は`--history-fetch paged100`。単一接続用の`--history-timing-log`もこの逐次指定と併用する。以下に残る実験記録の「既定はpaged100」は当時の条件であり、現在のcapture既定は本節のとおり。

## コマンド

リポジトリルートからbuildする。

```sh
go build -o <PRIVATE_TMP>/commiter-attribution-poc ./tools/attribution175go
```

実データ採取を行う場合は、対象rootと変更ファイルを明示する。`--path`は繰り返せる。出力先の親ディレクトリは事前に存在し、対象worktreeの外である必要がある。

```sh
<PRIVATE_TMP>/commiter-attribution-poc capture \
  --root /absolute/path/to/repository \
  --path src/example.go --path src/example_test.go \
  --out <PRIVATE_TMP>/attribution-input.json

<PRIVATE_TMP>/commiter-attribution-poc compare \
  --input <PRIVATE_TMP>/attribution-input.json \
  --out <PRIVATE_TMP>/attribution-comparison.json
```

既存出力は上書きしない。JSONは0600で新規保存する。固定入力には実diff・履歴本文・ローカルroot・内部参照IDが含まれ得るため、そのままGitHubへ公開しない。既知の機密値は検出時に拒否/除外するが、任意の機密/個人情報の完全検出は保証しない。

## capture

- 明示した安全なファイルだけを取得する。HEAD-to-worktreeの最終diffはstaged/unstagedをまとめた結果。untrackedは明示対象の生内容を取得する。未選択ファイルや秘密候補を一括収集しない。
- `.env*`、鍵・credential・secret候補、`.git`、相対逸脱、symlinkを拒否する。Git外部diff/textconvは使わない。
- HEAD、index hash、選択ファイルのdiff・内容hash・modeを取得前後で照合する。他のプロセスによる選択状態の変更を検出したら固定入力を不成立とする。全履歴時点での原子性や未選択ファイルの不変を保証するものではない。
- 履歴は保存済みApp Server APIだけを使用。thread/list→turns/list→items/listをcursorで取得する。通常/archivedを対象に、候補cwdを現在のworktreeに記録された会話へ限定し、実編集pathも同じworktree内に解決する。別worktreeからの編集は今回の取得範囲外とする。
- 現行契約と同様にHEAD commit時刻から採取開始時刻までに開始したTurnを対象とする。HEAD以前から継続するTurnの後半編集は取り逃がす可能性があり、今回この時間窓を変更しない。
- 完了済み編集のdiff、thread/turn/item/changes index、対応依頼と任意の最終応答を固定する。空/欠落diffと未対応kindは比較時に区別する。依頼がない/unsafeなTurnや機密diffの省略は記録し、取得を完全扱いしない。
- 通常モードでは採取用の共有5秒deadline、履歴の累積受信4MiBを維持。Git出力・個別ファイルにも4MiB制限、保存/読取する固定入力JSONに8MiB制限。終了後のGit不変確認のみ別の最大5秒を使うため、採取コマンド全体が必ず5秒以内とはしない。
- 取得量のthreads/turns/items、受信bytes、対象期間、採取経過msを記録する。対象外Turnや読み取り不能範囲を含む全履歴coverage率ではない。
- 本PoCはLLMへ投影しない。既存の補助JSON 8KiB枠は変更せず、その投影は後続Gateで扱う。

## compare

同じ入力で、各Git変更ブロックに次を出す。

- `path_baseline`：完了済み・同一pathの操作候補。現行方式の対応候補を比較するための一覧で、現行Plannerを実行した結果ではない。
- `matched`：追加/削除行、方向、順序、重複、EOFを含めて完全一致するブロックが唯一の操作に対応し、競合/欠落や取得不足がない。
- `ambiguous`：部分一致、複数操作、同一内容の繰り返し位置、rename未解決、取得途中の完全一致など。同一Turnでも別操作は統合しない。`single_turn`は候補が同一Turnにまとまるかの補助情報。
- `unmatched`：取得済み範囲に内容の対応根拠がない。
- `error`：履歴diffが壊れている、Git変更形式が未対応など。`reason`と`reasons[]`に分類を保持する。取得失敗はトップレベル`errors[]`にstage/code付きで保持し、各ブロックの`history_incomplete`と区別する。

ブロックはhunk内の連続した追加/削除行。context行で区切り、行番号・contextだけでは照合しない。hunkの行数を検証し、行内容の空白・改行欠落を勝手に正規化しない。add/deleteの生内容は先頭`-`等をdiff構文と誤認しない。renameは`old_path`と新パス`path`を保持し、Git検出の両端を選択して固定する。Gitの100% renameと履歴の空diff＋明示moveは純粋移動として照合し、編集付きrenameは方向付き変更ブロックを比較する。旧新パス・内容が一意なら`matched`、競合は`rename_multiple_candidates`、読取不能・内容不一致・候補なしは`rename_unresolved`とする。言語構文解析を前提にしないため、ソースが構文エラーでも内容一致で続行できる。破損したpatchを行数検証なしで一致へ昇格させることはしない。両端を選んでいない既知renameは`git.paths/rename_pair_required`で採取条件の不成立を示す。未追跡宛先へのmoveやGitがrenameと認識しない低類似度変更を網羅するものではない。

同じoperation IDのページ重複は1件とするが、異なる内容で同一IDを名乗る固定入力は拒否する。複数位置で同じブロックがある場合も位置を捏造しない。A→B→C合成、取消しの相殺、Patch Replay、Tree-sitterは実装しない。同一Turnのitem昇順を`item_order`に保持し、A→B、B→Cがあり現在がA→Cと一致する場合だけ、`history_superseded_or_rewritten`の観測根拠を付記する。Turnをまたぐ順序や単なる不一致からは推定しない。

`matched`は内容対応の成立であり、因果や最終残存の証明ではない。履歴がpartial/unavailableなら結果に`git_diff_only_fallback=true`を付け、matchedへ昇格させない。既存Plannerのfallbackを変更したという意味ではなく、診断用の状態表示。

## エラー契約

stderrは本文を含まないJSONで、例は次のとおり。

```json
{"error":{"stage":"history.items","code":"receive_limit","category":"error","reason":"history_unavailable"}}
```

| stage | 代表的なcode | 保存/継続 |
| --- | --- | --- |
| `git.root` / `git.paths` | `command_failed`, `unsafe_path`, `invalid_or_duplicate_path` | 固定入力不成立。可能なら失敗情報のみ保存 |
| `git.head` / `git.index` / `git.diff` / `git.untracked` | `command_failed`, `read_failed`, `sensitive_content`, `no_selected_changes` | 固定入力不成立 |
| `git.time_window` / `git.worktrees` | `invalid_timestamp`, `command_failed` | 対象期間/候補範囲を固定できず失敗 |
| `history.start` / `history.initialize` | `command_failed`, `rpc_error` | Git入力は保持し、history unavailableを保存 |
| `history.threads` / `history.turns` / `history.items` | `rpc_error`, `invalid_schema`, `receive_limit`, `deadline_exceeded`, `repeated_cursor` | 完全取得済みTurnだけ保持。partial/unavailableと失敗箇所を保存 |
| `history.filter` | `sensitive_evidence_omitted` | 機密候補を保存せずpartial |
| `git.verify` | `snapshot_changed`, `snapshot_unavailable` | 固定入力不成立。後者はcauseに元のstage/codeを保持 |
| `input.read` / `input.decode` / `input.schema` | `open_failed`, `size_limit`, `invalid_json`, `conflicting_duplicate_operation` | 比較失敗 |
| `output.path` / `output.write` / `output.encode` | `must_be_outside_target_worktree`, `already_exists`, `create_failed`, `write_failed`, `input_limit` | 保存失敗。既存ファイルは保持 |

成功はexit 0。Git/出力の失敗と履歴の部分取得はexit 1。履歴だけの失敗でもGit入力入りJSONは保存されるので、`history.status`と`history.error`を確認したうえでcompareできる。Git自体の失敗は失敗報告JSONでありcompare用入力ではない。RPCの生エラーメッセージ、command stderr、履歴本文を診断エラーへ混ぜない。

## 検証

```sh
go test ./internal/historyattribution ./tools/attribution175go ./internal/codexcontext ./internal/planning
go vet ./internal/historyattribution ./tools/attribution175go
```

テストは偽App Serverと一時Gitリポジトリを使い、実Codex履歴を読まない。完全/部分/反対方向/空白差/複数操作/同一Turn/別セッション/重複ページ/同一位置の曖昧性、raw add/delete、EOF、形式不正、秘密path、staged+unstaged+untracked、取得段階別RPC失敗、timeout、4MiB上限、取得中のGit変更、失敗時の入力保持と出力権限を確認する。

実履歴の採取と独立参照による精度/取り逃がし/誤帰属・時間/メモリの比較は、この実装後の別検証。現行Python人工結果や既存fixture/Goldを上書きしない。

## 理由の分類と研究用無制限モード

`category`は取得失敗の`error`と取得情報の欠落を表す`ambiguous`を区別する。低レベルの`stage`/`code`は残す。比較結果の`reason`は主理由、`reasons[]`は同時成立した全理由で、履歴不足によって複数候補・部分一致の事実を上書きしない。

| 分類 | reason | 判定条件 |
| --- | --- | --- |
| error | `git_snapshot_unavailable` | Gitの取得・検証に失敗 |
| error | `snapshot_changed` | 取得前後のHEAD/index/選択変更が不一致 |
| error | `history_unavailable` | API・通信・通常モード予算による取得失敗 |
| error | `history_parse_failed` | JSON/schema/履歴diffの形式不正・欠落 |
| error | `unsupported_change_type` | バイナリ等の未対応Git差分 |
| ambiguous | `multiple_history_candidates` | 同じ内容の完全一致操作が複数 |
| ambiguous | `partial_match` | 方向付き行の一部に対応根拠がある |
| ambiguous | `overlapping_candidates` | 同一ブロックがGitまたは履歴の複数位置にある |
| ambiguous | `history_superseded_or_rewritten` | 同一Turn内の明示順序付きA→B→Cに現在A→Cが対応 |
| ambiguous | `history_incomplete` | API未完走・対象情報の安全フィルタ除外 |
| ambiguous | `rename_multiple_candidates` / `rename_unresolved` | 上記rename規則 |

`split_merge_mapping_uncertain`は、明示選択した複数path内で、1履歴ブロックに2現在ファイルの追加/削除断片が非重複で完全包含される場合、または現在1ブロックに異なる2履歴pathの断片が非重複で完全包含される場合に付ける。因果的な分割・統合の断定ではなく、copy等も含む範囲対応の不確実性。EOF印はこの曖昧検出だけで除外し、matched判定では保持する。未選択の元/先pathや非連続・書換えを伴う分割統合は検出できない。単なる不一致から原因を推測しない。

今回の研究では`capture --unlimited-history`と`compare --unlimited-history`を使用できる。履歴のdeadlineと累積受信上限、および固定JSONの8MiB上限を外す。1行のRPC応答もReaderで読み切り、Scanner由来の暗黙上限を残さない。Git初期取得/不変確認の5秒・Git出力/個別ファイルの4MiB、安全path・機密除外、HEAD時刻からのTurn時間窓は維持する。無制限は履歴取得予算の診断用で、本番経路や通常PoCの予算変更ではない。

`history`には`unlimited_acquisition`, `received_bytes`, `max_message_bytes`, `rpc_requests`, `elapsed_ms`, `retained_json_bytes`を保存する。wire受信量には非対象itemの内容も含まれ、保持量は選択操作/依頼/応答のJSONを再エンコードしたbytes。両者をLLM入力トークン数と混同しない。外部`time -l`によるRSSは別ファイルへ保存する。

`TestMeasureUnlimitedLiveHistory`は明示opt-inの実データ計測専用。固定planにあるroot・roots・paths・期間で共有1回取得し、ファイルごとの保持量を集計する。通常テストではskipする。共有取得をファイル数分の独立測定と数えない。

## 確定した照合用フィールド限定スキーマ v2

`history-projection-v2.schema.json`を本研究の固定履歴形式とする。App Serverの実在する応答形式・要求パラメータではない。現在は受信後のローカル投影であり、サーバーからの受信量を削減しない。

| 対象 | 採用フィールド |
| --- | --- |
| 履歴全体 | schema_version=2, projection=attribution, threads, next_cursor, coverage |
| thread | thread_id, turns |
| Turn | turn_id, started_at, status, user_messages, file_changes, final_messages |
| ユーザー発話・最終応答 | item_id, item_order, text |
| 完了済み編集操作 | item_id, item_order, status, changes |
| 変更 | change_index, path, kind, move_path, diff |
| 完全性 | status, dropped, reasons, errors（category/reason/stage/code/cause） |

途中経過・計画・推論本文・コマンド実行情報・ツール引数/結果・Web検索・画像・UI付加情報は保持しない。現在のPoCは従来形式でもこれらを保存していなかった。新形式の主な変更は階層化と発話の出典ID保持であり、保持量の大幅な削減を前提としない。

`next_cursor`は固定入力ではnull。途中のAPIページには実際のcursorを使うが、今回のv2はページ応答そのものではなく取得終了時のスナップショット。nullだけで完全取得とは判断せず、coverageを必ず参照する。partial/unavailableはそのまま保存する。diffのnullと空文字は区別し、欠落を純粋renameへ変換しない。

発話textは従来のテキスト結合・末尾改行を維持する。最終応答は従来仕様と同じ最後のfinal_answerを保持する。複数のユーザー発話は個々のIDを残す。item_orderは同一Turn内で順序比較できる位置を保持し、投影後の短い配列の添字に振り直さない。既存の古い保存入力に発話IDやTurn情報がなければ捏造せずv2化を拒否する。取得量・経過時間・走査件数は測定ファイルへ分離し、照合用履歴には入れない。Gitの固定入力情報は外側に維持する。

```sh
<PRIVATE_TMP>/commiter-attribution-projected-v2 capture \
  --unlimited-history --project-history \
  --root /absolute/path/to/repository --path src/example.go \
  --out <PRIVATE_TMP>/projected-input.json

<PRIVATE_TMP>/commiter-attribution-projected-v2 compare-projected \
  --unlimited-history --input <PRIVATE_TMP>/projected-input.json \
  --out <PRIVATE_TMP>/projected-result.json
```

v2の読取は未知フィールドを拒否し、従来と同じ安全性検証・照合エンジンへ渡す。通常capture/compareと本番CLIの形式は変更しない。`TestEvaluateHistoryProjectionLive`は固定planを明示した研究用opt-in。前回の10pathと現在.gitignoreの同じ期間・cwd範囲で順番に取得し、同一取得応答から従来形式と新形式を作る。11ケースのGit差分は前回の固定入力を使い、decode/validate時間とcompare本体時間を分ける。各ケース1回ずつで、実行順は交互。速度改善の統計的主張には用いない。

## 承認済みの取得方式切り替え

研究用captureに`--history-fetch`を追加。現在worktreeに記録された会話だけを取得し、別worktree列挙は行わない。返却側に別cwdのthreadが混ざった場合も本文取得前に除外する。編集pathも対象worktree内だけを採用する。readHistoryに渡されたrootsも現在rootだけへ限定するため、古い研究planの複数worktree指定を使っても従来の広い範囲には戻らない。過去の実測原本は変更せず、新しい測定は範囲が異なる別実験とする。

| モード | thread一覧 | Turn一覧 | item一覧 |
| --- | ---: | --- | --- |
| paged20 | 20 | 本文なし20 | 各Turnを20ずつ |
| paged100（capture既定） | 100 | 本文なし100 | 各Turnを100ずつ |
| full5 | 100 | 本文込み5 | 個別item要求なし |

full5は`itemsView=full`の各Turnのitemsを同じ選別・安全性検証へ渡す。要約モードは使わない。itemsフィールド欠落はinvalid_schemaで停止し、完全取得や空履歴に偽装しない。明示的な空配列は有効。Turnページ・itemページのcursor検証と取得期間条件を維持する。境界ページでは対象期間外Turnの本文も返り得るため、全体受信量は実測する必要がある。ページサイズは要求値で、実サーバーの上限/返却件数は次の実測で確認する。

```sh
<PRIVATE_TMP>/commiter-attribution-fetch-modes capture \
  --history-fetch paged100 --unlimited-history --project-history \
  --root /absolute/path/to/worktree --path src/example.go \
  --out <PRIVATE_TMP>/attribution-paged100.json
```

比較時は`paged20`、`paged100`、`full5`だけを替え、出力先はそれぞれ新しいファイルとする。履歴期間・Git固定条件・機密検出・照合エンジンは変更しない。5秒/4MiB制限は通常時に維持し、研究用の無制限化は従来どおり明示フラグで指定する。非投影の履歴にはfetch_modeを保存する。v2最小履歴では計測情報を省くため、実行条件は別のplanに記録する。本番CLIには接続しない。

回帰検証は6Turn・各104itemを偽App Serverから返し、20/100件のitemページ境界と5TurnのTurnページ境界を通過させる。3方式で操作・依頼/応答・照合結果が同じこと、要求回数が減ること、別cwdの本文へアクセスしないこと、full本文の欠落時に失敗することを確認する。実履歴での速度・返却上限・メモリのA/Bは今回の実装作業には含めない。

## 取得内訳ログ

captureの`--history-timing-log <PRIVATE_TMP>/history-timing.json`で内訳計測を有効にする。既存ファイルやcapture出力と同じ宛先は拒否し、対象worktree外に0600で新規保存する。履歴本文・実path・内部ID・cursor・要求引数はログに入れない。

起動手続き、全体（終了処理込み）、item選別、Turn文脈検証、path検証、安全性検証、終了処理を記録する。要求ごとにメソッド・送信・待ち受信・外枠/result JSON解析・RPC全体・応答bytes・失敗有無を保存する。path/安全性は選別/文脈検証の子区間であり二重加算しない。initializeはサーバー起動の受付待ちも含むため、exec手続きの時間だけを起動全体と見なさない。応答待ち・受信はサーバー処理とローカル転送を分けない。ログへのJSON保存自体は計測終了後。

研究用速度probeでもCOMMITER_FETCH_TIMING=1を明示すると、セルごとのtiming JSONを保存できる。無指定なら保存しない。再現条件とメソッド別集計はfetch-timing-07のgoal参照。

## 500件要求・1応答300MB上限の研究モード

`--history-fetch paged500`はthread/Turn/itemの要求件数を500にする。履歴取得全体の累積量とは別に、JSONLの1メッセージ（末尾改行込み）を300,000,000bytesに制限する。研究用の無制限フラグを指定しても、このモードの1応答上限は維持する。固定長のreader断片を検査し、上限を超える断片を追記・JSON解析する前に`response_size_limit`で取得を止める。取得済みTurnだけを残し、未完了の履歴をcomplete/matchedにしない。入力bytesの上限であって、Goのバッファ拡張・コピー・App Serverを含むRAM全体の上限ではない。

ローカル実測では、item一覧を500件指定しても100件だけ返り、nextCursorが残った。100件指定と実返却が同じだったため、現在のローカル版では500指定による要求回数削減は得られなかった。現在の既定paged100は維持する。今後のバージョンでも同じ上限になると保証するものではない。

## 少数並行取得の研究モード

`--history-fetch parallel100`は、現在worktreeのthread/Turn一覧を1つのApp Serverで列挙し、そのプロセスを閉じた後、2つの独立したApp Server接続でTurnを取得する。thread/Turn/itemの要求ページは100件、各応答300MB上限。1つのApp Server内で同時RPCを多重化する実装ではない。初期化は一覧用1回＋worker用2回になる。同時に動くworkerは最大2。

Turn単位の共有queueを使い、同じTurnのページは逐次取得する。結果は完了順ではなく列挙順へ戻す。item_orderはTurn内の全item位置へ統一し、方式/workerの割当てによって変わらないようにする。投影後の少ないitemだけで番号を付け直すものではない。編集差分・発話ID・本文・安全フィルタ・期間・coverageは共通処理。worker失敗時は他workerへcancelし、未完了Turnを完全取得に昇格させない。

この研究モードは、現在の単一接続向け詳細タイミング/種別統計とは併用不可。並行では各待ち時間を足して全体時間にできないため、今回は壁時計の取得時間・合計要求数・受信量と外部メモリ監視で比較する。既定paged100は変更しない。非空のfixtureでsource/順序/照合結果の一致、要求が実際に最大2件重なること、失敗時のfallbackを検証し、race検査も通した。

## 重複検出と段階的メモリ保護

`parallel100`（最大2）、`parallel100x4`（最大4）、`parallel100x8`（最大8）を比較候補として選択できる。既定は従来の`paged100`を維持し、実測前に8並列を採用しない。一覧は一度だけ取得し、thread+Turnの一意な仕事をqueueへ登録する。同一Turn内のページ取得は逐次。同じitem IDは一度だけ扱い、重複によってitem_orderを進めない。同じIDで採用フィールドが異なる場合は`conflicting_duplicate_turn` / `conflicting_duplicate_item`、理由は`history_parse_failed`として記録する。競合Turnの証拠は保持せず、他の取得完了済みTurnを残す。異なるIDで本文が同じ記録は候補として残す。v2投影の直前にもTurn/操作の同一IDを検査し、同じ内容は一件、異なる内容は`conflicting_source_identity`で拒否する。LLMへの接続は今回の範囲外。

並列モードでは開始前と500ms間隔で、PoC自身・所有App Server・その子孫の合計RSSとmacOSのメモリ圧迫値を調べる。本文・コマンド引数・環境変数を監視ログへ記録しない。`history.memory_protection`へ数値の時系列、最大同時Turn数、中断理由を保存する。

| 状態 | 制御 |
| --- | --- |
| 合計RSS 1GiB以上、またはpressure=2 | 並列増加を停止し、接続は再利用する |
| 合計RSS 1.5GiB以上 | 新規Turn割り当てを停止し、取得終了後の接続を閉じる |
| RSS 1GiB未満かつpressure=1を連続2サンプル | 再開。1.5GiB到達後は指定最大並列数の半分（最低1）を上限とする |
| 合計RSS 2GiB以上 | `memory_limit_exceeded`で中断 |
| pressure=4 | `critical_memory_pressure`で中断 |
| 監視が取得不能 | `monitoring_unavailable`で中断 |
| 割り当て停止から30秒間回復しない | `memory_backpressure_stalled`で中断 |

中断時は所有接続をcancelして閉じ、完了済みTurnと不完全理由を返す。無関係なCodexプロセスは停止しない。監視はmacOS専用であり、他OSでは並列モードは監視不能として停止する。1応答300,000,000bytes上限は全workerに適用する。600MBの合計入力予約は今回は未実装。RSS監視はサンプリング制御なので、瞬間的なメモリ超過やOSのOOMを完全に防ぐ保証ではない。実履歴での2/4/8並列の速度・監視コスト比較は別途必要。

## 一覧と本文取得を重ねる比較経路

`pipeline100x4`は本文取得最大4workerのまま、一覧から一意なTurnが見つかるたびに本文取得へ渡す。queue容量は16Turnで、満杯なら列挙側が待つ。同一Turnのページは逐次。結果は列挙順へ戻す。一覧用接続と本文用接続が同時に存在するため、最大接続数は5。全所有プロセスを同じRSS監視に含める。最大4は本文の同時取得数であり一覧RPCを含む総RPC数ではない。列挙途中の失敗や競合では、不確かな証拠を残さないよう、この経路で取得した本文証拠を保守的に破棄して不完全を返す。既定paged100は変更しない。

`COMMITER_PIPELINE_WAITS=1`を明示したlive probeでは、worker仕事待ち（合算）、queue満杯待ち、メモリ許可待ち、Turn列挙番号ごとの本文時間/受信bytes/要求数を保存する。加えて一覧接続と各workerで独立したRPC計測を持ち、送信・応答待ち受信・JSON解析・終了処理を区別する。本文/ID/pathは計測ログへ保存しない。並列の区間合計は壁時計時間へ加算しない。一覧処理の戻り時刻は接続終了を含むため、cleanup_msを別に確認する。

## 通常CLIとの共通ETLエンジン

実装と回帰テストは`internal/historyattribution`にあり、研究用コマンドは同じエンジンへの入口を維持する。過去の採取原本・測定成果物は変更しない。通常CLIは`internal/cli/attributed_history.go`から共通の取得・照合処理を直接呼び出し、hostの照合結果を3フェーズへ供給する。

通常経路はHEADのコミット時刻から処理サイクル開始時刻まで、現在ワークツリーだけを対象とする。100件ページ、本文最大4並列、一覧中の本文取得、メモリ保護、1応答300MB上限を使う。通常CLIの履歴取得は5秒以内で、取得失敗は理由を残してGit差分へフォールバックする。固定Git入力の変更は停止する。

`go test ./internal/historyattribution ./internal/cli ./internal/planning`で検証する。`TestCLIHistoryETLToHelper`はApp Server/helperの別プロセスfixtureを使い、モデルをロードせず個人の実履歴も読まない。macOSのメモリ監視を使うためこのCLI ETLテストはmacOS向けで、共通RPC取得テストは監視samplerを注入して他環境でも実行する。
