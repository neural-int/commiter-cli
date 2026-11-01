# H184-HARNESS-004の多ファイル派生入力・再検証

状態: 9/11ファイルは両経路終了。7は単独3phaseのみ終了し、伝播2phaseは未測定。合計13生成応答が正常終了。初回の検証器実行権限不備の原本を保持し、7の追加サイクルはユーザー確認待ち。全15phase完了とは扱わない。

## 入力と接続

`tools/benchmark184/multifile004.py`は、canonicalの2メッセージ要求を、実際に採用した004の保存済み3メッセージ指示を基準に変換する。共通system・phase別指示を保持し、動的なフィールド説明と完全なSchemaを1回だけ表示する。runtime grammarのSchemaは別に保持する。

Grouping/Textの共有側に完全なfiles/raw_diff/change_blocksとfinal_responses履歴を置き、phase側には参照と固有情報を置く。Categoryは履歴と、各group内の完全な差分を使用する。task/schema_version・空user_requests・block_id/execution_refを表示からのみ省く。元データ・照合状態・候補・差分・非空本文は保持し、元fixtureは変更しない。Message-IDのfix訂正は前回の確認済みコピーから引き継ぐ。

Grouping/Textの先頭2メッセージの完全一致、phase間のファイル証拠一致、動的フィールド説明が004の保存済み指示と同じ規則で生成されることを事前検査した。

## 条件

Qwen3.5-4B 4bit pinned revision、Thinking OFF、grammarあり、native KV、32K/4096、temperature .6/top_p .95/top_k20/seed144、legacy prefill保護を固定。時間制限・warmup・自動repair・モデル条件を変更する再試行なし。App Server取得や実Git変更は含めない。

研究用`--benchmark-multifile`は1helper内でG/単独C/単独T/伝播C/伝播Tの5要求を順番に扱う。初回Groupingのprefix snapshotから、2回のTextそれぞれへ独立forkする。通常Plannerの4ファイル上限と経路は変更しない。snapshot/cache処理は採用済み004の型別実装を使用し、追加したのは研究用の要求順序のみ。

| ファイル数 | G入力tokens | 単独C入力tokens | 単独T入力tokens | G→T共通prefix tokens | cache型 |
|---|---:|---:|---:|---:|---|
| 7 | 6,457 | 4,912 | 4,876 | 4,024 | KVCacheSimple/MambaCache |
| 9 | 8,362 | 6,359 | 6,319 | 5,338 | KVCacheSimple/MambaCache |
| 11 | 10,031 | 7,590 | 7,548 | 6,436 | KVCacheSimple/MambaCache |

全要求は出力予約4096を含め32K内。Categoryは独立prefill。GのprefixReuseStatus=created、TのreusedとcacheReusedTokens>0、C/TのmodelReused/grammarPreparationReused=trueを要求し、不適用のまま004成功とは扱わない。

## 計測上の区別

Gのprefill総計にはprefixPrefillSecondsと通常runtimePromptSecondsの両方を含める。Textは実際に再計算したsuffixのprefill時間を報告し、full input tokensを全て再計算した速度と混同しない。MLX peakMemoryは同一helper起動以来の高水位で、各phase固有の新規ピークとは呼ばない。RSS/GPU等はphase区間のサンプリング値を区別する。

単独G/C/Tの連続経過時間は取得する。伝播経路は独立評価を挟むため、G+実C+実Tのphase時間合計と入力構築時間を示し、寄り道を含む経過時間を純粋な伝播TATと扱わない。

## 初回停止

7ファイルのG/C/Tは正常生成し、Gのsnapshot作成・モデル/grammar再利用・Tで4,024 tokensのKV再利用を確認。その後、コピーしたホスト検証器に実行権限がなくPermissionErrorで停止した。モデルの失敗やmax_tokensではない。実行権限を修正し、保存済み応答のホスト検証は受理。生成済み3応答は消さず、7の伝播が未測定であることを記録。未実行の9/11は条件変更せず続行する。

アダプター・評価のPython5テスト、Releaseビルド、Metal smokeが成功。新規依存なし。生履歴・Thinking・token IDs・個人pathは公開しない。

## 品質結果

| ケース | Grouping | Category単独 | Category伝播 | Text | validator |
|---|---|---|---|---|---|
| 7/4目的 | 4/4、完全一致 | 4/4 | 未測定 | 単独では途切れなし。ただし「無効な件数拒否」等は具体性が弱い | 単独は保存応答を再検証して受理。伝播未測定 |
| 9/5目的 | 5/5、完全一致 | 5/5 | 5/5 | 両側とも途切れなし。ただしscope「付着」や「件数検証強化」等は不自然/抽象的 | 両側受理 |
| 11/6目的 | 6/6、完全一致 | 5/6 | 5/6 | 単独側に「全角数字拒否処理」という逆の記述。伝播側のMessage-ID群をUIテストと記述する誤りなど | 両側受理 |

11ファイルのMessage-IDはCategory両側でfeatと誤分類（期待fix）。Groupingの誤配属は今回出なかったが、Categoryは改善していない。単独Textのmailbox IDは、実差分のテストが全角数字を受理するのに拒否と記述した。伝播TextのG003は実入力でF3/F5と正しく対応し、共通データもGroupingと一致しているが、「期限日ツールチップテスト」と出力した。ホストの参照生成でUI群へ置き換わったものではない。

13応答すべてcompleted・構造検証成功。max_tokensなし。breaking_evidence_refはすべてnone。Textの短文化・途切れ解消を意味品質の全面改善とは扱わない。各ケース1回であり、旧系統との品質差をKV再利用や特定の投影変更だけへ帰属しない。旧/004のプロンプト・配置・メタデータ省略・再利用等がまとめて変化した比較である。

## 性能

単独側は連続G/C/T、伝播側は共有G+実C+実Tの要求時間合計。Gは実験全体へ1回だけ計上する。下表はprefix準備の時間をprefillに含める。7の伝播値は未測定。

| ケース | 経路 | 3phase要求時間合計秒 | prefix準備を含むprefill合計秒 | decode合計秒 | 最大MLX高水位MiB |
|---|---|---:|---:|---:|---:|
| 7 | oracle | 53.465 | 39.542 | 10.744 | 3394.05 |
| 9 | oracle | 71.385 | 53.502 | 14.302 | 3586.12 |
| 9 | propagated | 72.549 | 54.611 | 14.301 | 3586.12 |
| 11 | oracle | 89.091 | 66.626 | 18.855 | 3694.72 |
| 11 | propagated | 97.208 | 69.173 | 23.691 | 3694.72 |

### phase別

Cは再利用なし。Tのprefillはsuffix計算のみ。MLXピークは起動以来の高水位。検証時間単独は未取得。

| ケース | 経路/phase | full入力tokens | 生成tokens | load秒 | 入力準備秒 | grammar秒 | prefix準備秒 | runtime prefill秒 | decode秒 | 要求秒 | decode tok/s | KV再利用tokens | model再利用 |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 7 | shared/grouping | 6457 | 72 | 1.168 | 0.063 | 0.693 | 12.308 | 8.465 | 2.717 | 25.444 | 26.50 | 0 | False |
| 7 | oracle/category | 4912 | 94 | 0.000 | 0.049 | 0.003 | 0.000 | 15.831 | 3.515 | 19.402 | 26.74 | 0 | True |
| 7 | oracle/text | 4876 | 103 | 0.000 | 0.053 | 1.106 | 0.000 | 2.939 | 4.512 | 8.620 | 22.83 | 4024 | True |
| 9 | shared/grouping | 8362 | 92 | 1.536 | 0.087 | 0.685 | 17.152 | 10.901 | 3.667 | 34.062 | 25.09 | 0 | False |
| 9 | oracle/category | 6359 | 117 | 0.000 | 0.074 | 0.003 | 0.000 | 21.839 | 4.557 | 26.479 | 25.68 | 0 | True |
| 9 | oracle/text | 6319 | 127 | 0.000 | 0.056 | 1.095 | 0.000 | 3.610 | 6.078 | 10.844 | 20.89 | 5338 | True |
| 9 | propagated/category | 6359 | 117 | 0.000 | 0.054 | 0.003 | 0.000 | 22.845 | 4.541 | 27.445 | 25.77 | 0 | True |
| 9 | propagated/text | 6319 | 127 | 0.000 | 0.066 | 1.164 | 0.000 | 3.713 | 6.093 | 11.042 | 20.84 | 5338 | True |
| 11 | shared/grouping | 10031 | 114 | 1.406 | 0.097 | 0.680 | 20.771 | 14.027 | 4.667 | 41.685 | 24.43 | 0 | False |
| 11 | oracle/category | 7590 | 140 | 0.000 | 0.071 | 0.004 | 0.000 | 27.475 | 5.590 | 33.145 | 25.04 | 0 | True |
| 11 | oracle/text | 7548 | 163 | 0.000 | 0.082 | 1.221 | 0.000 | 4.352 | 8.598 | 14.261 | 18.96 | 6436 | True |
| 11 | propagated/category | 7590 | 140 | 0.000 | 0.069 | 0.004 | 0.000 | 28.747 | 7.919 | 36.743 | 17.68 | 0 | True |
| 11 | propagated/text | 7548 | 151 | 0.000 | 0.081 | 1.960 | 0.000 | 5.627 | 11.105 | 18.780 | 13.60 | 6436 | True |

### 資源

CPU/GPUはマシン全体。GPU割当は専用常駐量ではない。圧縮/swapoutは要求監視区間のOS全体カウンター増分。

| ケース | 経路/phase | CPU中央値% | GPU中央値% | GPU割当ピークMiB | helper RSSピークMiB | MLX高水位MiB | swapout増分 | 圧縮増分 | 熱状態サンプル |
|---|---|---:|---:|---:|---:|---:|---:|---:|---|
| 7 | shared/grouping | 12.7 | 100.0 | 8079.83 | 875.20 | 3394.05 | 98992 | 374243 | {'nominal': 12} |
| 7 | oracle/category | 11.3 | 100.0 | 5508.34 | 228.50 | 3394.05 | 0 | 0 | {'nominal': 9} |
| 7 | oracle/text | 11.3 | 93.5 | 5596.62 | 261.69 | 3394.05 | 0 | 0 | {'nominal': 4} |
| 9 | shared/grouping | 12.5 | 100.0 | 8548.55 | 1050.61 | 3586.12 | 2968 | 555189 | {'nominal': 16} |
| 9 | oracle/category | 11.2 | 100.0 | 5817.81 | 261.95 | 3586.12 | 0 | 0 | {'nominal': 12} |
| 9 | oracle/text | 12.6 | 98.0 | 5804.08 | 300.92 | 3586.12 | 0 | 0 | {'nominal': 6} |
| 9 | propagated/category | 11.4 | 100.0 | 5546.59 | 300.92 | 3586.12 | 0 | 0 | {'nominal': 12} |
| 9 | propagated/text | 13.2 | 87.0 | 5747.14 | 308.28 | 3586.12 | 0 | 0 | {'nominal': 4, 'fair': 1} |
| 11 | shared/grouping | 12.2 | 100.0 | 9651.95 | 2645.88 | 3694.72 | 42328 | 381130 | {'nominal': 19} |
| 11 | oracle/category | 11.8 | 100.0 | 5648.81 | 1583.80 | 3694.72 | 0 | 0 | {'nominal': 11, 'fair': 5} |
| 11 | oracle/text | 15.0 | 92.0 | 5752.56 | 1618.38 | 3694.72 | 0 | 0 | {'fair': 6} |
| 11 | propagated/category | 11.7 | 100.0 | 5776.42 | 1618.72 | 3694.72 | 0 | 0 | {'fair': 17} |
| 11 | propagated/text | 12.4 | 85.0 | 5978.77 | 1625.39 | 3694.72 | 0 | 0 | {'fair': 9} |

実温度・周波数・厳密なThinking/最終回答token内訳・ホスト検証単独時間は欠測。低レベルの数値同等性を新たに証明した実験ではない。004の既存Qwen検証にもText差異の制約がある。生時系列・会話・token列は非公開で保持する。
