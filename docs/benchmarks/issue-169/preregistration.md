# 推論0回の事前登録

A/B/Cは同じGit変更、file ID集合、8共通fields、File順序、model/revision、system prompt、出力schema、seed、decoder、token/時間制限で比較する。差はAのstructural evidence、Bのraw diff、Cのraw diffに追加する所属宣言だけ。実requestとschemaはfrozen-inputs/とinput-schema-A/B/C.jsonに保存した。Aの9 Git fields/4位置fields除去は#168と同じで、内部原本はsource-mapping.jsonへ保持する。

最初の生成前に実入力・gold・抽出version・SHA256・入力bytes・実token数を固定する。tokenization-preflight.jsonlは同じnative chat processorの生成なし処理で得た観測であり、目的生成・Groupingの結果として数えない。計測helperとpreflight helperはPackage.resolvedが一致し、bounded-groupingのprocessor.prepareとthinking設定も一致する。実生成時のtoken数照合でこの同等性を確認する。

既知K01は#168 K1を再利用して実行確認に使う。H01〜H04は今回の新規合成変更で、具体的goldはfixtures.jsonに固定する。目的数は1/2/2/2、計7目的。K01既知3call、holdout4fixture×2順×3arm=24call、Phase 1計27call。Gate 1 GOなら同じholdoutで直接Grouping24call、最大51生成process。prompt調整、追加model、retry、repair、第二LLM検証は0回。各生成は120秒まで。既知結果のオフライン評価を記録後、prompt/gold/Gateを変更せずholdoutへ進む。

CはGoのBefore/After Tree-sitterでfunction/methodを対応付ける。変更runはcontext行で区切る。追加はAfter、削除はBefore、置換は双方の一意な所属宣言集合が一致した場合のみ採用する。1行に複数宣言、top-level、blank、解析失敗、不一致は省略する。宣言名は関数名、methodはpointer修飾を除いたreceiver型名+'.'+method名。重複を除きhunk_index/宣言名の順でsortし最大32entry/file、宣言名は128UTF-8 bytesまで。超過は省略して理由を記録し、raw diffを切らない。raw diffはGit HEADとworktreeの完全なU3差分、最大64KiB/file・128KiB全体。曖昧さとGit不一致は区別し、後者は停止する。

semantic coverageはgoldの具体的前後変更と対象を説明した場合のみ加点する。関数名や「処理を改善」だけはtarget-only。正しい推測でも入力に根拠がなければgrounding=false/hallucination=true。purpose exactは全goldと説明の一対一対応、余分/統合/分割なし、全説明に根拠あり。各説明の全文、gold対応、根拠と理由を保存する。オフラインagent評価であり独立人手評定ではない。purposeの過不足はfile partitionのFM/FSと区別する。

Gate 1絶対基準は未使用holdout coverage>=90%、exact>=80%、grounding>=95%、overmerge/oversplit/hallucination=0、schemaとsmall-input受入100%。BまたはCがこれと資源基準を満たし、Aに対するcoverageまたはexactが25pp以上改善、両指標を悪化させず、幻覚/過剰分割を増やさない場合のみGO。資源基準は全生成観測あり、wall p95<=120秒、helper RSS<=6GiB、前後swap増分<=256MiB、pressure<4、Git保持100%、実入力token数とpreflight一致。RSSはprocess peak、pressure/swapは前後値であり短いpeakや共存アプリの因果を分離しない。

Cの採用根拠にはBよりcovered goldを1以上、exact cellを1以上増やし、過剰統合/分割/幻覚を悪化させずgroundingを下げないことに加え、input tokens/wall p95がBの1.25倍以下を要求する。C自身の絶対Gate達成も必要。Bが絶対基準を満たしCの増分が確認できなければB優先。双方未達は双方NO-GO、Cのみ絶対達成でも増分不足なら保留。小標本の同等結果を効果不存在の証明にしない。

Gate 2は直接Groupingの完全割当/schema100%、exact>=80%、FM/FS=0、fallback/unresolved=0、同じ資源基準。AよりFMが減りFS/fallbackとexactを悪化させない候補だけ別Issue検討。CはさらにBよりFMを減らすかexactを1cell以上改善し、FS/fallbackを悪化させず費用1.25倍以内が必要。同一ファイル複数目的のH04はfileを分割せず、双方のtestを一つの不可分なfile groupにまとめる。Phase 1がNO-GOならPhase 2は未評価。

S16は同じGo変更16file×150変更行のtokenization-only stressで、入力枯渇とraw差分増量を観測する。S16の不達をsmall-input意味誤答へ換算せず、時間0や成功として数えない。16file採用の支持には使用しない。本研究のGOも16file約80%・production採用を意味しない。別作者/実履歴holdoutは今回含めず、適用範囲をこの合成変更へ限定する。

準備中の監査でraw diff全体のhunk件数メタデータが0になる不具合を再現した。header正規表現のmultiline指定欠落を修正し、回帰検査を追加した。AST解析とLLMに送るraw diffは変更せず、source-mappingの件数を完全差分に再照合した。事前登録前、目的生成0回の修正である。
