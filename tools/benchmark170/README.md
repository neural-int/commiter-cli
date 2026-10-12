# Issue #170 System Prompt比較

既存 `tools/benchmark169` の collector、native tokenize-only preflight、once-only runner、schema validatorを再利用する。`prepare.py` は collector のB-arm Raw Diffを三条件へ同一bytesで複製し、System Promptだけを変更する。既存helperを変更せず、ローカルcacheだけを使う。fixtureソースはデータであり実行しない。機密候補を含むrepositoryは収集しない。

新規出力先で `prepare.py --runtime <local-runtime-json>`、`preflight.py --runtime ...` を実行し、事前登録とhashを公開した後に `run.py --cohort known --runtime ...`、既知出力のoffline評価、`run.py --cohort holdout --runtime ...` を実行する。評価はgoldとRaw Diffの照合であり第二LLM呼出しはしない。`evaluate.py` がGateを計算し、GOのときだけ同じrunnerの `--phase grouping` を許す。再試行・結果に追従したprompt/gold変更はしない。

研究成果物はこのtools/docsのみ。#169の基盤をbranch ancestryに含めるが変更しない。製品移行、mainへのmergeはこの研究の対象外。
