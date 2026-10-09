# Issue #156 検証専用prototype

productionのplanner、default model、4-file上限を変更しない。Skill、新しい依存、cloud推論、外部repositoryのtest実行を使用しない。

- `planner.py`: #151固定抽出器に対するfile fallback、完全byte所有権検証、使い捨てrepoのtemp-index replay。
- `selective.py`: bounded group proposalの順位とmember全体の適合性を分けるB。NO-GO結果を閾値変更で修正しない。
- `audit_c.py`: 使用済みgoldのsubset stage可能性のみ。oracleは意味予測やruntimeの割当に使用しない。
- `evaluate_d.py`: 確認済みAだけをfinal candidateとし、現行three-phaseと固定H23を比較する。
- `build_history_adapter.py`: 固定commit e88f61bbから自repositoryのbenchmark/internal資産のみを`/private/tmp`に展開し、公開履歴入力用の薄いadapterを追加してbuild。H23のgrouping/observer/metadataを変更しない。
- `fixtures151`: 使用済み診断。directory名にindependentがあっても本Issueの新holdoutではない。
- `corpus`: 未使用公開repositoryに由来する固定snapshotと出典。全repositoryのコピーではなく選択変更ファイルのみ。元repositoryコードは実行しない。

## 入力とgold

入力snapshotとモデルへのpublic diffは研究成果物に固定する。goldは評価器側にだけ置く。16ファイル移行は19 changed fileのうち事前固定の辞書順16 selected filesであり、元commit全体の評価ではない。4/8/16は同じ履歴からの相関するprojectionのため、独立した3作者・3履歴として数えない。tomlのupstream test同期はmultiple-valid goldとしてprimaryから分離する。

## 再実行

既存のlocal cached Qwen/Gemmaと固定helperが必要。新しいmodelのdownloadや依存追加は行わない。runtimeの変更でdigestが異なった場合は同一条件の再現と呼ばない。

```sh
GOCACHE=/private/tmp/benchmark156-gocache go build -o /private/tmp/benchmark156-validate ./tools/benchmark156/validate
GOCACHE=/private/tmp/benchmark156-gocache python3 tools/benchmark156/build_history_adapter.py
PYTHONPYCACHEPREFIX=/private/tmp/benchmark156-pycache python3 -m unittest discover -s tools/benchmark156 -p 'test_*.py' -v
```

測定scriptは`open('x')`を使い、既存結果を上書きしない。独立したoutput directoryを作り、保存済みpreregistered JSONをそこへ複製し、Python moduleのOUTだけをそのdirectoryへ設定して`run()`を呼ぶことで、固定入力・source digestのまま新しい観測を保存できる。例：

```sh
PYTHONPYCACHEPREFIX=/private/tmp/benchmark156-pycache python3 - <<'PY'
import pathlib, shutil, sys, tempfile
sys.path.insert(0, 'tools/benchmark156')
import evaluate_d
output = pathlib.Path(tempfile.mkdtemp(prefix='benchmark156-replay-'))
shutil.copy2(evaluate_d.OUT / 'iteration-4-preregistered.json', output)
evaluate_d.OUT = output
evaluate_d.run()
print(output)
PY
```

これはモデル推論を再実行するため、固定call/時間予算を消費する。必要な根拠なしに成功を得るまで繰り返さない。

## 安全性と測定の区別

通常worktreeや既存indexをstage/replayに使わない。source snapshot→temp-index→最終tree一致はGitの構造検証であり、中間commitで元projectのtestsが成功するという主張ではない。Aのwall timeには重い正逆順stage監査が含まれる。runtime planner latencyやpure LLM latencyへ読み替えない。process peak RSSは子process集合の累積maximumであり、各callの独立peakではない。

prototypeはrename、symlink、submodule、mode-only、競合をサポートしない。fixed unit/context budget超過や不正mappingは明示的に停止し、部分的なGit変更を行わない。意味unknownは合法なfile fallbackを返すが、fallbackをexact正解に数えない。

## A2

`coarse.py`は同じ#151行抽出器で所有権を確定し、`refine`要求時のみ元のinline上限内で細分化する。refinement予算超過時は検証済みcoarse所有権を保持する。`evaluate_a2.py`は使用済みDの構造診断であり新holdoutではない。immutableなiteration-5の事前登録と結果を保存する。意味unknown・file fallback契約は変更していない。
