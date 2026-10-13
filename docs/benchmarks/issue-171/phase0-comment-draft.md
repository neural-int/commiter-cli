## 要約
Phase 0 を完了しました。Gold作成規則v1.0に基づく未使用合成fixture12件を作成・監査し、実Direct Grouping第1段階のBaselineを捕捉して4条件×通常/逆順の96入力を初回推論前に固定しました。モデル生成は現時点で0回です。次に固定した有限比較を実行します。

## 検証結果
- 専用worktree `codex/issue-171-direct-grouping`、取得元HEAD `97a98633223b0daee57ab9b31cdfb8b102bbaa37`。mainの既存 `.gitignore` 変更は保持。
- fixture12件、変更ファイル33件。confirmed11件、undetermined1件(F12)、multiple_valid0件。確定GoldにR6/R7 Fallback4ファイル、Fallback由来の目的分断7件。割当漏れ・重複0件。
- 実装/対応テスト、producer/consumer、cross-directory契約、same-package独立目的、依存があっても独立した機能目標、共有テスト、1行内3目的、test-only、情報不足を含めました。#168〜#170のbefore/after source bytesとの一致0件。
- F11はホストの明示仕様からGoldを確定し、両入力の根拠不足として主分母に残します。F12は仕様・依存根拠を確定できず、推論前から主評価除外（1/12、8.33%）。複数許容解の実例を裏付ける根拠は今回なく0件と記録し、採点器の許容解集合処理のみ契約テストで検証しました。
- 既存 `ThreePhaseGenerator.Generate` の実第1リクエストを捕捉してその境界で停止。A通常順は実捕捉入力/prompt/schemaと一致。A/B・C/Dでprompt共通、A/C・B/Dでinput共通、Schema・ID・task・メタデータ・decoder・context/outputは4条件共通。#170 Bの意味判断規則は保持し、task/outputだけ適応。GoldのR6/R7規則をpromptへ追加していません。
- 96入力のnative tokenize-only受理96/96、input tokens907〜4611、生成tokens0。Raw Diff切り詰め0。通常順は既存Plannerのパスソート、逆順はfile配列だけ反転し全armに適用。
- 実BaselineのGo入力はAST種類/名前/座標中心で、変更式・リテラル・call対象を保持しません。A/Cではconfirmed11件について具体的契約/独立性の十分な根拠を確認できず、B/DではF11以外の10件で必要なRaw Diff根拠を記録しました。入力情報充足性はGold確定状態と分離しました。
- 隔離した一時index/object directoryによる全12fixtureの選択ファイルbyte・全tree一致、元fixture HEAD/index/worktree不変更を確認。6契約テストPASS（Exact正規化、FM/FS、Fallback、multiple_valid/undetermined、重複/欠落/未知ID拒否、pin改変拒否）。
- 事前登録UTC `2026-10-13T02:48:05.308096+00:00`、151ファイルとモデル関連9ファイルをSHA256固定。同じGemma E4B 4bit revision `475b9088d29754a3379866cf5aeb6b41acd313c2` と#170のhelperを使用。seed144、temperature0、context16384、output768、native thought512、timeout120秒、96call上限、retry0、RSS8GiB/増分swap1024MiB上限。
- 固定Gate：DのExact18/22以上かつAより4cell以上改善、FM/FSともA以下、順序一致10/11以上、F01/F02/F03/F08の通常/逆順Exact、全条件の完全割当/Schema100%、Git・byte/tree・資源条件。1fixtureの2順序を独立標本として扱いません。
- 事前エラーも保存：Python同名module/import順、捕捉引数不足、F07の既存byte-bound拒否、観測用Git statusのindex cache更新。F07は推論前にテストを整形し、1行内3目的を保持して受理確認。index driftはroot別監査でF03に限定され、no-optional-locks修正後、同じsourceで再作成した捕捉requestの一致と監査PASSを確認。初期診断でF01を再作成した経緯も残しています。

成果物：`tools/benchmark171/fixtures.json`、`docs/benchmarks/issue-171/{gold-audit,input-audit,integrity-audit,preregistration}.json`、`preregistration.md`、`frozen-inputs/`、`pre-freeze-errors.json`。

## 考察
Goldへの唯一割当と採点の構造条件、および要因比較の入力契約を確認しました。これらはGroupingの意味品質を示す結果ではありません。Baselineでは具体的な変更前後を欠くため、誤分割の原因を入力不足と判断誤りに分けて分析する必要があります。Raw Diffに根拠があるケースでも正しいGroupingが得られるとは限りません。

R6/R7によるsingletonはGold上の正解ですが目的分断を含むため、通常の同目的統合成功と別記します。今回は合成小標本であり、実repository/別作者・16file約80%・Production正式採用への一般化はできません。新Planner、新stage、新依存、Production default、4file上限、#169/#170 NO-GOは変更していません。

## Next Steps
- 凍結した作業履歴をcommit/pushし、remote headを照合する。初回生成前のGold・Gate・入力固定を第三者が確認できる状態にするため。
- A/B/C/Dの96cellを各1回実行し、A→B、A→C、B→D、C→DとD−B−C+Aを評価する。入力要因とprompt要因を分離するため。
- Exact/FM/FS、順序一致、完全割当、Fallback/偏り、tokens/wall/RSS/pressure/swapを固定Gateで判定し、GO/NO-GOで有限研究を終了する。結果に追従した再試行・Gold/prompt/threshold変更はしない。
