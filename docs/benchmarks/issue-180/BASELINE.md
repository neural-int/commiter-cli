# Issue #180: 標準ハーネス H180-004（8bit KV・正式採用）

## 構成と起動

| 項目 | 値 |
| --- | --- |
| 標準ハーネス | `H180-004` |
| 状態 | ユーザー判断で正式採用（2026-10-19） |
| 前標準 | H180-003 / iteration09（BF16） |
| 基準測定 | iteration10のC01-KV8 / C02-KV8 |
| 次回比較契約 | `baseline_harness_id: H180-004` |
| KV方式 | K/Vとも8bit affine、group64、最初のprefill tokenから |
| 適用層 | global4層のみ量子化、native rotating20層はBF16 |
| 未採用候補 | H180-005（4bit、同iteration10） |

前標準の[契約保存版](H180-003.md)を継承する。model/revision、prompt/schema/出力/context条件、thinking予算512/384/384、seed144、temperature0、topP1、topK0、model/tokenizerとgrammar準備の計画内保持、phase別constraint/matcher/native state、KV prefix一致とコピー方法を維持する。

512tokenごとに同期的eval(cache)を実行し、明示的Stream().synchronize()/Memory.clearCache()は4chunkごと＋prefix末尾。C01は5chunk/2回、C02は36chunk/9回。通常長context prefill callbackは変更しない。chunk境界active/cache・同期解放時間とprefix/phase末尾のKV dtype/shape/nbytesを記録する。

8bit指定漏れを防ぐため、次回の評価runnerには以下の起動ラッパーをhelperとして渡す。native helperはリポジトリのビルド手順で生成し、環境変数で明示する。

```sh
export COMMITER_180_HELPER="$PWD/tools/benchmark180/h180004-helper.sh"
```

ラッパーは `COMMITER_180_KV_BITS=8` を必ず設定し、`--benchmark-cache` のみ受け付ける。`COMMITER_180_H180004_NATIVE_HELPER` にはビルド済みnative helperへの実行可能パスを指定する。未指定または実行不能なら明確なエラーで終了する。標準のbuild手順は[benchmark180 README](../../../tools/benchmark180/README.md)に記録する。native helperを直接呼ぶ場合は `COMMITER_180_KV_BITS=8` が必須で、未指定のBF16をH180-004として扱わない。

追加推論前にruntime/fixture/参照のhash、条件・要求上限・停止条件・監視を固定する。前回の最大12要求は8/4bit比較時の上限であり、次回に4bit候補まで再実行する指示ではない。

## 要約

2026-10-19、ユーザー判断により8bit KV保持のH180-004を正式採用し、次回以降の標準ハーネスとする。比較元は `baseline_harness_id: H180-004`、基準測定はiteration10のC01-KV8/C02-KV8。H180-005（4bit）は未採用の比較候補として保存する。

## 検証結果

| ケース | 旧H180-003全体 | 採用8bit全体 | 採用8bit KV準備 | 旧→8bit peak active | 旧→8bit prefix active＋cache標本最大 |
| --- | ---: | ---: | ---: | --- | --- |
| C01 | 67.729秒 | 68.366秒 | 4.053秒 | 4.796→4.796GiB | 4.748→4.679GiB |
| C02 | 212.772秒 | 229.748秒 | 49.537秒 | 5.197→4.936GiB | 6.521→5.911GiB |

8bitのprefix KV配列全体はC01 45.844MiB、C02 172.828MiB。global4層をK/Vとも8bit affine・group64で最初のprefill tokenから保持し、native rotating20層はBF16を維持する。既存のchunk512・同期的eval・4chunkごと＋末尾の明示同期/clear・grammar保持を継承。KV dtype/shape/nbytes計測も同じ構成で維持する。

8bitは両ケースの入力6要求が旧標準と一致、Validator受理。実適用4量子化層＋20 rotating層、再利用と同期解放回数を確認。Grouping参照一致は旧1/2→8bit2/2、分類は固定参照groupで2/2。C01要約は要件充足、C02は一時index・構造/replay検証の説明不足が継続。参照はagent作成・人間未承認で、一般的な精度改善の確定ではない。

## 考察

正式採用はメモリ削減を確認した検証構成の判断。全体高速化は観測していない。C02全体は約16.98秒長く、C01も約0.64秒長い。各条件1回・保存baseline・別build・追加計測・熱/他プロセス・生成出力の違いを分離していない。監視は比較全体で305有効サンプル、無効除外/監視途絶なし、Heavy158件。摂氏温度・プロセス別GPU値は未取得。

KV nbytesは配列容量とscale/biasを含み、物理buffer共有を追跡した常駐メモリとは異なる。MLX active/cacheと重なるため加算しない。旧標準はKV単体bytes未記録であり、BF16同容量換算を実測baselineと扱わない。C02後段は固定参照groupであり、通常経路の意味品質や120秒適合は未解決。

## Next Steps

次の比較はH180-004（8bit）からの差分として行う。同一構成の再測定はIDを維持し、新しい変更候補には新IDを割り当てる。過去実行契約の比較元H180-003は書き換えない。今回の正式採用更新では追加推論なし。

採用範囲は保存済み検証helper・起動ラッパー・baseline契約。通常経路への再接続、main merge、リリース完了は含まない。main向けPRで検証ハーネスと記録を統合する。過去の未採用記録は測定時点の履歴として維持し、現在の標準はBASELINE.mdを正本とする。

## 実装の識別

- source SHA256: `71e6d85b471cfb3c33c5861d6856e3f1c9f4d0cbd17e4a776970295ae2f91f2a`
- native helper SHA256: `d134b651cde53552c217d83e44e077bad74f63475f22459e2af2707e1147370e`
- runner SHA256: `c4aa5bcdabd56acbf1244662fcc2ec7c66d3be0832992beb6121b588b4db4aab`
- Metal SHA256: `0214ad23cc2fa1df961eb7e2118881bd4b2bdb0183546ef537fff1a2a61e2ada`

## 証跡

- [iteration10報告](iteration-10/results-report.md)、[実行契約](iteration-10/execution-contract.json)、[評価rubric](iteration-10/quality-rubric.json)
- [採用経緯](H180-004-adoption.md)

raw入力・応答・監視時系列/詳細集計・resource snapshot・詳細ログ・生成されたper-run結果はGitに含めず、必要な場合はローカルまたはCI Artifactsで管理する。Gitには再現用コード、fixture、Gold、実行条件、評価方法と主要スコアを含む簡潔な報告を残す。共通方針は[benchmark data policy](../README.md)を参照。

## Issueの区切り

ユーザー判断によりH180-004を現時点の成果として正式採用し、#180の検証を区切る。[最終採用と終了判断](CLOSURE.md)を参照。通常CLIへの接続やproduction 120秒適合は未実施・未解決として保持する。mainへのmergeはGitHub上のPR状態で確認し、文書更新だけで完了と扱わない。
