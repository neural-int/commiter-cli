# フィールド限定スキーマv2の確定と実履歴照合の再検証

## Goal

採用済みのフィールド選別案を研究用PoCのv2固定履歴形式として確定し、前回と同じ対象path・履歴期間・cwd集合・保存Git差分で再検証する。従来形式と新形式で判定、候補、曖昧理由、fallbackが変わらないか、取得・形式変換・読取検証・照合の時間とデータ量を分けて確認する。

## Result / Decision

11ケースすべてで従来形式と新形式の結果が一致。内訳は歴史Git差分9件ambiguous、1件input.schema/sensitive_contentによる拒否、現在.gitignoreの1件unmatched。候補参照や曖昧理由も含むComparison全体を照合した。新形式による一般的な精度改善を立証したものではない。

**今回実装したのは受信後のローカル投影。App Server側のフィールド限定は未対応。受信量や取得時間を減らしたA/Bではない。** 従来のPoCも不要な種類の本文を保存していなかったため、新形式の保持量はほぼ同じ。共有10path履歴JSONは従来56,969 bytes → 新形式56,827 bytes（0.25%減）。発話の出典ID・Turn情報を追加したため、ファイル別入力では大きくなるケースもある。

## Frozen fields

正式な照合用履歴契約は`tools/attribution175go/history-projection-v2.schema.json`。外側のGit固定入力はHEAD/index hash/snapshot hash/files/期間/rootを維持する。計測情報は別ファイルに分離する。

| 階層 | 採用フィールド |
| --- | --- |
| 履歴 | schema_version, projection, threads, next_cursor, coverage |
| thread | thread_id, turns |
| Turn | turn_id, started_at, status, user_messages, file_changes, final_messages |
| 依頼・最終応答 | item_id, item_order, text |
| 完了済み編集 | item_id, item_order, status, changes |
| 個別変更 | change_index, path, kind, move_path, diff |
| 完全性 | status, dropped, reasons, errors（category/reason/stage/code/cause） |

除外は途中応答・計画・推論本文・コマンド実行本文/出力/付加情報・ツール引数/結果・Web検索・画像関連・UI付加情報。対象ファイルの不一致候補や競合候補は落とさない。diffのnullと空文字は保持し、renameはpathとmove_pathを維持する。発話本文の結合/改行、最後のfinal_answerを使う従来仕様を維持する。

固定入力は取得停止/終了後のスナップショットなのでnext_cursorはnull。これで完全取得とは判断せずcoverageを参照する。ページ取得中の実cursorは従来のAPI取得処理で使う。未知フィールドはv2読取時に拒否。参照元Turn/発話情報の欠落を捏造・黙殺せずエラーとする。

## Conditions / method

- 前回の10path planのroot/10cwd/path/since/untilをそのまま使用。約76.6時間の履歴範囲。現在.gitignoreも前回保存入力と同じroot/期間、同じ10cwd集合を使用。
- Git差分は前回保存したR01–R10と現在.gitignoreの計11入力を使用。実worktreeの変更を作り直していない。
- 履歴の時間・累積受信量は無制限、安全フィルタ・対象path条件を維持。追加モデル呼び出しなし。
- 10pathを取得後、現在.gitignoreを順次取得。並行取得なし。各取得は1回で、再試行なし。
- 各取得から従来形式と新形式を両方生成。同一応答の比較であり、別々の再取得間の差をスキーマ効果と扱わない。
- 入力/バイナリ/source hashとquery条件を実行前に私有planに記録。従来形式は、新たに取得した発話ID・Turnメタデータだけを除いて旧保持形式を再現。
- 各ケースで形式読取/検証とcompare本体を別計測。実行順は従来→新、新→従来を交互にし、各1回。十分な反復を伴う速度benchmarkではない。
- 履歴保存状態は再取得時に変わり得る。同じquery条件であっても前回と同じbytesにはならない。現在のAPIはTurn開始時刻で期間判定するため、既に対象内の進行中Turnには後続itemが増え得る。

## Acquisition / data volume

| 取得範囲 | 前回の受信量 | 今回の受信量 | 前回の履歴取得 | 今回の履歴取得 | 今回の状態 |
| --- | ---: | ---: | ---: | ---: | --- |
| 10path共有 | 7.007MB | 7.105MB | 10.909秒 | 9.909秒 | partial・除外1・22操作/5Turn |
| 現在.gitignore | 42.210MB | 42.303MB | 22.573秒 | 18.522秒 | complete・対象操作0 |

MB/KBは10進。以前の二つの測定は一部並行、今回は順次実行であり、さらに計測コードも異なる。上記の時間差をスキーマによる高速化と結論しない。App Server要求に投影パラメータは追加していない。

| 保持JSON | 従来形式 | v2形式 |
| --- | ---: | ---: |
| 10path共有履歴 | 56,969 bytes | 56,827 bytes |
| 現在.gitignoreの空履歴＋管理情報 | 278 bytes | 149 bytes |

従来56,969bytesには取得メタデータを含む。現在の空履歴の差は管理情報の違いであり、編集情報の削減ではない。形式変換自体は10pathで約0.014ms、空履歴で約0.0003ms。

## What was received but not retained

受信時にitem種別の件数とJSON item全体のbytesだけを集計。生本文は統計へ出力しない。ここでいうbytesはコマンドの標準出力だけではなく、command等のフィールドを含むitem全体。ページ外枠/thread一覧等は含まないので総受信量とは一致しない。

| 項目種別 | 10path共有 | 現在.gitignore |
| --- | ---: | ---: |
| コマンド実行情報 | 4.827MB | 32.584MB |
| MCPツール情報 | 0.697MB | 3.552MB |
| 途中応答 | 0.177MB | 0.779MB |
| 推論情報 | 0.180MB | 0.984MB |
| Web検索 | 0.176MB | 0.381MB |
| その他の非採用item | 0.020MB | 0.050MB |
| 上記の非採用item合計 | **6.075MB（総受信の85.5%）** | **38.331MB（90.6%）** |

コマンド実行item単独で総受信量の約67.9% / 77.0%。今回の取得では、現行照合で使わない種類の大きな本文を受信していることを確認した。画像がuserMessage/tool結果内に埋め込まれた場合のbytesは別分類していないため、画像本体の有無・比率をこの集計から断定しない。

これらをAPI応答作成前に省ければ転送量を削減できる可能性はあるが、API/保存履歴の走査・JSON化・RPC往復等の時間内訳は未測定。上記比率をそのまま時間短縮率へ換算しない。

## Data matching / time

単位ms。読取/検証には従来JSONのdecode/安全性検証、v2 JSONの厳密decode/従来照合入力への復元/安全性検証を含む。compare本体は同じGo照合実装。今回は同一プロセス内の時間で、前回の約20ms級のsubprocess起動込み時間と直接比較しない。

| ケース | 判定 | 従来と新形式 | 読取・検証 従来 → 新 | 照合本体 従来 → 新 |
| --- | --- | --- | ---: | ---: |
| R01 | ambiguous | 一致 | 0.702 → 0.671 | 0.047 → 0.012 |
| R02 | ambiguous | 一致 | 2.353 → 2.415 | 1.230 → 1.716 |
| R03 | ambiguous | 一致 | 0.715 → 0.711 | 0.021 → 0.016 |
| R04 | ambiguous | 一致 | 1.247 → 1.325 | 0.043 → 0.044 |
| R05 | ambiguous | 一致 | 1.190 → 1.192 | 0.067 → 0.064 |
| R06 | ambiguous | 一致 | 0.801 → 0.785 | 0.033 → 0.037 |
| R07 | ambiguous | 一致 | 1.399 → 1.395 | 0.091 → 0.085 |
| R08 | ambiguous | 一致 | 0.466 → 0.477 | 0.010 → 0.013 |
| R09 | ambiguous | 一致 | 3.636 → 3.450 | 0.368 → 0.353 |
| R10 | 入力拒否 | 一致 | 0.455 → 0.487 | — |
| current | unmatched | 一致 | 0.066 → 0.054 | 0.005 → 0.004 |

照合できた10ケース（R10の入力拒否を除く）の中央値は、compare本体が従来0.0448ms → 新0.0406ms、読取/検証が0.9960ms → 0.9881ms。compare合計は1.915ms → 2.345msと逆方向であり、単発測定の揺れもある。速度優劣は未確定で、大きな改善を示していない。取得は秒単位、照合本体は今回の小標本でミリ秒未満中心という段階差は観測できた。

R01/R02/R05/R07/R08/R09はpartial_matchとhistory_incomplete、R03/R04/R06は完全一致候補があってもhistory_incompleteを維持。R10は両形式で同じsensitive_content拒否。現在.gitignoreは両形式でno_content_evidence。独立した因果Goldがないため、11/11一致を正確率100%とは呼ばない。

## Validation / artifacts

関連3パッケージのgo test、PoC go vet、研究用CLI buildが成功。回帰テストは形式往復での出典ID・操作・比較結果保持、rename/null diff/partial、除外フィールド拒否、集計に本文を混ぜないこと、参照元のない操作の拒否を確認。測定後に読取境界を補強した最終CLIでも、保存済み11入力の結果一致を追加確認した。履歴再取得はしていない。

正式JSON Schemaファイルは契約文書。実行時はGoの厳密decodeと意味検証を使用し、外部JSON Schema validatorの実行確認はしていない。追加依存なし。本番Collector/Plannerへ接続していない。

私有原本: `<PRIVATE_TMP>/issue182-attribution-projection-03/`。plan、2取得のsource履歴/v2履歴、item別集計、11組の入力/結果、timeログ、最終CLI確認を保存。旧原本を上書きしない。公開用は本goalとsummary.jsonのみで、生diff・発話・内部参照ID・ホームpath・機密候補値を含めない。コミット/pushやIssueへの投稿は今回の作業では行っていない。

## Next gate

フィールド契約と結果の互換性は確認済み。commiter内の形式変更による大幅高速化・受信量削減は確認していない。次に取得量を改善するなら、App Server側で不要なitem種別/フィールドを返さない取得経路の可否を調べ、同じ候補範囲で別実験にする。対象pathや履歴期間の絞り込みを同時に変更しない。
