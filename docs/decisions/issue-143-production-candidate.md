# Issue #143: 暫定 production 候補の選定

2026-10-04、利用者の判断により、Gemma と3段 planner を暫定 production 候補として選定し、組み込み既定値と新規設定テンプレートへ反映する。検証 Issue の終了は、この選定の完了を意味する。旧時間ゲートの達成や未知の入力に対する production 品質の保証を意味しない。

## 選定した構成

| 項目 | 候補 |
| --- | --- |
| planner / backend | `three-phase` / `mlx` |
| model | `mlx-community/gemma-4-E4B-it-4bit` |
| revision | `475b9088d29754a3379866cf5aeb6b41acd313c2` |
| quantization | MLX 4bit |
| context | 固定16,384 tokens |
| sampling | temperature 0、top-p 1、top-k 0、seed 144 |
| 1段目 | 各ファイルの group membership、native thought 上限512、JSON grammarあり、生成上限768 |
| 2段目 | 各 group の type と breaking evidence reference、native thought 上限384、JSON grammarあり、生成上限512 |
| 3段目 | 各 group の scope と summary、native thoughtなし、JSON grammarなし、生成上限768、事後の厳密検証 |
| サイクル予算 | 3呼び出し以内、生成枠の合計2,048 tokens、共有120秒、retry / repairなし |

Go は対象ファイルを source-first とパス順に提示し、モデル用 ID を `path:<new path または old path>` にする。membership 出力の全件・一意割り当てを確認して元の file ID に戻す。同じラベルは同じ group を表し、ラベル名のみ正規化する。決定済み group の統合・分割は行わない。

Swift helper は固定したモデルと段階別 generation profile を確認し、実際に token 化した入力と出力枠が context に収まることを確認する。native thought の内容は返却せず、final channel の JSON だけを返す。旧 helper が profile を無視した場合も Go 側で成功扱いにしない。

各段階で明示的な `completed`、JSON の重複キー・未知フィールド・group/file ID、機密値を検証する。最後に従来の `planning.Validate()` を適用する。途中停止、予算超過、未解決 evidence、構造違反の場合は部分 plan を返さず、Git 変更へ進まない。

## 根拠と限界

| 検証 | 観測結果 | 解釈の範囲 |
| --- | --- | --- |
| A285 | path-ID で既知の公開入力2件とも full gate を通過。70.397 / 76.330秒 | 使用した2入力での成立。旧時間ゲートを満たす根拠ではない |
| A287 | fresh synthetic 2入力 × numeric/path ID の4サイクル、12呼び出し。completed・grouping・Go validity は4/4、false merge / false split は0 | full gate は2/4。checksum の type が期待 `fix` に対し `feat` |
| A288 | checksum の要求・正解 type の根拠が入力に存在しないことを確認 | A287 の type 不一致だけで誤分類とは断定できない。正解も確定できない |

検証履歴は [Issue #143](https://github.com/neural-int/commiter-cli/issues/143) を参照する。path-ID の一般的な優位性、未知入力の精度、大規模入力、summary の全面的な意味評価は未確認である。速度以外の課題がすべて解消したとは扱わない。

通常 CLI の入力 adapter と helper 接続は今回追加した実装であり、過去の benchmark 成績がそのまま再現されるとは扱わない。追加のローカル試験は接続・停止条件の検証であり、production の精度推定ではない。

## 暫定運用の境界

- 対象は選択された最大4ファイル、圧縮していない入力、固定16K context に限定する。段階ごとの実際の prompt を独立に計測し、超過時は停止する。
- breaking evidence は host が観測した事実だけを利用する。通常 CLI には API/CLI/configuration/stored-format の全面的な observer はまだなく、空の evidence pool を互換性の証明にしない。モデルが `unresolved` を選ぶ場合は停止する。host evidence が存在する group では `none` を許可しない。
- テストのみの group は、現在は Go の `_test.go` に限定して `test` を拘束する。他言語の全面的な test 判定は提供しない。
- モデル未準備・helper 不在・対応外 platform の場合は停止する。モデルの自動取得や Ollama への自動 fallback は行わない。
- 既存設定ファイルは書き換えない。他モデル、Ollama、広い context、4ファイルを超える入力を使用する際は `llm.planner = "single-pass"` を明示する。既存の単段生成・repair・transport retry の契約はその構成で維持する。

今後の評価対象は、未知入力の意味評価、summary、breaking evidence の観測範囲、規模拡大、および時間である。これらを暫定採用済みという理由で通過扱いにしない。helper の旧単段経路に関する既知の OSLog 課題 #116 も、この選定では解決扱いにしない。

## コード反映時の検証

- Go の全テスト、および Go build が成功した。既存 Ollama / MLX の単段試験は、その構成を明示して継続する。
- Swift の32テスト、および helper の release build が成功した。
- 保存済みの固定モデルと release helper を用いた公開可能な1ファイルの合成入力では、通常 CLI の入力経路から3段を完了し、40.683秒で有効な plan を返した。計画生成前後の Git 状態は一致した。この値は A285 の2入力と条件が異なるため、速度改善の比較には使わない。
- setup / doctor の3段能力 probe も、同じ保存済みモデルと helper で35.503秒で完了した。
- 実機試験で発見した bounded channel の区切り抽出不整合を修正した。native 制御が JSON grammar に切り替わる `<channel|>` の直後を検証し、思考本文を返さない。
- 実機 smoke は `COMMITER_CANDIDATE_SMOKE_HELPER` と `COMMITER_CANDIDATE_SMOKE_CACHE` を指定した `go test ./internal/cli -run TestGemmaThreePhaseDefaultPlanning -count=1 -v` で再実行できる。指定がない通常テストは隔離した mock helper とモデル metadata を使う。モデルの取得処理は含めない。

ローカル検証とリリースは別の状態である。この選定記録は、production の精度保証やリリース完了を意味しない。
