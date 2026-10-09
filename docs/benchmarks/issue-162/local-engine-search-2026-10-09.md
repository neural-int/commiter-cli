# ローカル分割エンジン候補の調査（2026-10-09）

ユーザー条件: モデル保存サイズが大きくても、推論実行時3〜5GBなら可。4file超の拡張必須、先に変更を言語化する方式は不採用。今回の作業は一次情報・公開metadata・実機の資源確認だけ。download、依存追加、外部code/test、推論0。

## 結論

第一候補はGemma 4 26B-A4B IT＋TurboFieldfare。SSD expert streamingにより、通常MLXで全重みを常駐させず大きいモデルを使う候補。公式にcommit分割品質を測った記録ではなく、作者の短〜中入力の資源実測が条件に近いという選定。第二候補はGodwit＋Qwen3-30B-A3B、第三候補はGodwit＋GPT-OSS-20B。軽量比較候補はQwen3.5-4BのMLX4bit。

## 確認した事実

|候補|保存容量|作者報告の常駐/footprint|速度・条件|資格状態|
|---|---|---|---|---|
|Gemma4 26B-A4B＋TurboFieldfare|text約14.3GB|約1.9〜2.1GB footprint、4K KV付近|8GB M2 decode5.1〜6.3tok/s、M5 Pro31〜35tok/s。公開prompt約6〜3015tokens|最優先の資源検証候補。M3・16K・分割未測定|
|Qwen3-30B-A3B＋Godwit|16.1GiB|resident2.5GiB≒2.68GB|M4 Air decode2.3tok/s、短い初回TTFT4.5秒|3〜5GB候補、長入力peak/速度未測定|
|GPT-OSS-20B＋Godwit|11.2GiB|resident4.1GiB≒4.40GB|M4 Air decode2.8〜3.2tok/s、短い初回TTFT約5秒|上限の余裕小。長入力peak未測定|
|Qwen3.5-4B MLX4bit|safetensors3,034,300,695bytes|今回実測なし、推論時は重み＋KV＋scratch|新世代モデルの軽量対照。量子化repo名だけでruntime互換を認めない|5GB内は候補に留め、実測必須|

メモリ値は作者が用いた異なる指標。resident/footprint/RSS/Metal allocationを一つの同じ計測値として順位付けしない。短promptのTTFTを16file時間へ外挿しない。モデルのactive parameter数が小さいだけでは全体重みが省メモリになるわけではなく、上記はSSD expert streamingという別runtimeが必要。

TurboFieldfareは共有core約1.35GBをmapped Metal bufferに置き、必要expertをSSDから明示的なbounded slotへ読む。source量子化値を再量子化せずrepack。FP16 KVはcontextで増える。expert slot全容量約1.50GiB、OSのfile cacheも別に存在し、~2GBという短入力測定から16Kでの上限は保証できない。16K context設定はruntimeにあり、長入力prefillはchunk処理。同モデルの通常MLXとの生成token差も長めの比較で記録されており、custom engineの分割能力・出力終了・template・数値互換性を実測する必要がある。

公開M2の1017prompt/128generatedはprefill36.729秒、TTFT37.656秒。M5 Proの3015prompt/256generatedはprefill23.038秒。この差はhardware/SSD/workload差を含む。M3の16file総latencyは不明。

## 優先を下げた候補

- GPT-OSS-120B＋Godwit: "2GiB trunk"だけではない。README合計resident5.7GiB≒6.12GB、KV等の追加もある。現設定で3〜5GB条件外。保存59.2GiBも実機の内蔵空き30GiBを超える。
- Qwen3.5-9B MLX4bit: safetensors5,950,221,072bytes。重み全常駐の標準経路は5GB条件を重みだけで超える。3bit repoの調査では実重みを含まないものがあり、名前だけで実行候補としない。
- MLX-Flash: M3 Max36GBの"2.1GB free"は使用量2.1GBの意味ではない。3〜5GB推論の実証として扱えない。
- serve-mlx: 長contextの測定はM4 24GBでMLX peak12.86GiBのstress run。cache上限だけでtotal3〜5GBを保証しない。
- Qwen3-Coder-Next: coding専用80B/active3Bは興味深いが、今回3〜5GB総メモリの確認がなく、先行候補にしない。

## 実機適合性

Apple M3、16GB。macOS26.6.2、Swift6.4。TurboFieldfareの公開要件（Apple Silicon/macOS26/Swift6.2）に版番号上適合。内蔵Data volume空き約30GiBを確認。約14.3GBのstreaming installは容量上候補だが、build cache/余裕/実行時資源までの実証ではない。実機への取得やbuildは行っていない。

## 次に固定する資源試験

TurboFieldfare一候補に絞り、source/model revisionと依存・取得容量・入力token数を固定する。4K→8K→16K相当の入力でload/prefill/decodeのpeak physical footprint、RSS、Metal counters、OS pressure/swap、TTFT/totalを別計測。出力budget/timeoutも事前固定。小入力~2GBを根拠に16fileの合格としない。

資源資格が成立したら原diffからの独立/連動境界と全体8/16fileの妥当partitionを別に評価。最初からproduction helperに組み込まない。既存metadataモデルは別processにして同時常駐を避ける構成候補。メモリ低減成功と意味能力成功を別Gateにする。

## 調査pinと一次資料

- TurboFieldfare HEAD: 77e8f9c02ec5345f4d9b633961c938bbafefb757（2026-09-27）。https://github.com/drumih/turbo-fieldfare
- 作者資源実測: https://github.com/drumih/turbo-fieldfare/blob/main/docs/BENCHMARKS.md
- memory ownership/repack: https://github.com/drumih/turbo-fieldfare/blob/main/docs/SYSTEM_DESIGN.md
- context/runtime controls: https://github.com/drumih/turbo-fieldfare/blob/main/docs/RUNTIME_CONTROLS.md
- pinned量子化元: mlx-community/gemma-4-26b-a4b-it-4bit@0d77464eeb233a2da68ebf9d7dc4edaac7db956d（作者仕様。実取得未照合）
- Gemma元モデル: https://huggingface.co/google/gemma-4-26B-A4B-it
- Godwit HEAD: 094a91fbb202cb294d355a052964092ccce98f49（2026-08-07）。https://github.com/rayl15/Godwit
- Qwen3.5-4B量子化repo: mlx-community/Qwen3.5-4B-4bit@0e7ffd5c629ef7719d4cbc04069232580bfa9d9c（Hub APIでrevision/weight byte数確認）
- Qwen3.5-9B量子化repo: mlx-community/Qwen3.5-9B-4bit@8b2b98c00a6b4d291155e4890773ca8f769aee53（同API照合）
- Qwen作者の4B比較指標: https://huggingface.co/Qwen/Qwen3.5-4B （汎用benchmark、commit分割の改善証明ではない）
- 追加候補一次資料: https://github.com/szibis/mlx-flash / https://github.com/IDAH-BITBOX/serve-mlx / https://huggingface.co/Qwen/Qwen3-Coder-Next
