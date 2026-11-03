# H184-005 / runtime-prefill-ba

## 目的と固定条件

前回ABの順序を反転し、従来保護legacy→8K以下保護軽減adaptive8kの順で各3phaseを1回、新規計6生成。前回の原本保持、保存済みRelease helperとGo runnerを再利用しSHA一致確認。新規ビルドなし。同一helper内のモデル・共通文法準備再利用、scope候補の先行選択、Go実改行表示は両群共通。prefill保護設定だけが異なる。

Qwen3.5-4B affine4bit/group64、C184-02補完済み固定fixture、Thinking OFF、JSON文法/scope制約ON、native KV、入力32768/出力4096、sampling .6/.95/20、seed144、eager、時間制限なし。フェーズ間KV再利用なし。Pythonは監視起動のみで推論proxyなし。

## 順序別結果

| 実行 | 順番 | 条件 | 3phase秒 |
| --- | --- | --- | --- |
| AB | 1 | adaptive8k | 57.221 |
| AB | 2 | legacy | 58.163 |
| BA | 1 | legacy | 54.600 |
| BA | 2 | adaptive8k | 56.839 |

両順序とも後半の条件が遅くなり、速い保護方式は逆転した。ABの差はlegacyが約0.942秒遅く、BAではadaptive8kが約2.239秒遅い。2観測ずつの記述的平均はlegacy約56.382秒、adaptive8k約57.030秒。この平均差から一般的な性能差を断定しない。

## 今回BAの品質・資源

| 条件 | Grouping | Category | 最終validator | RSS最大MiB | system swap MiB | nominal/fair観測数 |
| --- | --- | --- | --- | --- | --- | --- |
| legacy | 3/3 | 3/3 | 通過 | 1200.7 | 818.62–818.62 | 28/0 |
| adaptive8k | 3/3 | 3/3 | 通過 | 1315.7 | 818.62–818.62 | 17/11 |

| 条件 | phase | input/output tokens | prefill秒 | decode tok/s |
| --- | --- | --- | --- | --- |
| legacy | 01 | 5123/52 | 16.391 | 25.66 |
| legacy | 02 | 4027/71 | 12.503 | 27.02 |
| legacy | 03 | 3911/83 | 12.711 | 20.19 |
| adaptive8k | 01 | 5123/52 | 17.244 | 25.74 |
| adaptive8k | 02 | 4027/71 | 13.675 | 26.66 |
| adaptive8k | 03 | 3911/83 | 13.729 | 21.17 |

全3phase要求は今回両群と前回ABでbyte一致。生成JSONも前回と全て一致。固定された後続入力を使う前に、先行生成の一致を確認した。Textの既存表現上の課題も同じで、validator通過は意味の全正確性を保証しない。

Category/TextでmodelReusedとgrammarPreparationReusedがtrue、全phase cacheReusedTokens=0。prefillProtectionEnabledはlegacyでtrue、adaptive8kでfalse。監視56サンプル、異常なし、helper最大1プロセス。

## 判断と限界

先行条件がnominalのみ、後続条件にfairが現れる関係は前回ABと今回BAで共通。順序・開始時状態の影響と整合する証拠が増えた。ただし動作周波数・温度・熱の発生元は測れていないため、ハードウェア要因だけに確定しない。今回のswapは両群818.62MiBで一定、前回の74.37秒のような測定中swap増加はない。

8K保護軽減の安定した高速化は確認できない。前回74.37秒の遅延を保護方式単独の原因とする証拠もない。比較結果を保存し、採用済みハーネス登録は変更しない。今回モデル推論を追加でやり直してはいない。

今回の追加は測定記録と一時privateランナーだけ。実装差分は比較切替を含む前回の未コミット状態を維持する。各順序1回、長入力の新規検証なし。phase runner時間はhelper起動前を含まないため、process_wall_secondsも計測値へ保存。Git/App Server取得を含む全CLI E2Eではない。
