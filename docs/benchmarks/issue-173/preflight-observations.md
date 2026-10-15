# Token計測の制限

推論開始前に既存tokenize-only補助バイナリへA候補版のメッセージを与えたが、3件とも`malformed_request`で拒否された。

1. 元のCategory requestに`preflight_only: true`を付与。
2. 同じメッセージのまま、補助バイナリの既知のGrouping契約に合わせ`generation_profile: bounded-grouping`、`output_tokens: 768`へ変更。
3. 2に加え、既存#171のGrouping用外部Schemaを指定。メッセージ内のCategory Schema指示は変更しない。

いずれもモデル生成は開始されていない。拒否結果は`preflight-errors.json`へ保持した。成功したtokenizationではないため、入力token数と出力token数は未取得である。byte数をtoken数として扱わない。固定した本番補助バイナリはtoken件数を返さず、今回その実装を変更していない。生成結果の意味評価とこの計測不足は別に記録する。
