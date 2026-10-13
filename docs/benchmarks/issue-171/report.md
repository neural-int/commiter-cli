# Issue #171 検証報告

fixture作成とPhase 0の妥当性監査は完了した。Phase 1は初回モデル呼出し後に事前登録のswap上限を超えたため停止した。実行資源GateはNO-GO、4条件の意味品質・要因効果は比較不能である。Issueの全完了条件とGoalは未達。

## 固定した評価データ

未使用の合成fixture12件、変更ファイル33件。Gold作成規則v1.0に従ってconfirmed11件、undetermined1件、multiple_valid0件を初回生成前に記録した。複数許容解の実例に十分な根拠がなく、恣意的に作らず0件とした。許容解集合に対応する採点器は契約テストで検証した。

| Fixture | 根拠・目的 | Gold |
|---|---|---|
| F01 | 上限値修正＋対応テスト、独立した表示接頭辞 | 2目的の分離 |
| F02 | producer/consumerの共通単位変更、独立retry | 2目的の分離 |
| F03 | 同packageのwidth/badge変更とそれぞれの対応テスト | 2目的の分離 |
| F04 | encode/decodeの同一契約の依存する2変更 | R5で統合 |
| F05 | 2独立目的を含む実装、別々の対応テスト | 実装はR6/R7 singleton |
| F06 | 独立したpool/expiryと共有テスト | 共有テストはR6/R7 singleton |
| F07 | 同じ1行の依存するunit計算＋独立title/retry、共有policyテスト | 3目的、2ファイルはR6/R7 singleton |
| F08 | genericなpadding機能と、それに依存するjob表示機能 | R3の依存だけで統合しない |
| F09 | config/producer/consumer/docsの共通separator契約 | 4ファイル統合 |
| F10 | 不変helperを使うzero/negativeの独立テスト拡張 | 独立目的の分離 |
| F11 | ホスト仕様にある共通batch契約、モデル入力には仕様なし | confirmedのまま入力不足を別記 |
| F12 | opaque option値変更、仕様・関連callerなし | undetermined、事前除外 |

R6/R7 Fallbackは4ファイル、Fallback由来の目的分断は7件。これらはGoldと一致すればExact正解で、Gold不一致のFSとは別に扱う。全割当漏れ・重複0件。#168〜#170のsource bytesとの一致0件。学習データとの重複や実repository/別作者での一般化は未検証。

## Phase 0 の監査

既存ThreePhaseGeneratorの実Grouping第1リクエストを捕捉し、その境界で停止した。A通常順は捕捉requestと一致。BはRaw Diffへの入力表現変更、Cは#170暫定Bの意味判断規則、Dは両方。共通Schema、ID、task、安全指示、メタデータ、モデル、decoder、context/outputを固定し、意味規則以外のprompt適応内容を全文保存した。

4条件×12件×通常/逆順の96入力を用意し、native tokenize-onlyで96/96受理、907〜4611 input tokens、生成tokens0を確認した。通常順は既存Plannerの非テスト/テストパス辞書順、逆順はfile配列だけを反転し全armに同じように適用した。Raw Diff切り詰めはない。

実BaselineのGo入力はAST種類・名前・位置を含むが、変更式・リテラルや呼出し先を保持しない。A/Cは具体的な変更契約と独立性を十分に裏付けない。B/DはF11以外の10 confirmedで根拠を含む。これは事前の入力監査であり、モデルの成否から導いた判定ではない。confirmedを入力不足だけで主分母から除外していない。

一時index/object directoryだけで全12fixtureの選択byte・全tree一致を確認した。元fixture HEAD/index/worktree、main HEAD/index/既存diffは事後監査でも維持された。初期観測処理のindex cache更新による監査失敗は保存し、no-optional-locks修正と同source再作成後の実request一致を確認した。F07のbyte-bound拒否と、Python module読み込み、捕捉引数、原文Goalの空白チェック失敗も事前エラーとして保存した。

最初の凍結commitは `f94ec345`。原文GoalをJSONで完全保持しMarkdown空白のみを整形した、生成前の訂正commitは `e94e0f3f`。訂正前事前登録を `preregistration-v1.json` に残し、比較変数は変更していない。最終事前登録は153ファイルhashとモデル関連9ファイルhashを固定した。GitHubへの事前投稿は自動承認レビューで拒否され、pushも保留した。公開が生成前に済んだとは主張しない。

## Phase 1 の観測結果

| 項目 | 観測 |
|---|---|
| 実行cell | F01・A・通常順だけ、1/96 |
| 残りcell | 95件未実行、paired比較0件 |
| backend / Schema / 完全割当 | completed / valid / 全3ファイル exactly-once |
| 予測分割 | clamp実装＋対応テスト、prefixは独立 |
| Exact / FM / FS | 1 / 0ペア / 0ペア（この1cellだけ） |
| input / output tokens | 3365 / 560 |
| wall / load / TTFT | 35.873秒 / 3.576秒 / 12.274秒 |
| helper peak RSS | 644.844 MiB |
| MLX peak allocation | 4.796 GiB |
| pressure | 2 → 1 |
| system swap | 6100.94 → 9823.25 MiB |
| swap差分 | +3722.31 MiB、固定上限1024 MiBを超過 |
| Git・凍結hash・model artifacts | 事後一致 |

swap観測はシステム全体の前後差であり、増加全量がhelperだけに起因するとは立証していない。固定した停止条件が実際の観測値に達したため停止した。RSSとMLX/unified allocationは異なる指標なので同一のメモリ使用量と解釈しない。

採点器のNO-GOには未実行cellをExact0に残す固定契約が含まれる。Aの1/22や他armの0/22は主分母上の実行未達の計数であって、各方式の意味品質の観測率ではない。`evaluation.json` の未実行を含む差分は要因効果として使わず、`comparison-availability.json` でA→B、A→C、B→D、C→D、交互作用をすべて比較不能として明示した。

## 判断と未到達

資源と完全実行のGateはNO-GO。1例の成功ではRaw Diffや暫定Bの有効性・順序安定性を判断できない。4条件paired比較、FM/FS差、Fallback/偏りの比較、wall p50/p95の比較は未到達。Production採用、default、4file上限、#169/#170 NO-GOは変更していない。モデルretry0、stop以後の追加生成0、結果を理由にしたGold/prompt/Gate変更0。

局所契約テスト6件、研究用Go捕捉器build、Gold監査、96入力の静的/token受理監査、事後hash・Git・byte/tree監査、対象diff checkはPASS。指定goal.mdはIssue Verification全達成後に最終品質ゲートを要求するため、Phase 1未達の現時点では全repositoryのtest/vet/buildを実行していない。これらをPASSと扱っていない。

## Issue 完了条件の状態

- [x] 既存Direct Groupingの実装・入力・出力契約と有限Gateを生成前に記録
- [x] Baseline/Treatmentの入力・全文prompt/schema、同一モデル・Gold・入力順序を固定
- [x] 誤統合・誤分割・識別不能のケース別根拠監査
- [ ] Phase 1で4条件のpaired比較と品質・順序・完全割当・資源評価を完了
- [x] 意味品質・安全・費用を分けたNO-GO、未到達、次の検証条件を明記
- [x] Productionと元Git状態を維持し、有限研究runを停止

これはローカルの達成監査であり、remote Issue本文のチェックは未更新。paired比較未達なのでGoalは未完了。

## Next Steps

- この停止結果と固定成果物をGitHubへ公開する。具体的draft・3コミットをユーザーに提示して外部公開の承認を得るため。Issue本文は達成済み5項目だけをチェックし、Phase 1は未チェックに残す。
- ユーザーが比較継続を選ぶ場合は、他プロセスとの共存とmemory/swap観測条件を整えて、新しい有限事前登録として扱う。今回の停止後に同じrunを再開したり、上限を事後緩和したりしない。初回F01は既知結果になったことも明示する。
- 次の実装判断は、4条件のpaired比較と固定品質・資源Gateを実際に満たしてから行う。初回1cell成功や構造監査だけで製品変更へ進まない。
