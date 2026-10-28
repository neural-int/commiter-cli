# 現在worktree限定の取得方式速度比較

## Goal

paged20 / paged100 / full5を現在worktreeに記録された会話だけに限定して比較し、取得時間・要求数・受信量と選択証拠/照合結果の互換性を確認する。

## Result / Decision

主checkoutの履歴取得はpaged20の17.355秒に対しpaged100は9.606秒、今回の単発測定では44.7%短縮。要求数は1,144→605。full5は268要求へ減ったが27.523秒・143.983MBとなり悪化。現時点ではpaged100を既定候補として維持し、full5の性能採用はしない。3モードの実装自体は比較用に保持する。統計的に最適な値を確定したものではない。

## Frozen conditions

前回projection比較の保存query期間/path/歴史Git差分11件を使用。ただし取得cwdは各queryのrootだけ。関連worktree集合を使った以前の時間とは直接A/Bしない。10pathのqueryでは20→100→full5、主checkoutではfull5→100→20の順で各1回、全6セルを逐次実行。計測中の並行検証なし。各セルでApp Serverを起動し、初期化から取得/選別終了までを測定。時間・受信量の上限なし、安全フィルタと取得期間条件は維持。測定用の追加item種別集計は無効に揃えた。

query条件、保存Git入力hash、source/test binary hashを実行前に私有planに固定。原本は新規保存し、再試行やモデル生成なし。履歴はliveなので同じqueryでも内容が増える可能性はあるが、今回の選択証拠と結果の一致を検査した。

## Main checkout results

| モード | 取得時間 | 要求数 | 受信量（10進MB） | thread / 対象Turn / item |
| --- | ---: | ---: | ---: | --- |
| paged20 | 17.355秒 | 1,144 | 42.305 | 87 / 443 / 15,097 |
| paged100 | 9.606秒 | 605 | 42.098 | 87 / 443 / 15,097 |
| full5 | 27.523秒 | 268 | 143.983 | 87 / 443 / 15,097 |

主checkoutの対象は前回と同じ.gitignore。全方式で対象編集操作0件、history complete、unmatched/no_content_evidence。同じthread/Turn/item数は、選択期間内で処理した件数であり、full5が受信した期間外item数を含まない。

paged100はpaged20比で約1.81倍の取得速度。要求回数は約47.1%減で、受信量はほぼ同じ。これは要求回数削減の有効性を示す観測だが、往復待ち/サーバーの再読込/JSON解析それぞれの寄与は未分解。各1回でcache/実行順/システム負荷の影響があり、安定した改善率の保証にはしない。

full5は約3.4倍の受信量で、paged20より約58.6%長い。コードではTurn本文の受信後に期間を判定し、全Turnページを走査するため期間外本文も受信する。今回の受信増加要因と考えられるが、期間外bytesの厳密な内訳は集計していない。要求数を減らすだけでは速くなるとは限らない。

## Ten historical paths: scope effect

対象worktreeのcwdで記録されたthreadは0件だった。全方式で3要求・556bytes・本文0件。paged20 375.836ms、paged100 25.191ms、full5 23.051msだが、起動/初期化/空一覧だけの計測であり、ページサイズの速度効果として解釈しない。

以前の22操作は別cwdで記録された会話から取得していたことになり、承認済みの現在worktree限定方針では対象外。別cwdの会話を再び含める条件変更はしていない。R01–R09は全方式でcomplete/unmatched、R10は従来どおりinput.schema/sensitive_content拒否。

## Consistency / validation

11保存入力×3方式の結果と、選択された履歴操作/依頼/最終応答/coverageは方式間で同じ。ただし今回どちらのqueryも対象編集操作が0件であり、実編集の照合成功や非空候補の取り逃がし率を立証したものではない。期間内の非対象itemは大量に取得しているが、破棄した全item本文の同一性を比較したものでもない。非空の操作と照合結果の互換性は前段の6Turn×104itemの偽サーバー回帰テストで確認済み。

実測後に、既存の別Turnへlimit=20/100を指定する小さな読み取り確認を実施。実返却数20/100・両方nextCursorありを確認し、100指定が20件へ固定されていないことを確認した。これは測定6セルの外で、追加モデル生成なし。

測定前にPoC go test成功、実行ファイルをbuild。測定後go vetとdiff --check成功。既存の本番Collector/PlannerやGit変更は操作しない。追加依存なし。

## Artifacts / Next gate

私有原本: <PRIVATE_TMP>/issue182-fetch-speed-06/。plan、6セルのsource履歴/照合結果/metrics、全体timeログ、limit確認を保存。公開用は本goalとsummaryのみで、内部ID・発話・diff・ホームpathを含めない。過去原本を上書きしない。commit/push・Issue投稿は今回行っていない。

現段階ではpaged100を第一候補とし、full5は性能上不採用。初回ページ増量の観測結果と、対象worktree制限で実編集候補がなくなった問題は分けて扱う。安定した改善率を採用Gateにするなら同じ範囲の反復測定が必要。非空実編集の精度検証には、現在worktreeのcwdで記録された履歴を持つ別の固定標本が必要で、今回は自動追加していない。
