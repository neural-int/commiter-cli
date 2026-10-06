## 要約

H15として観測E-IDとpurpose G-IDを分離したが、3case全てを一括groupへ結合した。weak16 FM120、cross12 FM36、guardrail FM12。anchor方式のfalse splitは解消した形でもfalse mergeを増やしており、採用しない。

## 検証結果

H13の全観測情報・observer・contrastを維持し、global output/taskを具体的E-root所属から抽象purpose G-ID assignmentへ変更。複数entity contractを含めるpurposeをdiff/codeから判断する契約。entity root代表の拘束は外したが、未知/欠落/重複selected IDと未知G-ID/unresolved、不正JSONの拒否とpartial planなしは維持した。

| fixture | exact | FM | FS | complete | unresolved | calls | input tokens | output tokens | total wall秒 | observer probes/samples/unknown |
|---|---|---:|---:|---|---|---:|---:|---:|---:|---|
| weak-edges-independent-16 | false | 120 | 0 | true | false | 1 | 2888 | 192 | 15.037 | 0/0/0 |
| contract-cross-boundary-12 | false | 36 | 0 | true | false | 1 | 5434 | 176 | 19.472 | 4/16/0 |
| shared-callee-independent-6 | false | 12 | 0 | true | false | 1 | 2476 | 92 | 10.424 | 2/8/0 |

全call completed、各case1回、timeout/context overflow/不完全出力なし。固定Gemma 475b9088d29754a3379866cf5aeb6b41acd313c2、native0/output1536/context16K/call120秒/whole600秒、temperature0/top_p1/top_k0/seed144、repair/retry0。全caseの最終partitionは全fileを含む1group。モデル内部の判断理由は記録しておらず、原因を理由テキストから確認したとは主張しない。

focused anchor/contrast/interaction testsと既存のinvalid membership/duplicate JSON拒否テスト、benchmark build成功。任意code実行0、fresh protocol8/metadataの追加測定なし、production変更なし。

## 考察

G-ID出力に変更しただけでは、共通purposeと広い変更categoryを適切に区別できる根拠を得なかった。anchor方式で得られたweak16/guardrailの独立性を失い、cross12も異なるexpiry/cents目的をまとめた。形状/complete assignmentの成功は品質の根拠にならない。

現在のH15はnative0 contractで直接purpose assignmentを生成する。出力が一括結合した事実から内部推論の有無や原因は断定できない。以前のH7 native1024試験はcanonical declaration IRであり、今回のcaller/test/4snapshot/contrastとpurpose identityを分けたtaskではなかった。既存bounded deliberation contractを新taskへ適用する検証は未実施。ただし単なる予算増加自体を採用根拠にせず、過去failureと同一条件の反復は行わない。

## Next Steps

- 既存bounded-global-contractのnative1024/output1536の責務・stop/reservation契約を監査し、H15のabstract purpose判断との未試験の組合せを明確化する。モデル内部理由を憶測せず、生成contractとtaskの境界を確認するため。
- 次のH16を行う場合は同profile一つへ固定し、prompt/seed/orderや予算を探索しない。caller/test/causal観測を持つ新taskを、事前定義されたbounded deliberation phaseで扱えるかを独立評価するため。
- weak16/cross12/guardrailを各1回、全品質・費用を記録し、全資格通過時のみ未測定protocol8へ進む。失敗なら同taskの予算反復調整をせず、現capabilityと未検証architectureの残余を監査するため。
