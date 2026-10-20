# H182-003: Ministral公式tokenizer適合後の再測定

H182-002のMinistral Instruct/Reasoning各C01/C02を各1回比較する。前回の成果物は保持する。今回の簡潔化プロンプト候補は適用しない。

## 公式仕様と実装

- 公式モデルカードの指定に従い、検証専用Python 3.12環境へ `mistral-common==1.8.6` を導入した。Swift Package依存は追加していない。Pythonと推移依存はrunの `requirements.lock.txt` に記録する。
- 公式tekken.jsonのV13 serializerでsystem/user roleと連続userを処理し、出力token IDをSwiftのLMInputへ直接渡す。Swift側の独自role結合・chat templateはこの経路では使わない。
- Swift/MLXによるweightロード・prefill・decode・KV8bit/group64と既存JSON grammar/validatorを維持する。出力のbyte vocabularyは公式と照合した変換済みtokenizerを使う。
- Reasoning版は全131072語彙が一致。Instruct版は通常語彙が一致するが、ID34/35が公式では予約名、変換版ではTHINK名という差がある。入力は公式serializerの結果を使い、InstructではThinkingを有効にしない。この差を隠さずauditへ残す。
- BOS=1、EOS=2。Reasoning版のTHINK開始=34、終了=35。EOSとTHINK終了は区別し、THINK終了前のEOSを正常回答として扱わない。
- Reasoning版は公式SYSTEM_PROMPT.txtをcustom systemへ付加する。公式の一般的なMarkdown指定については、最後に「このタスクの最終回答は所定JSONのみ」と明示する。独自のTHINK指示追記と開始token強制をこの経路では使わない。
- Category/Textの公式token列とSwift実入力列を、両モデル×C01/C02の全4組で照合し、完全一致を確認した。共有prefix長も新しい列から計算し直した。
- 生成予算・Thinking予算・120秒cycle制限は設けない。進捗の停止300秒、監視失敗、RSS12GiBの安全停止は独立して扱い、中断を成功に含めない。
- 初回tokenと64tokenごとに数値進捗をstderrへ保存する。進捗の経過秒にはprefillを含む。最終decode tokens/sは完了メトリクスの生成時間を使う。

既存Xcodeビルドキャッシュに別checkoutへの参照が残っていたため、新規scratch directoryのnative SwiftPM Debugで今回のソースをビルドし、39テスト成功を確認した。前回binaryの流用はせず、Go側の評価binaryとfixtureのみ前回の固定版を使う。build経路の違いも含むため、時間差をtokenizer単独の効果と断定しない。

## 固定した公式asset

| モデル | 公式repository revision |
| --- | --- |
| Instruct | `b35d4dfe56c142746f54dbd64f579faab2744308` |
| Reasoning | `4a36357c811bf511a7b625d132e12f22408aac91` |

weightはH182-002と同じMLX量子化revisionを維持する。公式assetはweightとは別revisionとして記録し、manifestにファイル別SHA256を残す。

## 実行

前提: 上記revisionのtekken.json / SYSTEM_PROMPT.txt / manifest.json、固定済みMLX weight、専用Python環境、新規ビルドしたSwift helperと対応metallibをruntime directoryへ用意する。外部通信はダウンロード時のみで、推論はローカル実行する。

```sh
PYTHONDONTWRITEBYTECODE=1 /private/tmp/issue182-ministral-runtime/venv/bin/python \
  tools/benchmark182/mistral_retest.py \
  --runtime /private/tmp/issue182-ministral-runtime \
  --previous /private/tmp/issue182-suite-v3 \
  --out /private/tmp/issue182-ministral-retest
```

再実行時は新しいout directoryが必要。同じsuiteを上書き・再試行しない。

## 参照

- https://huggingface.co/mistralai/Ministral-3-3B-Instruct-2512
- https://huggingface.co/mistralai/Ministral-3-3B-Reasoning-2512
- https://github.com/mistralai/mistral-common/tree/v1.8.6
