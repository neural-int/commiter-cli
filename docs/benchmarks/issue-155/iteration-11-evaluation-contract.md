# 次のattribution評価契約（準備段階・推論未登録）

## 目的と責務
#155のpositive/negative/unknownとcross-boundary条件を保持する。LLMの責務は検証済みassertionに対する実装編集の効果仮説であり、戻り値の再計算、source列対応の創作、final groupやpair scoreではない。

## 入力で区別するもの
- hostが検証したsource binding: table行、入力式/列、期待値、call/result、predicate、到達条件、version、UTF-8 spanとsource hash。
- 未検証のcontrol binding: missing/ambiguousとして残す。根拠の一部を省略してcomplete anchorにしない。
- ChangeUnitと再構築manifest。symbol/callerだけの対応は因果関係や同一intentとしない。
- before/afterの実測値がある場合は別の観測factとして渡す。値が取得できないだけで、source assertion自体が存在しないとはしない。
- fixture名、目的label、gold relation/groupはモデル入力に含めない。

## 出力と検証の層
各評価対象unit/anchorについて、観測条件を変える/その観測に独立/unknownの主張、source根拠ref、unknown時の不足項目を返す。独立は具体的な観測条件に対する主張で、全入力や作者のcommit目的への独立と混同しない。

hostで検証するのはID・source位置・完全割当・unknown形状・矛盾・budgetである。source refが正しくても意味主張が正しいとは限らない。意味主張は独立評価goldとの照合で別に評価し、構造validを意味validにしない。runtimeでは未検証の意味主張をhard unionやGit mutationに使わない。

## 現時点の未達条件
- ParseBytesのerr/continueとmulti-result、ignoreのcolumn guardを含む7行の完全source bindingが未証明。
- 既存18行/6履歴はsource監査に使用済み。正式独立holdoutとして採用しない。
- 独立目的negative、共有test/calleeの反例、cross-boundary positive、testなし/観測不足、同一行の別intentを含む評価入力と識別可能なgoldを未確保。
- 未使用データ・閾値・call/token/time・順序・source/helper/model hashes・baseline比較の事前固定が未実施。

## 計測開始の条件
上記データとgold根拠を推論前に固定し、native grammar/ID contractをモデルcall0で監査する。固定local model/helperを使い、prompt/閾値/model sweep、retry/repairによるgate緩和を行わない。baselineとattributionが異なる出力taskであること、単独attribution指標とpartition exact/FM/FSを分けることを明記する。今回の文書は計測の事前登録ではなく、その成立に必要な条件である。

## A/B/C境界
ここでは固定sourceの観測条件とunit効果を評価する。部分適用候補を生成・実行して誤候補を修正するB、一般的なprogram slicingを導入するC、metadata/final commit planは開始しない。source binding自体の成功を#155のGOとしない。
