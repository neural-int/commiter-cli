## 要約

固定した根拠付きattribution候補は診断NO-GO。8callすべてbackendはcompletedだが、attribution4件は全件host reject。引用不足だけでなく、共有testの独立条件を全unitへ結び付ける、観測される入力では振る舞いが変わらない実装をpositiveにする、観測なしでunknownを返さない誤りがraw出力にあった。拒否出力のqualityはnullで保持し、正しい対応やFM/FS=0として集計していない。

これは使用済み診断3件と開発用1件の固定candidate不採用であり、あらゆるattribution方式の不可能性、専用モデルの必要性、#155の独立能力GO、旧#150の達成を意味しない。B/Cとproductionへ進めない。

## 検証結果

事前条件はcommit `1a2ea8d3`で保存し、#155 comment 6060294286 / #154 comment 6060294799へ測定前に掲載した。全4caseの両armでpayload hashは同一。model/helper/profile/予算は不変、retry0、repair0。Qwen3-8B pin、neutral grammar、temperature0/topP1/topK0/seed144、16K context・output1536、120s/call。全callは8〜44秒でcompleted、timeout/invalid JSON/output budget超過は0。

| ケース | attributionのauthoritative判定 | direct grouping（atom pair） |
|---|---|---|
| Fee ceiling / Wrap delimiter | missing_observation_evidence、quality=null | FM0 / FS9、exact=false |
| shared testのLeft / Right | missing_observation_evidence、quality=null | FM4 / FS0、exact=false |
| Countの負入力処理とdiagnostic変更 | missing_observation_evidence、quality=null | FM0 / FS0、exact=true |
| testなしValue変更 | missing_contract、quality=null | FM0 / FS0、exact=true |

attribution: 4call、backend complete4、host-valid0、wall計129.795s、input8787/output1033tokens。
direct: 4call、backend complete4、host-valid4、atom partition exact2/4、FM4/FS9、wall計85.470s、input7711/output414tokens。
両者は異なる出力task。attributionのrelation精度とdirectのpartition exact/FM/FSを同じ指標として比較しない。attributionのprecision/recall、unknown成功率はaccepted outputがないためnull。

拒否されたraw出力の観測（採用可能なrelationの集計ではない）：
- Fee/WrapではU010をimplementsとassertsの両行に重複して含め、実装行にもtest unitを含めた。authoritative validatorは先に引用不足で拒否したため、rawの重複観測を別に記録する。
- shared testでは、Leftの実装、Rightの実装、各assertionの全4unitにC001とC002の両方を付けた。コード上の各条件はLeft(2)とRight(2)で別である。
- Countでは、負入力へのbranch追加の19unitをCount(3)!=3のC001に関連付けた。提供コードはbefore/afterともCount(3)=3であり、この観測はそのbranch変更で変わらない。message変更20unitもassertsに分類した。
- testなしではcontracts=[]にもかかわらずimplements_changed_behavior/direct_source_evidenceを返した。unknownは返していない。

grammar/tokenizer監査ではモデルweightsをロードせず、合法16出力許可・不正8出力拒否・合法出力penalty0、全prompt568〜3131tokensがoutput1536込みで16K内であることを確認した。意味的に正しい出力のschema到達可能性と予算内のtoken列を事前に確認している。計測release helper SHAは不変。Go3テスト、Python5テスト、Go vet、Swiftのwireテスト1件、diff checkは成功。Swiftは既存tmp helperのtest/buildでありproduction helperの採用証明ではない。

独立データ候補としてgoogle/uuidの公開commitをsourceのみで監査した。外部コードは実行せず、モデルへの入力にもしていない。
- `e8d82d30a3eb641530570da83295395651911778`：Compare追加とmonotonicity test書き換え。変更if bodyはErrorfとbreakを含み、現observerの単独報告body対象外。変更条件anchor0、context7114bytes。
- `9ee7366e66c9ad96bab89139418a713dc584ae29`：Validate追加とtable/subtest。nested callbackが現observerの対象外でanchor0、context8057bytes。
既存benchmarkの変更test symbol検索は両候補とも0件だが、holdout認定はしていない。source SHA・commit/parent・検索範囲での結果はiteration-3-public-candidates.json。source観測未対応をsemantic independenceやpositive成功へ置き換えない。

## 考察

入力に根拠が存在し、合法出力がdecoderで到達可能でも、この固定global attribution出力は成立しなかった。source/testという種類の区別だけでは、具体的な観測条件にどの編集が影響するかを判断できていない。特にshared testとCount(3)は、sourceの意味に反する関連付けであり、引用IDや重複IDをhostで補正するだけでは解決しない。

第一失敗を全探索の終了とは扱わない。一方、promptの引用要求を強める、出力keyを変える、閾値を変えるだけの反復は今回の意味誤判定を説明する新仮説にならない。このglobal role/contract assignment candidateを同じ情報で再試行しない。

次に検証する合理性がある責務差は、任意のunit→contract関連を出させる前に、LLMが観測条件の具体的な入力についてbefore/afterの結果とsource上の分岐・return根拠を正しく取り出せるか、という局所的な事実判断である。例は「Count(3)は変更前後で何を返すか」。新しい仮説は引用の自動補完ではなく、意味判断をrelation選択から観測結果・source witnessの抽出へ変更する。hostによるsource mappingと関係の導出はgold非依存で設計し、test PASSや同時FAILを同一intentのmust-linkにしない。

この局所事実taskが成功しても最終commit境界の証明にはならない。また、既存#149 H5/H12の実行事実をglobal groupingへ再投入する方式との差分を事前に明記する必要がある。Aの独立能力はまだ未証明であり、部分適用によるplanner修正を行うBやProgram Slicing Cへは移行しない。

## Next Steps

- 現固定candidateをNO-GOとして保存し、wording/ID修復/model sweepを行わない。shared-test・Count・unknownに意味上の失敗があり、表面的な出力修正では原因を解消できないため。
- Aの次の最小仮説を「観測入力に対するbefore/after結果とsource witnessの局所抽出」として定義し、既存H5/H12および本candidateとの責務差、支持/反証条件を推論前に固定する。role関連の選択失敗と、sourceから具体的な事実を取り出す能力を切り分けるため。
- 局所事実のsource検証・不明時unknownを先に監査し、goldやfixture固有規則を持ち込まない。検証可能な根拠なしでmodelの説明を事実と認めないため。
- 独立実例のparser未対応と意味誤判定を別に記録する。外部2例のanchor0をモデル能力の失敗や成功へ数えず、Aの独立評価に必要なcoverageを確認するため。
