# 現在worktree限定と取得方式の実装

## Goal

承認された現在worktreeのみの取得範囲、通常ページ20→100件、本文込み5Turn方式を研究用ハーネスへ実装し、比較用の20件基準も残す。

## Result / Decision

3モードを実装。capture既定はpaged100。paged20は現在worktree限定の基準、full5はthread一覧100件・本文込みTurn一覧5件で、個別item取得をしない。サーバー側のフィールド投影ではなく、既存APIでページ数・取得単位を変える方式。

対象cwdは現在worktreeだけとし、別worktreeの列挙を削除。別cwdで記録された会話は対象worktreeへの絶対パス編集を含む場合も対象外。pathは引き続き対象worktree内に限定する。過去の複数worktreeを含む測定とは別条件。

## Validation

6Turn・各104itemの偽App Serverでページング境界を含む回帰検証を実施。3方式で同じ操作、依頼/最終応答、内容照合結果を取得し、要求回数がpaged20 > paged100 > full5になることを確認した。パラメータの要求値も偽サーバー側で検証。別cwdのthreadを混ぜても本文にアクセスしないこと、full方式で個別item要求が発生しないこと、full本文フィールド欠落がinvalid_schemaとなることも確認。

最初の追加テストはfull5分岐漏れを検出した。本文込み取得後も個別item取得に進んでいた箇所を修正し、再検証は成功した。

関連3パッケージのgo test、PoCのgo vet、研究用CLI buildが成功。既存の原本・固定Git差分・本番Collector/Plannerは変更しない。依存追加なし。

## Limits / Next gate

実履歴の速度比較は未実行。100件要求が実サーバーで何件返るか、full5の応答サイズ・メモリ・取得時間は次の測定で確認する。既存の安全フィルタ、期間条件、snapshot確認、通常時の5秒/4MiB予算は維持する。本文込み方式では対象期間の境界ページに不要Turnの本文が含まれ得る。

3方式の実測時は現在worktreeに範囲を揃え、過去の広い取得範囲の時間差をページサイズの効果と混同しない。今回はローカル実装・検証まで。commit/push・Issue投稿は行っていない。
