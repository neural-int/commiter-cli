# Issue #110: Ministral 3 3B での Ollama / MLX 再計測（2026-09-25〜26）

## 結論

**暫定結果。** Ministral 3 3B は両 backend で正常系を完走した。MLX は Qwen3.5-4B で失敗した複数 commit ケースの構文・ドメイン検証を 5/5 件通過したが、独立した変更を全件 1 commit にまとめ、期待グループには 0/5 件しか一致しなかった。Ollama も実装とテストを一貫して別 commit に分け、期待グループへの一致は 0/5 件だった。64K 設定の MLX は 2 試行とも不採用、24 ファイルは両 backend とも本番の出力枠で不採用だった。この結果から既定モデル・backend や bounded-output の量的上限は変更しない。

## 条件

- Apple M3、16 GiB unified memory、macOS 26.6.2、arm64。repository HEAD は `6e087f5cffbaa0c6594e6f01df968ef4fdcb12d3`。本計測で production コードと依存は変更していない。
- Ollama 0.33.3、Metal GPU、[`ministral-3:3b-instruct-2512-q4_K_M`](https://ollama.com/library/ministral-3:3b-instruct-2512-q4_K_M)。manifest SHA-256 は `f04aa1c738f64e13c625b82ae92504fc0260fa6723b509ed1ece0fa188179b1d`、モデル層は 2,953,825,504 bytes。
- MLX release helper はこの checkout の固定 `mlx-swift-lm` revision `c6446cf7bfb7cea76408013b614d4b2c530eaa03` を使用。モデルは [`mlx-community/Ministral-3-3B-Instruct-2512-4bit`](https://huggingface.co/mlx-community/Ministral-3-3B-Instruct-2512-4bit) の `a962dcb09eee4169c890e544c9eb938f1113fdee` に固定し、ハッシュ検証して取得した。保存対象ファイル合計 2,779,150,244 bytes。Ollama の Q4_K_M と MLX の affine 4bit は量子化形式が異なる。
- [計測器](../../tools/benchmark110/main.go) は同一の合成 diff・planning schema・renderer・context preparation・generator・validation を production adapter 経由で使用する。repository 収集、Git 変更、実データ入力は含めない。記録するのは数値、違反コード、commit type、ファイル ID のみで、prompt と生成文は保存しない。試行は順番に実行した。
- 正常系、複数 commit、日本語は各 backend で 5、5、3 回。大きめ diff、64K 設定、24 ファイルは 1 回ずつ（MLX の 64K 設定は原因切り分けのため再試行）。通常の出力枠は 1024 token、24 ファイルの本番枠は `48 × 24 = 1152` token。wall time は入力準備後から計画検証までで、helper 起動と許された 1 回の修復を含む。
- 最初の sandbox 内 MLX 試行は Metal デバイス非公開で helper が起動時に終了したため、測定値から除外した。Metal にアクセスできる環境で `--smoke-metal` が `metal_ok` を返し、以降の MLX 試行をそこで実施した。

## 同一フィクスチャでの結果

| フィクスチャ | 入力 / context 設定 | Ollama | MLX | 計画品質 |
| --- | --- | --- | --- | --- |
| 実装 + テスト | 3,690 bytes / 8K | 検証 5/5、中央値 7.39 秒（7.04–7.74） | 検証 5/5、中央値 5.26 秒（5.11–5.55） | 期待する同一 commit は Ollama 0/5、MLX 5/5。Ollama は毎回実装とテストを分離。 |
| 独立した 2 変更 | 4,598 bytes / 8K | 検証 5/5、中央値 19.49 秒（10.89–32.57） | 検証 5/5、中央値 7.50 秒（6.80–7.63） | 期待する 2 グループは両方 0/5。Ollama は毎回実装・テスト・文書の 3 グループ、MLX は毎回全 4 ファイルを 1 グループ。 |
| 日本語 summary | 3,719 bytes / 8K | 検証 3/3、中央値 13.35 秒（12.81–15.00） | 検証 3/3、中央値 6.74 秒（6.69–6.75） | 日本語検証は全件通過。期待グループは Ollama 0/3、MLX 3/3。 |
| 大きめ diff | 26,400 bytes / 32K | 検証 1/1、45.77 秒 | 検証 1/1、28.58 秒 | Ollama は実装・テスト・文書を分離、MLX は同一 commit に配置。 |
| 64K context 設定の構造入力 | 54,692 bytes / 64K | 検証 1/1、215.65 秒 | 検証 0/1、227.14 秒。再試行も 0/1、221.88 秒 | MLX の最初の試行は backend error。再試行では JSON は完了したが、初回・修復とも `invalid_assignment`。 |

上の 64K ケースは **64K token に近い prompt を実測したものではない**。54,692 bytes の入力から `contextinput` が 64K の設定を選んだケースであり、MLX には入力 token 数の telemetry もない。64K 上限近傍の品質やメモリ安全性の証拠として使わない。

通常の 13 試行では両 backend とも grammar・domain 検証が全件成功し、修復は不要だった。期待グループ一致は Ollama 0/13、MLX 8/13。ただしこの参照グループは明確な 3 種の合成例に限る。大きめ diff の比較は単一試行である。

## ファイル数と出力枠

| ケース | 結果 |
| --- | --- |
| MLX、12 ファイル、1024 token | 検証成功、12.65 秒。全ファイルを 1 commit にまとめたため、このケースは割当成功の証拠であり、意味的な commit 分割の合格判定ではない。 |
| MLX、16 ファイル、1024 token | 初回・修復とも `invalid_assignment`、43.15 秒。 |
| MLX、20 ファイル、1024 token | 初回・修復とも `invalid_assignment`、86.06 秒。 |
| MLX、24 ファイル、1152 token | 初回・修復とも `invalid_assignment`、117.29 秒。 |
| MLX、24 ファイル、2048 token | 初回・修復とも `invalid_assignment`、86.87 秒。応答サイズは 1152 token 枠の試行と同じで、token 枠を増やしても割当が改善しなかった。 |
| Ollama、24 ファイル、1152 token | 初回・修復とも 1152 decode token で `invalid_json`、257.10 秒。adapter は終了理由を返さないため、token 上限到達は decode count からの推定。 |
| Ollama、24 ファイル、2048 token | 検証成功、84.73 秒、1193 decode token。24 ファイルを各 1 commit に割り当てた。 |

失敗した計画はすべて不採用となり、部分的なファイル割当は受け入れていない。

Ollama の 24 ファイルは 2048 token なら単発で完走したため、1152 token の本番予約量はこのモデル・入力では足りない。ただし 1193 token を次の production 上限として採用できる根拠にはならない。MLX は 2048 token でも割当違反であり、両 backend に共通する安全な量的上限はまだ出せない。

## 既存 Qwen3.5-4B 計測との対照

[前回のレポート](issue-110-2026-09-25.md) では Qwen3.5-4B の複数 commit ケースで Ollama が検証 4/5・期待グループ 4/5、MLX が検証 0/5 だった。今回の Ministral では両 backend が検証 5/5 だが、期待グループは双方 0/5。MLX の構文・ドメイン検証は改善したが、期待する計画品質には達していない。前回は 64K 設定が両 backend で検証成功したが、今回の MLX は 2 試行とも失敗した。小標本、異なるモデル容量・量子化、キャッシュと熱状態の影響があるため、中央値だけで安定した速度順位を付けない。

## 指標の範囲と未解決事項

- 最初の正常系 pilot は Ollama 7.88 秒、MLX 6.08 秒。OS キャッシュを消去していないため真の cold start ではない。
- production Ollama リクエストは `keep_alive: 0`、MLX は毎回新しい helper を起動する。retained model の warm reuse は実装されておらず測定不能。Ollama の正常系 5 回で報告された load duration 中央値は 1.33 秒。
- Ollama 正常系の prefill 中央値は 349.7 token/秒、decode 中央値は 34.3 token/秒。MLX helper は token 数と prefill/decode 時間を返さないため、この内訳は測定不能。
- 比較可能な peak unified memory telemetry はない。process RSS は GPU の割当を正確に表さないため、peak 値を報告しない。
- 固定版 `mlx-swift-lm` には生成文の prefix を診断ログに出す既知の未解決箇所がある。本計測は合成入力のみ。実データの benchmark と既定採用には、公開・固定された修正と実行時検証が必要。
- schema は commit 数と各 `file_ids` の配列長を入力ファイル数で制限するが、`scope` と `summary` の数値上限は未確定。`contextinput` の出力予約は `max(1024, 48 × files)`、MLX helper の上限は 8192。今回の失敗から安全な production limit は確定できない。#88 の Acceptance Criteria と日英 SRS はこの時点で変更していない。

## 再現

```sh
GOCACHE=/tmp/commiter-issue110-gocache go run ./tools/benchmark110 -describe -fixture all
GOCACHE=/tmp/commiter-issue110-gocache go run ./tools/benchmark110 -backend ollama -ollama-model ministral-3:3b-instruct-2512-q4_K_M -fixture normal -repeats 5
GOCACHE=/tmp/commiter-issue110-gocache go run ./tools/benchmark110 -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper mlx-helper/.build/release/commiter-mlx-helper -fixture normal -repeats 5
```

MLX モデルは取得前に明示確認を受け、上記 revision に固定して `-prepare-mlx-model` で検証・インストールした。Ollama モデルも同じ確認後に取得した。推論はローカルで実行した。

## 品質ゲート

`go test ./...`、`go vet ./...`、`swift test --package-path mlx-helper -c release`（5 tests）、MLX helper の `--smoke-metal` は成功した。Swift test は最初に sandbox の `sandbox-exec` 制限で manifest 検証に失敗したため、Metal が利用できる実行環境で再実行して成功した。
