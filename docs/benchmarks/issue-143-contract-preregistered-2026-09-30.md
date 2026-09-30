# #143 候補表現比較の事前固定条件（2026-09-30）

前回結果8c951fdを起点とする別実験。既使用fixtureの診断的比較であり、新規holdoutではない。入力十分性・goldの独立承認を前提にしない。人間確認についての所感はIssueコメントの報告で、例ごとの独立回答は未記録。前回240callを変更・追加再集計しない。

## 独立変数と固定項目

従来partition-listと新file-membershipの2arm。新armはcandidateのgroup列挙をfile別所属表へ可逆変換し、systemに符号化の読み方を追加する。candidate_idsが列順、各fileのgroup_idsが同順、同一候補列内で同じgroup IDなら同commit。group IDは候補内だけのラベルで順位ではない。列挙形式と説明文を含むcontract一式の比較で、説明文・byte長・token化差の単独効果は分離しない。

元task、repository_input bytes、raw diff、relation context、全file IDs、candidate IDs・partition集合、gold、output JSON schemaを固定する。group labelsは各候補partitionをcanonical化してG001から割当。元candidate順は正逆で反転し、新armでは対応する列順を反転する。schema enum順は従来どおり固定。追加のsemantic evidence、summary、pair-local model判断、candidate pruning、repair/retryなし。

## 対象と予算

モデルは前回完了結果が得られたMinistral(index0)、Granite(1)、Phi(2)、Gemma(4)の既存pin4bit。Nemotron(3)は48/48 internal_errorだったため、この品質比較では実行せず別互換性問題として保持する。結果による途中除外なし。新規download・依存追加なし。

既知4例と元holdout8例、正順・逆順・逆順・正順の4call/arm/model/fixture。4モデル×12例×2arm×4call=384call。各arm/modelで既知16、元holdout32call。2048 output tokens、120秒/call、context8192、warmup/repair/retryなし。前回helper SHA-256を固定。backend wallは起動・モデルロード込み。出力token count未取得ならunavailable。

run→fixture→model→arm順。モデル順は(position+fixture_index+run_index)%4、arm順は(offset+original_model_index+fixture_index+run_index)%2で反転する。GPU同時推論なし。従来armも同時期に再測定し、arm差は各モデルの同時期従来armに対して比較する。Ministral従来armをモデル間baselineとする。

## 検証と指標

実行前manifestに全48入力のprompt/schema/repository hash、親manifest、対象model indicesを固定。親12contract一致、従来prompt/schema hashが前回と一致、新表現から元partitionへのround trip、repository/task/output schemaのbyte一致、gold/rationale非漏洩を検査する。

全384予定callを分母にgold/完了率を記録。不正出力・runtime failureも失敗に含む。partitionなしのfalse merge/splitは算出不能。元holdoutのgold、false merge/split、guardrail false merge、正しいjoin、完了率、構造的不正、wall中央値を集計する。提示方向差と同一方向反復差はfixture単位で分ける。4callを独立した新規例とは扱わない。

方向依存への次段候補条件を推論前に固定:
- 元holdout gold28/32以上、同時期Ministral従来arm比+4call以上
- 同モデル従来arm比で元holdout方向差が2fixture以上減る
- guardrail false merge0、正しいjoin/false split/完了率にMinistral従来arm比悪化なし
- 構造的不正0、wall中央値がMinistral従来armの2倍以内
- 同モデル従来arm比で元holdoutの反復差が増えない

全条件未達でもモデル・contractの一般的不可能性や入力不足を確定しない。合成goldの妥当性、モデル能力、template/grammarとの相互作用は未分離。production採用判断、Pass2+planning.Validate end-to-endは対象外。通常CLI・production planner・SRS・既定model/backendは変更しない。

結果JSONLはfile/candidate IDs・hash・数値・failure codeのみ。raw prompt/response/生成summaryなし。結果コメントは実測条件・観測・未測定事項に限定する。
