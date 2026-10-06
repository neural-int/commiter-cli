# Iteration22事前登録: observation identityとpurpose identityの分離

H15 purpose-recordsはH13の全file/観測contract/test/caller/4snapshot値/contrastを維持し、E-IDを入力根拠に限定する。global出力は抽象G-ID assignmentへ変更し、複数entityの異なるcontractを含められるchange purposeをdiff/codeから判断するtaskへ分離する。具体的な一つのE-root選択/代表file拘束をpurposeの定義にはしない。gold partitionが旧schemaで数学的に表現不能だったという仮説ではない。初期H5自由assignmentと異なり、具体的entity記録・causal観測・有限比較を保持する。

未知/重複/欠落selected ID、未知G-ID、unresolved、不正JSON/途中出力/timeout/context overflowは従来partition/strict JSON gateで拒否し、partial planなし。duplicate JSON keysも拒否。latent author intentを入力せず、joint effectやsoft relationをhard unionにしない。metadataはfinal grouping後、final planning.Validateは維持。

fixed Gemma 475b9088d29754a3379866cf5aeb6b41acd313c2、native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。observer上限はH13と同じ。global task/output contractだけ変更し、prompt/seed/orderの反復調整はしない。

weak16/cross12/shared-callee independent6を各1回、順に測定。3case全てexact/FM0/FS0/completeなら未推論protocol8を1回。それも通過した場合のみrange/regression/order/metadata/Validateへ進む。失敗時は同条件反復で資格を取り直さず、観測結果から次の判断を行う。
