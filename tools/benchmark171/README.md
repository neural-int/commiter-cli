# Issue #171: Direct Grouping 2×2 比較

対象は既存 ThreePhaseGenerator の第1段階だけ。`main.go` の捕捉クライアントで実際の入力・prompt・schema を記録し、その呼出しで終了する。モデル呼出しはしない。`collect.go` は #169 の Git collector と syntax 準備を再利用する。fixture のコードはデータであり実行しない。新Planner・後続stage・Production変更・新依存はない。

12件の未使用合成fixtureを通常順/逆順・A/B/C/Dで比較する。confirmed 11件、undetermined 1件。multiple_valid は証拠を伴う実例が今回ないため0件と記録し、評価器の許容解集合処理だけを回帰検証する。実repository・別作者の一般化は主張しない。F11はホスト側の明示仕様によりGoldを確定するが、モデル入力にはその仕様を含めず、confirmed分母に残して情報不足を別記する。

1. `go build -o /private/tmp/commiter-171-prepare ./tools/benchmark171`
2. `python3 tools/benchmark171/prepare.py --runtime /private/tmp/commiter-171-runtime.json`
3. `python3 tools/benchmark171/preflight.py --runtime /private/tmp/commiter-171-runtime.json`
4. `python3 tools/benchmark171/audit.py /private/tmp/commiter-171-runtime.json` と `python3 -m unittest discover -s tools/benchmark171 -p 'test_*.py'`
5. Gate・資源・全入力・採点器・モデルpinを事前登録し、fixture/Gold確認と推論0回の成果をcommit/pushする。
6. `python3 tools/benchmark171/run.py --runtime /private/tmp/commiter-171-runtime.json`
7. `python3 tools/benchmark171/evaluate.py` で固定GateのGO/NO-GOを判定し終了する。

通常順は既存Plannerの決定則（非 `_test.go` のパス辞書順、続いて `_test.go` のパス辞書順）。逆順はID・schemaを変えずfile配列だけ反転し、全armに同じように適用する。Productionのソート処理を変更していないため、ソート実装の追加効果はこの試験では推定しない。order一致とGold正解は別の指標。

既存 #169 helper、tokenize-only preflight、厳密JSON/Grouping validator、資源観測を使用する。retry 0、上限96call、1cell1回、timeout120秒。入力不足・構造不正・未確定Goldを事後除外せず記録する。生成前の凍結ファイルは結果を理由に書き換えない。処理中断時もintent journalによる再実行を拒否する。
