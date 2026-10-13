# Issue #171 Phase 0 事前登録

## 範囲と仮説

既存Direct GroupingのGrouping第1段階に限定し、入力表現と意味判断プロンプトの単純効果・交互作用を比較する。後続Category/Text、Purpose Discovery/Assignment、新Planner、Production変更は行わない。Raw Diffの情報保持と正しいGroupingは別に検証する。

12件・4条件・通常/逆順・各1回、最大96call、retry0。未使用合成fixtureの小規模切り分けであり、16file約80%、実repository、別作者、正式採用への一般化をしない。既知#168〜#170のsource bytesとの重複を生成前に検査する。checkpoint学習データとの非重複は検証不能。

Aは実際のThreePhaseGeneratorの第1リクエストを実行境界で捕捉した既存入力/既存prompt。Bは入力だけRaw Diffへ置換、Cは意味判断promptだけ#170暫定Bへ適応、Dは両方。同一SchemaとID・task・安全指示・ファイルメタデータ・モデル・decoderを使用する。プロンプトの適応全文は `prompt-adaptation.json`、全実リクエストは `frozen-inputs/`。R6/R7の解答ルールをモデルpromptへ追加しない。入力予算の超過は切り詰めや再試行をせず、生成前の受理失敗として記録する。

通常順は既存Plannerの非テストパス/テストパス辞書順を全armに適用する。逆順はfiles配列だけ反転しID/Schemaを維持する。全件ソートの変更をProductionへ加えていないため、ソート導入効果のablationではない。

## Gold・採点

`commit-grouping-gold-rules.md` v1.0に基づき、確定11件・未確定1件(F12)・複数許容解0件。A/CのGo ASTは名前・位置のみで変更式を欠き、confirmed全11件で十分な契約根拠は確認できない。B/DのRaw DiffはF11以外の10件で必要根拠を含む。F11はホストの明示仕様でconfirmedだが両入力では根拠不足であり、主分母から除外しない。F05/F06/F07のR6/R7 Fallbackは4ファイル、目的分断は7件。Fallbackへの正しいsingletonはExact正解で、Gold不一致FSには含めない。fallback/全件singleton偏りを別記する。複数許容解は根拠を持つケースがなく実評価0件であり、評価器だけ契約テストで確認する。

各ファイル目的、目的間の独立/依存、具体的before/after、適用規則、唯一割当、Revert保証の限界、arm別識別可能性をfixtureとgold-auditに保存する。Goldと採点rubric、作者message、fixture分類はモデル入力へ含めない。各armには同じ選択ファイルを渡す。

confirmed 11×2=22cell/armをExactの主分母とし、通常順11件と逆順11件を併記する。逆順は独立標本とはみなさない。構造不正・timeout・未実行をExact0として残し、FM/FS算出不能はnull/unscorable。FM/FSペア件数、発生cell件数・率、scorableペア分母の率を区別する。未確定1件/12件(8.33%)は初回生成前から主評価除外し、構造・資源・予測分割は記録する。

## 固定品質Gate

研究候補Dに対するGOは、以下をすべて満たす場合に限る。満たさなければNO-GOで有限研究を終了する。GOでもProduction採用の判断ではない。

- 全96cellの実行記録があり、全armでSchema適合・完全割当・既存validator受理が100%。
- Dのconfirmed Exactが18/22以上（80%以上）かつAより4cell以上改善。
- DのFM・FSペア件数がともにA以下。
- Dのconfirmed順序一致が10/11以上（90%以上）。
- Dが全件merge/singletonに偏るだけで改善しないことを、mixed partitionのF01/F02/F03/F08が通常/逆順ともExactである条件で確認。
- 元Git状態・選択file所有権・byte/tree一致、context、RSS、pressure/swapの資源条件をすべて満たす。

A→B、A→C、B→D、C→DのExact/FM/FS/順序一致の差、およびD−B−C+Aを記録する。D対Aだけから一方の要因の効果を断定しない。小標本かつ決定的1回測定のため、統計的有意性・母集団の因果効果は主張しない。

## モデル・資源・停止

#170と同じMLX Gemma E4B 4bit revision `475b9088d29754a3379866cf5aeb6b41acd313c2`、既存measured helper・tokenize-only helperを再利用。seed144、temperature0、top_p1、top_k0、context16384、output768（native thought上限512を含む既存profile）、`bounded-grouping`、timeout120秒。新download・追加backend・多数決・第二LLMは行わない。

各callのinput/output tokens、wall、TTFT/load、helper RSS/MLX peak、before/after pressure・swapを保存。RSS上限8GiB、swap増加上限1024MiB/call、critical pressure>=4で即時停止。GOの資源評価にはpressureがnormalの1/2であることを要求する。観測不能・timeoutは成功に置換しない。p50/p95はモデルロードを含むプロセス時間、RSSはプロセスpeakであり、持続常駐時の費用と同一視しない。

Attempt journalをプロセス起動前にfsyncする。中断cellを再実行せず、hash漂移・Git漂移・hard resource stopは証跡を保存して停止する。96call以後の改善・Gold/threshold/prompt修正・追加ablationをしない。モデル生成前に全入力、Gold、評価器、閾値、モデルmanifest/weights、helperソース/バイナリをhash固定しcommit/pushする。
