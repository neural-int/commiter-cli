# Issue 162: evidence-based review relationship diagnostic

初回は作者commit境界の再現や最終グループ構築ではなく、自己作成4file入力内の対象2fileの関係診断。契約・入力・gold・helper/source/model/profileは事前固定。詳しくはdocs/benchmarks/issue-162/evaluation-contract.mdとiteration-1-report.mdを参照。

- `python3 -B -m unittest discover -s tools/benchmark162 -v`
- `python3 -B tools/benchmark162/benchmark.py preflight`
- 推論前に事前登録をcommit/pushし、`python3 -B tools/benchmark162/benchmark.py run`

出力は排他的作成のため、同じ成果物の上書き再計測は拒否する。再現時は新しい隔離checkoutで結果ファイルを保存退避し、同じpreregistered source/inputを使用する。元計測結果を置換しない。helper binaryとmodel cacheは実験環境の絶対path。helper sourceの固定公開baseと実際の変更byte archiveはenvironment.jsonのリンクに保存済み。新依存・helper build・model downloadは不要な計測だったが、別環境にcacheが無い場合は利用可能性を別途確認する。

`iteration-1-draft-preflight.json`は未実行の512token初稿。実際に使用したのは1536tokenの`iteration-1-preregistered.json`。helper neutral profileのfixed budgetから決定し、推論前に修正した。温度等はhelper sourceの固定設定。

source IDの有効性は説明の正しさを保証しない。API依存をtogetherと返したことは分類不一致であり、mergeしたcommitが不適切という意味ではない。unknownを返さないことも直ちにすべてのpartitionが不適切という意味ではなく、本診断の根拠不足検出の未成立を表す。小標本でのGemma優位を一般化しない。
