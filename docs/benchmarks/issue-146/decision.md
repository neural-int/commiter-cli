# Issue #146 の設計判断

## 結論と適用範囲

現行 Gemma / prompt / generation profile のまま、多ファイルの production 上限を拡大する案は見送る。測定範囲は4〜16ファイルであり、通常 CLI の上限は4ファイルを維持する。4ファイル以内でも現行 planner の意味的な正しさや plan 成功を保証する結果ではない。

本 Issue は Design / Verification として、方式比較、契約、コスト、失敗条件、採用・見送り判断を残す。多ファイルの機能提供や model / prompt の改善は本変更に含めない。追加の意味的改善や production 採用判断は親 #145 の後続作業として扱う。測定から採用の根拠が得られなかったことを、機能拡張の成功と扱わない。

## 採用する検証契約

- selected file ID の集合を正本とし、graph の node 集合との一致を backend 呼び出し前に検証する。
- 既存 relation extractor の graph は window の候補を決める。source/test と changed identifier 等の edge を用い、path proximity は接続とみなさない。soft edge は grouping の union を行わない。
- Stage 1 は最大4ファイル。edge を ID 順に seed とし、未処理 edge を最も多く覆う file を追加する。最大4まで、同点は ID 順。edge を覆うための overlap を認め、孤立 file も全件提示する。
- 局所 membership は same / different の file 対制約に変換する。same の推移閉包を取り、different が閉包内に入る場合や同一対の判断が食い違う場合は停止する。投票・推測・repair は行わない。
- bridge は未確認 group の最小 ID 代表を最大4つ提示する。全体 partition の候補を得る手段であり、他の file の意味的な関係を直接検証したことにはならない。
- audited は bridge 後も、直接同じ window に提示されていない全 file 対を追加提示する。最小未提示対を seed とし、未提示対を多く覆う file を追加する。全対を確認しても、別の順序・別の組合せ・通常入力での意味的精度は保証しない。
- 完全割当、一意性、全制約の整合性を確認するまで Stage 2 / 3 を開始しない。metadata は全体 group の全 file を保持し、最大4 group の packet と16K contextで扱う。全 category を先に終え、summary と最終 `planning.Validate()` まで成功して初めて plan を返す。
- 不正な局所 JSON、duplicate / missing / unknown ID、矛盾、未確認、context overflow、時間・window 上限、metadata 失敗のいずれでも部分 plan を返さない。

これらは実験用ハーネスの契約として採用する。production の多ファイル planner を採用したという意味ではない。

## 比較と見送り根拠

| 方式 | 検証結果と判断 |
| --- | --- |
| 全件を大きい model input に渡す | 最大4ファイルの Stage 1 契約を崩すため対象外。実装しない |
| graph-only | controlled 7ケース中5ケース、contract 3ケース中2ケースが未確認。欠落 edge を独立性に読み替えない限り確定できず、単独方式を見送る |
| bridge | controlled の完全割当7/7に対し exact は3/7。contract 6ファイルで FM12、12ファイルで contradictory pair。代表判断のみで production 上限を拡大する根拠はない |
| audited | controlled 8ファイルでは bridge の exact 成功後、5回目で transitive contradiction。追加提示は見逃していた不整合を検出した。一方 contract 6ファイルは全対監査後も FM12 のまま整合した。全対確認自体は意味的な正解の証明にならず、production 採用を見送る |
| soft edge による hard merge / 多数決 | Issue の安全契約に反するため実装しない |
| 現行4ファイル上限の維持 | 今回の production 判断として採用。canonical request の一致と実モデルの4ファイル比較を確認し、検証結果だけを理由に品質条件を緩めない |

## 規模と予算

検証ハーネスの対象は最大16ファイル。局所判断の上限は4ファイル、初期 window と overlap / bridge / audit を合わせて最大48 windows、全体600秒、各局所 generator は既存120秒以内で全体 deadline を共有する。metadata は最大 `2 × ceil(G / 4)` calls。G <= 16 なら実験用 backend call の上限は56。context は全 phase で16384 tokens、Stage 1 output 枠768、Stage 2は512、Stage 3は768とする。helper は実 tokenize 後にも context を検証する。

48 / 600秒は測定を有限に止める guard であり、利用者向け SLA や production 採用予算ではない。oracle で全対監査を完了した window 数は4ファイル1、5ファイル4、8・9ファイル8、12ファイル16、16ファイル20。意味的な失敗が早く起きて calls が減る場合を性能改善として扱わない。

この観測から確認できたのは16ファイルまでの契約検証と失敗測定であり、安全に拡張できる最大実用規模は確立できていない。production で今回拡大できる規模はない。16を超える推定や無制限の overlap は行わない。

## 4ファイル互換性と測定の限界

4ファイル以内は全件を1つの window にする。metadata も source-first path 順、group の first-seen canonicalization を現行実装と揃え、メッセージ、schema、生成オプションが一致する回帰テストを通した。実モデルの contract 4ファイル比較は現行・候補とも exact、FM0 / FS0、input3542 / output1075 tokens、3calls、Stage 3 失敗だった。現行が失敗しているところを候補の独自成功として数えない。

controlled suite の gold は固定参照であり、一方向の定数変更を含む。contract suite は変更前後の実行可能なテストで意図を具体化し、実 parser / extractor を用いたが、初回結果を踏まえた追加診断である。3ケースの単発測定を通常入力の精度や独立 holdout と扱わない。期待値はモデル入力から除外し、既存の gold と測定結果を上書きしていない。

実 token は計測用 helper が報告した値。output は native thought / channel の token を含む。wall time は helper load を含むハーネス実行時間で、fixture / graph の事前構築、build、モデル準備、Git 入力収集を含まない。生成本文・thought は保存せず、Stage 3 の追加診断は件数・文字数・schema 成否だけを保存する。

contract 6ファイルの追加数値診断は、Stage 3 の group 数1 / 1、missing / unknown 0、strict schema 成功、scope4文字、summary72文字だった。停止原因は summary の48文字上限超過であり、grouping の FM12 と独立した失敗である。

wall time はローカル観測値であり、推論中に軽い Go build / 契約テストも実行した。CPU・熱・他プロセスを隔離した latency benchmark ではないため、数秒の差や production latency の根拠にはしない。

## Issue 完了条件との対応

| 完了条件 | 証拠 |
| --- | --- |
| 実装前の bounded architecture 測定・比較 | runtime 無変更の standalone harness、Iteration 1 / 2 の graph-only / bridge / audited 比較 |
| window が final commit boundary にならない | oracle の9・16ファイル単一 group、global metadata に完全な group を渡す契約テスト |
| overlap / reconciliation contract | 本文の window規則・same/different・bridge / audit・予算の定義 |
| soft relation の irreversible merge をしない | edge は提示候補のみ、soft edge 単独で union しない回帰テスト |
| contradiction / unresolved fail-closed | 実12file直接矛盾・8file推移矛盾、停止前に metadata がない raw trace、予算・不正ID・cancel テスト |
| 5〜8 / 9〜16 の完全割当評価 | controlled 5 / 8 / 9 / 12 / 16、contract 6 / 12、停止ケースも欠測と分けて記録 |
| exact / FM / FS / unresolved の独立記録 | 全 JSONL の各フィールド。未解決ケースの exact / FM / FS は null |
| call / wall / token cost | 全実 calls の per-window / phase tokens・wall・stop、oracle は backend0 / tokens null |
| 現行4file baseline の回帰判定 | controlled と contract の実モデル比較、canonical message/schema/options 一致の回帰テスト。意味的改善なし、plan成功の新たな根拠なし |
| Stage 2 / 3 は全体確定後のみ | 実停止trace、全体未確定時・監査矛盾時に metadata 不実行の契約テスト |
| 採用・見送り根拠 | 検証契約と現行上限維持を採用、多ファイル production 採用を見送り。比較表と実例を明記 |

この対応は Design / Verification の成果を示す。安全な多ファイル production 拡張を実現したという完了判定は行わない。
