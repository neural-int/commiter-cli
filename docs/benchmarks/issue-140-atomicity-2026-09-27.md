# Issue #140: 独立目的の分離指示を追加した最終比較（2026-09-27）

## 条件

- 推論前に[比較条件と正解ラベル](issue-140-atomicity-labels.md)を `fe103f3` で固定した。既存4 fixture と新規 `holdout_split` / `holdout_join` の計6 fixture を使用した。
- 現行 production と同じ `full` に対し、benchmark 専用の `full+atomicity` は `constraints.grouping` の末尾に `Put files with independent change purposes in separate commits.` の一文だけを追加した。テストでその一文以外の prompt JSON と file ID が一致することを確認した。出力 schema は同じ。
- 各 fixture × 2 arm × 2回の24試行。2巡目は arm 順を逆にした。既存 Release helper（SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`）とローカル MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee` を使用。生成上限1024 token、各試行の timeout は2分。新規ダウンロードとクラウド API は使っていない。
- exact grouping は commit 順序を問わない file ID 集合で判定。false merge / false split は誤った file pair の数。失敗した plan には grouping を採点しない。

## 結果

下表の `full → +atomicity` は各2試行の比較。false merge / false split は**有効な plan のみ**の合計であり、`mixed_24` の追加文側は欠測とする。

| fixture | 有効 plan | exact grouping | false merge | false split | backend calls | repair calls | wall time 合計 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `multi_commit` | 2/2 → 2/2 | 0/2 → 0/2 | 8 → 8 | 0 → 0 | 2 → 2 | 0 → 0 | 16.8s → 15.4s |
| `cross_directory` | 2/2 → 2/2 | 2/2 → 2/2 | 0 → 0 | 0 → 0 | 2 → 2 | 0 → 0 | 12.1s → 11.6s |
| `same_directory_independent` | 2/2 → 2/2 | 0/2 → 0/2 | 2 → 2 | 0 → 0 | 2 → 2 | 0 → 0 | 13.2s → 12.2s |
| `mixed_24` | 2/2 → 0/2 | 0/2 → 欠測 | 288 → 欠測 | 0 → 欠測 | 2 → 4 | 0 → 2 | 66.2s → 225.6s |
| `holdout_split` | 2/2 → 2/2 | 0/2 → 0/2 | 8 → 8 | 0 → 0 | 2 → 2 | 0 → 0 | 24.1s → 24.0s |
| `holdout_join` | 2/2 → 2/2 | 2/2 → 2/2 | 0 → 0 | 0 → 0 | 2 → 2 | 0 → 0 | 20.7s → 20.6s |

全体では `full` が有効 plan **12/12**、exact grouping **4/12**、backend calls **12**、repair calls **0**、wall time **153.2秒**。`full+atomicity` は有効 plan **10/12**、exact grouping **4/10（2試行は欠測）**、backend calls **14**、repair calls **2**、wall time **309.4秒**。追加文により各 prompt は64 byte 増えた。

`mixed_24` の追加文側は2試行とも初回と repair 後の検証で `invalid_assignment` だった。各 backend 応答の stop reason は `completed` であり、timeout や出力打ち切りとしては分類されなかった。修復を通しても `planning.Validate()` の割当契約を満たさなかったため、不正な plan は採用していない。その他の有効 plan では `full` と追加文側の file grouping は同一で、有効 plan が得られた分離正解の3 fixture はいずれも過剰結合のままだった。結合正解の2 fixture には false split は発生しなかった。

試行別の grouping、prompt bytes、修復、所要時間、失敗分類は[生データ](issue-140-atomicity-mlx.jsonl)に保存した。生成された plan 本文、prompt 本文、出力 token 数は保存していない。MLX adapter から output token 数を取得できないため、JSONL では `unavailable` と記録した。

## 判断

事前に定めた「分離例で false merge を減らし、結合例で false split を増やさず、完全割当を維持する」という条件を、この一文は満たさなかった。既存・新規の分離例で改善はなく、24-file では有効 plan が **2/2 から 0/2** に悪化した。この追加文を production prompt に採用する根拠はない。

この結果は、検証した一文と単一のローカル model、少数の合成 fixture に限る。prompt/input の別の調整全般が無効だとは言えず、Issue #139 の設計変更を自動的に正当化するものでもない。次の判断では、どの誤結合を許容しないかと、追加の機械的な境界付けが必要かを別途検討する。今回の検証では production の planning 指示と実装は変更していない。

## 再実行と検証

既存のローカル model と上記 SHA-256 の helper を用いる。

```sh
GOCACHE=/tmp/commiter-issue140-gocache go run ./tools/benchmark110 \
  -issue140 -issue140-probe atomicity -backend mlx \
  -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit \
  -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee \
  -helper <helper-path> -fixture all -repeats 2 \
  -timeout 2m -output-tokens 1024
```

`go test ./...`、`go vet ./...`、`git diff --check` を通した。
