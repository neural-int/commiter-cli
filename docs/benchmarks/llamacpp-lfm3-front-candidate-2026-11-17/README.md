# LFM3前段2フェーズの固定入力候補

LFM3-1.7B Q8_0で、GroupingとCategoryの最終回答がそれぞれ満点になる条件を確認した。Thinking ONで自然な思考終了を待ち、その後のJSONへ元のSchemaを適用する研究候補として固定する。

| 項目 | 固定条件 |
|---|---|
| モデル | LiquidAI/LFM3-1.7B-GGUF、Q8_0 |
| ランタイム | 固定llama.cppとLLGuidance。revision・digestはprofile.jsonに収録 |
| Thinking | ON。自然に生成した思考終了タグ後の全体を最終JSONとして検証 |
| 文法 | 思考部を許可し、終了タグ後は元のフェーズ別JSON Schemaを適用 |
| コンテキスト／総出力枠 | 32,768／8,192トークン |
| Sampling | temperature 0.1、top_k 50、repeat_penalty 1.1、top_p 0.95、min_p 0.05、seed 144 |
| Grouping入力 | 元の差分・作業履歴・コマンド履歴・指示・Schemaを保持 |
| Category入力 | 元の固定3群と差分を保持。共有の履歴本文を省略し、分類規則を具体化 |

## 完了した最終回答の結果

| フェーズ | 意味評価 | Schema／停止 | 総出力トークン | 実時間 |
|---|---|---|---:|---:|
| Grouping | 3群のファイル集合が一致 | 通過／EOS | 5,824 | 150.35秒 |
| Category | 種別3/3、破壊的変更参照3/3 | 通過／EOS | 3,224 | 82.32秒 |

両フェーズは独立した固定入力で実行した。Categoryには期待する3群を渡した。Groupingが生成したラベルはREADMEがG001、UI実装と対応テストがG003、正規化実装と対応テストがG002だった。固定Categoryのラベルとは異なるため、Groupingはファイル集合で採点した。

<details>
<summary>入力の変更内容と採点条件</summary>

Categoryでは共有メッセージの履歴本文を空配列に置き換えた。差分側の履歴参照メタデータは保持した。README、UI、正規化の掲載順を維持した。

Categoryのsystem指示で文書だけの変更をdocs、プログラムの新しい動作をfeat、不具合修正をfixと区別した。実装に対応するテストは実装の目的に従う。内部実装の変更だけでは公開互換性の破壊と判定しないことも明記した。実際の英語指示全文とSchemaをprofile.jsonに保存した。

EOSで自然終了し、単一の思考終了タグの後にSchema準拠のJSONがあることを合格条件とした。Groupingは全5ファイルの重複・欠落を確認した。Categoryは全3群の種別とbreaking_evidence_refを照合した。期待する参照値はすべてnoneである。

</details>

<details>
<summary>比較した条件と確認範囲</summary>

Thinking OFFで文法を解除し、出力枠だけを8,192に増やす条件では4,143トークンで自然終了した。思考終了タグの後に正しい最終JSONがあった。ただしOFFの応答全体はSchemaに違反した。

Categoryを1群ずつ独立に渡す条件は合計で種別1/3、破壊的変更参照1/3だった。履歴省略と分類規則の具体化だけでは種別2/3、参照2/3だった。履歴と参照メタデータを省き、Grouping指示を短くしたOFF条件も5群に分割した。各結果はresults.jsonに収録した。

今回の採用条件は1 fixture・1 seed・各1回である。Categoryでは入力と指示の複数要素を変更しており、個々の寄与は未分離である。実行中の熱状態はCategoryがnominal、Groupingがnominalとfairだった。速度の優劣を判定する対照測定は実施していない。

通常CLIへの組み込み、生成結果を伝播するE2E、Textフェーズ、3フェーズ合計時間は未検証である。本ディレクトリは固定入力の研究条件と測定結果を保存する。非公開fixtureの生差分と作業履歴はローカル実験記録に保持する。

</details>
