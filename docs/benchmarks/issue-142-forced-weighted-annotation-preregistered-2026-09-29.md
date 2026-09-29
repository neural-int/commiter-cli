# Issue #142: forced ranking・weighted lexical・relation annotation の事前登録

## 範囲と固定条件

- [修正済みの考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5889934887)に従い、3 probe を独立に測る。この文書を実装・モデル実行・新しい集計より前にコミットする。
- 前回までの合成 fixture 13件（既使用9件、新規作成4件）を用いる。結果を見て設計した fixture が含まれる探索的検証であり、実開発データ、無作為標本、盲検 holdout ではない。
- モデルは既存の MLX `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。新規依存・モデル取得は行わない。production planner、Document schema、Pass 2、通常 CLI は変更しない。
- output 2048 tokens、各 call 2分上限、retry/repair 0。生の path/diff/prompt/response は結果 JSONL に保存しない。

## 1. Forced-choice ranking

- `lexical_crossdir_pairs` は lexical 後 C001/C002/C003（gold C003）のみを selector に渡し、gold 不在 baseline は candidate recall failure として記録する。
- `lexical_collision` は baseline C001/C002（gold C001）と lexical 後 C001/C002/C003（gold C001）の両 arm を比較する。`lexical_paraphrase_gap` と `lexical_bridge` は両候補集合で gold 不在のため selector call をせず、候補生成結果のみ報告する。
- 各 arm で提示順を正順・逆順・逆順・正順の4回。3 arm × 4 = 12 call。candidate ID/grouping、repository input、モデル、output、system/task、JSON Schema enum の ID 順は arm 内で固定。`none` は指示と enum から除く。arm の先行順は反復ごとに交互にする。
- fixture/arm ごとの完了、有効候補ID、gold 選択、停止理由、wall、prompt/schema hash を記録する。候補が2件と3件の比較は候補数・内容と時点が異なるため、候補追加の単独因果効果とは呼ばない。未完了は誤選択に数えず、全 call 分母と有効回答分母を別に示す。

## 2. Weighted lexical evidence の探索

- 対照は既存の二値 lexical rule: path + diff から英字 token を分割、小文字化、4文字未満と固定 stopword を除外、共有 token 1つ以上を edge とし、連結成分から partition を1件作る。
- 固定する weighted variant: 各 token の fixture 内 document frequency を df（その token を持つ file 数）とし、共有 token の重みを `1/(df-1)` とする。file pair の score は共有 token の重みの和。`score >= 0.75` と `score >= 1.25` の2つの閾値について、edge と連結成分の partition を1件ずつ作る。境界を含む。新たな stopword、token-source 補正、community detection はこの probe に加えない。
- 各 fixture の既存候補に、二値 arm は既存 lexical partition 1件、weighted arm は閾値0.75、1.25の順に最大2件を末尾追加する。canonicalize/dedupe、既存候補のID/順序、全体 cap8は維持。二値 arm と weighted arm はそれぞれ既存候補から開始し、互いの候補を混ぜない。
- 既使用13件の fixture 別に、gold 包含、候補数、新規 gold/非 gold 候補、cap 到達、各閾値の edge の gold 内/間、partition exact を記録する。追加候補数が異なるので、同じ候補予算の制御実験とは解釈しない。モデル ranking はこの weighted probe では行わない。

## 3. Relation annotation

- `verify_misleading_relation`、`verify_join_present`、`holdout_spurious_test_link` の既存 C001/C002 pair を用いる。候補は current relation context を含む入力から先に生成して両 arm で固定。
- current arm の complete-partition forced-choice prompt を対照とし、annotation arm では `repository_input.relation_context` object に文字列フィールド `relation_interpretation: "structural_hint_not_shared_purpose_proof"` を1つ追加する。他の prompt field、system/task、relation の元データ、schema、候補 ID/grouping、output 設定は同一。検証専用 prompt のみで production Document schema は変更しない。
- 各 fixture・arm を正順・逆順・逆順・正順の4回ずつ、3 × 2 × 4 = 24 call。arm の先行順は反復と fixture に応じて交互にする。完了、有効候補ID、gold 選択、方向別回答、停止理由、wall、prompt/schema hash を記録する。入力差分が上記1 field のみであることをテストする。少数の合成例であり、内部機序や production の一般的な有効性は推定しない。

## 記録

- 実行順は weighted generator、forced ranking、relation annotation。中断・失敗は記録して除外理由を明記する。結果からルールや閾値を変更して同じ fixture で再評価しない。
