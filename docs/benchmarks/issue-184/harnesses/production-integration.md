> 現行のコンテキスト・prefill方針は[revision 3](H184-HARNESS-004-r3.json)と[実token判定の記録](actual-token-context.md)を参照。以下の16K/32K/64K選択・legacy保護の説明と既存の測定表はrevision 2時点の記録である。

# H184-HARNESS-004 / revision 2 の3-Phase接続

004の混合配置・型別KV prefix再利用をThreePhaseGeneratorに接続する。履歴的なrevision 1の記録は書き換えず、[revision 2](H184-HARNESS-004-r2.json)で実装条件を区別する。本書はローカル実装・検証の記録であり、mainへのマージやCI成功の記録ではない。

## 変更範囲と維持する制限

| 項目 | revision 2 |
| --- | --- |
| 入口 | ThreePhaseGeneratorのみ。単一呼び出しの旧Generatorは変更しない |
| ファイル上限 | 通常3-Phaseの4ファイルを維持 |
| 時間制限 | 通常3-Phaseの1サイクル120秒を維持 |
| summary | SchemaのmaxLength 48を維持 |
| 出力予算 | 各フェーズ4096 tokens |
| 入力context | Preparedの予算内で最初の要求に収まる16K/32K/64Kを選び、サイクル内で固定。後段が収まらなければ停止 |
| Thinking / grammar / KV | OFF / JSON Schema制約あり / native |
| prefill保護 | 従来どおり各chunkの同期とallocator cache解放を維持 |
| 処理 | 同一helperでGrouping→Category→Text。モデルと共通文法準備を保持 |

32K固定は保存済み研究fixtureの条件であり、通常Plannerのcontext自動選択を32K固定に変更したわけではない。64K固定・16K保護閾値・解放頻度変更も採用していない。

## 入力とキャッシュ

固定の指示文は`internal/planning/prompts/harness004.json`をGoへembedする。Pythonアダプターも同じファイルを読む。非公開の保存済み要求を指示文の供給元にしない。Schemaは実際のファイル・グループIDから生成し、プロンプト内への完全掲載は一回とする。

1. Grouping: 共通system、完全なfilesと履歴、フェーズ指示・Schema・ファイル参照。
2. Category: 分類規則を含むsystem、履歴、フェーズ指示・Schema・グループ内の完全なファイル差分。
3. Text: Groupingと同一の先頭2メッセージ、フェーズ指示・Schema・確定した所属/type/breakingとファイル参照。

表示からだけtask/schema_version、user_requests（非空でも省略）、block_id/execution_ref等を省く。保存fixtureの元データと出力検証は保持する。Schemaの辞書キーは自然順で整列し、配列の意味的な順序は保持する。raw_diffのヘッダー・hunk・追加削除記号は残す。実改行化はGoのMLX BackendでLLM用本文だけに適用し、Go–Swiftの外側の通信は正規のJSONを維持する。生成時のPython Proxyは不要。

固定Text指示には、48文字以内で具体的かつ完結した文を選び、非本質的な識別子を省き、識別子や句の途中で終わらせないよう記す。48文字のSchema上限を緩める変更ではない。この指示の採用と、意味品質が実験で改善したこととは区別する。

helperは最初の共通チャット部分を同じtokenizer/templateで処理し、実際の入力token列と先頭から一致する範囲だけを保存する。Grouping用とText用のキャッシュは独立コピーとし、Categoryは独立prefillする。再利用条件はモデル・設定・token列の一致。型の選択は実キャッシュ型を基準とし、未対応型や不一致は通常prefillへ戻る。現在の有効化対象はmanifestのSHA固定Qwen/Gemmaだけである。

通常経路の履歴供給は、既存の任意`codex_context`とパス関連付けを維持する。研究fixtureの`agent_history_contexts`を使う内容照合PoC全体を本番へ移植したという意味ではない。取得や内容照合は本変更の測定範囲外であり、対応関係を推測で補完しない。

2026-11-03のユーザー指示により、履歴本文はfinal-onlyへ変更した。旧CollectorとPlannerの「依頼必須・最終応答を先に省略」という規則を削除し、安全な非空の最終応答だけを保持する。収まらない最終応答は記録全体を除外し、依頼へのフォールバックはしない。旧取得経路と004の完全な契約との差は[整合性監査](004-production-alignment-audit.md)に記録した。取得経路・64K固定・実token計数まで整備済みと扱わない。

## 保存fixtureの準備・実行

検証専用runnerは通常Plannerの4ファイル制限と120秒制限を変更せず、保存済み7/9/11ファイル要求を専用helperモードで扱う。App Server取得、Git stage/commit、モデル取得は行わない。新規依存は不要。

標準のRelease helper/Metalビルド手順を完了した後、validatorと熱状態probeを作る。

```sh
go build -o /path/to/private/host-validator ./tools/benchmark184/multifile-validate
swiftc tools/benchmark184/thermal_probe.swift -o /path/to/private/thermal-probe
```

manifestの例。相対パスはmanifestの配置ディレクトリを基準にする。

```json
{
  "cases": [
    {"name": "7-files", "requests": "7-files/requests", "gold": "7-files/gold.json"},
    {"name": "9-files", "requests": "9-files/requests", "gold": "9-files/gold.json"},
    {"name": "11-files", "requests": "11-files/requests", "gold": "11-files/gold.json"}
  ]
}
```

requestsには正規JSON本文を持つ`grouping.request.json`、`category.request.json`、`text.request.json`を用意する。32K/4096、モデルの固定revision、final-only、各ファイルの根拠が3フェーズで一致することを検査する。入力の依頼や差分を公開リポジトリに保存しない。

```sh
python3 -B tools/benchmark184/run004.py \
  --manifest /path/to/private/cases.json \
  --helper /path/to/release/commiter-mlx-helper \
  --validator /path/to/private/host-validator \
  --thermal-probe /path/to/private/thermal-probe \
  --output /path/to/private/new-preparation
```

既定は準備だけで生成しない。全ケースの入力整合性、実行権限、正常計画の受理とファイル欠落計画の拒否を、モデル起動より先に確認する。元要求・gold・派生要求・指示/helperハッシュを別保存し、既存出力先は上書きしない。

実推論は新しい出力先を指定し、明示的に`--execute`を追加する。Metalの疎通後に熱・負荷・空きメモリを確認し、各ケースの開始前にnominal状態等が60秒継続することを待つ。固定時間の待機だけで同等の熱状態と断定しない。Grouping一回、正解上流のCategory/Text、実上流を伝播するCategory/Textの最大5生成を各ケース一回実行し、失敗・未完了を保存して再試行しない。例外や監視異常で停止した結果を成功扱いにしない。

応答にはhelperの時間・token・キャッシュ計測値を保存する。CPU/RSS、GPU公開統計、swap/メモリ圧迫、電源・熱状態はmonitorへ保存する。OSが返さない資源指標は欠測であり、推定しない。`state=complete`は予定ケースの処理終了を表し、意味正解・全生成の完走とは別である。failed_cases、phase_calls、各evaluationと応答を照合する。生の会話・Thinking・個人パスを含む成果物はprivateで管理し、公開時は集計だけを選ぶ。

## 今回の確認結果

| 確認 | 結果 |
| --- | --- |
| Go全テスト | 成功 |
| Swift Release全テスト | 57件成功、Metal読込成功 |
| 実Qwen / 通常3-Phase入口 | 1ファイルの人工差分で3フェーズ・最終validator通過、Textで191 tokens再利用 |
| 実Gemma / 通常3-Phase入口 | 同じ疎通テスト通過、Textで200 tokens再利用 |
| 搬送・表示 | 外側JSON成立、本文のみ実改行、指示/ホスト元データ不変をテスト |
| validator事前確認 | 保存7/9/11要求で正常受理・欠落拒否。実行権限不足もモデル起動前に検出 |

これらは接続の確認であり、新revisionで7/9/11ファイルの意味品質やTATを再測定した結果ではない。以前のQwenで見られた再利用有無によるText差分と、数値的な完全同等性の未確定状態も解消したとは扱わない。CI・remote・mainの状態はこのローカル検証とは別に確認する。
