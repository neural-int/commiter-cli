# Iteration 9 事前登録: global task contract / grouping-only routing

入力はH6のcanonical entity/contract IR、global instruction/schemaは同じ。prototype helper copyはnative1024/output1536のbounded-global-contractと、native0/output1536のbounded-routed-groupingを追加する。context16K、temperature0/top_p1/top_k0/seed144、120秒/call、repair/retry0。production helper/source/defaultは変更しない。新依存とmodel downloadなし。既存build cacheはAPFS cloneで再利用し、locked resolution/skip updateでbuildした。

まずGemma extendedをweak16/cross12各1回。routingはQwen3.5、Phi4mini、Granite4.2の順でweak16を各1回。weak16がexact/FM0/FS0/completeならそのrouteのcross12を1回測る。structural failure/semantics failureならqualification不成立で同条件反復を追加しない。routeとgeneration contractを同時に変えるためmodel単独効果を主張しない。

固定cached revisions: Qwen3.5-4B 32f3e8ecf65426fc3306969496342d504bfa13f3、Phi4mini ac1c269cb4222a4e136a3d09edad301056c1f36a、Granite4.2-3B 0c6f39b1827afd5eb2c1c3b13751929857434953。Gemma revisionは従前。native0 routeはJSON grammarなしで生成しstrict JSON/ID検証で失敗時停止する。別modelはglobal groupingだけ、metadataは元のGemma/profileを使う。暗黙fallbackなし。

両資格成立後に4/6/8/9/16、独立評価、共通callee guardrail、canonical input逆転、metadataとplanning.Validateまで確認する。予算probeやgrouping-onlyの成功をproduction candidate成立へ読み替えない。
