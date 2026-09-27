# Issue #141: file-centric Pass 1 の事前比較条件

この文書と benchmark コードを推論前にコミットする。#141 の[最初の比較条件](issue-141-preregistered-2026-09-28.md)で固定した6 fixtureと正解 groupを再利用し、full、group-centric Pass 1、file-centric Pass 1 の3条件を同じ実行内で比較する。

| fixture | 正解 group | 用途 |
| --- | --- | --- |
| multi_commit | F001,F002 / F003,F004 | 分離 |
| cross_directory | F001,F002 | 結合 |
| same_directory_independent | F001 / F002 | 分離 |
| mixed_24 | 奇数 ID 12件 / 偶数 ID 12件 | 24 file の分離と完全割当 |
| holdout_split | F001,F002 / F003,F004 | 分離 |
| holdout_join | F001,F002,F003 | 結合 |

## file-centric 契約

Pass 1 は各 file ID を key、group label を value とする assignments object のみを返す。JSON Schema にはすべての file ID を required として列挙し、additionalProperties を禁止する。Go は生の応答から JSON object の重複 key を Unicode escape 解釈後も拒否し、未知 ID、欠落 ID、空・不正な group label を検出する。通常の map decode や backend の schema 制約を exactly-once assignment の唯一の根拠にはしない。group label の名前自体は正解の意味に影響しない。

repository input と relation context は既存の full と同じ contextinput.Prepare 済み Document から作る。Pass 1 では type/scope/summary/breaking を要求しない。構造的失敗には1回だけ repair を許し、有効だが誤った grouping は repair しない。Pass 1 が完全割当かつ正解 grouping の場合だけ、先行検証と同じ Pass 2 を実行し、Go で元の group に metadata を結合して planning.Validate() する。

## 実行条件と判定

- 6 fixture × 3条件 × 2回の36試行。2回目は条件の順序を逆にする。前回の12試行/armと同じ反復数。
- ローカル MLX model は mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee、既存 helper SHA-256 は da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48。追加ダウンロードとクラウド API は使わない。
- 各 backend request の output 上限1024 token、plan cycle 全体の timeout は2分。モデル・backend・fixture・証拠・Pass 2・修復回数・採点法は既存比較と同じ。
- 主指標は Pass 1 の完全割当、exact grouping、false merge / false split pair、per-file assignment accuracy。構造的失敗と有効割当の semantic grouping failure を分け、不正な割当には grouping 指標を付けない。
- 補助指標は Pass 1/Pass 2/repair/transport retry/総 backend calls、wall time、prompt/output bytes、stop reason。output token count が取れない場合は unavailable と記録する。

採用候補となるには、分離4 fixture で有効な割当の false merge が減り、結合2 fixture の false split を増やさず、完全割当を維持する必要がある。構造的失敗が残る場合は semantic grouping の限界とは判定しない。構造的に有効な割当でも過剰結合が続く場合は、この model/evidence/prompt/budget における直接 partition の品質不足と判断する。2回ずつの合成 fixture だけで production 採用を確定しない。

file-centric の schema サイズや prompt 文面は出力表現とともに変わる。mixed_24 の初回 prompt は full 15,720 bytes、group-centric 14,123 bytes、file-centric 15,281 bytes と事前確認した。結果の差を「構造だけ」の純粋な因果効果と断定しない。production planner、SRS、通常 CLI は変更しない。
