## 要約

fixture作成とPhase 0の妥当性監査は完了しました。Phase 1は初回呼出し後に事前登録のswap上限を超え、1/96callで停止しました。実行資源GateはNO-GO、4条件のGrouping品質と要因効果は比較不能です。Phase 1の完了条件とGoalは未達です。

## 検証結果

- 専用ブランチ `codex/issue-171-direct-grouping` で研究のみを実施。最初の凍結commit `f94ec345`、原文Goalの空白だけを修正した生成前訂正commit `e94e0f3f`。比較変数は変更していません。
- 未使用合成fixture12件、33変更ファイル。Gold規則v1.0に基づくconfirmed11件、undetermined1件(F12)、multiple_valid0件。Fallback4ファイル、目的分断7件。R1〜R10の根拠と唯一割当、arm別入力識別可能性を保存。#168〜#170とのsource bytes一致0件。
- 実装/テスト、producer/consumer、cross-directory契約、同packageの独立目的、依存と同目的の区別、共有テスト、1行内3目的、test-only、根拠不足を含みます。multiple_validの実例は根拠不足のため0件とし、許容解集合の採点処理だけ契約テストで確認。F11はhost仕様からGoldを確定し、入力不足でも分母に残します。F12は推論前から主評価除外（1/12、8.33%）。
- 既存ThreePhaseGeneratorの実Grouping第1requestを捕捉。A通常順は捕捉input/prompt/schemaと一致。A/Bはprompt共通、A/Cはinput共通、B/Dはinput共通、C/Dはprompt共通。Schema・ID・task・安全指示・共通metadata・model/decoder/budgetを固定。#170 Bは意味規則を保持してtask/outputだけ適応。
- 96入力のnative tokenize-only受理96/96、input907〜4611tokens、生成0tokens。通常順は既存Plannerのパスソート、逆順はfilesだけ反転し全armに適用。Raw Diff切り詰め0。Gold/rubric/fixture分類/作者messageは入力に含めていません。
- BaselineのGo入力はAST種類/名前/位置を含み、具体的変更式・リテラルやcall対象を欠きます。A/Cのconfirmed11件で十分な契約根拠を確認できず、B/DではF11以外の10件で必要根拠を記録。これは生成前の監査であり、Gold確定状態とは分離しました。
- 最終事前登録は153ファイルとモデル関連9ファイルのhashを固定。同じGemma E4B 4bit revision `475b9088d29754a3379866cf5aeb6b41acd313c2`、#170と同じhelper、seed144、temperature0、context16384、output768、native thought512、timeout120秒、96call上限、retry0、RSS8GiB、swap増分1024MiB上限。
- 実行したF01/A/通常順はcompleted、Schema valid、全3ファイルexactly-once、Exact1、FM0ペア、FS0ペア。clamp実装＋テストを統合し、prefixを分離しました。
- 同callはinput3365/output560tokens、wall35.873秒、load3.576秒、TTFT12.274秒、helper RSS644.844MiB、MLX peak4.796GiB。pressure2→1、システムswap6100.94→9823.25MiB、増分3722.31MiBで上限1024MiBを超過。固定条件どおり停止し、残り95cellは未実行。paired比較0件。
- システムswap前後差の観測であり、増加全量をhelperだけの原因とは断定していません。停止後の追加生成0、Gold/prompt/Gate調整0、retry0。
- 全12fixtureの一時index/object directoryでのbyte/tree一致、元HEAD/index/worktree不変更、元mainの既存diff保持、全凍結/モデルhashの事後一致を確認。局所契約テスト6件、捕捉器Go build、Gold/入力/token/整合性監査、対象diff checkはPASS。
- Phase 1未達のため、指定goal.mdの最終品質ゲート（全repository test/vet/build）は未実施。Issueチェックも承認までは未更新です。詳細は `docs/benchmarks/issue-171/report.md`、raw測定は `measurements.jsonl`、停止証跡は `termination.json`、比較不能項目は `comparison-availability.json` に保存しました。
- 事前投稿は自動承認レビューで拒否され、生成前pushも保留しました。事前登録はローカルcommitで固定しており、公開が生成前に済んだとは主張しません。事前準備のエラー・修正・公開手順差異も保存しました。

## 考察

今回のNO-GOは実行資源と完全実行のGate未達です。F01の1例はBaselineの正しい分割を示しますが、Raw Diffや暫定Bの効果、交互作用、順序依存、Fallback/偏りの優劣を比較する根拠になりません。未実行cellをExact0とする固定採点の計数や差分を、各armの観測品質や要因効果として解釈しないでください。

Gold作成・唯一割当、入力の要因比較契約、byte/tree一致は確認できましたが、意味品質の比較と16GBでの共存可能性は未到達です。合成小標本、複数許容解の実評価0件という限界もあります。16file約80%や実repository/別作者への一般化、正式採用は判断できません。Production default、4file上限、#169/#170 NO-GOを変更していません。

## Next Steps

- 停止した有限runをこの証跡で保存・公開する。結果を伏せず、資源停止と意味品質未判定を区別するため。達成済み5項目だけをチェックし、Phase 1は未チェックに残します。
- 人間が比較継続を選ぶ場合は、他プロセスとの共存・memory/swap観測条件を整え、新しい有限事前登録を作る。今回のrunを停止後に再開したり上限を事後緩和したりせず、F01が既知結果になったことを明示するため。
- Production実装は、4条件paired比較と品質・安全・資源Gateが実際に満たされるまで進めない。1例の成功と構造監査だけでは製品品質を保証できないため。
