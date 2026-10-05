# Issue #146: 第1回 bounded planning 比較

2026-10-05、最大4ファイルの Stage 1 を再利用する window / reconciliation の成立と、固定 Gemma の出力を分けて評価した。全体の一意割当は7入力すべてで成立したが、固定参照との grouping 一致は3/7、最終 plan の成功は1/7だった。代表だけを追加判断する方式の production 採用は、この測定では支持できない。

## 条件と測定範囲

- baseline commit: `f275d95`。Apple M3、16GiB。
- Gemma: `mlx-community/gemma-4-E4B-it-4bit@475b9088d29754a3379866cf5aeb6b41acd313c2`、MLX 4bit、固定16K context。
- temperature 0、top-p 1、top-k 0、seed 144。現行の段階別 generation profile と生成枠を維持した。
- Stage 1 は最大4ファイル。最大4ファイルの制約は membership 判断に適用し、確定済み group の Stage 2 / 3 は全ファイルを保持して独立に固定16K context を検証する。
- 実験用 guard: 最大48 window、1 fixture の cycle は600秒。局所 generator の120秒上限も維持した。現行 baseline は3呼び出し・120秒をそのまま使用した。これらの実験 guard を production の採用予算とは扱わない。
- モデルは保存済み cache を使用し、取得していない。計測専用 helper は現行 source のコピーへ数値 telemetry だけを追加した。生成設定、model / backend の既定値、通常 CLI は変更していない。

input tokens は chat processor が作った token 列の長さ、output tokens は runtime の generationTokenCount で、native thought / channel を含む。byte 数の推定値ではない。phase / window ごとの数値は JSONL に保存した。wall time は window 構築から局所判断、統合、metadata、最終検証までで、事前の fixture / graph 構築、build、モデル準備、Git 入力収集は含まない。helper の load は各呼び出しの時間に含む。

## モデルを使わない契約比較

正解を返す oracle では `bridge` が7/7の参照 grouping と全件一意割当を達成し、metadata / `planning.Validate()` も成功した。実 backend calls は0で、モデル精度や実 token cost の観測に数えない。

| 入力 | files | graph-only の window 数 | graph-only | bridge の window 数 | bridge |
| --- | ---: | ---: | --- | ---: | --- |
| baseline | 4 | 1 | 確定 | 1 | exact、完全割当 |
| 同一 directory の独立変更 | 5 | 2 | 未確認が残り停止 | 4 | exact、完全割当 |
| implementation＋tests | 8 | 2 | 未確認が残り停止 | 3 | exact、完全割当 |
| directory を跨ぐ単一 intent | 9 | 3 | 確定 | 3 | exact、完全割当 |
| 境界を跨ぐ複数 intent | 12 | 4 | 未確認が残り停止 | 5 | exact、完全割当 |
| edge 欠落の単一 intent | 16 | 4 | 未確認が残り停止 | 5 | exact、完全割当 |
| 弱い edge の独立変更 | 16 | 4 | 未確認が残り停止 | 20 | exact、完全割当 |

`graph-only` で未確認が残る5入力は、関係がないことを別目的の根拠にせず `unresolved` にした。9ファイルの chain と16ファイルの edge 欠落入力では、window 境界を跨ぐ全体 group が成立した。

直接矛盾、same の推移閉包と different の衝突、duplicate / missing / unknown ID、入力 context 超過、cancel、window 予算超過を targeted test で検証した。通常 Stage 1 の構造検証を通らない出力は全体統合へ渡さず、grouping が未解決の場合は metadata を呼び出さない。metadata の duplicate key / unknown group / unresolved evidence / context 超過でも部分 plan を返さない。

## 固定 Gemma の測定

FM / FS は file 対の誤統合・誤分割数である。exact は group 名・表示順を除いた固定参照との一致を表す。complete は全件・一意割当であり、意味的に正しい grouping を表す指標ではない。

| 入力 | files | exact | FM | FS | complete | grouping unresolved | 最終 plan | calls | wall 秒 | input tokens | output tokens |
| --- | ---: | --- | ---: | ---: | --- | --- | --- | ---: | ---: | ---: | ---: |
| baseline | 4 | 不一致 | 4 | 0 | yes | no | Stage 3 停止 | 3 | 51.441 | 2,753 | 932 |
| 同一 directory の独立変更 | 5 | 不一致 | 10 | 0 | yes | no | Stage 3 停止 | 5 | 90.254 | 3,586 | 1,698 |
| implementation＋tests | 8 | 一致 | 0 | 0 | yes | no | 成功 | 5 | 131.127 | 5,586 | 2,076 |
| directory を跨ぐ単一 intent | 9 | 一致 | 0 | 0 | yes | no | Stage 3 停止 | 5 | 140.752 | 6,085 | 2,216 |
| 境界を跨ぐ複数 intent | 12 | 不一致 | 36 | 0 | yes | no | Stage 3 停止 | 7 | 196.626 | 8,353 | 3,038 |
| edge 欠落の単一 intent | 16 | 一致 | 0 | 0 | yes | no | Stage 3 停止 | 7 | 223.650 | 10,487 | 3,270 |
| 弱い edge の独立変更 | 16 | 不一致 | 120 | 0 | yes | no | Stage 3 停止 | 7 | 196.195 | 9,423 | 2,696 |

39呼び出しはすべて `completed` で、helper の context overflow / timeout / incomplete output は0だった。accepted grouping の duplicate / missing / unknown selected file ID は0。grouping unresolved は0/7だが、これは参照に対する誤統合を検出したことを意味しない。最終 plan の成功は1/7で、残る6入力は Stage 3 の group ID または長さの検証で停止した。保存した集計値から、その内訳を区別することはできない。

現行 `ThreePhaseGenerator` の同じ4ファイル入力では、FM=4、FS=0、complete=yes、3呼び出し、51.926秒、input/output は2,753/932 tokens、最終検証は `invalid_schema` で停止した。候補の Stage 1 は同じ全件 window と契約を使用し、同じ参照差を観測した。ただし候補 metadata adapter の file / group 表示順はまだ現行と完全に揃えていないため、この測定だけで Stage 2 / 3 の無回帰を確定しない。

診断を追加する前の最初の baseline probe も別 artifact に保存した。3呼び出し、54.631秒、2,753/932 tokens、最終検証停止を観測したが、Stage 1 grouping を保持していない。この probe の complete=false / exact=null を、membership が欠落した証拠に使わない。

## 考察と次の比較

局所 membership が参照どおりなら、window を跨ぐ9 / 16ファイルの group と、独立した16 group を Go 側の制約で表現できる。graph の soft relation を union に使わず、未確認を別 group と決めつけない設計は、モデルを使わない試験で成立した。

実モデルでは、同一 group と判断されたことで未確認対がなくなり、window の追加が早く終わった入力がある。16個の独立変更は oracle の20 window に対して5 window で終了し、FM=120だった。calls が少ないことを scalability の利点として採用すると、grouping quality の低下を見落とす。

これらの初期入力には、異なる定数を同じ方向へ更新するだけの合成変更を含む。固定参照への一致は測定できるが、それだけでは実際の変更意図の曖昧さと、モデルの意味判断の失敗を切り分けられない。通常変更の精度推定、モデル内部の判断根拠、production の規模保証には使用しない。

次の iteration では、次の比較を行う。

1. metadata の source-first / group 正規化を現行と揃え、4ファイルでは実際の prompt、schema、profile が同じであることを確認する。metadata の表示順を回帰比較の交絡要因にしないため。
2. 期限境界・金額丸め・zero capacity の異なる修正契約と対応テストからなる4 / 6 / 12ファイル fixture を、before / after の Go テストを通して固定し、既存の syntax / relation extractor で比較する。理想化した graph と、独立した修正目的を持つ実抽出 graph を分離するため。
3. 代表間の観測だけで確定する方式と、残る file 対を最大4ファイルの window で監査する方式を比較する。未観測の対に context 依存の矛盾がないと仮定する部分と、その検証 cost を確認するため。
4. Stage 3 の診断へ group ID の件数と scope / summary の文字数だけを加える。生成テキストや生成条件を変えず、停止の種類を分けるため。

追加 fixture は、この初期観測を受けて設計した補助入力である。採用用の独立 holdout として扱わない。通常 CLI の4ファイル上限は維持し、代表だけで確定する方式の production 採用は見送る。

## Artifact

- `iteration-1-oracle.jsonl`: 固定参照を返す契約比較。
- `iteration-1-mlx.jsonl`: 7入力の実 helper 測定。
- `iteration-1-baseline.jsonl`: 現行4ファイル経路の比較。
- `iteration-1-baseline-diagnostic.jsonl`: 診断追加前の probe。
- `environment.json`: model / helper / fixture の固定値と測定範囲。
- `tools/benchmark146`: 再実行するハーネス、計測 helper の作成、契約 test。
