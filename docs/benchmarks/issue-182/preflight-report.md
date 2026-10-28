# Issue #182: 三段階推論エントリーポイントの互換性確認

この文書は研究用モデル差し替えモード追加前の確認記録。後続の`H182-001-model-defaults`では、明示したbenchmark modeでモデル固定を外して比較する。通常CLI経路についての以下の観測は保持する。

2026-10-19、PR #181 のマージcommit
`69bfc0775d23e62481cd45bcb4bf73608a355757` を基点に実行した。
**比較計測は保留**。現行の三段階推論経路は候補3モデルを受け付けない。
モデルの品質や性能を理由にした不採用判断ではなく、Issue #182 の完了条件も未達である。

| モデル | CLI設定検証 | helperのbounded-grouping要求 | 推論 |
| --- | --- | --- | --- |
| Gemma 4 E4B 4bit | 受理 | 未実施 | 未実施 |
| Qwen3.5-4B 4bit | 拒否 | malformed_request | 未実施 |
| Qwen3.5-2B 4bit | 拒否 | malformed_request | 未実施 |
| Phi-4-mini-instruct 4bit | 拒否 | malformed_request | 未実施 |

## 実行条件と確認範囲

- `tools/benchmark182/preflight.go` をGoでビルドし、productionの`config.Resolve`と`mlx.Client`を呼び出した。
- helperは当該worktreeのマージ済みSwiftソースからDebug buildした。固定済み依存を使用し、追加依存なし。ハッシュは`preflight-contract.json`に保存した。
- 同helperの`--smoke-metal`は`metal_ok`。これはMetal演算の動作確認で、モデルの正常生成を示さない。
- 3候補の要求は実在する空ディレクトリと合成メッセージを使用した。モデルID検証はweightロードより前に実行される。candidate weight・tokenizer・chat templateのロード試験ではない。
- 1 invocationあたり最大3要求、要求ごと15秒、retry/repair/fallbackなし。初回確認と終了コード検出追加後の再確認を実行し、計6件の拒否を確認した。モデル推論は0回。
- 公開APIから4モデルのrevision、model_type、量子化設定、weight/tokenizer配布ファイルの存在を確認した。全候補の設定は4bit/group64、GemmaとQwenはaffine表記、Phiのmodeは未記載。実効bits、weight整合性、template動作、context対応は未検証。
- 検証OSはmacOS 26.6.2（25G83）。推論が未実施のため、推論時のCPU/GPU・メモリ・熱・電力の時系列、監視負荷、性能スコアは未取得。

## 停止理由

`internal/config/config.go` の三段階planner検証はGemmaのモデル名・revision・4bitを固定し、他モデルを拒否する。`mlx-helper/Sources/commiter-mlx-helper/CommiterMLXHelper.swift` もbounded profileとGemmaの固定IDを照合し、異なるIDを`malformed_request`とする。3候補とも両方の拒否を実行確認した。

さらに、helperのnative thought処理はGemmaのchannel token IDを前提とし、研究用prefix cache構築もGemma型に限定される。これらはソースから確認した制約で、候補モデルでの失敗実測ではない。モデルIDの照合だけを外しても、H180-004をそのまま共通条件にはできない。

固定依存`mlx-swift-lm@c6446cf7bfb7cea76408013b614d4b2c530eaa03`のLLM factoryには`qwen3_5`と`phi3`の登録がある。ただし、factory登録とcommiter-cliのエンドツーエンド対応は別であり、weightロードや正常生成の成功は主張しない。

## 次の実装・計測に必要な条件

候補モデルを比較する研究用経路を追加し、通常CLIの固定モデル契約と分ける必要がある。研究構成ではモデルごとのchat template・thinking制御・KV型の対応を定義し、Gemmaにも同じ条件を適用して新たに測定する。既存H180-004の結果をその新構成の実測baselineとして流用しない。

その後、公開可能な16ファイル相当のfixtureと正解partition/category、同一prompt/schema/validator、timeout、warmup・反復回数、推論上限と停止条件、資源監視方法・負荷を推論前に固定する。実Groupingを後段へ渡し、正解group注入による後段評価は別集計とする。

現時点でデフォルトモデルは変更していない。比較が成立していないため維持・変更の優劣判断は保留。

## 検証とローカル証跡

- native helper build: 成功
- Metal smoke: 成功
- `go test ./internal/config ./internal/mlx -count=1`: 成功
- `go build ./tools/benchmark182`: 成功
- `go vet ./tools/benchmark182`: 成功
- `git diff --check`: 成功

生の実行出力・buildログは`<PRIVATE_TMP>/issue182-preflight/`および`<PRIVATE_TMP>/issue182-*.log`に保持し、Gitへ追加していない。初回の検証コマンドbuildは存在しないClientフィールドを指定して失敗したため、既存の`mlx.NewClient`へ修正してbuildを通した。失敗ログも保持した。公開可能なモデルmetadata、再現コード、条件、要約だけを本ディレクトリに残す。
