# H182-006 Releaseビルド比較（完了）

## 目的と条件

ユーザー指示によりH182-005の4ケースをRelease helperで各1回再測定する。Debugは保存済み測定値を使用し再実行しない。変更はSwift helperとそのSwift依存のReleaseコンパイル。ソース、依存revision、GPU shaderのmlx.metallib、Go driver、weight、入力、prompt、sampling、seed144、KV8/group64、Thinking ON・各phase1024、出力予算・時間制限なし、文法制約・進捗・監視・固定partitionを保持する。

Gemma C01/C02は簡潔化、Qwen4B C01/C02は現行prompt。warmup/retry/repairなし、新規4セル・最大12phase、直列実行。RSS12GiB・monitor failure・進捗300秒なしの安全停止は維持する。

## 事前監査

Releaseビルドとテスト後に生成を伴わない再captureを行い、Debug側と全要求、Category/Text実token列とKV構成が一致することを確認して契約を凍結する。既存原本は上書きしない。

## 判定

cycle秒、生成tokens/runtimeGenerationSeconds、Thinking数・強制終了、実Grouping、validatorと文章品質を分けて比較する。出力が違う場合は生成数の差と生成速度を区別する。各1回・実行時期の異なる比較なので、時期やバックグラウンド負荷の影響を除いた因果効果は未確定。ハードウェアやハーネスの他要因を同時に変更しない。

## 結果

全4ケースでplan受理・実Grouping基準一致。全12phaseの応答本文、生成tokens、Thinking数、強制境界の有無がDebug側と一致した。出力短縮による見かけの速度向上ではない。

| ケース | Debug token/s | Release token/s | 生成速度比 | Debug cycle秒 | Release cycle秒 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Gemma C01 | 16.95 | 22.36 | 1.32x | 135.46 | 95.86 |
| Gemma C02 | 10.38 | 11.46 | 1.10x | 350.89 | 357.37 |
| Qwen4B C01 | 15.70 | 17.33 | 1.10x | 189.59 | 167.01 |
| Qwen4B C02 | 11.13 | 12.44 | 1.12x | 458.45 | 416.63 |

生成token/sは全3phaseの生成tokens合計/runtimeGenerationSeconds合計。Thinkingを含み、prefill/load/文法準備を含めない。cycle秒は全体の待ち時間。Gemma C02は生成速度が上がった一方、cycle時間は僅かに増えた。

## 解釈と限界

観測した生成速度は全4ケースで約1.10〜1.32倍、Releaseでも11.46〜22.36 token/s。Debug設定には速度差が観測されたが、今回の30〜70 token/sとの隔たりをRelease化だけで解消する結果ではなかった。残る差の主因がハードウェア、Swiftモデル実装、同期、文法制約、KV設定のどれかは、この比較だけでは特定できない。

各条件1回で時期と背景負荷が異なるため、速度比をビルド設定だけの厳密な因果効果とはしない。GPU utilizationが高いことも帯域上限到達の証明ではない。通常の生成経路との比較やprofilingは今回行っていない。

Qwen C01 Category、C02 Grouping/Categoryの計3phaseで予算到達後のJSON移行が成立し、他phaseは自然終了。Qwen C02 scopeのinternal/gitstatなど、Debug側の文章品質上の問題も同じままである。形式受理は意味品質・本番適合を保証しない。後段は固定partitionで、end-to-end成功率ではない。

## 検証と成果物

ReleaseビルドのSwift40テスト成功。生成前に全要求・実token列・KV構成がDebug側と一致し、モデル・source・依存・shader・設定のhashを凍結。全12実送信要求のprompt変換と不変条件、全完了Thinkingカウンタ1024以下を実行後にも監査した。

OS監視215サンプル全件有効、monitor_failed=false。4セルのdriver時間は1075.24秒。

原本は<PRIVATE_TMP>/issue182-release-comparison/。公開用全テキスト証跡はraw/。source snapshot、Releaseビルド・テストログ、準備ログ、監査、要求/応答、進捗、OS監視、summary/completionを含む。weight/binary/metallib/build cacheは含めずhashで識別する。公開コピーはメール形式文字列とホーム絶対pathを置換し、raw/SHA256SUMS.txtに公開コピーhashを保存する。契約内のhashは置換前原本の値。

登録した4ケースは完了。採用promptとThinking条件を変更せず、追加測定・最適化は行っていない。Issue #182全体のGoalは未完了。
