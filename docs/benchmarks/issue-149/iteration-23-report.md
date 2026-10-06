## 要約

H16としてH15 purpose判断を既存bounded deliberation contractで生成した。weak16はunresolved_or_missingとしてhostが拒否し、cross12 FM36/guardrail FM12。採用条件を満たさず、同taskの予算反復調整はしない。

## 検証結果

helper sourceを監査。native上限1024、native/channel/finalを含むtotal output1536、enable_thinking=true、final JSON grammar、runtime stop alignment、prompt+outputのcontext予約を確認した。これはnative量だけの比較ではなくgeneration contract全体の組合せ試験。helper/source/model pinは変更なし。

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens（native含む） | total wall秒 | host結果 |
|---|---|---:|---:|---|---|---:|---:|---:|---:|---|
| weak-edges-independent-16 | 未評価 | 未評価 | 未評価 | false | true | 1 | 2890 | 809 | 40.623 | unresolved_or_missing |
| contract-cross-boundary-12 | false | 36 | 0 | true | false | 1 | 5436 | 1146 | 59.398 | completed assignment |
| shared-callee-independent-6 | false | 12 | 0 | true | false | 1 | 2478 | 1086 | 49.819 | completed assignment |

全backend callはcompleted、timeout/context overflow/不完全backend出力は観測しなかった。weak16のhost membershipはunresolved_or_missingで拒否し、groupsなし/complete=false。そのexact/FM/FSを0や成功とは扱わない。cross12/guardrailは全fileを含む1group。各case1回。

fixed Gemma 475b9088d29754a3379866cf5aeb6b41acd313c2、native1024/total output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。observer制限とunknown処理は同じ。raw native思考/IR/promptは保存せず、内部理由を推測しない。追加取得/dependency/code変更、fresh protocol8/metadata試験なし。

## 考察

H15 native0では3case一括結合だった。今回weak16は未解決で停止し、cross12とguardrailの誤結合は同じ。bounded deliberationとgrammarでoutput contractを成立させても、purpose分離のsemantic改善を得なかった。生成予算やshapeの追加だけを採用根拠にはできない。

現時点のpurpose G-ID taskで評価したcheckpointはGemmaのみ。Qwen3-8Bはcausal contrastを持つE-root taskでcross12 FS30だったが、observation identityとpurpose identityを分けたH15 taskでは未評価。この未試験の組合せを残して、available capability全体の限界を結論することはできない。

## Next Steps

- H17として、H15 purpose-records/native0の入力/task/schema/gatesを固定し、取得済みQwen3-8Bの同revisionへgrouping-only routeする。以前の8B E-root試験と異なるidentity/taskの未評価組合せを一度確認するため。
- weak16/cross12/guardrailを各1回、既存native0/output1536等の固定budgetで順次測定する。追加取得、同taskの予算sweep、prompt/seed/order tuningを行わず、checkpointとtaskの組合せを独立評価するため。
- 全資格通過時のみ未測定protocol8へ進む。失敗なら試験済みfamily/既存Issue evidence/利用可能capabilityと合理的な未検証候補を棚卸しし、追加仮説の根拠または再開prerequisiteを明文化する。有限の失敗から普遍的不可能性を主張しないため。
