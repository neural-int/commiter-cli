# Issue #140: 正解 grouping 付き4条件と production `full` の比較（2026-09-27）

## 条件

- 推論前に[正解 grouping と判定基準](issue-140-grouping-labels.md)をコミット `8a54a9b` で固定した。`multi_commit`、`cross_directory`、`same_directory_independent`、新規 `mixed_24` の4 fixture を使用した。`mixed_24` は同一ディレクトリ内の login rule 12ファイルと report rule 12ファイルを交互に並べ、正解を2 group とした。
- `baseline`、`guidance-only`、`statistics-only`、`guidance+statistics` に production の入力契約を満たす `full` を加えた。前者3つの部分 relation context は benchmark 専用で、そのまま製品に投入できない。`full` は既存の renderer と context preparation を通る。
- 各 fixture × arm を2回測り、2巡目は arm の実行順を逆にした。既存 Release helper（SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`）とローカル MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee` を使用。生成上限1024 token、各試行2分。新規ダウンロードやクラウド API は使用しない。
- 完全割当と `planning.Validate()` の成功を grouping の正しさと分けて記録する。false merge / false split は誤った file pair の数。出力 token 数は MLX adapter から取得できず、`unavailable` とする。

## 結果

全40試行で完全割当と `planning.Validate()` は成功した。下表の値は **各 arm の2試行とも同じ**。`B / G / S / G+S / F` は順に baseline / guidance-only / statistics-only / guidance+statistics / full。

| fixture | 正解 group 数 | exact grouping match B/G/S/G+S/F | false merge（各試行）B/G/S/G+S/F | false split（各試行）B/G/S/G+S/F |
| --- | ---: | --- | --- | --- |
| `multi_commit` | 2 | 0/2 / 0/2 / 0/2 / 0/2 / 0/2 | 4 / 4 / 4 / 4 / 4 | 0 / 0 / 0 / 0 / 0 |
| `cross_directory` | 1 | 2/2 / 2/2 / 2/2 / 2/2 / 2/2 | 0 / 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 / 0 |
| `same_directory_independent` | 2 | 0/2 / 0/2 / 0/2 / 0/2 / 0/2 | 1 / 1 / 1 / 1 / 1 | 0 / 0 / 0 / 0 / 0 |
| `mixed_24` | 2 | 0/2 / 0/2 / 0/2 / 0/2 / 0/2 | 144 / 144 / 144 / 144 / 144 | 0 / 0 / 0 / 0 / 0 |

分離が正解の3 fixture では、すべての arm が全ファイルを1 group にした。`mixed_24` の false merge 144 は、login 12ファイルと report 12ファイルの異なる group 間にできた `12 × 12` 個の誤った file pair を表す。`cross_directory` はすべての arm で関連する2ファイルを正しく結合した。

| arm | 完全割当 | exact grouping match | backend calls 合計 | repair calls 合計 | wall time 合計（秒） |
| --- | ---: | ---: | ---: | ---: | ---: |
| baseline | 8/8 | 2/8 | 8 | 0 | 112.2 |
| guidance-only | 8/8 | 2/8 | 10 | 2 | 217.8 |
| statistics-only | 8/8 | 2/8 | 8 | 0 | 113.3 |
| guidance+statistics | 8/8 | 2/8 | 8 | 0 | 120.2 |
| full | 8/8 | 2/8 | 8 | 0 | 130.5 |

`guidance-only` の `mixed_24` は両試行で初回が `invalid_assignment` となり、1回の repair 後に完全割当へ回復した。ただし最終 grouping は1 group で不正解だった。各 arm・fixture の prompt bytes、wall time、生成 byte 数、request ごとの検証結果は[生データ](issue-140-grouping-mlx.jsonl)に記録した。prompt や生成 plan 本文は保存していない。

## 判断

4つの診断条件と production で有効な `full` の間に、今回の labeled grouping 品質の差は見られなかった。特に `full` は分離が必要な3 fixture で baseline の false merge を減らせず、事前に定めた「baseline の誤りを残したままなら品質改善とは判断しない」基準を満たさない。guidance / statistics の単独投入を production grouping 改善案とする根拠も得られなかった。

先行する `many_files` 24ファイルでは baseline の完全割当は `0/2` だったが、正解 grouping を付けた今回の `mixed_24` では baseline も `2/2` だった。ファイルの内容・パス・配置が異なるため、先行する完全割当の改善シグナルは24ファイル一般へ広げられない。今回の試行では、完全割当は全 arm で上限に達し、品質比較の差は過剰結合に現れた。

各条件2試行、単一 model / backend、合成 fixture の結果である。`mixed_24` は意図を名前に明示した人工的な入力で、実変更の意味判断を代表しない。wall time は helper 起動と model load を含み、出力 token 数や計算資源量は不明。今回の結果だけで一般的な効果の不在を証明せず、production の入力契約・planner architecture は変更しない。

## 再実行

既存のローカル model と上記 SHA-256 の helper を用いる。

```sh
GOCACHE=/tmp/commiter-issue140-gocache go run ./tools/benchmark110 \
  -issue140 -issue140-probe grouping -backend mlx \
  -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit \
  -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee \
  -helper <helper-path> -fixture all -repeats 2 \
  -timeout 2m -output-tokens 1024
```
