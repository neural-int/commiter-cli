# Issue #142: pair-local positive control 事前登録

## 対象と固定条件

- [修正済み考察](https://github.com/neural-int/commiter-cli/issues/142#issuecomment-5892734760)に従う。既存の合成 fixture `new_test_pair_shared` の F001/F002（gold `same`）と `new_crossdir_shared` の F001/F002（gold `same`）を使う。fixture 内容・gold は変更しない。前者は source/test の `RetryInvoicePayment`、後者は code/docs の `backfill` を両ファイルに含む。
- 既存の `pair-local` と同じ `issue142PairLocalInput`、system、task、schema形、raw diff 構築を再利用する。prompt は対象2ファイルの `{id,path,raw_diff}` のみとし、complete candidate ID、partition、relation context、第三ファイルは含めない。
- 各 fixture 4 callの順序を固定する: 1 `(F001,F002; same,different)`、2 `(F002,F001; different,same)`、3 `(F002,F001; same,different)`、4 `(F001,F002; different,same)`。各組み合わせは1回のみ。過去の fixture は再実行しない。
- モデルは `mlx-community/Ministral-3-3B-Instruct-2512-4bit@a962dcb09eee4169c890e544c9eb938f1113fdee`、helper SHA-256 は `da8a7c24770047827ec159649254e04554cf50736182d191e634cdfb0fa6ae48`。output上限2048 tokens、各 call 2分上限、retry/repair 0。実行前に検証コードを別コミットで固定する。新規依存とモデル取得は行わない。

## 記録と解釈の範囲

- fixture/run、ファイル順、enum順、stop reason、有効ラベル、選択、合成 gold との一致、wall、prompt/schema hash、output bytes/token可用性を JSONL に記録する。生 path/diff/prompt/response は結果 JSONL に保存しない。
- この2件で `same` を観測した場合、その入力で固定 contract が `same` を返せることまでを示す。両件で `different` の場合、この対象範囲で `same` を観測できなかったことまでを示す。relation type と語・path の手掛かりは同時に異なるため、fixture間の差から単一原因を推定しない。各組み合わせ1回なので、反復変動や順序の因果効果を推定しない。
- model capability、paraphrase固有の限界、production grouping の精度、#142の停止条件をこの8 callだけで確定しない。generator、candidate集合、scorer、Pass 2、通常 CLI は変更しない。
