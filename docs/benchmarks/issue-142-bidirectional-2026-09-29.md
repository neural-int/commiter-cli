# Issue #142: 二方向 pair 比較と決定的集約の実測結果

## 条件

[事前条件と実装](issue-142-bidirectional-preregistered-2026-09-29.md)をコミット `e99b692` で固定した後、前回の9合成 fixture と同じ候補集合を用いた。全21 unordered pair を `A,B`、`B,A`、`B,A`、`A,B` の順に4回ずつ、計84 call で比較した。候補IDと grouping の対応、および各 pair の JSON Schema enum 順は4 call で固定した。同方向の prompt hash は同じで、逆方向とは異なった。pair 内の schema hash は4 call で同じだった。

MLX model `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`を使用した。output 2048 tokens、call ごとに2分上限、repair/retry 0。新規依存、モデル取得、production planner、Pass 2、通常 CLI は実行していない。

## Fixture 別結果

pair 内訳の列は `stable / direction_disagreement / repeat_variation / incomplete` の順。確定候補は全ての他候補に stable edge で勝つ候補がある場合のみ記した。

| 区分 | fixture | 候補数 | pair 内訳 | 確定ID | 合成 gold 一致 | calls | wall |
| --- | --- | ---: | --- | --- | :---: | ---: | ---: |
| 既使用 | `verify_join_present` | 2 | 1 / 0 / 0 / 0 | C001 | ○ | 4 | 17.0s |
| 既使用 | `verify_split_present` | 3 | 3 / 0 / 0 / 0 | C001 | ○ | 12 | 54.5s |
| 既使用 | `verify_join_absent` | 3 | 2 / 1 / 0 / 0 | 未確定 | 未確定 | 12 | 46.7s |
| 既使用 | `verify_split_absent` | 3 | 2 / 1 / 0 / 0 | 未確定 | 未確定 | 12 | 64.3s |
| 既使用 | `verify_misleading_relation` | 2 | 1 / 0 / 0 / 0 | C001 | × | 4 | 16.3s |
| 前回 holdout | `holdout_crossdir_semantic` | 3 | 2 / 0 / 0 / 1 | C001 | ×（gold 候補なし） | 12 | 211.8s |
| 前回 holdout | `holdout_same_directory_pairs` | 3 | 3 / 0 / 0 / 0 | C001 | ○ | 12 | 69.9s |
| 前回 holdout | `holdout_spurious_test_link` | 3 | 3 / 0 / 0 / 0 | C002 | ○ | 12 | 60.6s |
| 前回 holdout | `holdout_atomic_feature` | 2 | 0 / 1 / 0 / 0 | 未確定 | 未確定 | 4 | 19.7s |

全21 pair の内訳は stable 17、direction_disagreement 3、repeat_variation 0、incomplete 1。完了した20 pair に限ると17 / 3 / 0。direction_disagreement の3 pair は、`verify_join_absent` と `verify_split_absent` の C001/C002、`holdout_atomic_feature` の C001/C002 で、いずれも4回答が各回に先に提示されたIDを選んだ。同方向の2回答が割れた完了 pair はなかった。

84 call のうち82件は `completed` かつ有効ID、2件は `max_tokens`。後者は共に `holdout_crossdir_semantic` の C002/C003 pair を逆順に提示した call（2回目と3回目）だった。この pair は incomplete とし、安定 edge を作らなかった。他の2 pair により C001 が C002 と C003 の両方に stable に勝ったため、fixture の確定IDは C001。全 call の output token telemetry は `unavailable`。

集約で候補が確定した fixture は6/9、未確定は3/9。安定 edge の cycle は0/9。合成 gold への exact 一致は全9 fixture で4/9、確定6 fixture では4/6、gold 候補が存在する8 fixture では4/8、gold 候補が存在して候補が確定した5 fixture では4/5だった。既使用5件は確定3/5・exact 2/5、前回 holdout 4件は確定3/4・exact 2/4。calls は既使用44、前回 holdout40、合計84。wall は各198.8秒、362.0秒、合計560.8秒。

前回の tournament 記録は29 pair calls、wall 322.2秒、gold 候補が存在した16試行中12試行で最終合成 gold 一致だった。今回は9 fixture あたり1つの確定候補または未確定を返す別の比較・集約条件で測定した。両結果は異なる日時の実行で、同じ分母の精度比較ではない。

## 記録と範囲

[結果 JSONL](issue-142-bidirectional-mlx.jsonl)は9行で fixture 名は一意、pair 21件、call 84件。pair status、stop reason、選択ID、prompt/schema hash、wall、集約結果を含み、生の prompt、response、repository content は含まない。今回は新規 holdout を追加していない。2回の同方向反復から反復変動率や候補位置の因果効果は推定していない。合成 gold と実開発者判断の一致、production 分布・精度は測定していない。

```sh
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe bidirectional -describe -fixture all
GOCACHE=/private/tmp/commiter-issue142-gocache go run ./tools/benchmark110 -issue142 -issue142-probe bidirectional -backend mlx -mlx-repo mlx-community/Ministral-3-3B-Instruct-2512-4bit -mlx-revision a962dcb09eee4169c890e544c9eb938f1113fdee -helper <上記 SHA-256 と一致する helper> -fixture all
```
