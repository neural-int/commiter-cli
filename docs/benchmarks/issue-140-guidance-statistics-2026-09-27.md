# Issue #140: guidance / statistics の4条件比較（2026-09-27）

## 条件

- [先行する #140 の検証](issue-140-2026-09-27.md)と同じ `many_files`（独立した24ファイル）、MLX model revision `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、Release helper（SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`）を使用した。新規モデルや依存のダウンロードはない。
- 各条件を2回実行し、2巡目は実行順を逆にした。1試行の上限は生成1024 token、wall time 2分。wall time には helper 起動と model load を含む。MLX adapter は出力 token 数を返さないため `unavailable` とする。
- `baseline` は既存の relation context なしの prompt と同一。ほかの3条件は benchmark 専用 renderer により、relation-aware guidance と graph statistics の有無だけを切り替える。`guidance+statistics` は先行検証の `guidance-stats-only` と同じ prompt。部分 relation context は production の `Document` 検証を満たさないため、各条件の結果は製品入力をそのまま変更できる根拠ではない。
- 完全割当は `planning.Validate()` を通った全 file ID の一意割当を指す。`many_files` には正解 grouping がないため、commit 分割の品質はこの比較では判定しない。

## 結果

| 条件 | 完全割当 | 失敗分類 | backend calls | repair calls | wall time（秒） | prompt bytes | 推定 input tokens | 成功時の group 数 |
| --- | ---: | --- | --- | --- | --- | ---: | ---: | ---: |
| baseline | 0/2 | `invalid_assignment` ×2 | 2, 2 | 1, 1 | 63.8, 85.9 | 13,780 | 15,188 | — |
| guidance-only | 2/2 | なし | 1, 1 | 0, 0 | 22.4, 31.5 | 14,083 | 15,491 | 1, 1 |
| statistics-only | 2/2 | なし | 1, 1 | 0, 0 | 24.0, 35.0 | 14,137 | 15,545 | 1, 1 |
| guidance+statistics | 2/2 | なし | 1, 1 | 0, 0 | 24.5, 27.7 | 14,418 | 15,826 | 1, 1 |

8試行すべての backend retry は0。成功した6試行はすべて、24ファイルを1つの commit に割り当てた。出力 token 数はすべて `unavailable`。試行別の出力 byte 数、stop reason、失敗分類は[生データ](issue-140-guidance-statistics-mlx.jsonl)に記録した。prompt と生成された plan 本文は保存していない。

## 判断

この固定 fixture・model・backend・budget では、guidance 単独と statistics 単独のどちらも完全割当が `2/2` だった。両者を同時に含めることが成功の必要条件という先行する仮説は支持されない。また、今回の完全割当に対して一方だけを主因とは特定できない。

成功は全ファイルを1 commit にまとめた結果であり、独立変更の分割品質が改善したことは示さない。各条件2試行で prompt 長と内容も変わるため、効果の再現性や作用機序を一般化しない。次の判断には、正解 grouping のある複数 fixture で完全割当と false merge / false split を併せて測る必要がある。production の入力契約と planner architecture は今回変更しない。

## 再実行

正解 grouping を付けた追加比較は[grouping 検証](issue-140-grouping-2026-09-27.md)を参照。

既存のローカル model と Release helper を用いる。`<helper-path>` は上記 SHA-256 のバイナリのパスに置き換える。

```sh
GOCACHE=/tmp/commiter-issue140-gocache go run ./tools/benchmark110 \
  -issue140 -issue140-probe guidance-statistics -backend mlx \
  -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit \
  -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee \
  -helper <helper-path> -fixture many_files -repeats 2 \
  -timeout 2m -output-tokens 1024
```
