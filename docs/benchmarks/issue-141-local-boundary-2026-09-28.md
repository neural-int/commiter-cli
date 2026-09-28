# Issue #141: 局所 boundary 判定の検証結果

## 結論

2ファイルの変更内容に焦点を当てた局所判定は、今回の8 fixture・2反復では guardrail の誤結合を避けたが、主比較で false split が大きく増え、`multi_commit` では false merge も生んだ。**この benchmark contract は production 候補としない。** 候補生成、判定、結合を分離して測った結果、24-file の失敗は候補連結性の欠落だけでは説明できない。小標本・合成 fixture・固定モデルであり、局所 boundary 判定全般の不可能性は示さない。

## 条件

[推論前に固定した条件と実装](issue-141-local-boundary-preregistered-2026-09-28.md)は `07d8108`。前回と同じ6主 fixture・2 guardrail、MLX model revision `a962dcb09eee4169c890e544c9eb938f1113fdee`、Release helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`、1024 output tokens/request、2分/試行、2反復を使った。Pass 2 は実行していない。今回の prompt は2ファイルの path / language / raw diff / summary のみを提示し、relation 種別や全体 graph は提示しない。8 pair ごとの batch 判定を Go が検証し、true pair の推移的結合で grouping を作った。

## 主比較

以下の既存3条件は[前回の同一 fixture・モデルの結果](issue-141-edge-and-keyed-2026-09-28.md)であり、今回は再測定していない。local-boundary の wall / prompt は各 pair に絞った別形式の入力であり、単純な速度比較はしない。

| 条件（6 fixture × 2反復） | 完全割当 | exact grouping | false merge pair | false split pair | calls | 合計 wall |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| file-centric | 12/12 | 4/12 | 0 | 16 | 12 | 179.2s |
| soft edge を全て hard 縮約 | 12/12 | 8/12 | 0 | 6 | 12 | 166.8s |
| edge accept/reject | 12/12 | 8/12 | 0 | 6 | 20 | 212.9s |
| **local-boundary** | **12/12** | **6/12** | **4** | **214** | **22** | **104.1s** |

今回の局所判定は主比較124候補 pair を評価し、TP 34、TN 60、FP 4、FN 26。候補判定の正解は94/124だが、grouping では false split が214 pairに拡大した。1つの正解 group をつなぐ肯定判定が欠けると、その後の推移的結合では補えないためである。全12試行で構造的失敗はなく、全 file ID の完全割当を保った。

| fixture（各2反復） | exact | false merge | false split | 候補 pair / run | TP / TN / FP / FN（2反復計） |
| --- | ---: | ---: | ---: | ---: | ---: |
| `multi_commit` | 0/2 | 4 | 2 | 6 | 2 / 4 / 4 / 2 |
| `cross_directory` | 2/2 | 0 | 0 | 1 | 2 / 0 / 0 / 0 |
| `same_directory_independent` | 2/2 | 0 | 0 | 1 | 0 / 2 / 0 / 0 |
| `mixed_24` | 0/2 | 0 | 208 | 45 | 24 / 46 / 0 / 20 |
| `holdout_split` | 2/2 | 0 | 0 | 6 | 4 / 8 / 0 / 0 |
| `holdout_join` | 0/2 | 0 | 4 | 3 | 2 / 0 / 0 / 4 |

`mixed_24` は各反復で真候補22・偽候補23。真候補22は正解2 group をそれぞれ連結できる構成だったが、モデルは各反復で真候補10件を別目的と判定した。真の全 pair 132件に対する候補 pair recall は22/132であり、連結性は満たすが pair 網羅率は低い。失敗は候補連結性の欠落ではなく、連結に必要な candidate の false negative と、合成 fixture の正解ラベルをモデルが読み取れるかという問題が重なっている。`multi_commit` は2つの別目的 pair を各反復で誤って同一目的と判定し、false merge 2 pair/反復となった。

## Guardrail と制約

| 条件（2 guardrail × 2反復） | exact | false merge | calls |
| --- | ---: | ---: | ---: |
| file-centric（前回） | 4/4 | 0 | 4 |
| soft edge 全強制縮約（前回） | 0/4 | 4 | 4 |
| edge accept/reject（前回） | 0/4 | 4 | 8 |
| **local-boundary** | **4/4** | **0** | **4** |

guardrail の偽 relation は4/4で別目的と判定できた。主比較の `multi_commit` には別目的 pair の false positive が4件あるため、「誤結合を避けられる」と一般化できない。今回の全試行で、否定した pair が別の肯定経路により同一 cluster に入る `negative_closure` は0件だったが、推移的結合の安全性保証ではない。

`mixed_24` は同じ stem を持つ12ファイルを同一目的とする合成ラベルで、個々の差分は異なる関数を追加する短い行である。このラベルを prompt から再構成しにくい可能性があり、結果は**このラベル集合との一致**として読む。gold label を候補生成や prompt には使っていない。

## 再現と判断

[数値・file ID のみの16行](issue-141-local-boundary-mlx.jsonl)を記録した。各 `(fixture, run, variant)` は一意。主比較の prompt bytes 合計58,820、guardrail 4,070、出力 bytes は主比較2,742・guardrail168。output token 数は backend から取得できず全行 `unavailable`。モデルの raw 応答、prompt、生成 summary は保存していない。

```sh
GOCACHE=/tmp/commiter-issue141-gocache go run ./tools/benchmark110 -issue141 -issue141-probe local-boundary -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <helper-path> -fixture all -repeats 2 -timeout 2m -output-tokens 1024
```

8 fixture では候補連結性を満たし、全ての response は構造検証を通った。それでも主比較の false merge と false split を両立できない。#141 の局所 boundary 案を今回のまま拡張せず、候補集合・gold label の妥当性・3B model に与える目的情報を再評価するのが次の設計判断となる。Pass 2 keyed batch の動的 grouping 接続は本試験では未実施であり、独立した残課題である。production planner / SRS / 通常 CLI、新しい依存・モデルは変更していない。
