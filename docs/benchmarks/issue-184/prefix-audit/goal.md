# H184-005 / prefix-audit

## 目的と方法

採用済み固定入力の実際のMLXチャットテンプレート適用後token列を取得し、先頭からの完全一致長を測る。Qwen3.5-4B、Thinking OFF、32K/4096、元と同じモデル・構造化出力説明。モデル生成・KV再利用・prefill計算は0回。ファイル5件の情報と履歴3contextsが3phaseで全て等しいことを確認した。

--benchmark-input-auditを追加し、全3profileで入力準備直後にtoken IDsを返す。COMMITER_184_AUDIT_WIRE=1はauditOnlyの場合だけ、完成したwireをそのままテンプレートへ渡す。通常生成では作用しない。Releaseビルド成功。元入力token数5123/4027/3911を再現した。原文・token IDsはprivate保存し、公開するのは集計だけ。

## 結果

| 条件 | Grouping入力tokens | Category入力tokens | Text入力tokens | G→C共通prefix | C→T共通prefix |
| --- | --- | --- | --- | --- | --- |
| 現行 | 5123 | 4027 | 3911 | 3 | 5 |
| user内項目順だけ変更（値の表記を保持） | 5123 | 4027 | 3911 | 3 | 5 |
| 共通情報を先頭へ置く探索案 | 4940 | 3926 | 3810 | 2871 | 2873 |

別途、JSON再serializeを伴う並べ替えも監査したが、表記変更も混じるので順序だけの評価には使わない。元の各fieldの文字列表現を保持した追加監査で、項目順だけではprefixが増えないことを確認。

## 原因と入力順

現行順はフェーズ固有system指示→同じsystem内の出力契約・動的Schema→user本文（履歴・files/groups等）→assistant生成開始マーカー。最初のsystem文がGroup/Generate…と異なるため、user内に同じ差分や履歴があっても連続prefixにはならない。途中の共通文字列だけに以前のKVを挿すことは、この種のprefix再利用ではできない。

探索案の順は共通system規則→userメッセージの共通Git差分・履歴→userメッセージのフェーズ固有指示・Schema・残りの入力。共通files/historyを先頭の辞書へ一度だけ置き、後半のfiles/groupsはfile_idで参照する。system指示の一部を後半のuserメッセージへ移し、共通systemが後続phase指示を適用する構成とした。現在の要求と同じ情報を持つが、役割・権限・参照形式が変わるため、単なる無損失な並べ替えとは扱わない。元の出力Schemaはホスト側で保持している。

3phase共通部分は少なくとも2871tokens。最初のphaseでそのprefix状態を確保できれば、Category/Textでそれぞれ約2.87Ktokensの再計算を避ける余地がある。これはtoken列の一致だけの結果で、秒数の短縮を測ったものではない。因果attentionの位置を含む全cache状態の正しい複製が条件。

## 実装上の注意と次の判断

当該Qwenは8 attention layers/24 other layersで、既存コードはMambaCacheも扱う。attentionのKVだけでなく再帰状態も正しく保持・複製・分岐する必要がある。現行のcacheReusedTokens=0は変えていない。元の共通systemを長くするだけでは数千tokensのdiff/historyを再利用できない。

採用前に、まず探索案の入力配置そのものをKV再利用なしで元入力と推論比較する。Grouping/Category/Textのスコア・最終validator・指示の権限・履歴を命令として扱わないことを確認し、その後で同じ探索案のcacheなし/ありを比較する。入力順変更とcache実装の影響を分離する。今回精度・速度は未測定で、ハーネス登録や本番入力順を変更しない。
