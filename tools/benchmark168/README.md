# #168 Purpose Discoveryの有限評価

研究用ツールと公開fixtureだけを扱う。製品CLI・ThreePhase・validator・4file上限を変更しない。元のrepository index/worktreeにcommitを作らず、自己作成一時repositoryからsnapshotとsyntax evidenceを収集する。実履歴fixtureもsourceの読み取りと解析だけを行い、当該code/testは実行しない。

`prepare.go`のfixture/collect/prepareと`projection.go`は#167の実装を再利用した。固定helperの`bounded-grouping`profileを使うが、目的発見のprompt/schemaを入力する。新しいdecoder/helper/model/依存は追加しない。

今回の固定入力・モデル・source/helper/binary hashesは`docs/benchmarks/issue-168/preregistration.json`。v1の無効request4回は別receiptとv1契約へ保持し、結果を隠したやり直しにしない。v2は2known+10当初holdout、総process呼び出しは16。

公開requestの`model_path`とfixture rootはplaceholder。実際のmodel cache/fixture/helperの絶対パスは公開対象外のruntime receiptで解決する。再現には同revisionの既存local model、同digestの#167 measured helper、Go、Python標準機能が必要。新規downloadは行わない。

```sh
go build -o /tmp/commiter-168-prepare ./tools/benchmark168
/tmp/commiter-168-prepare -cache <installed-model-cache> -out <new-input-directory> -runtime <local-runtime-json>
```

作成されたlocal runtime JSONへ`helper`と`prepare`の実行ファイルパスを追加する。再現用入力を使う場合は独立した新規事前登録を作り、既存のreceiptを上書きしない。今回の固定`run.py`は保存済みfixture rootとhashを使い、同じcellを再実行しない。

```sh
python3 tools/benchmark168/run.py --cohort known --runtime <local-runtime-json>
# known出力を追加LLM生成なしでgold/inputに照合してknown-evaluation.jsonへ記録する
python3 tools/benchmark168/run.py --cohort holdout --runtime <local-runtime-json>
# 各説明のsemantic/target gold binding・grounding・理由をholdout-evaluation.jsonへ記録する
python3 tools/benchmark168/summarize.py
python3 -m unittest discover -s tools/benchmark168 -p 'test_*.py'
```

`semantic_gold`は具体的な振る舞い変更に対応するgold、`target_gold`は同じ変更対象を示すgold。後者だけ一致した一般的な機能説明をcoverageへ数えない。over-merge/splitはtarget対応、groundingはモデルが実際に受け取った入力、hallucinationは未根拠の作用・追加・型などの主張を別々に記録する。オフライン評価は判断を含むため、本文・入力・goldと理由を併記し、機械的テストPASSと混同しない。

通常/逆順は独立fixture件数ではない。H4の既存fixtureとの重複、H6の受入失敗、同一作者由来の実履歴という限界は結果報告に分離して残す。Gate 1未達ならAssignment/paired Grouping/mixed fallback/Metadata/productionは未評価のまま終了する。
