# Issue149 exploration

本番code/default/model/backendを変更せず、保存済みpinned modelと計測用helperのみ使用する。modelの取得処理は含めない。goldはpayloadへ渡さない。semantic IR、raw response、prompt、native thoughtはRAMのみ。stdoutはfixture名、numeric telemetry、stop分類、固定selected IDsのgroupingだけ。

```sh
go build -o /tmp/benchmark149 ./tools/benchmark149
/tmp/benchmark149 -architecture semantic-ir -fixture contract-independent-6 \
  -helper /path/to/instrumented/commiter-mlx-helper \
  -cache /path/to/existing/mlx-models
```

`baseline`は現行ThreePhaseGeneratorのStage1をそのまま実行し、検証済みgroupingがcategory入力へ渡った時点で計測を止める。metadata成功・plan成功には数えない。`raw-global`はIRを経由しない全体判断の診断用対照。

N<=16、per-call120秒、whole-fixture600秒、N+1calls、repair/retry0。モデルの完成stop、strict JSONとselected IDの全件一意割当を確認。unknown/missing IDやunresolvedで部分planを返さない。本ツールはplanを生成せず、Git mutationもしない。IRの事実忠実性やsemantic正解をJSON schemaだけで保証しない。

既使用contract/controlled fixtureと独立評価holdoutを分ける。holdoutは24b166bで初回結果確認前に固定した著者定義の合成要求で、実利用者判断のgoldやproduction精度分布ではない。採用に用いたら以後fresh holdoutに戻さない。

helperの数値instrumentationは#146の計測用コピーを再利用し、binary hashをenvironment.jsonへ保存。出力tokensにはnative thought/channelを含む。wallには毎callのmodel loadを含み、fixture構築、build、Git収集を含まない。初回推論と契約testは一部並行し、熱/CPUを隔離したlatency評価ではない。
