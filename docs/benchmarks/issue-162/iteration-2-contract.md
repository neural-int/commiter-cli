# Iteration 2: 複数許容分割とhost-owned dependency

**初回の5分類Gateを緩めず、別の責務分離仮説を評価する。** Gemmaの局所レビュー対応能力を、新しい未使用の自己作成Go入力で確認する。ユーザーの明示した次試験実行に基づく。これは逐次グループ構築やproduction導入ではない。

## 評価する範囲

4/6/8 selected filesの8 workloadに、固定対象EA/EBの局所境界を一つずつ持つ。残りのchanged filesは独立な定数変更としてsingletonを保持する。LLMは対象2file全文＋未変更のgo.mod/support/test＋hostのsource factsを受け取る。他changed filesの意味判断は要求しない。8file全体の目的推論を達成したという主張はしない。

初回のPython fixtureを再利用せず、trim/bucketの実装とassertion、shared helper/same packageの独立変更、新APIとcallerの両方向、並行timeout/flag変更を新規作成。単一作者controlled入力であり、自然な公開履歴や独立作者の採用holdoutではない。source/gold監査時点で使用済みになる。目的を変更したため、初回8/16との数値を同一指標の改善と呼ばない。

## 許容される分割

- 対応実装＋変更assertion: 同時レビューを要求し同一group。片方だけの適用は今回fixtureのtestsで失敗することを推論前監査する。
- 独立したobservable behavior: 同一package・shared helperだけで統合しない。対象2fileは別group。
- 新API＋caller: 同一groupも、provider→consumerの順序付き分離も許容。後者のrevertはconsumer→providerの順序を明示する。任意のgroupを独立revertできるという主張と区別する。
- timeout/flagsの多義的変更: mergeもsplitも許容。作者のpolicyを推測して一意goldへ固定しない。deferも合法な保守計画へ変換できる場合は許容。
- 構造不正/部分適用のtest失敗: 最終計画として採用しない。LLMの不正応答は保守案へ退避するが、応答Gateは失敗のまま保持する。

## 機械とLLMの責務

Go標準parser/ASTで、同一package内の新規top-level functionと新規unqualified callからprovider/consumerを抽出する。これは限定的なcompile依存の根拠であり、意味同一性ではない。method、import alias、build tag、rename、全behavior依存の網羅は非対応。Go host sorterが順序を作る。metadataは検証用の固定chore/summaryであり、目的やbreaking情報の生成能力は未評価。

Gemmaはmerge/keep_separate/deferだけを返す。依存方向を選ばせない。mergeには両対象のsource IDを要求するが、IDの存在は意味根拠の正しさを証明しない。元source参照を保持する。

## 比較対象

1. file-only: 入力順で各fileをsingleton。
2. static-new-api-order: singletonを静的依存のtopological orderで並べる。
3. test-assisted-pair: authored fixtureのbefore/only-A/only-B/both/afterを実測。両単独stateがfailならmerge、一方だけfailなら方向付き分離、双方passなら分離。goldをruntime判定に使わない。固定2file単位の限定baseline。
4. Gemma＋static host: 同じsource/static factsから局所境界を提案し、hostが順序・assignmentを確定。

3はモデルには見せないtest実測情報を使うため、4との差を純粋なLLM処理の因果効果とは呼ばない。品質・追加calls・検証costの代替方式比較。source-only比較は2対4。どの方式も全機械的手法の最良を代表しない。

## 検証・予算

各計画についてunchanged authoritative planning.Validate、全ID exactly-once、使い捨てGit index/treeの各状態のbyte-exact確認、forward適用・dependency reverseのrevert順でGo testsを検証。さらに各groupを最終stateから単独revertして、独立性と依存付き可逆性を別記する。fixtureのGo標準libraryのみ、GOPROXY/GOSUMDB off、外部repo source/test無し。root repo/indexへGit mutationしない。副作用防止のため継承したGIT_*環境を使い捨てGit操作から除外する。

tests PASSはfixtureに含むassertionの範囲のみで、未観測の仕様/全runtime correctness/レビュー同一目的の証明ではない。順序違いでも同じcase/ordered partitionの監査結果はcache再利用し、初回監査時間をtotalに含める。常駐/uncached production latencyとは別。

固定Gemma/helper/revision/neutral grammarは初回と同じ。temp0/top_p1/top_k0/seed144/native thought0、context16384/output1536、16call、per-call30秒、whole1800秒。mandatory audit用120秒を推論開始判定で予約する。baseline監査・推論・結果検証込み。timeout/tokens nullを0へ捏造しない。source/model/helper/binary/input/prompt hashを推論前に保存してcommit・push。

Gate: 16/16 completed/応答valid、16/16の許容分割＋ordered tests＋reconstruction＋authority valid、8/8提示順で同じpartition、対応2caseの両順でstatic baselineより改善、独立変更の誤統合0、全予算内。これは局所境界判断の次段限定GOであり、16fileやproduction GOではない。未達なら残存原因を凍結し、閾値/入力を事後変更しない。
