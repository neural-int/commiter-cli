# H184-002 履歴候補の補完後に行ったCategory英訳比較

## Goal / Conditions

抽出安全性検査の誤検出に対応し、元apply_patchの追加内容がGit差分と一致することを確認した上で、UI実装4ブロックへH3/exact、メールテスト1ブロックへH2/exactを追加した。原本H184-KIMDEL-FIXED-001を保持し、補完版H184-KIMDEL-REPAIRED-002を使用。取得状態partial、全9ブロックのambiguous/history_incompleteは維持した。これは候補参照補完だけの効果を見る実験であり、抽出実装を修正してcomplete/matchedへ昇格した実験ではない。

C184-02: Qwen3.5-4B Affine4bit/group64、固定revision、Swift Release、Thinking OFF、json_schema、native KV、32768/4096、temperature0.6/top_p0.95/top_k20、penalty1.0、seed144、eager、時間制限なし。モデル・helper・system・Schema・翻訳文・現在Git差分は前回と同じ。Groupingは前回の実応答3/3を両群で再利用、Category/Textを各群1回、計4生成。英訳はCategoryの依頼3本文・最終応答3本文だけ。Textは日本語履歴と各群の実Categoryを受け取る。

## Result

| 変更目的 | 正解 | 補完後ja | 補完後en |
| --- | --- | --- | --- |
| README品質チェック | docs | docs | docs |
| 期限日tooltip追加 | feat | feat | fix |
| Message-ID不具合修正 | fix | feat | feat |

Categoryはja 2/3、en 1/3。補完前はja 2/3、en 1/3。Text最終validatorはja False、en False。禁止scopeはja {"G001": "docs"}、en {"G001": "docs", "G002": "fix", "G003": "feat"}。JSON生成完了と実タスク成功は区別する。

## Measurements

| arm | phase | 入力tokens | 出力tokens | prefill秒 | decode tok/s | helper wall秒 | MLX peak GiB |
| --- | --- | --- | --- | --- | --- | --- | --- |
| ja | category | 4996 | 71 | 21.475 | 19.79 | 30.307 | 3.153 |
| ja | text | 4880 | 74 | 16.600 | 24.47 | 30.592 | 3.140 |
| en | category | 4897 | 71 | 17.233 | 26.53 | 23.248 | 3.133 |
| en | text | 4880 | 82 | 17.517 | 24.71 | 24.196 | 3.140 |

host各区間とruntime各区間、CPU user/system、メモリ、監視集計はmeasurements.jsonへ保存。helper wallはproxyと非推論contract probeを含む。Grouping再利用分の過去計測は新しい推論性能として集計しない。

## Validation / Limits

補完後fixtureの34ファイル検証、原本との差が候補配列5件だけであること、ja/enの差が履歴6本文だけであること、6文字列の逆置換によるuser入力byte一致、元Grouping応答のbyte一致、ソースハッシュ維持、監視正常・同時helper最大1を確認。補完元patchの識別情報は非公開provenanceに保存。

単一fixture/seed・各1回。候補は補完済みだが全体の履歴不完全フラグは意図して維持している。実抽出経路の修正効果や完全履歴での精度を主張しない。背景負荷・OSキャッシュは固定していない。native KV dtype/bytesは未export。Apple翻訳は未使用。準備段階の履歴レコード読み出しフィールド誤りは推論前に修正し、失敗コピーを保持。モデル失敗の再試行なし、原本保持。
