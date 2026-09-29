# Issue #142: 新規合成 holdout と ranking / relation context 切り分けの事前登録

## 固定条件と解釈範囲

- 2026-09-29 に、下記 fixture と合成 gold、比較 arm、測定方法をモデル実行および新 fixture の生成結果を見る前に固定する。前回の lexical 規則（path + diff の英字分割、4文字以上、固定 stopword、共有 token 1個以上、連結成分、重複排除、候補上限8）は変更しない。
- 新 fixture は既存9件の結果を見た後に人手で作った合成例であり、実開発データからの無作為標本でも独立した盲検 holdout でもない。gold はモデル入力に含めない。
- 既存の pinned MLX 3B モデル、revision a962dcb09eee4169c890e544c9eb938f1113fdee、helper SHA-256 da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48 を再利用する。新規依存・モデル取得・production planner・Pass 2 への変更は行わない。

## 新規合成 fixture と gold

1. lexical_crossdir_pairs: src/queue/dispatch.go は backfill job を dispatch、docs/runtime/operations.md は backfill の運用を説明、src/accounts/update.go は profile update、docs/account/guide.md は profile update の手順を説明。gold は {F001,F002} と {F003,F004}。
2. lexical_paraphrase_gap: src/storage/evict.go は古い entry の削除、docs/guide/retention.md は一時 record の期限切れ規則、src/alerts/notify.go は alert の送信。gold は {F001,F002} と {F003}。関連2ファイルに明示的な共通語がない条件。
3. lexical_collision: src/audit/record.go は export 操作の監査記録、docs/report/export.md は report export の書式、src/alerts/notify.go は通知。gold は {F001}, {F002}, {F003}。共有語があっても別目的の条件。
4. lexical_bridge: src/auth/session.go と docs/security/auth.md は session token renewal、src/metrics/usage.go と docs/ops/metrics.md は API token usage の計測。gold は {F001,F002} と {F003,F004}。token の共有による目的間の連結を調べる条件。

## Generator

- 上記4件を既存候補生成と既存 + lexical で比較する。fixture 順序と file ID は記載順に固定。
- fixture 別に候補数、gold 包含、追加 candidate ID と grouping、cap 到達、lexical partition の exact、lexical edge の gold 内 / gold 間、追加された非 gold 候補数を記録する。集計は分母を明示し、既使用9件との合算を主要結果としない。

## Ranking

- 4件すべてについて、既存候補と lexical 後候補をそれぞれ comparative selector に渡す。候補が同じでも両 arm を実行し、同時期の対照とする。gold 不在 arm は none を正解扱いにし、gold 選択と正解判定を別に記録する。
- 同一 fixture で repository input と relation context、既存候補の ID / grouping / 順序を固定。lexical 候補は末尾追加。候補表示の正順と逆順を各1回、arm の実行順を fixture ごとに交互とする。JSON Schema enum は各 arm 内で正順固定。
- 4 fixture × 2 arm × 2方向 = 16 call。output 2048 tokens、各 call 2分上限、retry/repair 0。完了、有効 ID、none、gold、正解、stop reason、calls、wall、prompt/schema hash を記録。候補数が変わる場合、比較は候補追加の複合効果であり、単独の語彙因果効果とは呼ばない。

## Relation context removal

- 既使用の verify_misleading_relation、verify_join_present、holdout_spurious_test_link の各 C001/C002 pair を対象とする。relation context 付きの prepared input から候補を先に生成し、固定した pair を両 arm へ渡す。削除 arm は repository_input.relation_context フィールドだけを省略する。
- system/task、他の repository input、候補 ID/grouping/順序、schema enum、model/output 設定を固定する。各 arm で正順、逆順、逆順、正順を実行し、arm の先行順を交互にする。3 fixture × 2 arm × 4 = 24 call。
- 完了、有効 ID、gold 選択、方向別回答、stop reason、wall、prompt/schema hash を記録する。期待する prompt 差が relation_context フィールドだけかをテストする。合成 fixture 内の効果を測り、原因の一般化や production の採用判断には用いない。

## 記録

- 生の path/diff/prompt/response を結果 JSONL に保存しない。fixture 名、候補 grouping、共有 token、出力 ID、計測値のみ保存する。呼び出しは generator → ranking → relation context removal の順。失敗も結果に含め、事後除外しない。
