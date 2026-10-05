# Issue149 architecture exploration

本番planner/default/model/backendを変更せず、保存済みpinned checkpointと計測用helper copyのみ使用する。model取得処理は含めない。goldはmodel payloadへ渡さない。prompt、IR、raw response、native thoughtはRAMのみ。stdoutにはfixture名、数値telemetry、stop分類、固定selected IDのgrouping、最終validation結果だけを出力する。Git mutationは行わない。

```sh
go build -o /tmp/benchmark149 ./tools/benchmark149
python3 tools/benchmark149/prepare_helper.py mlx-helper /tmp/new-experiment-helper
# 既存のlocked dependencies/build cacheを再利用してhelperをbuildする。
/tmp/benchmark149 -architecture assertion-facts -fixture contract-independent-6 \
  -helper /path/to/instrumented/commiter-mlx-helper \
  -cache /path/to/existing/mlx-models
```

`prepare_helper.py`のdestinationは新規ディレクトリを指定する。production helperは変更しない。実測に使ったhelper source/binary hashは`docs/benchmarks/issue-149/environment*.json`に記録する。現在のscriptはIteration10までのexperimental profilesとcached revisionsを許可するため、初期iterationの再現には対応commitのscriptを使用する。

| architecture | semantic responsibility |
| --- | --- |
| baseline | 現行ThreePhaseGenerator。既定はStage1のgroupingを確認後categoryの直前で停止 |
| semantic-ir | per-fileのbefore/after/changed-contract/symbolをmodel抽出し、全体へ渡す |
| batch-ir | 最大4fileのkeyed extractionを全体へ渡す |
| raw-global | 圧縮なしの全file raw diffを1callで比較する診断対照 |
| grounded-facts | hostが変更前後のliteralを観測し、soft graphと全体判断へ渡す |
| contract-facts | Go ASTのunique caller/callee観測を追加 |
| assertion-facts | 対応testのinputと期待条件観測を追加 |
| observed-contracts | 具体的subject/before-after/member/根拠を伴う全体contract出力。draft診断はfinal gateと別の数値で記録 |
| observed-anchors | host保持の観測契約E-IDへのglobal所属。root実在/自己所属を検証 |
| canonical-contracts | source/path順とmodel IDをhostで固定し、宣言・定数・関数body観測も追加 |

Go observerは公開合成fixtureのmodule `fixture`とfull before/after diffを対象にしたprototypeである。Go type checkerや汎用module resolver、partial Git diffからのfull source収集は実装していない。解決できないsymbolを推測しない。soft relation、test-pass、graph componentを不可逆なcommit boundaryにしない。

`-reverse`はincoming file順だけを反転する。`-global-profile bounded-global-contract`はGemma native1024/output1536、`bounded-routed-grouping`はnative0/output1536で、`-group-model` / `-group-revision`の保存済みcheckpointをgroupingだけへ割り当てる。`-group-cache`を明示するとgrouping checkpointだけ別の既存pinned storeを使用できる。base Gemma cacheの移動や複製は不要。metadataは元のGemma/profileを使う。暗黙fallbackはない。routingモデルの固定revisionはenvironment-routing.json参照。

N<=16、context16K、per-call120秒、grouping-only全fixture600秒、最大N+1calls、repair/retry0。各方式の実callsはJSONLへ記録する。最終JSONをstrict decodeし、duplicate key、unknown/missing selected ID、unknown group、unresolved、non-completed stopを拒否する。停止時のFM/FSはnullで、部分割当を正解として数えない。structural validationだけで意味的正解を保証しない。

`-metadata`を明示した場合だけ、completeなglobal grouping後にcurrent Japanese category/text契約を使用し、最大4groupずつ生成して`planning.Validate()`をauthoritative gateとする。4fileのwhole cycleは現行120秒、5〜16fileはexperimental240秒。失格candidateにmetadataを実行して採用扱いすることはない。metadata含む4file比較はIteration10のserialized結果を参照する。

既使用contract/controlled fixtureと独立評価holdoutを区別する。holdout6は24b166bで初回結果確認前に固定した著者定義の合成要求で、実利用のgoldやproduction精度分布ではない。holdout16には5つの既使用purposeと3つの追加purposeを含み、全部をfreshと呼ばない。採用に用いたfixtureをfresh holdoutへ戻さない。

input tokensはhelperのinput.text.tokens.size、outputはinfo.generationTokenCountで実測し、native thought/channelも含む。nullは取得不能であり0ではない。wallには各callのmodel loadを含み、fixture構築/build/Git収集は含まない。初期の一部はhost tests/buildと並行。Iteration10のinitial full4は別model推論とも並行したためlatency比較から除外し、原記録を保持してserialized比較を追加した。熱やCPUを隔離したproduction latency評価ではない。

入力監査testは、同じ観測入力と異なる著者partitionを比較する識別可能性の検査である。counterfactual goldをmodelへ送信しない。runtime boundary testは公開cross12 fixtureの5中間状態だけをtemporary directoryで実行する。利用者repositoryの任意コードを実行する機能ではない。両監査はmodel精度の測定ではなく、観測入力・test-passだけでは著者の変更目的を一意に証明できない範囲を記録する。

未推論fresh protocol8はIteration11 cross12結果確認前に固定した共有protocol変更＋独立health変更。現時点で資格試験に失敗しているため推論を開始していない。H8/H9の全体結果を参照し、rootの実在・groundedな構造をsemantic proofへ読み替えない。
