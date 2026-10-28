# App Serverのフィールド限定取得可否の調査

## Goal

不要な種類の本文を受信前に除外し、編集diff・依頼・最終応答を保持できるか、ローカルApp Serverの公開リクエストと実動作で確認する。

## Result / Decision

2026-10-22のインストール済みCodex CLI 0.155.1では、必要な種類/フィールドだけを返す指定は確認できなかった。メタデータだけ・表示要約だけの取得は可能だが、照合用の編集diffは取得できない。commiter側のリクエスト変更だけで、採用済みv2スキーマ相当の応答を返させることはできない。

## Sources

- 実行バイナリから当日に再生成したexperimental込みJSON Schema。
- [公式App Serverドキュメント](https://learn.chatgpt.com/docs/app-server)：Turn一覧のitemsView、itemページング、通知opt-outの範囲。
- 完了済みの実Turn1件を使った読み取り専用リクエスト。モデル生成、resume、履歴変更なし。

## Supported requests

| リクエスト | 確認した指定 | 制約 |
| --- | --- | --- |
| thread/items/list | threadId, turnId, cursor, limit, sortDirection | item種別・フィールドの選別指定なし |
| thread/turns/list | threadId, cursor, limit, sortDirection, itemsView | notLoaded / summary / fullの3段階 |
| thread/read | threadId, includeTurns | 全Turnを含めるかだけで、編集だけを選べない |
| thread/timeline/list | threadId, cursor, limit | 種別・フィールド指定なし |
| thread/searchOccurrences | threadId, searchTerm, cursor, limit | 表示される依頼/最終応答の部分検索。編集diff取得の代替にならない |
| initialize capabilities | optOutNotificationMethods | 通知に適用。リクエストの応答/エラーには適用しない |

## Read-only probe

前回の保存履歴に含まれる完了済みTurnを、既存のopaque cursorを使って同じ1件に固定した。各条件1回の診断であり、一般的な性能比較ではない。IDや本文は集計に出さず、結果を私有保存した。

| itemsView | 応答bytes | 時間ms | item内訳の要点 |
| --- | ---: | ---: | --- |
| notLoaded | 536 | 36.82 | itemなし |
| summary | 2,039 | 42.36 | userMessage 1 / agentMessage 1のみ |
| full | 216,721 | 214.49 | fileChange 3、commandExecution 93等を含む |

summaryではfileChange 3件がすべて省かれた。fullにあるdiffの285 / 2,785 / 1,887bytesはいずれもsummaryに存在せず、採用必須フィールドを満たさない。userMessageもfullの2件に対しsummaryは1件で、表示要約を全ユーザー発話の保存としても扱わない。

次にthread/items/listの同じTurn・先頭20件について、通常要求と、未定義のitemTypes/fields/excludeTypesを付けた要求を各1回比較した。両方54,655bytesで応答内容も同一。後者にもcommandExecution 8件とreasoning 6件が残り、fileChangeのみという指定は効いていない。成功応答が返ることを機能対応と誤認しない。これらの未定義フィールドを実ハーネスへ追加していない。

## Implication / next gate

現APIで使える軽量取得は、一覧/表示用要約/通知抑制であり、必要な編集diffを残したまま不要本文だけを除外する投影ではない。現行ハーネスのnotLoadedによるTurn列挙は既に軽量取得を利用している。対象Turnの編集を網羅するには、現在はitem一覧を取得してクライアント側で選別する。

取得前の除外を実現するには、App Serverのサーバー側に種別/フィールド選別を追加する、または別の履歴読み取り経路を検討する必要がある。どちらも今回実装・採用していない。summaryから候補Turnを絞る場合も、タイトル/本文にない編集を取り逃がさない保証は別途必要。

## Artifacts

公開可能な集計は本goalとsummary。私有原本は<PRIVATE_TMP>/issue182-appserver-field-probe/のsummaryと生成Schema、probeスクリプトは同じtmp配下。実応答の本文・IDは公開用に保存しない。既存ハーネス・元の検証結果は変更していない。commit/push・Issue投稿は今回行っていない。
