# H182-009 生成経路オーバーヘッドの切り分け（準備）

## Goal

MLX生成とcommiter独自ハーネスの差を、Go側の経路とSwift内processorの差に分けて観測する。今回はprepareのみ。生成測定、本番設定変更、依存追加は行わない。

## Cases

| モデル・fixture | 段階1（各arm 3phase） | 段階2（各arm 3phase） |
| --- | --- | --- |
| Gemma E4B C01・簡潔化 | Go→direct | bounded-cap→standard-cap |
| Gemma E4B C02・簡潔化 | direct→Go | standard-cap→bounded-cap |
| Qwen3.5-4B C01・現行 | Go→direct | bounded-cap→standard-cap |
| Qwen3.5-4B C02・現行 | direct→Go | standard-cap→bounded-cap |

各段階8セル・最大24phase、合計16セル・最大48phase。各arm1回。GPUは直列。各stageは明示的な別コマンドでのみ開始し、自動連結しない。

## Stage 1: transport comparison

新しい同一Release helperで両群を再測定する。Go群は既存evaluation.test→Python proxy→helper。direct群は保存済みの同一actual-request 3件をhelper --benchmark-cacheに順に入力。モデルをarmごとに新規プロセスへロードし、同一armの3phaseでモデル・grammar preparation・Category/Text prefixを保持。Go群の後段には既存参照partitionを注入。direct群はその同一要求を再生するのでGrouping結果を後段要求へ反映しない。入力token、実応答、Thinking/終了、decode速度が一致するかを確認して全体時間を比較する。Goが実際に送った要求とdirectの再生要求を実行後にも全phase照合し、要求または応答が異なる場合、全体差をGoの負荷と断定しない。captureのCategory stubはfixで実生成C02のfeatとは異なるため、directはcaptureでなく保存actual-requestを再生する。

Go群の全体時間にはfixture準備・proxy・planner/validator・helper起動が含まれ、direct群のdriver時間には保存要求ファイルI/O・起動・通信が含まれる。差は経路全体の差であり、Go言語やIPC単独の負荷ではない。通常のgenerationWallSeconds/runtimePromptSeconds/runtimeGenerationSeconds/load/grammar/prefix計測も分ける。詳細per-token CPU timerはOFF。

## Stage 2: generation-components comparison

両群ともdirect入力、同一MLXLMCommon.generate/TokenIterator、同一KV8と独自prefix prefill/reuseを使用。bounded-capは従来A/eagerのGrammar/NativeThoughtBudget/BenchmarkProgress processorを保持。standard-capはGenerationComponents()でcustom logit processorを与えず、grammar準備、文法制約、Thinking状態制御、token単位progressを省く。MLX標準sampler、通常のEOS/stop処理、AsyncStreamのtext出力は維持する。これは標準generateを同じhelper内で呼ぶ比較であり、独立した公式CLIやMLX全体の最高速度測定ではない。

段階2だけ両群のmaxTokens=1024/phaseを設定する。EOS/stopで早く終了し得るため固定長を保証せず、実tokens・decode時間・生のstop理由を報告する。length到達を診断の正常終了として保存し、raw textを返す。JSONでない出力や途中Thinkingをtask成功と扱わず、validator品質比較の対象にしない。bounded-capのThinking予算1024は保持するが、同じ1024の生成上限に先に達する場合があり、Thinking終了誘導や最終JSONのコストを網羅しない。standard-capのtemplateはThinking ONだが独自Thinking予算制御はない。全体差は複数processorをまとめた差で、各要因の寄与は段階3まで未特定。

両群共通のAsyncStream consumerでraw textを蓄積し、chunk受信時に最大5秒ごとの進捗通知を行う。追加のtoken.itemは呼ばない。出力chunk構成による観測負荷の差は残る。通常のresponse容量安全上限も維持。

## Fixed conditions

重み4bit、KV8/group64、eager、同一Release binary/metallib、同一依存revision、同一入力・schema・chat template・seed144、prefix長と再利用を固定。Gemma temperature/top_p/top_k=1.0/0.95/64、Qwen4B=0.6/0.95/20。両モデルtemplate Thinking ON、Gemma簡潔化・Qwen現行prompt。段階1はThinking予算各phase1024・最終出力予算/時間制限なし。段階2の生成上限は診断専用で採用条件を変更しない。warmup/retry/repairなし。Metal traceなし、詳細CPUタイマーなし。C01/C02は探索用参照、独立した人手Goldなし。

RSS12GiB、monitor failure、出力進捗300秒なしで停止。system swap/pressure/thermalとMLX memory/KV snapshotを記録し、環境差や生成長差を制御処理単独の効果と断定しない。失敗原本を保持し、自動再試行しない。

## Preparation gate

prepareで直前KV8実行の入力要求と一致を確認し、token/cache audit（生成なし）を保存する。stage1/stage2開始時はsource、入力、config、binary、shader、driverをhash照合し、既存結果を上書きしない。Releaseテストを通す。モデル生成を伴うbranchの実動確認は測定開始後に行うため、prepare成功を実測成功とは扱わない。

## Commands / Artifacts

- `python3 tools/benchmark182/path_compare.py prepare`：capture、token/cache監査、契約固定。生成なし
- `python3 tools/benchmark182/path_compare.py stage1`：8セルの経路比較。未実行
- `python3 tools/benchmark182/path_compare.py stage2`：8セルのprocessor比較。未実行

原本は<PRIVATE_TMP>/issue182-path-comparison/。raw requestやmodel pathを含む契約はローカル保持。公開時は既存のsecret/path除去とmanifest検証を行う。過去の時間値を今回のbaselineへ流用しない。

## Preparation correction

初回prepareのC02 Text要求比較で、captureのCategory stub=fixと保存実生成のfeatの差を検出し停止。生成は行っていない。初回監査は<PRIVATE_TMP>/issue182-path-comparison-prepare-attempt1/へ保持。captureは過去のcaptureと照合し、直接再生には保存actual-requestを使用するよう修正した。新旧helperに同じactual-requestを入力した生成なしtoken/cache監査で一致を確認する。
