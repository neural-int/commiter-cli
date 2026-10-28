# H182-010 検証用ファイル数上限撤廃

## Goal

5ファイル・3目的のfixtureを一括入力できるよう、研究用経路の件数上限を外す。

## Result / Decision

通常Plannerは4件、旧build.pyの研究overlayは16件、直近の履歴E2Eテストはパス列・収集結果の両方で4件だった。`build.py --unlimited-files`を追加し、Plannerの件数ガードと履歴E2Eの2つの件数ガードをoverlayだけで除去した。旧引数なしビルドは16件を維持する。通常ソースに上限撤廃は適用しない。

## Verification

5件と17件がmockクライアントへ到達するGoテスト成功。入力がcontext上限を超える場合の既存拒否テストも成功。Pythonの既存・新規計5テスト成功、CLI検証バイナリ生成成功、git diff --check成功。fixture変更・App Server履歴取得・モデル推論は実施していない。

追加試験の初回はPrepareが選択したcontext tierの指定不足、次は秘匿化されたbackendエラー本文の照合で失敗した。試験のcontext ceilingとクライアント呼び出し件数に基づく判定へ修正して成功した。原本失敗ログは保持。実装のガード除去内容は同じ。

## Limits

外したのはファイル件数ガード。context容量、IPC上限、validator、cycle timeoutは維持する。件数無制限でも任意サイズの入力を受理するわけではない。履歴取得と生成の時間制限は別条件であり、今回変更していない。commit/push/main反映は未実施。
