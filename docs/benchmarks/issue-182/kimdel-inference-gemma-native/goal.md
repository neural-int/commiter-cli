# H182-015 Gemmaプロンプト変更＋native KV比較

## Goal

GemmaだけでThinking ON/OFF×grammarあり/なし4条件を測る。H182-014の固定fixtureと32K入力・4096出力・計測を維持し、プロンプトとKV量子化を同時変更する。

## Result / Decision

全4条件・12phase正常測定終了。536.589秒（約8分57秒）。監視268件すべてvalid、fatalなし。Grouping全正解4/4、Category全正解3/4、validator受理3/4。Grouping/Category/validatorを全て満たし、文章も差分・テストに沿う条件はON両条件とOFF grammarありの3/4。OFF noneはCategoryのメール修正をfeatと誤り、Text scope17文字でも拒否。出力上限到達0。再試行・モデル採用なし。終了後helper/evaluation不在を確認。

## Changes / Conditions

Gemma E4B固定revision、sampling1/.95/64、seed144、eager、各phase別Release helper、固定Git差分/履歴、32K/4096、時間制限なし、warmup/retry/repairなし。実Groupingを後段へ渡す。モデルweight/fixture不変。モデル入力は今回もコマンド14件・会話3件のGrouping文脈を保持。
変更1: Gemma用compactは基本ルールを維持し、Categoryにdocs=文書のみを追加。Textは全type語を禁止scopeとして列挙。全ファイルG001/type=featに偏る自動構造例を研究用env flagで除去し、フィールド/型/必須/完全Schemaは維持。
変更2: COMMITER_180_KV_BITS=native。KVキャッシュそのものを無効化した実験ではない。SwiftはkvBits=nil/quantized cache構築分岐を通らずgenerateへcache=nilを渡し、MLX標準native cache生成に委譲する。requestedKVBitsは全phase未設定。
重要な計測制約: native経路ではworkingCacheがnilのままruntime内部でcacheを生成するため、phaseEndKVLayersは空。実際のnative KV各層のdtype/bytesは今回exportされていない。前回KV8との実容量比較は不可。native指定とソース経路の検証を、実配列dtype/容量の実測と混同しない。追加計測・再推論は行わない。
プロンプトとKVの2要因を同時に変更。今回の改善をどちらか単独の効果とは断定しない。各1回、時間や熱条件も固定できていない。

## Cell measurements

Categoryは正解集合の群だけを採点。decode速度は全phase生成tokens/全decode時間。Thinking込みでload/prefillを除く。

| 条件 | Grouping | Category | validator | 秒 | 生成tokens | decode token/s |
| --- | ---: | ---: | --- | ---: | ---: | ---: |
| gemma-on-json_schema | 3/3 | 3/3 | 受理 | 191.67 | 3709 | 24.99 |
| gemma-on-none | 3/3 | 3/3 | 受理 | 209.37 | 3745 | 22.51 |
| gemma-off-none | 3/3 | 2/3 | 拒否 | 62.63 | 300 | 18.77 |
| gemma-off-json_schema | 3/3 | 3/3 | 受理 | 62.86 | 201 | 17.13 |

## H182-014 comparison

| 条件 | 前回Grouping→今回 | Category正解数→今回 | 前回秒→今回秒 | decode token/s 前回→今回 |
| --- | --- | --- | --- | --- |
| gemma-on-json_schema | 0/3 → 3/3 | 0 → 3 | 179.01 → 191.67 | 24.38 → 24.99 |
| gemma-on-none | 0/3 → 3/3 | 0 → 3 | 191.62 → 209.37 | 21.92 → 22.51 |
| gemma-off-none | 3/3 → 3/3 | 1 → 2 | 66.27 → 62.63 | 15.94 → 18.77 |
| gemma-off-json_schema | 1/3 → 3/3 | 0 → 3 | 63.61 → 62.86 | 16.68 → 17.13 |

前回ON両条件は全5ファイル1群。今回は3群へ分かれ、Categoryもdocs/feat/fixに正解。OFF grammarありも2群から正解3群へ改善。OFF noneはGrouping正解を維持しCategory1/3から2/3に改善したが形式拒否。

ONの生成量は前回各3246に対し今回3709/3745。Thinking rawも3160から3506へ増加。従ってONの全体時間は短縮していない。回答が異なるためKVの速度効果として扱わない。OFFは約63–66秒から約63秒で大幅な短縮なし。

## Thinking / Text

ONのThinking rawはG1444、C1020、T1042で両条件同じ。forcedClose=false、全phase自然終了。ONのTextはREADME品質チェック、UI残日数tooltip、空の山括弧からloc:キー生成を各群に記述。loc:キーは追加された回帰テストに明示されており、生成された説明を幻覚とは判定しない。OFF grammarありも3目的を日本語で記述。OFF noneはscope=mail/store_readerが17文字でmaxLength16に違反し、Categoryでメール修正をfeatとする誤りが残る。

## Performance / resources

helper RSS最大3.571GiB、MLX peak最大4.796GiB（別指標、加算不可）。CPU busy平均9.99%、system GPU範囲[0, 100]%。thermal {'nominal': 82, 'fair': 186}。swap MiB開始/終了/最大[2662.94, 2654.94, 2662.94]。free%範囲[29, 69]。同時helper最大1。
2秒間隔監視、温度実数・消費電力・プロセス別GPUは未取得。背景処理を含む。全phaseのload/prefill/grammar/decode/host/CPU user/system/prefixはphase-timing.mdとmeasurements.json。native KV dtype/bytesのみ前述の理由で未取得。

## Limits / Next

今回のfixtureで改善を観測したが、一般精度・再現性・プロンプト単独効果は未確定。次に因果を切り分けるなら改訂prompt+KV8または旧prompt+nativeを別実験として比較する。現時点で追加試行はしない。原本と前回結果を保持。生の履歴・要求・応答を公開しない。
