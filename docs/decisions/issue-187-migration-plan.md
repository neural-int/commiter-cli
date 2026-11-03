# Issue #187: Harness 004への通常経路統一

状態: 着手時の実装計画。移行コード・性能評価・リリースは未完了。
確認日: 2026-11-03。

## 基準

- 対象: [Issue #187](https://github.com/neural-int/commiter-cli/issues/187)。
- 実装の基点: マージ済みPR #185を含む`fa8fcd2e877afe5e149cac67f3dbd01f7c8c7dc5`。
- 履歴取得・内容照合・投影の移植元: `codex/issue-182-model-comparison`の`f75c88fd005435d856d1b2464aeb9738f5521d4c`。確認時に作業ツリーの変更なし。`internal/historyattribution/`、`internal/cli/attributed_history.go`、`internal/planning/{grouping_history,grouping_input,metadata_input,command_safety}.go`を個別に照合して取り込む。研究ブランチ全体のマージは行わない。
- 確定フィールド: [Issue #183のスキーマ・ETL・コマンド補助入力の採用コメント](https://github.com/neural-int/commiter-cli/issues/183)。過去の依頼本文を送る部分は、後続のfinal-only方針で置き換える。
- 実行条件: [Harness 004 revision 3](../benchmarks/issue-184/harnesses/H184-HARNESS-004-r3.json)と[実token判定](../benchmarks/issue-184/harnesses/actual-token-context.md)。歴史的manifest・元fixtureは変更しない。

## 引き継ぐ先行条件

全フェーズの枠は65,536 tokens、出力予約は各4,096 tokens。最終入力はchat template適用後の実token数で最大61,440。入力全体が12,000 tokens以下なら追加の毎chunk同期・allocator cache解放を省き、12,001以上で実施する。再利用prefixを入力全体に含める。

通常は各フェーズ1要求・全入力準備1回。履歴超過時は必須入力を確保し、全件1回＋必須1回＋候補最大3回の上限を維持する。採用した準備済み入力を再利用し、計数専用IPCやGoのbytes/token判定は導入しない。tokenizerの失敗を履歴不在に置き換えない。

4ファイル上限、120秒サイクル、summary48文字、scope16文字、Thinking OFF、JSON Schema、native KV、モデル別sampling/seedを維持する。Grouping/Textの独立KVコピー、Category独立prefill、未対応cache型の通常prefillへのfallbackも維持する。

## 着手監査で確認した差分

| 領域 | 現在の通常経路 | 変更先 |
|---|---|---|
| 履歴取得 | `optionalCodexContext`から旧Collector。別worktreeの会話も探索 | 現在worktree限定・100件ページ・本文4並列・重複/競合検査の取得経路 |
| 帰属 | 完了fileChangeのパス関連付け | 固定した完全diffの全変更ブロックと編集本文の照合。rename・部分一致・曖昧さを保持 |
| 履歴本文 | final-onlyだが`paths/final_response`形式 | `context_ref`本文辞書とブロック候補参照。安全な非空最終応答のみ |
| コマンド | 未接続 | Groupingだけに安全なパス言及補助入力。編集成功・ブロック一致の証拠にしない |
| 予算 | helperの実token選別が旧`codex_context`専用 | 新しい辞書・候補・コマンド参照を壊さずに記録全体を選別 |
| ファイル | `path:` ID、Git状態コード、false/nullやmode等を送信 | 入力内のF1形式IDとホスト対応表、自然順、許可フィールド、metadata-onlyの空diff/空blocks |
| Category/Text | 古いファイル構造と証拠候補を共有 | 確定した専用投影。Textへ確定type/breakingを渡し、取得・投影の全体状態を送らない |
| 設定 | planner省略時にsingle-pass/Ollamaへ戻る | 明示した新契約か、移行方法の分かる設定エラー |
| モデル | Goは固定Gemmaだけ、helperは固定Qwenも受付 | 検証済みのモデル・revision・量子化の組を設定/setup/doctor/helperで一致させる |
| 入力準備 | Three-phaseの前に旧system/Rendererを生成 | Three-phase専用準備。再生成feedbackと安全性検査を維持 |

移植元の`budgetPhaseHistory`は旧バイト予算と依頼本文を含む。通常経路へそのまま接続しない。予算による最終応答の省略では本文を切り詰めず、候補参照・コマンド参照・省略理由も一緒に投影し直す。候補削減後の見かけ上の一意性をmatchedへ昇格させない。

## 実装順序

1. 取得・照合エンジンと回帰ケースを移植し、final-onlyへ統一する。固定Git入力の前後検査、ホスト専用出典表、再生成キャッシュの同一入力検査を接続する。
2. F形式ID・全変更ブロック・フェーズ別許可フィールドを接続する。実際のhelper受信要求で、元diffの不変性と参照整合性を検査する。
3. 新履歴投影と実token選別を統合する。安全性除外・履歴なし・取得失敗・全件収容・一部省略・全件省略を確認する。Grouping/Text共有入力とprefix再利用の整合性を維持する。
4. 通常CLIのsingle-pass分岐、設定の暗黙復帰、旧prompt準備、旧経路への案内を除去する。研究用途に必要な旧コードは通常CLIから到達不能な経路へ分離する。既存の失敗記録・fixtureは保持する。
5. 設定・setup・doctor・helperのモデル受付を統一し、Go固有の配置・分類強制を整理する。未検証モデルを一般許可しない。
6. 移行文書、日英README/SRS、設定スキーマ、破壊的変更のリリース情報・配布版更新手順を揃える。次バージョンは実施時の最新タグと配布規約で確定し、ここではタグ・配布・main mergeを実行しない。

## 確定した取得期限

2026-11-03のユーザー指定により、CLIとCollectの履歴専用5秒期限を両方撤去する。親contextのキャンセル、通信量・メモリ保護を維持する。Git snapshotの前後検査は独立した短い期限を維持する。

## 検証と完了判定

- 構造検査: 安全性、全ブロック、rename、部分一致、同一出典の重複/競合、別操作の同内容、欠落・不正ID、候補省略後の曖昧さ、全ファイルの完全割当、Git変化時の停止。
- 受信検査: 3フェーズ伝播、不要な旧prompt/フィールドの不在、非空user_requestsの不送信、最終応答欠落・危険・超過時の記録全体除外、参照先の存在、元diffとホストデータの不変性。
- 実行検査: Go test/race/vet/build、Swiftの関連テストとRelease helper、Pythonの表示・投影回帰、差分チェック。CIは実際の対象HEADの結果を別途確認する。
- 実モデル検査: 通常入口は4ファイル以内、研究入口は7/9/11ファイルと区別する。JSON成立、validator、意味品質、完走を分け、KV再利用・各フェーズ時間・TAT・メモリ/swap/熱状態を記録する。構造検査の成功を速度・意味品質の成功に置き換えない。
- Issueの完了条件が残る間は移行完了・本番検証済み・リリース済みと記載しない。
