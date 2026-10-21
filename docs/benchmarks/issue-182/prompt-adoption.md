# Issue #182 プロンプト採用決定

2026-10-21のユーザー指示による決定。

- Gemma 4 E4B：H182-004で測定したprompt-candidate.mdの簡潔化プロンプトを採用。
- Qwen3.5-4B / Qwen3.5-2B：現行プロンプトを維持。
- Thinking条件は維持。予算比較の対象はGemma ONとQwen4B ON。Qwen2BはOFFのみで今回の予算比較対象外。

この採用決定を研究ハーネスで固定する。通常CLIへの本番組み込みはこの検証に含めない。Gemmaで1回ずつ観測した時間短縮を安定性・文章品質の保証とは扱わない。

## H182-005 Thinking予算比較

各採用promptで、保存済みのH182-004無制限結果と、1024 tokensをC01/C02で各1回新規測定して比較する。新規4セル、最大12フェーズ、warmup/retry/repairなし。Qwen4B C02無制限はユーザー中断として比較し、完了時間・完了出力のある成功例として扱わない。無制限群は再実行しない。

変更するのは研究設定のthinkingBudgetのみ。モデルweight revision、tokenizer/template、sampling、seed144、KV8bit/group64、prompt、facts、schema、validator、固定partitionによる後段比較、無制限の最終出力予算、cycle時間制限なしを保持する。生成前に現在のfixture要求・実token列・KV構成を採用promptの無制限側と照合する。

上限はフェーズごとに1024 native prefix tokens。従来カウンタと同様、生成された開始markerを含み終了markerを含まない。Gemmaは開始markerを生成し、Qwenはtemplateが開始markerを入力済みのため、通常の本文token数が完全同一とは限らない。自然終了が上限より早ければ追加Thinkingしない。上限到達時はGemmaのchannel境界またはQwenの`</think>`を既存logit processorで強制し、その後は同じJSON grammarで回答を続ける。nativeThoughtForcedCloseで自然終了と強制遷移を区別する。EOSによる単純打ち切りは成功に含めない。

Qwen公式はThinking終了後に最終回答を生成する形式を示し、Qwen3向け予算例ではThinkingを閉じたassistant継続を使う。本実装はSwift上の連続decodeとgrammarへ適用するもので、公式Pythonの2呼び出し方式やQwen3.5固有のサービス実装と同一ではない。境界のテストと実測で回答遷移を検証する。

- https://github.com/QwenLM/Qwen3/blob/main/docs/source/getting_started/thinking_budget.md
- https://huggingface.co/Qwen/Qwen3.5-4B

結果では、自然終了/予算到達、Thinking数、最終JSONの成立、validator、実Grouping、summary品質、cycle秒、decode token/sを分ける。RSS12GiB・監視失敗・進捗300秒なしの安全停止は維持し、生成予算とは別に記録する。モデル・promptの追加調整はしない。

予算対応のためSwift binaryは再buildする。nil予算の分岐は従来と同じであることをテスト・差分・token監査で確認するが、同一binaryを同時期に両群へ適用した再測定ではないため、速度比較には時期とbinary差の限界を記載する。
