# Issue 162: evidence-based review relationship diagnostic

初回は作者commit境界の再現や最終グループ構築ではなく、自己作成4file入力内の対象2fileの関係診断。契約・入力・gold・helper/source/model/profileは事前固定。詳しくはdocs/benchmarks/issue-162/evaluation-contract.mdとiteration-1-report.mdを参照。

- `python3 -B -m unittest discover -s tools/benchmark162 -v`
- `python3 -B tools/benchmark162/benchmark.py preflight`
- 推論前に事前登録をcommit/pushし、`python3 -B tools/benchmark162/benchmark.py run`

出力は排他的作成のため、同じ成果物の上書き再計測は拒否する。再現時は新しい隔離checkoutで結果ファイルを保存退避し、同じpreregistered source/inputを使用する。元計測結果を置換しない。helper binaryとmodel cacheは実験環境の絶対path。helper sourceの固定公開baseと実際の変更byte archiveはenvironment.jsonのリンクに保存済み。新依存・helper build・model downloadは不要な計測だったが、別環境にcacheが無い場合は利用可能性を別途確認する。

`iteration-1-draft-preflight.json`は未実行の512token初稿。実際に使用したのは1536tokenの`iteration-1-preregistered.json`。helper neutral profileのfixed budgetから決定し、推論前に修正した。温度等はhelper sourceの固定設定。

source IDの有効性は説明の正しさを保証しない。API依存をtogetherと返したことは分類不一致であり、mergeしたcommitが不適切という意味ではない。unknownを返さないことも直ちにすべてのpartitionが不適切という意味ではなく、本診断の根拠不足検出の未成立を表す。小標本でのGemma優位を一般化しない。

## Iteration 2

対象pairの許容分割を評価し、新APIの依存方向はGo標準ASTのhostへ分離。`iteration-2-contract.md`が固定契約。初回の5分類NO-GOを変更しない。

- `go build -o /private/tmp/benchmark162-sourcefacts ./tools/benchmark162/sourcefacts`
- `go build -o /private/tmp/benchmark162-validate ./tools/benchmark162/validate`
- `python3 -B -m unittest discover -s tools/benchmark162 -v`
- `python3 -B tools/benchmark162/partition_trial.py preflight`
- 全fixture endpoint/counterfactual stateを監査し、事前登録をcommit/push後、`python3 -B tools/benchmark162/partition_trial.py run`

実験結果ファイルは排他的作成。元結果を上書きしない。draft/invalid-input/preparation-failureは推論前準備の履歴で、実使用入力は`iteration-2-preregistered.json`と`iteration-2-input-audit.json`。Gemma以外は今回推論しない。

runは自己作成Go fixtureを標準libraryのみ・network offでtemp directoryに作り、testsとtemp Git treeを検証する。外部repo code/test実行用ではない。test-assisted baselineはモデルへ渡さないtest実測を追加情報として使う。監査cacheを同じcase/ordered partitionで再利用するため、total wallはuncached production latencyと違う。

初回は関係分類、次試験は複数許容partitionの妥当性なので、8/16と13/16を同じ指標の改善と呼ばない。局所2fileの境界を評価したcontrolled4/6/8file workloadであり、全8fileの意味分割や#156の16file/80%採用成功ではない。
