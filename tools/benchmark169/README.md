# Issue #169 Raw Diff / Minimal Syntax Context の有限比較

`tools/benchmark169/` と `docs/benchmarks/issue-169/` だけを扱う研究用実装。製品入力、mode 排他 validator、Git commit mutation、default 4file 上限を変更しない。#168 の `prepare.go` と `projection.go`、Purpose Discovery の system/schema を再利用した。Tree-sitter は既存依存を利用する。

自己作成の一時 Git repository から snapshot、Before/After、raw diff、Evidence を取得し、元の index/worktree を照合する。fixture の内容はモデル入力用のデータであり、fixture の Go code/test は実行しない。機密ファイルの除外を含む `gitstate.Collect` を使用し、除外や選択集合の不一致があれば停止する。モデル処理は既存 local cache と helper だけで実行し、ネットワーク送信しない。公開 fixture/gold は自作ソースのみ。ローカル絶対パスの対応表は `/tmp` の runtime receipt に置き、研究成果物には含めない。

A は #168 と同じ9 Git管理 fields、4 Evidence位置 fieldsを除いた projection。B/C は同じ8共通fieldsと順序、Git `--unified=3` の完全な raw diff を使用する。C は `mode: raw_diff` に `syntax_context[]` を追加した研究用表現で、既存 product `contextinput.Build` に流さない。各 entry は hunk_index と enclosing_declaration のみ。Go function/method を対象とし、Before/After 不一致、構文エラー、top-level、同じ行に複数宣言がある場合は省略し、理由を別ログへ保存する。

再現手順（成果物を上書きせず、新規出力先と新規事前登録を用意する）:

```sh
go build -o /tmp/commiter-169-prepare ./tools/benchmark169
/tmp/commiter-169-prepare -cache <existing-model-cache> -out <new-input-dir> -runtime <local-runtime-json>
# runtimeに既存計測helper、preflight_helper、prepareのパスを加える
python3 tools/benchmark169/preflight.py --runtime <local-runtime-json>
python3 tools/benchmark169/audit.py
go test ./tools/benchmark169
python3 -m unittest discover -s tools/benchmark169 -p 'test_*.py'
# schema、fixture/gold、rubric、Gate、source/input/helperのSHA256を登録・公開した後だけ生成
python3 tools/benchmark169/run.py --cohort known --runtime <local-runtime-json>
# known出力を追加生成なしで入力/goldと照合しknown-evaluation.jsonへ記録
python3 tools/benchmark169/run.py --cohort holdout --runtime <local-runtime-json>
# 各説明のsemantic_gold/target_gold/grounded/hallucination/理由をholdout-evaluation.jsonへ記録
python3 tools/benchmark169/summarize.py
# Gate 1 GOの場合のみ:
python3 tools/benchmark169/run.py --phase grouping --cohort holdout --runtime <local-runtime-json>
```

preflight は既存 #163 helper の `preflight_only=true` により、同じ model/revision、Package.resolved、processor.prepare、enable_thinking=true を使って実 token 数を取得する。生成 branch より前に戻るため、purpose/grouping 推論は0回。sentinel `{}` は意味品質として採点しない。実生成は既存 #167 計測 helper をそのまま使用し、token 数の一致を確認する。decoder/model/prompt の調整や retry は禁止する。

`run.py` は process 開始前に試行を記録する。中断して receipt がない場合も同じ cell は再実行せず、未解決として残す。Phase 2 の起動は保存済み Gate 1 GO に限定する。schema 受理、意味品質、資源、source binding、実用性を別指標で扱う。

独立 holdout は過去に生成していない新規合成変更4件で、別作者や実repositoryからの無作為標本ではない。通常/逆順の8cellを独立8fixtureと数えない。S16 は大きい差分のtokenization-only予算確認であり、quality分母にも生成回数にも含めない。採用は本Issueの対象外。
