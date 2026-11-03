# Harness 004への設定移行

通常の計画生成をMLXのThree-phase Plannerに統一します。`single-pass`、Ollamaによる通常生成、planner省略からの旧設定復帰は廃止します。旧設定を検出した場合はエラーで停止し、別のPlannerへ切り替えません。

## 設定の変更

グローバル設定とリポジトリの`.commiter.toml`の両方を確認してください。モデルやbackendを指定する既存ファイルでは、`planner = "three-phase"`を明示してください。低優先度の設定が不正な場合は、高優先度の設定で上書きしてもエラーになります。確認・push・検証コマンド・機密判定の既存設定は必要な値を保持してください。

`[llm]`を次の構成に変更します。`schema_version = 1`は維持します。

```toml
[llm]
planner = "three-phase"
backend = "mlx"
model = "mlx-community/gemma-4-E4B-it-4bit"
model_revision = "475b9088d29754a3379866cf5aeb6b41acd313c2"
model_quantization = "4bit"
context = "64k"
max_context_tokens = 65536
```

検証済みの別構成は、`model = "mlx-community/Qwen3.5-4B-4bit"`と`model_revision = "0e7ffd5c629ef7719d4cbc04069232580bfa9d9c"`の組です。量子化は同じ`4bit`です。モデル名だけの変更、可変revision、任意のモデルの指定は受け付けません。`--model`もこの組の整合性検査を受けるため、モデルを切り替える場合は設定のrevisionも合わせて変更してください。

`llm.context = "auto"`も64K固定の別表記として受け付けます。8K/16K/32K枠や65,536以外の`max_context_tokens`は移行エラーになります。旧`endpoint`は通常MLX生成に使いません。設定に残す必要はありません。

変更後は、次の順に確認します。

```sh
commiter config show --effective
commiter setup
commiter doctor
commiter --dry-run -- path/to/changed-file
```

Apple Silicon macOSと同梱Swift helperが必要です。`setup`はモデルと固定revision、概算容量、保存先を表示し、確認後にモデルを取得します。`doctor`と通常生成はモデルを取得・更新しません。

## 入力・実行条件

- 通常入口は最大4ファイル、生成サイクルは120秒、各フェーズの出力は4,096 tokensです。summary48文字、scope16文字、JSON Schemaと最終validatorを維持します。
- Grouping、Category、Textはすべて65,536 tokensです。helperがchat template適用後の実token数を検査し、出力予約込みで収まらない必須入力は生成前に停止します。Goのバイト数は通信・メモリ保護に使い、token数の確定には使いません。
- 実入力が12,000 tokens以下なら追加の毎chunk同期・allocator cache解放を省き、12,001以上なら実施します。再利用prefixも全入力に含めます。
- 1回の生成に同じhelperを使い、Grouping/Textで型別KV prefixを再利用します。各フェーズの実行用KVは独立コピー、Categoryは独立prefillです。未対応型やprefix不一致では通常prefillに戻ります。

## 履歴の取得と省略

履歴の取得後も、CPU照合に親キャンセルと既存のメモリ保護を適用します。メモリによる待機中もキャンセルでき、停止した照合の途中結果を成功として使用しません。監視はサンプリングによるもので、瞬間的なメモリピークを完全に防ぐ保証ではありません。

App Serverとの各RPCは、応答データの進捗が30秒止まると打ち切り、history_unavailableとして任意履歴を除いてGit差分のみで続行します。通知だけでは応答の進捗としません。これは無応答検知であり、履歴収集全体の固定期限ではありません。

100%一致の純粋renameは、renameメタデータを指す仮想ブロックから照合します。旧新パスだけで内容の編集成功を断定せず、関連候補はambiguousとして保持します。

App Serverから現在worktreeの履歴を100件ページ・本文最大4並列で取得し、固定した現在Git差分の変更ブロックと内容を照合します。履歴専用の5秒期限は設けず、親処理のキャンセルと通信量・メモリ保護を維持します。Git snapshot検査は独立した期限で行います。

Turn開始時刻がHEAD以前であっても探索対象に含めます。個別編集の時刻がないため、内容一致は現在の差分との関連性を示すもので、HEAD以降に編集したことの証明ではありません。サイクル開始後に始まったTurnは除外します。古いTurnも探索するため取得負荷は増える可能性がありますが、キャンセル・応答サイズ・メモリ保護は維持します。

Categoryは、ホストが供給した破壊的変更の証拠候補のID・対象ファイルID・Factを受け取り、typeとbreaking_evidence_refを選択します。ホストが参照を検証してbreakingを確定し、Textには確定したtype/breakingだけを渡します。Textへ証拠候補・Factは再送しません。

通常経路は、固定Git差分の各変更ブロックからgit:F番号:B番号の参照候補を自動生成します。Factは対象パスと差分の行範囲を示します。これらの候補は非互換性を事前判定したものではなく、Categoryが実差分から破壊的変更かを判断します。非破壊ならnone、根拠不足ならunresolvedを選択できます。手動で渡した確認済みの証拠とは区別し、グループ外・架空の参照は拒否します。

commandExecutionの重複検査・安全性検査・パス言及の投影にも、照合と同じキャンセル・メモリ保護を適用します。

モデルへ渡す履歴本文は安全な非空の最終応答だけです。依頼本文・Thinking・途中経過を送らず、最終応答が欠落・危険・予算超過の場合は記録全体を省きます。依頼への置換、本文の部分切断、パス一致だけによるmatched判定は行いません。

候補を省略する場合は候補参照・コマンド参照も省き、影響する変更ブロックに省略理由を残します。曖昧さをmatchedへ昇格させません。Groupingのコマンド履歴は旧新パスへの言及という補助情報であり、実行・編集成功・内容一致の証拠には使いません。Category/Textにはコマンド履歴と取得・投影のトップレベル状態を渡しません。

既存のIPC通信枠を任意履歴が超える場合は、候補参照と省略理由を整合させた履歴完全省略の表示で続行します。必須入力自体が通信枠を超える場合は停止します。このバイト制限はtoken判定とは別の資源保護です。

全件収容時の全入力準備は1回です。超過時は必須入力を確保して候補を一括選別し、全件1回＋必須1回＋候補最大3回に制限します。採用した準備済み入力を生成へ再利用します。履歴不在・取得不能・部分欠落は入力の状態として保持し、Git差分だけで処理を続けます。tokenizerの失敗、必須入力の超過、Git snapshotの変化、validator違反は停止条件です。

## リリースと研究経路

この移行は設定と利用経路を変える破壊的変更です。PRは`feat(planning)!: unify production planning on harness 004`、Release categoryは`Changed`、Breaking changeは`Yes`として扱います。2026-11-03確認時の最新公開リリースは`v2.0.0`であり、次のmajor候補は`v3.0.0`です。リリース実施時にタグとmainの差分を再確認して確定します。

配布バージョンは`.github/workflows/release.yml`がタグから取り出し、`internal/cli.Version`へldflagsで埋め込みます。新しい静的versionフィールドは追加しません。リリース時はタグ、CLIの`version`出力、配布アーカイブ名・メタデータの一致を既存preflightで確認します。本移行作業ではタグ・配布・main mergeを実施しません。

過去のsingle-pass比較には`ResearchGenerator`と当時の記録を使います。旧bounded helper入力は`--research-legacy`を明示した研究実行に限定します。歴史的ログ・fixture・manifestは当時の条件の記録として保持し、今回の意味品質・性能結果へ流用しません。7/9/11ファイルの研究経路は通常入口の4ファイル上限と区別します。
