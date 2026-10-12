# 推論0回の事前登録

目的はA（#169実使用）とB/C（Issue提示本文）のpaired比較。Aは#169 H03-normal-B-discoveryのsystemから共通suffix以前をbytesで抽出する。Issueの表示上の改行は#169実使用では文間のspaceであったため、その実使用bytesを優先する。B/CはIssueの改行を保持する。共通の安全suffix、JSON Schema指示、Raw Diff、8 file fields、mode、task、順序、model/chat templateは全条件で一致する。full requestとsystem全文/hashを保存する。

モデルはgemma-4-E4B-it-4bit revision475b9088d29754a3379866cf5aeb6b41acd313c2。既存#167計測helper、#163生成なしpreflight helper、#169 collectorを同じhashで再利用。seed144、temperature0、top-p1、top-k0、context16384、output768、native thought512、bounded-groupingを固定。新たな生成stage、retry/repair、model downloadは0。既存collectorが同じsourceからHEAD/worktree/raw U3 diffを生成し、全index/worktreeの前後digestを照合する。token数はnative preflightと実生成時に照合する。

Known H01-H04は以前の結果を見た回帰fixture。normal順のみ4×3=12生成。reverseも事前登録するがknown品質は採用根拠にしない。新規N01-N04は未生成の自作合成fixture4件×2順×3条件=24生成。N01実装/test/consumer単一page-size契約、N02producer/consumer/test単一wire-key契約、N03同package/helperの独立slug/visibility、N04同fileの独立timeout/batchを含む。N03/N04は過剰統合を誘発する評価。目的goldは1/1/2/2の計6個。N04の目的goldは2個、whole-file grouping goldは不可分な1groupであり混同しない。Source、gold、labels、rubric、元commit messageはモデルへ渡さない。

未使用性は新規作成・初回実行journalと過去#168-169 sourceの完全一致監査と#167 generator sourceの対象symbol不在確認で確認する。過去と同種の合成構造を持つため、設計者独立性や実世界無作為性は主張しない。N01の同じ100→64はUI consumerとLimitの共通契約と事前定義し、testの別列挙を許さない。N03のnormalize共有は目的統合の根拠としない。意味が不明な説明はtarget-only/unknownで加点しない。

具体的coverageはgoldの具体的前後変更と対象を説明した場合に1目的として加点する。test/consumerにも同変更がある場合、説明がその契約を具体的に含めば同goldに対応する。単なるsymbol名や「testを更新」はtarget-only。各説明にsemantic_gold、target_gold、grounded、hallucination、理由を保存。one-to-oneで全goldを説明し余分・重複・統合・幻覚がない場合のみpurpose list exact。target_goldが複数ならpurpose overmerge=max(0,n-1)、同goldを複数説明で扱えばoversplit=max(0,count-1)。存在しない変更/根拠のない作者要件はhallucination。曖昧なtarget対応はunknownでexact不達、推定だけを加点しない。正常JSON以外、timeout、未受入は欠測/未解決として分母に残す。順序依存は説明の語句順を無視したgold対応集合と品質判定の差で測る。known/holdoutやnormal/reverseを独立標本にしない。

Gate1絶対基準：holdout coverage>=90%、purpose exact>=80%、schema/admission100%、grounding>=95%、overmerge=oversplit=hallucination=0、未知対応なし、順序不一致0。資源基準：全receiptあり、input/preflight一致、wall p95<=120s、helper RSS<=6GiB、前後swap増分<=256MiB、pressure<4、Git保持100%。RSSはprocess peak、swap/pressureは前後観測で短時間peakや他appの因果は判定できない。call timeout120s、critical pressure>=4またはswap増分>1024MiBなら停止。

B/Cは絶対基準を満たし、Aよりcoverageまたはexactが25pp以上改善し、両指標・oversplit・overmerge・hallucinationを悪化させない場合のみ候補GO。B/Cが双方GOならexact、coverage、oversplit、wall p95の順で比較するが小標本の同点は優劣未確定とする。相対wall p95<=1.25×Aも要求する。差はPrompt全体の効果であり文の単独因果効果ではない。BとCには否定文以外にも差がある。採用の外的妥当性はこの合成fixtureに限られる。

Gate1 GOならholdout24callだけ直接Groupingを追加する。比較するpurpose規則は各arm本文のまま、全条件共通のfile-to-group出力指示を追加する。schema完全割当100%、exact>=80%、FM=FS=0、unresolved/fallback=0、singleton逃げ0、同資源基準。Phase1の説明精度からGrouping精度を推定しない。最大生成はPhase1 36+Phase2 24=60、条件順はfixture index+order indexでABCをrotate。Gate1 NO-GOならPhase2未評価で研究終了。S16は#169の16file×150変更行をtokenizationのみで再利用し、生成/quality分母から除外する。長期目標16file約80%の達成とは呼ばない。

オフライン評価は同じagentによる入力/gold照合であり独立人手評定ではない。Prompt盲検化はしない。Phase1結果を見てgold/閾値/promptを変更しない。既知/holdout実行中は他のmodel/compile/test処理を並行起動しない。

初回事前登録の過去source列挙が0件となったため、生成0回のまま#168の明示pathからsourceを読んで完全一致なしを確認した。#167はGo generatorなのでexpanded sourceの完全一致は未検査であり、新しいtarget symbol不在の確認だけに限定する。初版hashはpreregistration-v1.jsonに保存する。prompt/gold/Gate/request bytesは変更しない。
