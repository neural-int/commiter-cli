# 004・9ファイルTextの要約指示比較

状態: 変更後Textを1回生成して正常終了。追加Grouping/Categoryのtoken生成なし、再試行なし。比較基準は保存済み004の9ファイル単独Textであり、最初の旧2メッセージ形式の48文字途切れ応答との単一要因比較ではない。

## 変更

004のphase固有メッセージ内の長さ指示だけを次へ置換した。共通systemと共有メッセージ、phase入力本文、group/分類、Schema、48文字上限は一致を検査。runtime sampling/seed/native KV/32K4096/Thinking OFFは維持。

```
Write a concrete, self-contained summary within 48 characters.
Prioritize the affected behavior and what changed.
Omit nonessential identifiers and details while preserving the meaning.
Choose wording that fits completely; do not end with an unfinished identifier or phrase.
```

`multifile004.summary_instruction_variant`として実験用に実装し、今回の実送信要求と完全一致することを検査した。004既定の指示を自動採用で上書きしたものではない。

## 結果

| 群 | 変更前summary | 変更後summary | 文字数 | 評価 |
|---|---|---|---:|---|
| G001 | 品質チェック手順を記載 | 品質チェック手順を記述 | 11 | 意味はほぼ同じ |
| G002 | 期限日ツールチップ追加 | 期限日ツールチップ追加 | 11 | 同一 |
| G003 | 空角括弧正規化強化 | 空角括弧正規化追加 | 9 | 追加という表現になり、修正の核心であるローカルキーへのfallbackは未記述。角括弧の呼称も不正確 |
| G004 | 件数検証強化 | 件数値の bool・負拒否 | 13 | scopeは付着からアタッチメントへ改善。boolへの言及が増えたが、負数拒否は既存動作であり変更の焦点が曖昧 |
| G005 | オプション日付変換 | 可変日付変換関数追加 | 10 | 可変日付はNone許容の意図を正確に伝えていない |

構造検証とホストvalidatorは受理。意味の完結・具体性・正確さの明確な改善は未確認。この1例から改善指示を採用しない。

## 計測

| 指標 | 保存済み基準 | 変更後 |
|---|---:|---:|
| 入力tokens | 6319 | 6354 |
| 生成tokens | 127 | 141 |
| suffix prefill秒 | 3.610284924507141 | 4.456525921821594 |
| decode秒 | 6.078314900398254 | 6.887244939804077 |
| KV再利用tokens | 5338 | 5338 |
| Text要求時間秒 | 10.844 | 12.556 |

モデル/grammarを保持したhelperで、同一の5,338 tokensのprefix snapshotを再構築し、独立forkからText生成した。prefix準備の応答はprefixPreparationOnly=true・generatedTokens=0であり、Groupingの推論結果ではない。TextではmodelReused/grammarPreparationReused=true・prefixReuseStatus=reusedを確認。

基準は以前のG/Cを実行した後のTextだが、今回はG/Cの出力生成を繰り返さず共通prefixのみを準備した。直前処理・ウォーム状態・測定時点とバイナリは異なるため、厳密な同時A/Bや速度の因果比較とは扱わない。元データと実送信本文/Schemaの一致範囲は検査済み。資源監視・raw応答は非公開で保持する。

Releaseビルド・Metal疎通・関連Python6テスト・git diff --check成功。7ファイルの保留中の伝播再実行とは別の実験で、その追加実行は行っていない。
