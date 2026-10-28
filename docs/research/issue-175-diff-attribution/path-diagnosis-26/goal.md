# 実履歴0件のパス判定調査（2026-10-25）

## Goal / Decision

除外280件はすべて現在ワークツリー外。安全性・パス解決失敗ではなかった。含まれていた.gitignore編集3件の実パスは、issue-175-codex-context作業ツリーのtools/benchmark175/.gitignoreが1件、issue181-clean-review-fixes作業ツリーのdocs/benchmarks/.gitignoreが2件。現在チェックアウト直下の.gitignoreとは別のファイルであり、現在のワークツリー限定方針では正しい除外。今回の0件をパス判定の不具合とは認定しない。

## Evidence

実App Serverを読み取る計測専用overlayで段階別に確認。完了編集283件、ワークツリー外280件、現在ワークツリー内3件、対象パス0件。取得したfileChange item195件。3件の.gitignoreはいずれも会話のcwdが現在チェックアウトを指していたが、編集パスは別ワークツリーの絶対パスだった。thread.cwdだけでは編集場所を決められず、編集パスによる追加判定が必要な実例となった。

## Limits / Next

対象期間・取得した会話内のfileChangeについて確認した結果。現在の直下.gitignoreがどの操作で作られたか、期間外の履歴やcommandExecutionに存在するかまでは未特定。シェル経由編集を現行fileChange照合で扱えると主張しない。実履歴付きE2Eには現在ワークツリーの現在差分と対応するfileChangeの組を選ぶ必要がある。別ワークツリーや同名ファイルを混ぜて成功扱いにしない。

## Delivery

通常ソースの変更・推論なし。診断原本はprivate tmpに保持し、報告には会話本文・出典ID・個人絶対パスを含めない。commit/push/main反映なし。
