# Issue #167 の Phase 2 比較

合成 fixture repository だけを作成し、既存 collector / syntax / document から同じ入力を両 planner へ渡す。既存の公開 CLI・default の上限は変更しない。参照 purpose はモデルに渡さず、生成 prompt・content・stderr を artifact に保存しない。プロトコル double のテスト値をモデル測定として利用しない。

推論前に `docs/benchmarks/issue-167/phase-2-preregistration.md` と fixture manifest を commit する。単体 regression / 原Git状態保持が成功しても実モデルの Gate 2 を満たしたとは扱わない。

```sh
go build -o /tmp/commiter-167-benchmark ./tools/benchmark167
/tmp/commiter-167-benchmark -manifest
python3 tools/benchmark167/prepare_helper.py mlx-helper /tmp/new-issue167-helper
# 同じ Package.resolved の保存済み依存で計測コピーをビルドする。
/tmp/commiter-167-benchmark \
  -helper /tmp/new-issue167-helper/.build/release/commiter-mlx-helper \
  -cache '/path/to/already-installed/mlx-models' > phase-2-measurements.jsonl
```

`-fixture` は事前登録の1 workloadだけを実行する。順序・反復数・上限は flag で変更しない。model の用意は `Store.Ready()` の確認だけで、新規 download はしない。各 run の wall time は前後の構造 replay と metadata を含み、fixture生成・collection・syntax分析・compileは含まない。stage/call時間を別保存する。first-observed と filesystem-warm は process/model cold の条件下での OS cache観測であり、常駐 model warm / storage cold の計測ではない。

helper 数値 telemetry は固定 generation 設定を変えず、input/output tokens、load、runtime first-token時間、MLX peak を追加する。macOS `/usr/bin/time -l` から helper peak RSS を取得する。pressure と swap の前後観測は共存負荷の影響を含み、当該 run の因果的メモリ消費とは呼ばない。
