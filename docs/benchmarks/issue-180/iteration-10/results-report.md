## 要約

最新標準H180-003（iteration09の保存結果）をbaselineに、H180-004（8bit）とH180-005（4bit）を比較した。K/Vはaffine・group64、global4層のみ先頭prefillから量子化し、native rotating20層はBF16を維持する。baseline再実行なし。H180-003のchunk512・同期的eval・4chunkごと＋末尾の明示同期/clear・grammar保持を維持。

## 検証結果

| ケース | 構成 | prefix KV 秒 | Category＋Text 秒 | 全体 秒 | peak active GiB | prefix active＋cache標本最大 GiB |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| C01 | H180-003 BF16 | 4.059 | 38.816 | 67.729 | 4.796 | 4.748 |
| C01 | H180-004 KV8 | 4.053 | 39.184 | 68.366 | 4.796 | 4.679 |
| C02 | H180-003 BF16 | 51.187 | 109.888 | 212.772 | 5.197 | 6.521 |
| C02 | H180-004 KV8 | 49.537 | 122.535 | 229.748 | 4.936 | 5.911 |
| C01 | H180-005 KV4 | 5.512 | 53.995 | 91.824 | 4.796 | 4.668 |
| C02 | H180-005 KV4 | 51.199 | 118.797 | 222.520 | 4.796 | 5.599 |

| ケース | 構成 | prefix KV全体 MiB | global量子化KV MiB | rotating KV MiB | allocator cache標本最大 GiB |
| --- | --- | ---: | ---: | ---: | ---: |
| C01 | H180-004 | 45.84 | 19.12 | 26.72 | 0.676 |
| C02 | H180-004 | 172.83 | 150.88 | 21.95 | 1.811 |
| C01 | H180-005 | 36.84 | 10.12 | 26.72 | 0.683 |
| C02 | H180-005 | 101.83 | 79.88 | 21.95 | 1.559 |

KV bytesは各cacheのinnerState配列nbytes合計で、確保容量とscale/biasを含む。同一物理bufferの共有を追うRSS測定ではない。baselineにはKV配列bytesの直接記録がないため、baselineの同指標は未取得。MLX active/cacheとKV bytesは重なるので加算しない。

| phase | 標本数 | CPU全体平均 | GPU全体平均 | OS free範囲 | thermal pressure |
| --- | ---: | ---: | ---: | --- | --- |
| H180-004/C01/Grouping | 13 | 16.3% | 88.4% | 26–39% | Nominal |
| H180-004/C01/Category | 10 | 13.0% | 92.5% | 37–45% | Nominal |
| H180-004/C01/Text | 8 | 14.4% | 89.7% | 38–65% | Nominal |
| H180-004/C02/Grouping | 52 | 19.8% | 86.2% | 33–49% | Moderate, Heavy |
| H180-004/C02/Category | 43 | 14.0% | 94.6% | 28–48% | Moderate, Heavy |
| H180-004/C02/Text | 15 | 27.0% | 84.9% | 37–66% | Moderate |
| H180-005/C01/Grouping | 18 | 33.7% | 82.5% | 28–43% | Moderate |
| H180-005/C01/Category | 13 | 27.1% | 86.6% | 28–50% | Moderate, Heavy |
| H180-005/C01/Text | 11 | 15.9% | 87.1% | 38–65% | Heavy |
| H180-005/C02/Grouping | 50 | 12.3% | 93.2% | 32–56% | Heavy, Moderate |
| H180-005/C02/Category | 42 | 11.7% | 94.6% | 32–52% | Moderate, Heavy |
| H180-005/C02/Text | 13 | 12.4% | 90.0% | 38–54% | Moderate |

実要求数12、最大12、追加retry/repair/judgeなし。有効監視305件、不正除外0件、監視失敗=None。失敗候補bits=[]。実際の量子化gate=True。入力・JSON一致と意味品質は分けて評価し、主要な品質判定はこの報告内に記録した。

## 考察

### 出力品質の参照比較

| 構成 | Grouping参照一致 | 分類（固定参照group） | C01要約 | C02要約 |
| --- | --- | --- | --- | --- |
| H180-003 BF16 | 1/2 | 2/2 | 要件充足 | 一部充足 |
| H180-004 8bit | 2/2 | 2/2 | 要件充足 | 一部充足 |
| H180-005 4bit | 1/2 | 2/2 | 要件充足 | 一部充足 |

8bitのC02は参照どおり1group、baselineと4bitは同じ3group。C02分類とTextは固定参照group注入後なので実Groupingを救済したend-to-end成功とは扱わない。C01要約は3方式とも404修正と正規化/パス解決の要点を保つ。C02要約は3方式とも一時indexや構造/replay検証の要件を十分に明示していない。8/4bitのC01要約は末尾句点を含み、表記も変化した。構造化JSON一致はraw文字列一致と別記（4bit Groupingの空白差は意味差ではない）。この評価は分析者によるもので、人間による独立評価とは扱わない。

### 実測からの判断

C02のpeak activeは5.197→8bit 4.936／4bit 4.796GiB、prefix active＋cache標本最大は6.521→5.911／5.599GiBに低下。prefix KV全体は8bit 172.828MiB、4bit 101.828MiBで、量子化global部分は150.875→79.875MiB、rotating部分21.953MiBは同じ。scale/biasと未量子化層があるので全KVが単純に半分になるわけではない。

同じ確保容量をBF16へ戻した計算上のprefix KVはC01 62.719MiB、C02 305.953MiB。この値は実測baselineではなく、量子化配列shapeからの参考換算である。global部分単体は8bitでBF16相当の53.125%、4bitで28.125%となり、scale/biasを含む圧縮を確認できた。

全体時間はC01 baseline67.729→8bit68.366／4bit91.824秒、C02 baseline212.772→229.748／222.520秒。今回、全体の高速化は観測できなかった。C01の8bitはNominalだが4bitはModerate/Heavyで、CPU全体平均もphaseで異なる。順序・熱・他プロセス・出力tokenの変化を分離した比較ではなく、4bit固有の速度悪化とは断定しない。2ケースで明確な追加意味品質崩壊は観測していないが、品質維持の一般的な保証ではない。


品質は固定参照のpartition/category一致とtext required meanings/forbidden claimsを区別する。参照はagent作成・人間未承認、C01/C02のみで一般的な回答精度ではない。C02後段は固定参照groupを注入するため、実Groupingの品質と後段を分離する。schema/Validator受理やJSON完全一致は意味品質そのものではない。

推論前のfreezeで前回監視/runtime参照の不一致を検出して停止した。0要求の状態で参照を修正してから正式契約を固定した。モデル条件の事後変更や追加推論はない。

各case各構成n=1、過去baseline・別build・追加instrumentation/observer負荷がある。順序は8bit C01→C02→4bit C01→C02で固定しており、時間差を単独因果効果と確定しない。OS swapは他プロセスと既存状態を含む。prefill最初から量子化した候補であり、decode開始時だけ量子化する方式とは区別する。

## Next Steps

登録済み一巡で終了。意味内容の判定は固定rubricに基づく分析者レビューとして別記し、追加LLM judgeは使わない。標準はH180-003を維持し、候補を自動採用しない。raw入力・応答・監視はローカル保持。commit/push・Issue更新は未実施。
