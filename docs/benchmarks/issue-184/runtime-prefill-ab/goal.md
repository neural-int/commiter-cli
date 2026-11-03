# H184-005 / runtime-prefill-ab

## 目的と条件

同一helperでモデルと文法用共通準備を保持し、scope候補の事前絞り込み・Go実改行表示を維持して、prefill保護だけを比較。両群とも同じ新しいSwift Releaseバイナリと同じGo runner。環境変数COMMITER_184_PREFILL_POLICYのみをadaptive8k→legacyへ変えた。既存原本は保持し、新規ディレクトリへ保存。

adaptive8k: context>16384かつ実入力>8192の場合のみchunk同期・解放。legacy: context>16384なら実入力長にかかわらず従来のchunk同期・解放を使用。

補完済み固定fixture、C184-02、Qwen3.5-4B affine4bit/group64、Thinking OFF、JSON文法とscope制約ON、native KV、入力32768/出力4096、temperature .6/top_p .95/top_k20、seed144、eager、時間制限なし。各3phaseを1回、計6生成。フェーズ間KV再利用なし。Pythonは監視・起動管理にだけ使用し、推論proxyなし。

固定phase入力を使用し、先行出力が元の後続入力の前提と一致することを確認してから続行。Git/App Server取得を含むCLI E2Eの再実行ではない。

## 結果

| 条件 | Grouping | Category | Text/最終validator | phase runner秒 | プロセス全体秒 | RSS最大MiB | swap MiB | 熱状態 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| adaptive8k | 3/3 | 3/3 | 完了/通過 | 57.221 | 57.583 | 819.5 | 826.62–826.62 | nominal |
| legacy | 3/3 | 3/3 | 完了/通過 | 58.163 | 58.169 | 2087.1 | 826.62–826.62 | fair/nominal |

全3phaseの要求はbyte一致、生成JSONも全て同一。Category/Textは両群でmodelReused/grammarPreparationReused=true、cacheReusedTokens=0。prefillProtectionEnabledはadaptive8kで全てfalse、legacyで全てtrue。

| 条件 | phase | input/output tokens | prefill秒 | decode秒 | decode tok/s | MLX peak GiB |
| --- | --- | --- | --- | --- | --- | --- |
| adaptive8k | 01 | 5123/52 | 18.204 | 2.125 | 24.47 | 3.142 |
| adaptive8k | 02 | 4027/71 | 12.458 | 2.927 | 24.26 | 3.142 |
| adaptive8k | 03 | 3911/83 | 12.868 | 4.089 | 20.30 | 3.142 |
| legacy | 01 | 5123/52 | 17.568 | 2.065 | 25.18 | 3.200 |
| legacy | 02 | 4027/71 | 14.135 | 2.825 | 25.13 | 3.200 |
| legacy | 03 | 3911/83 | 13.994 | 3.993 | 20.79 | 3.200 |

## 解釈

従来保護は57.221→58.163秒、約0.942秒（1.65%）長かった。今回、従来保護へ戻すことで速くなる証拠は得ていない。各1回で順序固定、後半はfairも観測しており、小さい差から安定した優位性は断定しない。

前回の保護軽減＋再利用74.373秒は今回は再現せず、同じ軽減設定が57.221秒となった。今回swapは両群とも既存826.62MiBから増えていない。前回の遅延を保護軽減だけの原因と断定できない。RSSの差も圧縮・共有GPUメモリ・サンプリング等の影響を受けるため、そのまま必要メモリ差と解釈しない。CPU/GPUサンプルの集計はmeasurements.json、詳細はprivate原本に保持。

生成スコア維持は確認できたが、Textには実ファイル名とは異なるwidget.pyのscope、修正を「追加」とする元の表現が残る。validatorは全ての意味的正確性を保証しない。

## 検証と状態

今回追加した比較切替を含むSwift ReleaseビルドとSwift53テストが成功。Go runnerは前回の最終EOF/終了状態確認を含む版を両群で使用。監視58サンプル、異常なし、helper最大1プロセス。追加の再試行・条件変更なし。

変更・記録は未コミット。採用済みH184-HARNESS-002 revision 1は上書きしない。比較用切替の既定値は前回の候補adaptive8kのまま。実入力8K超の新規推論は未実施。
