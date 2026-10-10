# Issue #165：事実抽出・関係解釈・境界判断の探索的診断

## 要約

固定Qwen2.5-Coder-3B 4bitと既知8case、2提示順、A/B1/B2/C1/C2の80生成callを実行した。事実抽出の欠落・誤記、関係分類と方向の誤り、最終境界の誤統合を、独立した出力上の失敗として確認した。事実抽出だけが誤統合の原因だとは識別できない。監査済み事実を追加してもBの関係labelとCの境界は変わらず、Bは全32応答がcompensation、Cは全32応答がmergeだった。内部推論の失敗箇所やpolicyによるdefer増加の原因は未確定である。

**人間による基準事実レビューは未確認。** 生成前の19事実と生成後の意味照合はCodexが手作業でsourceと照合したもので、独立した人間による監査とは呼ばない。Issueの「人手で監査した検証可能な事実と照合」に対応する確認を依頼済み。以下の意味評価は固定されたagent監査基準による暫定診断であり、Issue/Goal全完了を主張しない。基準/gold/promptは結果を見て変更していない。

## 検証結果

### 実験の固定と再現

- 参照revision `f8d21f937f550acae1f6f334a547bce9fca99649`、Qwen pin `3dd939c621c08e5753d5b89f35a2642cd83b98ca`、4bit/group64、旧helper/Metal/AST/validatorを再利用した。モデル9file、Swift source/Package/tests、Python/Go sourceのdigestは事前・事後で一致した。
- 事前登録commit `895381a1`、取得goal原文のJSON化 `2bc44904`、remote一致証跡 `fbccd541` をpushしてから生成した。remote manifestのbyte一致と生成開始時のclean worktree/remote head一致を確認した。goal原文Markdownの行末空白による初回diff失敗を保持し、実験入力を変えず保存形式のみ修正した。
- 80 load/tokenize-only callで生成0を確認した。native入力508〜1409tokens、context16384/output1536、temperature0/top_p1/top_k0/seed144/native thought0、neutral grammar、各call30秒、最大80生成/retry0。全80生成で事前token数と一致した。
- C1のsystem/user/schemaは#164 baselineとbyte一致。A/B1は同じraw source・before/after・unchanged context・AST factsと行番号表。B2/C2に追加する事実リストは同一で、関係label、must_merge、commit境界、test実行結果を含まない。A/Bの生成回答は後続条件へ渡さない。
- 全80backend完了・JSON/schema形状正常、timeout/invalid JSON/invalid schema/total budget拒否/retry/fallback0。Bのdirection規約と引用の意味は別評価であり、shape validityをsemantic validityとは呼ばない。

### A：事実抽出とAST baseline

生成された35claimを、事前登録19事実×2提示順の38項目と照合した。`fact-review.json` は各F、対応claim番号、判定理由、余剰claimを保存する。exact quoteは修復・空白正規化しない。

| 登録事実の判定 | 件数 / 38 | 意味 |
|---|---:|---|
| present_correct | 8 | 必須before/after意味を完全に記述。引用の正確性とは別 |
| partial | 19 | after等の内容は正しいがbeforeとの差分や契約が不足 |
| missing | 7 | 必須factに対応するclaimなし |
| wrong | 4 | nil/assertionの失敗条件、boolean極性/戻り値の誤記 |

余剰の誤りは2claim（Containsのpathをfixture.goと記述、既存Unchanged関数を追加と記述）。正確だが不変部分に関する余剰claim2件も保存した。partialは完全正解へ加えず、誤抽出とも区別する。

引用41fragment中23がsource ID/version/line/substringと一致。35claim中19はそのclaimの全引用が実在し、16応答中6は全引用が実在した。present_correct 8項目のうち5は対応claimの引用もすべて実在する。Admission逆順のCPU 2→3/Memory 4→6というclaim自体は正しいが、EA/EB引用IDが逆転している。引用が実在してもclaimの意味が正しいとは限らない。

代表的な誤り：nil逆順のTestNilについて「nilなら失敗」と記述したが、afterはnon-nilなら失敗する。boolean通常順はboolを返すDecodeFlagを「0を返す」と記述。逆順はEncodeFlagのafter極性をbeforeのtrue→1/false→0と取り違え、DecodeFlagもint恒等写像のように誤記した。新API通常順では既存Unchangedを追加と記述し、逆順では新RoundToTenの抽出自体が欠けた。

明示されたBound本体/codec roundtrip/Admission外部不足は各2応答で欠落した。新providerの追加は逆順で欠落した。Admissionのmissing_informationは両順とも空だった。TTL両順は全Fを完全に記述し引用も実在した。shared helper逆順は両変更のbefore/afterを完全に記述したが、不変Bound本体が不足した。

旧ASTは8caseでpackage/関数定義/裸identifier callを再抽出し、旧factsとbyte等価なデータを得た。新RoundToTenのprovider EB→consumer EAは機械baselineで回収できる。selector calls、literal/operator、assertion条件、roundtrip意味は既存抽出範囲外であり、constant-only TTLは関数definition/callが空。Aの値/極性/assertionの内容をこれらAST factsの反復と混同しない。ただしAのafterの一部読解や8項目の完全記述を、関係/境界の正解へ読み替えない。

### B：関係分類と根拠

| 指標 | B1 原source | B2 ＋監査済み事実 |
|---|---:|---:|
| JSON/schema形状正常 | 16/16 | 16/16 |
| 関係label一致 | 2/16 | 2/16 |
| direction規約一致 | 2/16 | 2/16 |
| labelとdirectionの両方一致 | 0/16 | 0/16 |
| 全引用の実在 | 7/16 | 13/16 |
| label＋directionの提示順一致case | 6/8 | 4/8 |

全32応答の関係labelはcompensation。label一致はboolean codecの2応答/条件だけだった。direction一致は新APIのEB→EAの2応答/条件だけで、両指標の正解行は重ならない。非dependencyではprovider/consumer=noneを要求したが全応答でconsumerをEAまたはEBにし、nominal labelが合っているcodecでも規約違反がある。

全reasonは明示的共同契約の維持を主張した。codecではsourceに両boolean roundtrip契約があり、full sourceに照らした補償理由は支持される。ただし実際の引用はEncodeFlag/DecodeFlagだけで、契約や両側極性の十分な引用を保証しない。他のケースのreasonは片側の値変更等を共同契約の根拠としており、その明示契約はsourceで確認できない。

B2によって引用実在性は7→13応答へ変わったが、関係labelは全16対応行で不変、direction変更は4行。引用の改善は意味関係の改善を証明しない。Bの6分類結果はCの入力へ渡しておらず、この診断はproductionの推論stage追加ではない。

### C：最終境界判断

| 指標 | C1 原source | C2 ＋監査済み事実 |
|---|---:|---:|
| JSON/schema/source ID正常 | 16/16 | 16/16 |
| 許容判断 | 8/16 | 8/16 |
| 独立誤統合 | 6/6 | 6/6 |
| 独立の正しい分離 | 0/6 | 0/6 |
| 必要merge確定 | 4/4 | 4/4 |
| 必要対応のkeep_separate誤分割 | 0/4 | 0/4 |
| 外部不足の必要defer | 0/2 | 0/2 |
| 判定可能行の不適切defer | 0/14 | 0/14 |
| 判定可能行の確定coverage | 14/14 | 14/14 |
| 提示順一致 | 8/8 | 8/8 |

全32境界はmerge。C1の16回答は#164 historical baselineと回答自体が一致し、C2も16/16で同じ。許容8は対応/補償4、依存2、複数許容2。独立3case×2orderと外部不足case×2orderが不許容である。確定coverage14/14は誤統合6行を含み、精度成功ではない。全確定案のauthoritative validator/完全割当/byte/tree/依存順apply/revert検査を実行・再確認した。分離案0なので正しい独立分離や単独revert能力の獲得を主張しない。

### case別の証拠

| case | A通常 / 逆順（F順） | B1/B2 | C1/C2 | #164 policy 通常/逆順 |
|---|---|---|---|---|
| nil/assertion | partial,partial / partial,wrong | 全compensation。規範correspondingと不一致 | 全merge、許容 | defer/defer |
| shared Bound | partial,partial,missing / correct,correct,missing | 全compensation。独立性を確定できず | 全merge、不許容 | defer/merge |
| time/permission | partial,partial / partial,partial | 全compensation、共同契約のsourceなし | 全merge、不許容 | defer/merge |
| search/sort | partial,partial / partial,partial | 全compensation、共同契約のsourceなし | 全merge、不許容 | defer/defer |
| boolean codec | partial,wrong,missing / wrong,wrong,missing | labelは一致、非dependencyのdirection違反 | 全merge、許容 | defer/defer |
| new API EB→EA | partial,partial / partial,missing | directionは一致、labelはcompensation | 全merge、許容 | defer/merge |
| TTL | correct,correct / correct,correct | 全compensation。明示共同契約を捏造 | 全merge、複数許容 | defer/merge |
| Admission | partial,partial,missing / correct,correct,missing | 全compensation。外部不足を識別せず | 全merge、不許容 | merge/merge |

correctはpresent_correctの略。完全claimでも引用ID/空白/引用符が不正な場合があり、詳細はfact-review/case-evidence/rawを参照する。policyは過去結果の再集計で、今回再生成していない。過去10件の不適切defer、必要merge確定0/4、期待defer0/2を保持し、今回baseline条件のA/Bからpolicy固有の原因を断定しない。

### 費用・資源・安全

記録された準備39.746秒＋生成/構造監査688.421秒＝728.167秒、post再集計/確定案再監査6.122秒。準備timerの終端後にmanifest保存/既存preflight、run timerの終端後に最終identity hash確認があるため、728.167秒を全作業の厳密なwall timeとは扱わない。準備/生成80callは3600秒の実験上限内で、post監査400秒の予約内。API/model料金0、download0bytes、新依存0。電気料金は未計測。

| 条件 | input / output tokens | TTFT中央値 秒 | process wall中央値 秒 | 採取footprint最大 bytes |
|---|---:|---:|---:|---:|
| A | 15756 / 3041 | 6.992 | 14.295 | 2817689880 |
| B1 | 16044 / 1681 | 6.421 | 10.397 | 2787461424 |
| B2 | 18694 / 1717 | 6.955 | 11.455 | 2865973552 |
| C1 | 9676 / 208 | 2.892 | 3.315 | 2608400616 |
| C2 | 12326 / 208 | 3.086 | 3.534 | 2507180360 |

全input72496/output6855tokens。B2/C2は対応する原source条件より各2650input tokens増加した。B2は引用の実在性が変わったが意味label不変、C2は全境界不変。異なる質問/schemaのA/B/C間の費用を同一品質の効率比較へ使わない。

全80processの採取footprintは5GB以内、最大2,865,973,552bytes（約2.87GB）。しかし**全80callでpressure level2が観測され、正常pressure条件は0/80**。全採取levelは1が2回、2が823回。3callにsystem swap増加を観測した。実験前のglobal swap usedは3805.25M、終了後3872.62M。これはmodel process単独のswap使用量ではなく、他の開発アプリも共存したsystem観測である。増分は丸められたvm.swapusage表示からの概算。モデル単独の原因や各段階の因果効果へ帰属しない。良好な資源資格を主張しない。

100ms samplingとlifetime peakの終了直前未観測、OS cache、共存負荷を限界として保存。RSS最大1,945,321,472、MLX peak最大2,321,187,812bytesを別列に保持し、footprintへ加算しない。独立Metal allocationはnull。helperごとにfresh processでexit0を確認した。network拒否sandboxとoffline envで全helper/自己作成fixtureを実行し、クラウド推論/外部diff送信/外部repository code/test/新モデル取得0。

## 考察

`fact-missing` はbeforeとの差分やsource-visible契約/外部不足の欠落で支持される。`fact-wrong` はnil assertionの逆転、boolとintの混同、極性の誤読、既存関数追加の誤記で支持される。ただしAの事実8項目は完全に記述され、partial19項目もafter等を部分的に認識している。モデルがsourceを一切認識できないという仮説は支持しない。

`relation-wrong` は独立変更/対応実装/assertion/API依存/複数許容/外部不足を全compensationへ分類したこととdirection規約違反で支持される。B2でもlabel不変であり、監査済みfact追加だけで関係解釈が成立する仮説はこの既知caseで支持されない。TTLはAが完全・引用正常でもBが誤分類し、新API方向はASTでも抽出できるがBのlabelに反映されない。これらは別質問の出力上の不一致で、同一推論の内部因果鎖を観測した証拠ではない。

`boundary-wrong` は独立6/6のmergeと外部不足2/2のmergeで支持される。C2も不変なので、sourceにあるfactを重複した自然文で追加すれば誤統合/必要deferが直る仮説は支持されない。一方、nil/codec/API/TTLのCは許容であり、A/Bの誤りが常にCの誤りへつながるとも言えない。全mergeという一律の出力だけで成功している正のcaseもあるため、必要merge4/4から適切な意味理解を主張しない。

`unsupported-or-ambiguous` は引用ID/line/quoteの不一致、sourceにない共同契約のreason、TTLの複数許容規範、Admissionの外部制約不足として保存する。TTLの規範は唯一の作者関係を正解とするものではない。Admissionは入力外のclient契約が欠けるので、その作者境界や真の補償関係をモデル能力不足のgoldへ転換しない。

`undetermined` は全caseで内部原因について残る。Bは補償labelへの偏りと不要なprovider/consumer値を返しており、schema/質問文の表面効果、decoderや学習済み形式への偏り、source意味の混同を本対照では分離できない。Oracle factsはnorm labelを含まずsourceへ結びつくが、言語化した内容自体が答えの手掛かりになり、情報量/長さ/format/salience対照もない。変化がない結果も、fact利用の有無や一意の失敗段階を証明しない。

#164 policyの不適切deferは現C1/C2では再現しておらず、A/Bにpolicy条件を与えていない。したがってpolicyで保留が増えた内部原因は未識別。今回のbaseline診断とhistorical policyを同一条件の因果比較へ混ぜない。さらに基準事実の独立人間レビューが未確認で、agent監査の誤り/解釈の余地は残る。この未確認事項を独立holdoutや精度改善の証明へ隠さない。

## Next Steps

- 人間がsource-auditの19事実を確認する。Issue指定の人手監査をagent作業で代替したとは主張できないため。修正が必要なら元の登録とrawを保持し、影響を再分類して、事後gold変更による成功を作らない。
- 本診断の追加生成は行わず、human監査成立後に全完了条件/チェックを再照合しgoal.mdの最終品質ゲートを実施する。既知8caseの固定80callで出力上の支持/反証/未識別は保存できており、同じ入力の反復で内部原因を識別できる設計ではないため。
- 最小の次仮説は「関係label/不要direction欄の形式効果か、関係内容の解釈失敗か」。別Issueで未使用の独立正負/補償/API/外部不足caseを事前登録し、label順またはdirection欄だけの単一要因対照を検討する。現Bの一律compensation/非dependency direction違反と引用改善のみの結果を切り分けるため。今回実行しない。採用/本番資格を主張するなら独立repo/familyの未使用評価は別途必要。
- 必要なら長さ/formatを合わせたfact対照や、固定policy自体のA/B/C診断を別の事前登録で検討する。今回のC2不変だけではfact認識/情報量の因果識別もhistorical不適切deferの原因識別もできないため。新モデルsweep/prompt調整/production stage追加へ直結させない。

## 保存した成果物と品質確認

`contract.md` / `preregistered.json` / `source-audit.json` / `machine-baseline.json` / `native-token-audit.json` / `input-contract-audit.json` / `remote-registration.json` が生成前契約。`results.jsonl` は全raw/生成文字列/resource、`summary.json` は機械集計、`fact-review.json` はCodexによる意味照合、`case-evidence.json` は履歴と支持/反証/未識別、`result-audit.json` は独立再集計・fresh plan再検査、`resource-audit.json` は資源/費用の別判定。

原goal.md全文はbyte SHA付き `goal-source.json`、Issue開始時の本文/チェックは `issue-source.json` に保持。raw引用やsourceを変更せず、再現時は保存済み応答を `audit_results.py` で検査できる。生成の再現は固定同digest helper/model/AST/validatorを用い `diagnose.py prepare`、生成前commit/push/remote登録後に `run`。exclusive-createで既存rawを上書きせず、別の専用worktree/登録を使う。全モデル出力や実装から内部推論を復元するとは主張しない。

targeted checksは構造拒否27件、入力差/引用/schema拒否5件、全80応答照合、8確定案のauthoritative/完全割当/再構築/ordered fixture tests、resource計測器lifecycle、Python構文、diff検査がPASS。研究専用docs/tools以外の変更0、main HEADと既存 `.gitignore` 変更を保持、production/default/4file/staging/metadata/依存変更0。unit tests追加0、機能更新/削除なしなので既存テストの削除は不要。

Issue成果条件の品質証拠としてGo test 20package、Go vet/build、CLI build、release-notes Python tests 25件、固定helperのSwift Testing 32件（8+24）がすべてPASSした。結果は `preliminary-quality-gates.json` に保存し、goal.mdの最終品質ゲートとは区別する。人間レビューと全Issueチェックの成立前にGoal完了や最終Gate実施済みとは報告しない。
