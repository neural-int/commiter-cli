# Issue #142: deterministic candidate pre-ranking と top-2 selector の事前登録

## 対象・制限

- [修正済み考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5890415686)に従い、既使用の13合成 fixtureを使う。前回の既存・二値 lexical・weighted lexical の候補生成規則と上限8を変えない。モデル実行・score結果確認前に本書と検証コードを別々にコミットする。
- 13件はすべて既使用の合成例で、gold と直近の失敗例を見た後にこの score を定めた。これは探索的な fit 検査であり、未知入力での精度推定や production 方針の確定に用いない。
- モデルを呼ぶ条件では既存の pinned MLX 3B `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee` と helper SHA-256 `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48` を再利用する。新規依存・モデル取得、production planner、Pass 2、通常 CLI は変更しない。

## 決定的 score と recall@K

- File pair の support は、relation context の soft source-test / matching_test_path に +4、soft direct-import / observed_import_path に +3、basename stem 一致に +1、同一ディレクトリに +0.1、前回固定した lexical token の共有 score（各 token は `1/(df-1)`、同一 fixture の file document frequency df）を加える。該当しない pair は0。
- Candidate score は、その partition が同一 group に置く全 unordered file pair について `support(pair) - 0.5` を合計する。別 group の pair は0。合成 gold、候補 ID、candidate provenance は score に使わない。
- score 降順、同点は candidate ID 昇順で順位付けする。候補集合ごとに `recall@1`、`recall@2`、`recall@3`、total recall、gold rank、top-2 ID、候補数、追加非 gold 候補を記録する。Kが候補数を超える場合は存在する全候補を対象にする。全13件分母と、gold が生成されている fixture の条件付き分母を別に集計する。
- 既存・二値 lexical・weighted lexical の3集合はそれぞれ独立に既存候補から構築する。候補の canonicalization、dedupe、cap8、IDと grouping は前回と同じ。二値と weighted の candidate family を混ぜない。

## Top-2 forced-choice

- 対象 arm を次の6つに固定する: `holdout_crossdir_semantic` / binary、`lexical_crossdir_pairs` / binary、`lexical_crossdir_pairs` / weighted、`lexical_bridge` / weighted、`lexical_collision` / weighted、`verify_misleading_relation` / weighted。
- 各 arm で score 上位2候補に合成 gold が存在する場合のみ selector を呼ぶ。gold が top-2 にない場合は pruning failure として記録し、selector の accuracy 分母には入れない。候補数はすべて2以上であることを確認する。
- 2候補の generic forced-choice 指示を使い、`none` を含めない。各 arm は正順・逆順・逆順・正順の4 call。最大24 call。各 arm の candidate ID/grouping、repository input、system/task、schema enum ID順、model/output を固定する。output 2048 tokens、各 call 2分上限、retry/repair 0。
- fixture/arm 別に停止理由、有効 ID、合成 gold 選択、方向別回答、calls、wall、prompt/schema hash を記録する。候補集合が異なる arm 間の差は候補数・内容を含む複合差として扱う。

## 条件付き pairwise fallback

- weighted 候補集合で total recall が true なのに top-2 recall が false の fixture が1件以上あれば、固定 fixture 順の先頭2件までを対象とする。0件なら fallback のモデル call は0件。
- 対象 fixture の全 unordered candidate pair を ID 昇順に列挙し、各 pair を `A,B` と `B,A` の2 call で比較する。候補 ID/grouping、repository input、schema enum ID順、model/output は pair 内で固定。`none` を含めない。output 2048、2分上限、retry/repair 0。
- 各 pair の2回答が同一 ID か、方向で異なるか、未完了かを記録する。全 pair が完了し、ある候補が他の全候補に同一 ID の2回答で勝った場合だけ、その候補を strict winner と記録する。そうでなければ winner なし。gold は pair 選択にも winner 決定にも使わない。
- 各方向1反復のため、repeat variation、position bias の因果効果、4反復から2反復への安全な削減は推定しない。calls・wall と incomplete も含めて記録する。

## 記録

- 実行順は決定的 score → top-2 forced-choice → 必要な場合の pairwise fallback。score の結果を見て重み・ペナルティ・対象 arm を変更しない。生の path/diff/prompt/response は JSONL に保存しない。
