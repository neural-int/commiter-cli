# Context-only attribution ablation

既存7caseの入力/goldと旧A-only/impact結果を保持する。追加7callだけでcontext-onlyを測る。impactと同じfield名・semantics metadata・未変更sourceを保持し、observationsを空にする。この条件差でcounterfactual観測の増分を区別する。system/schema/helper/model/context16384/output1536/120secを変更しない。新規probeを実行しない。

これは使用済み入力の要因分離であり、B Goの未使用独立検証とは呼ばない。各caseのexact/FM/FS/complete/token/wallを比較する。FS回帰を隠さず、旧結果/goldを変更しない。context-onlyは純粋なsourceだけでなく同一soft-evidence metadataも保持する条件である。
