# Iteration14事前登録: 未導入8B checkpointの能力資格試験

実行前提: ユーザーによる4,623,782,544 bytesのmodel取得承認と、10GiB以上の空きまたは余裕ある別cache保存先。既存ファイルの削除は含めない。未承認・容量不足の間はdownload/build/inferenceを行わない。

checkpointはmlx-community/Qwen3-8B-4bit@545dc4251c05440727734bcd94334791f6ab0192、4bitへ固定。既存mlxmodel.Storeのdigest検証とReady契約を使い、public registryへのmodel取得だけを行う。repository dataやpromptを外部送信しない。追加dependency、production default/backendの変更なし。helperの変更はexperimental copyのbounded-routed-grouping許可model IDに限定。別保存先の場合は-group-cacheを使い、base Gemmaは既存cacheを使用する。

H9 observed-anchorsの入力・schema・canonicalization・host validationを維持し、grouping checkpointだけ変更する。native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0、no fallback。metadataは従来Gemma/profileでfinal grouping後のみ。

weak-edges-independent-16とcontract-cross-boundary-12を各1回、同時推論せず実行する。実tokens/phase-wall/whole-wall/model pin/stop/exact/FM/FS/complete/unresolvedを記録。いずれかsemanticまたはstructural failureなら資格不成立。同条件反復、seed/prompt/順序による調整、native budgetの事後変更で資格を取り直さない。

両方exact/FM0/FS0/completeなら、未推論fresh holdout-protocol-and-health-8を1回。それも通過した場合のみ各レンジ、旧regression/holdout、shared-callee guardrail、順序、final metadata/Validateへ進む。8Bというparameter数自体を改善根拠にせず、取得やhelper load成功をproduction candidate選定と混同しない。
