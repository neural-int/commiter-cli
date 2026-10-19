## 要約

現行仕様（共有120秒、overlayなし）で既存2ケース×履歴なし/ありを各1回実行した。最終Validator受理は2/4、探索的参照と意味が一致した完了例はC01-Bのみ。C02は両条件timeoutとなり、現行上限では安定して完了できない課題が残る。追加生成・retry・repairは行っていない。

## 検証結果

A=履歴なし、B=履歴あり。固定モデルGemma 4 E4B 4bit、temperature=0、seed=144。合計9要求（上限12）、完了したnative receiptは7。timeoutした2要求はreceiptなしだがGo側の要求・応答・errorを保存した。

| 条件 | 秒 | 要求数 | 最終Validator | 内容 |
| --- | ---: | ---: | --- | --- |
| C01-A | 63.49 | 3 | 受理 | 1 group一致 / refactorで参照fixと不一致 / 要約は404目的欠落 |
| C01-B | 84.51 | 3 | 受理 | 1 group一致 / fix一致 / 404修正と正規化の意味を充足 |
| C02-A | 120.27 | 1 | 失敗 | Groupingでtimeout、後続未評価 |
| C02-B | 120.28 | 2 | 失敗 | Groupingは3 groupsで参照1 groupと不一致 / Categoryでtimeout、Text未評価 |

- C01-Aの要約:「静的アセットの解決を導入し、画像コンポーネントとバッジを更新しました。」
- C01-Bの要約:「静的アセットのインポートを文字列URLに正規化するヘルパーを導入し、画像表示の404エラーを修正」
- C01のscope/summary長はA=16/35、B=16/48で契約内。C02-AはGrouping120.26秒、C02-BはGrouping110.32秒＋Category9.95秒で共有期限に達した。
- Go runnerは成功終了したが、これは失敗記録まで完遂した意味であり全ケース成功ではない。実行前のgo test ./...、go vet ./...、go build ./...は成功。
- 固定ソース148件・機械capture・Gold・helper・MetalのSHA一致、fixture2件のHEAD/index/対象ファイルbytes不変を完了監査で確認した。
- 完了7要求のnative helper最大RSSは約1.0–3.65GiB。GPU/Metal最大量ではなく、timeout要求分も含まない。途中のシステムsnapshotはfree36%、swap使用7776.56MiB。他作業は制御しておらず、この数値からtimeout原因を断定しない。

## 考察

今回の2fixtureでは機械capture12要求が旧captureとbytes一致した。旧処理でもRaw Diffが選択されていたため、今回のRawDiffProjection共通化・旧evidence選択廃止による品質改善/退行の比較にはならない。Text grammarも前回実モデル評価に含まれていた。これは現行120秒の再現確認である。

前回の360秒条件ではC01-A69.13秒/C01-B105.67秒で受理、C02-A280.26秒/C02-B285.58秒でTextまで進みscope=docs拒否だった。C01の分類・要約の傾向とC02-Bの3分割は今回も同じ。ただし時間予算が異なるので単純な高速化率として扱わない。今回C02-AはGroupingでも期限に達した。実行時間変動があり、各1回からコード退行や履歴の速度効果を因果推定しない。

Goldは従来のagent作成の探索的参照で、独立した人の確定Goldでもholdoutでもない。文章の意味評価もagentによるルーブリック照合であり、人手判定完了とは扱わない。scope=docsの禁止は現行コードに残るが、今回C02はその最終検証へ到達していない。

## Next Steps

- 今回は一巡で終了。条件延長、prompt調整、fixture縮小、追加生成をせず失敗を保持する。
- 現行120秒の完了率、C02のGrouping、scope規則をそれぞれ別の論点として扱う。Category/TextのKV再利用は#180の未実装検討事項のまま。
- 現行のraw結果・native receipt・analysis・完了監査はdocs/benchmarks/issue-178/iteration-03に保存。前回360秒のraw結果もiteration-02に保存して区別した。
- 入力契約変更と今回の検証記録はローカル未commit。今回、commit/pushや仕様修正は実施していない。
