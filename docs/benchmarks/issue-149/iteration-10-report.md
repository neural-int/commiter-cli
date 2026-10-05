## 要約

残る保存済み3checkpointを資格試験したが採用routeは得られなかった。Ministral Reasoningはweak16を正しく分割した一方、cross12を12個に分断してFS30だった。4fileの順次実行では現行baselineとH5の双方がexactかつfinal Validate成功。利用者の回答により、今後も変更目的はdiff・コードの観測情報だけを根拠にする。

## 検証結果

| route / fixture | exact | FM / FS | complete | unresolved | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | ---: | --- | ---: |
| Nemotron Nano4B / weak16 | null | null / null | false | true | 1 | null / null | 0.805 |
| Ministral Instruct / weak16 | null | null / null | false | true | 1 | 3653 / 1536 | 115.839 |
| Ministral Reasoning / weak16 | true | 0 / 0 | true | false | 1 | 3653 / 214 | 34.921 |
| Ministral Reasoning / cross12 | false | 0 / 30 | true | false | 1 | 6034 / 162 | 42.888 |

Nemotron stopはinternal_errorでinput計測前に停止、詳細原因は未特定。現runtimeのregistryにはnemotron_hがあるためunsupported modelと断定しない。Instructはmax_tokens。Reasoningはcompletedで全件一意割当、unknown/duplicate/missing ID0。route側metadata未実行。HF Qwen3-8B cacheはtokenizer/configのみでweightsがなく、利用可能checkpointに数えなかった。model downloadなし。

| 同じcontract-baseline-4 / serialized | exact | FM / FS | complete / unresolved | calls | input / output合計 | wall s | final planning.Validate |
| --- | --- | --- | --- | ---: | --- | ---: | --- |
| current three-phase | true | 0 / 0 | true / false | 3 | 3542 / 1096 | 54.526 | pass |
| H5 assertion-facts + current metadata | true | 0 / 0 | true / false | 3 | 3780 / 1087 | 61.563 | pass |

順次実行の6callsは全completed、timeout/context overflow0。phase別tokens/wallはJSONL参照。categoryは双方1271/435、textは双方1155/91。初回のfull4は別model推論と並行し、currentは73.858秒でValidate成功、H5は120.258秒・metadata_backend_failureだった。その原記録を保存し、resource条件が異なるlatency比較から除外した。計測backendがcontext期限終了をtimeoutとして保持するよう補修し、期限切れctxの回帰testはpass。旧unknown stopを改変していない。

公開cross12 fixtureではsource+testの6pairを順に適用した5中間状態が全てgo test成功。このpartitionは著者goldの2groupに対してFM0/FS24だった。test-passだけではglobal semantic groupingを識別できない反例であり、利用者repoを実行する機能は追加していない。

weak16の観測requestを固定して独立16groupと一括1groupの反事実goldを比較した入力監査は、request hash一致・不一致pair120・real model calls0。元goldは変更していない。ユーザーはdiff・コード観測だけを根拠にする方針を選択した。

## 考察

Reasoning routeには独立定数を分ける能力の観測があるが、実装/testを統合する要件を同時に満たさない。共通callee guardrailがあるため、呼出しgraphをhard unionへ変えて補修しない。runtimeのgreen状態も変更目的そのものを保証せず、runtime authorityのみの方式は棄却する。

入力監査の反事実一括意図はcode上で観測できない。ユーザーの明示した根拠範囲では、この不可観測な意図を採点goldにしたり、探索天井や入力不足の根拠にしたりしない。既存の観測可能な異なるentity/behaviorを独立とするweak16 goldを維持する。有限のroute失敗を全architectureの不可能性へ読み替えない。

4fileでは最終planまで成立したが、5〜16fileへ安全に拡大するcandidateはまだ未選定。4file成功や速いrouteだけで採用条件を緩和しない。Skill未使用、production変更なし。

## Next Steps

- H8として観測根拠を伴う変更contractの集合をglobal outputにする方式を評価する。flat file→group IDだけでは、別entityの2→3を共通契約へ抽象化してしまうため、before/after・具体的subject・observed evidence IDsを明示する責務へ変える。
- hostは根拠IDの実在、memberごとの観測根拠coverage、complete assignmentを検証する。内容のsemantic correctnessは別途goldで測り、構造検証だけを意味的証明として扱わないため。
- same inputのweak16/cross12を各1回資格試験し、通過時のみholdout・大規模・順序・metadataへ進む。既に失敗したflat-output routeやprompt wordingだけの再試行を避けるため。
