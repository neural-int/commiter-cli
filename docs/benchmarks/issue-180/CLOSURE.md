# Issue #180: 最終採用と検証終了

## 要約

2026-10-19、ユーザー判断によりH180-004（8bit KV保持）を正式採用し、現時点の実装・証跡をmain向けPRにまとめて#180を区切る。新しい最適化や追加推論はこの終了作業に含めない。

## 採用する構成

- 一計画内でmodel/tokenizerと不変grammar準備を保持し、phaseごとのconstraint/matcher/native stateは独立させる。
- Category/Textの一致する共通prefixだけを再利用し、Categoryの思考・回答は引き継がない。
- cache一括評価、512tokenごとの同期的eval、4chunkごと＋末尾の明示同期/Memory.clearCache()を維持。
- global4層のK/Vを最初のprefillから8bit affine・group64で保持。native rotating20層はBF16。
- tools/benchmark180/h180004-helper.shで8bitを固定。H180-005（4bit）は未採用の比較記録。
- 基準測定はiteration10の8bit。次回比較はbaseline_harness_id: H180-004。

## 検証結果

C01/C02の8bit一巡6要求は入力bytes一致、Validator受理、量子化・grammar再利用・同期解放回数のgateを通過。8bit/4bit比較全体は12要求、監視305有効サンプル、監視途絶なし。Swift既存37tests、Metal smoke、Go関連testsが成功。PR準備時にも全Go test/vet/build、diff checkと起動口を確認する。追加LLM推論・judge・retryなし。

C02のMLX activeピークはBF16基準5.197→8bit4.936GiB、prefix active＋cache標本最大6.521→5.911GiB。全体時間は212.772→229.748秒で、高速化は確認していない。各条件1回・過去比較・熱/CPU負荷/出力差の制約を保持する。

## 未完了・未解決として残す事項

- 保存された通常CLIはphaseごとにhelperを起動する経路であり、研究用のcycle/prefix保持を通常CLIへ接続する変更は含まない。
- 通常経路へ接続した過去の未commitコードは現checkoutに存在しない。mainへの証跡統合を、その復元やproduction適用と扱わない。
- productionの既定120秒は変更していない。検証timeoutなしの完了を120秒内適合と扱わない。
- C02後段は固定参照group注入。意味品質の参照はagent作成・人間未承認で、一般的な品質保証ではない。C02要約には一時index/構造replay検証の説明不足が残る。
- 摂氏温度・プロセス別GPU使用率は未取得。swap/熱/他プロセスの因果帰属はしない。

このIssueの終了は、上記制約を残したH180-004検証成果の採用判断であり、当初のproduction導入条件すべてが達成されたという宣言ではない。

## 証跡

[標準契約](BASELINE.md)、[採用経緯](H180-004-adoption.md)、[iteration10報告](iteration-10/results-report.md)、[実行契約](iteration-10/execution-contract.json)、[固定rubric](iteration-10/quality-rubric.json)。過去報告の未採用・NO-GO・失敗記録は書き換えない。raw入力・応答・監視詳細はGitに含めず、必要な場合はローカルまたはCI Artifactsに保持する。
