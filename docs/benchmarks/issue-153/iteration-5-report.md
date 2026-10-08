## 要約

実履歴fd042894の修正と回帰testの対応を四状態で確認。旧実装＋新testのみ失敗、新実装＋新testは成功。ただしinline表現は2fileで285unitとなり、現Cの8unit上限とBの64unit上限を超える。入力を間引いたり上限を緩めず、モデルcallは0。現候補NO CANDIDATEを維持する。

## 検証結果

出典fd0428947c752766d936275f2d42caa10732993d、parent c643154b0498e5d441efdb864a6cee26e3deb2e0。対象はinternal/relation/extract.goとextract_test.go。秘密情報や作者情報をfixtureへ保存せず、source/testとSHAだけを使用した。

| 実装 | テスト | 対象回帰test |
| --- | --- | --- |
| 旧 | 旧 | pass |
| 旧 | 新 | fail: resolved import also reported unsupported |
| 新 | 旧 | pass |
| 新 | 新 | pass |

対象はTestDirectImportUsesObservedSyntaxAndUniqueChangedTarget、各count=1。全suiteの証明ではない。新testは正常に解決したimportがunsupportedにもなる挙動を検出し、実装側はimport evidence kindを限定している。

inline extractionは実装146unit、test139unit。両fileの全再構築とforward/reverse stagingは成功。C solverの2〜8unit、B payloadの64unit条件を超えるため、意味品質計測は未実施。semantic goldはnullのまま保存した。目的が対応する証拠は得られたが、全285atomが一つのgold intentか、他の変更との独立性は未認定。

## 考察

合成の単文字変更と違い、通常のコード置換は大量のUTF8 atomになる。この実例で細粒度表現をそのままall-pairs/Bell enumerationへ渡す方式のscaling制約が具体化した。抽出が正確でも、候補の処理能力に適合するとは限らない。上限を増やすだけでは解決にならない。

次の課題はgold非依存の粗粒度operationと、必要に応じた分割の責務境界。単純な同symbol全結合は複数intent境界を潰す可能性があり、Aの既存18ケースを保持できるかを検証する必要がある。今回のtest対応をgoldとしてunit統合することはしない。実例1件は独立holdout一般化の証明ではない。

## Next Steps

- 新規A専用worktreeで、goldを参照しないoperation groupingの最小仮説を事前固定する。実コードのatom爆発を抑える表現がC探索に必要なため。
- 既存18 intra-file fixtureの境界表現と今回の実例の再構築/stagingを同時確認する。粗粒度化によってA GOの根拠を失わないため。
- 構造能力が保たれなければ候補を棄却し、285unitの間引き・gold別統合・solver上限だけの緩和は行わない。B停止、C現候補No-Go、D条件未達は保持する。
