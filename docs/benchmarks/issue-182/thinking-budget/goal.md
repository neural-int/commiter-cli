# H182-005 Thinking予算比較

## 要約

Gemma簡潔化とQwen4B現行プロンプトを固定し、Thinking ON・各フェーズ1024 tokensの4セルを測定した。全4セルでplan受理・実Groupingの探索基準一致。Gemmaは全6フェーズで自然終了し上限は発動しなかった。Qwen4BではC01 Category、C02 Grouping/Categoryの計3フェーズで予算到達後に境界を閉じ、有効JSON回答へ移行した。Qwen4B C02は最終Textまで完了した。

Thinking予算の本番採用は今回決めない。形式受理はsummary/scopeの品質やend-to-end成功の保証ではない。

## 条件と事前監査

ユーザー指定の採用決定は../prompt-adoption.md。Gemmaは簡潔化、Qwenは現行。Qwen2BはThinking OFFを維持し今回追加測定しない。新規4セル・12フェーズ、各1回、warmup/retry/repairなし。出力予算とcycle時間制限は無制限。sampling、seed144、weight revision、template/tokenizer、KV8bit/group64、facts、schema、validator、後段の固定partitionは保持した。

無制限側はH182-004の保存済み採用prompt結果を使用し再測定しない。全4セルでlive fixtureの再capture要求、Category/Textの実token列・KV構成が無制限側と完全一致。不正な0予算は生成前に拒否。Swift40テスト成功。

上限は各フェーズのnative prefix tokensで、従来どおり生成開始markerを含み終了markerを含まない。GemmaとQwenでは開始markerが生成側か入力済みかが異なる。上限前の自然終了を延長しない。予算到達時はGemma channel境界またはQwen </think>を既存logit processorで強制し、同じJSON grammarで最終回答を続ける。自然終了と強制遷移はnativeThoughtForcedCloseで区別する。

Qwen公式のThinking境界と予算例を参照したが、本実装はSwift連続decode上の境界制御であり、公式Pythonの2呼び出し方式と同一ではない。根拠と限界は../prompt-adoption.mdに記載。

## 結果

| ケース | 採用prompt | 無制限秒（保存済み） | 1024秒 | plan / 実Grouping |
| --- | --- | ---: | ---: | --- |
| Gemma ON C01 | 簡潔化 | 144.63 | 135.46 | 受理 / 基準一致 |
| Gemma ON C02 | 簡潔化 | 400.72 | 350.89 | 受理 / 基準一致 |
| Qwen4B ON C01 | 現行 | 267.12 | 189.59 | 受理 / 基準一致 |
| Qwen4B ON C02 | 現行 | 中断（約150分時点） | 458.45 | 受理 / 基準一致 |

### Phase別実測

G/C/TはGrouping/Category/Text。値は生成tokens（Thinking tokens）/decode tokens/s。*は予算到達による強制遷移で、全てcompleted・有効JSONを取得した。

| ケース | G | C | T |
| --- | ---: | ---: | ---: |
| Gemma C01 | 422 (360) / 19.04 | 678 (651) / 16.63 | 717 (676) / 16.20 |
| Gemma C02 | 652 (563) / 10.96 | 1049 (1022) / 9.32 | 810 (769) / 11.60 |
| Qwen4B C01 | 575 (515) / 19.50 | 1050 (1024) / 13.94 * | 824 (781) / 16.11 |
| Qwen4B C02 | 1097 (1024) / 10.44 * | 1050 (1024) / 10.88 * | 559 (513) / 13.44 |

### 解釈と文章品質

Gemmaの2ケースは全3フェーズの応答内容・生成tokensが無制限側と完全一致。最大Thinkingは1022で上限が発動せず、今回の時間差をThinking予算削減の効果とは扱わない。

Qwen4B C01は無制限のCategory Thinking1657から1024へ減り、633 tokens削減。生成tokens合計3082→2449。最終応答内容は全3フェーズで同一。cycleは267.12→189.59秒だが、同時期・同一binaryの対照実験ではなく因果的な短縮率とはしない。

Qwen4B C02無制限はGrouping Thinking57984 tokens・9037.53秒の進捗でユーザー中断。1024群はGrouping/Categoryで上限に達しJSONへ移行し、TextはThinking513で自然終了。成功した無制限対照がないため速度改善率を算出しない。

Qwen4B C02のscopeはinternal/gitstatで、実ディレクトリinternal/gitstateと一致しない表記を残した。summaryは「Issue#167のPhase1で、一時indexとobjectを保持した構造経路の検証」。Gemma C02のsummaryは「Issue #167 Phase.1「ファイル優先構造」を実装・検証完了。」。形式検証の受理を、記述した検証完了の実証や文章品質の保証としない。

## 実行後監査と成果物

全12実送信要求で予定どおりのprompt変換と他入力の不変を再確認。全完了応答のThinkingカウンタは1024以下。OS監視233サンプルは全件有効。driverは4セル・1166.94秒、monitor_failed=falseで自然終了した。監視・driver時間をモデルの生成区間時間と混同しない。

原本は<PRIVATE_TMP>/issue182-thinking-budget/。公開用全テキスト成果物とsource/checksはraw/。メール形式文字列とホーム絶対pathを公開コピーで置換し、raw/SHA256SUMS.txtで公開コピーのhashを記録する。実行契約内hashは原本の値。前回の無制限群と中断ログはH182-004の保存資料を参照し、上書きしない。

## 限界と次の判断

各条件1回のC01/C02探索比較。独立人手承認Goldと16ファイル評価は未実施。後段の固定partitionを使うためend-to-end成功とは区別する。nil予算の処理は従来と同じだが、予算を接続するためhelper binaryを再buildし、無制限群とは実行時期も異なる。token数・境界成立・出力一致の観測と、時間差の因果解釈を分ける。

今回登録した4セルは完了。Gemma簡潔化/Qwen現行というユーザーの採用決定を維持する。1024予算は回答遷移が成立した実験結果として記録し、予算の採用・追加反復・本番組み込みは別の判断とする。Issue #182全体のGoalは未完了。
