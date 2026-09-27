# Issue #141: grouping-first 比較の事前条件

この文書と検証コードを推論前にコミットする。#140 の最終検証と同じ issue140AtomicityFixtures() の6 fixture、事前に固定した正解 group を使う。対象は合成 diff のみ。production の planner、schema、SRS は変更しない。

| fixture | 正解 group | 判定 |
| --- | --- | --- |
| multi_commit | F001,F002 / F003,F004 | 分離 |
| cross_directory | F001,F002 | 結合 |
| same_directory_independent | F001 / F002 | 分離 |
| mixed_24 | 奇数 ID 12件 / 偶数 ID 12件 | 分離と完全割当 |
| holdout_split | F001,F002 / F003,F004 | 分離 |
| holdout_join | F001,F002,F003 | 結合 |

## 比較

- full: #140 と同じ production-valid relation context、planning.Generator、planning.Validate()。
- grouping-first: 同じ contextinput.Prepare 済み repository input を使う。Pass 1 は group_id と file_ids の partition のみを生成し、Go 側で JSON/schema、空 group、未知 ID、重複、欠落を検査する。初回の構造的失敗には1回だけ repair を許す。semantic grouping error は repair しない。
- Pass 1 の結果が**完全割当かつ正解 grouping**のときだけ、固定された group の type/scope/summary/breaking を1回の bounded Pass 2 request で生成する。Pass 2 は file ID を出力せず、Go が元の partition に metadata を結合した後 planning.Validate() する。group を変更できない。
- 各 fixture を2回測り、2回目は arm 順を逆にする。モデルはローカル MLX mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee。helper SHA-256 は da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48。各 request の生成上限1024 token、各 plan cycle の wall-time 上限2分とする。新規モデルは取得しない。

主指標は Pass 1 と full の exact grouping、false merge pair、false split pair、complete assignment、per-file assignment accuracy。structural failure と semantic grouping failure を区別する。成功 plan のみの false merge だけで判断せず、失敗件数を併記する。Pass 2 は正解 Pass 1 のみを母数とし、metadata validation 成功、grouping boundary 維持、backend call 数、repair 回数、wall time、prompt/output bytes を記録する。MLX が output token count を返さない場合は unavailable と記録する。

採用候補とみなすには、分離例で false merge が減り、結合例の false split を増やさず、完全割当を維持し、Pass 2 の既存 validation を通る必要がある。2回ずつの合成 fixture と単一モデルだけで production 採用を決めない。Pass 2 の group ごとの個別 request と、production の call budget は今回の1バッチ probe の結果を踏まえ別途判断する。

## 前提レビューで明確にした境界

- #140 が否定したのは**試した** relation/guidance/statistics/atomicity の入力条件であり、single-pass prompt tuning 全般の不可能性ではない。
- 先行 provisional intent は「完全な自由形式」ではなく、title と evidence_file_ids を要求する JSON schema だった。1024/2048 token の両条件で6件中5件が未完了で、唯一完了した cross_directory では evidence の重複があった。これは当該 prompt/model/budget の結果に限る。
- #140 の atomicity 付き mixed_24 は invalid_assignment で、grouping 指標は欠測である。「2 group に分けられなかった」と数値化できるのは有効な full 側だけ。今回も不正 plan は false merge に換算しない。
