## 要約
Iteration 4の局所fact抽出候補は診断NO-GO。6回すべてbackend completedだが、値・return根拠・unknownのhost検証を通過したケースは0/6。固定モデルに戻り値計算と証拠不足判定まで任せる方式は、この最小probeでも成立しなかった。#155全体の限定能力GOは未証明。B/Cとproductionには進まない。

## 検証結果
- 事前固定commit: 63c4120452eeb53011f2acd2d1a5343a49316120。専用worktree/branch: codex/issue-155-local-observation。
- 同一ローカルQwen3-8B、neutral grammar、6calls、retry0/repair0。wire監査は合法12/12、不合法6/6拒否、合法penalty0、全promptのtoken上限適合。
- Fee(101): afterの正解2に対し1。shared Left(2): 正解4と別関数のreturnに対し6とその参照。Count(3): 正解3に対し0と負数分岐のreturn。
- 観測入力なし: unknownが必要なのにbefore0/after1と断定。Delegateの定義欠落: unknownが必要なのにbefore3/after4と断定。
- 公開google/uuid.Compareにもvalue_mismatch。これは実sourceへの人工入力probeであり自然な変更目的holdoutではない。
- 拒否した6回答のqualityは全てnull。attribution precision/recall、最終partition exact/FM/FSの成功値を捏造しない。
- calls=6, backend completed=6, host valid=0, 合計wall=140.689秒、input tokens=9459、output tokens=988。
- Go/Pythonの限定observerとhost検証テスト、native wire replay通過。任意repository codeは実行せず、依存・モデル追加なし。

## 考察
参照IDを含む構造的に完了した回答でも、算術・分岐・対象関数を誤り、unknownを具体値で埋めた。したがってschemaやcitation形式だけの修正では解消を示せない。今回の結果は全LLMの能力限界ではなく、固定candidateと入力条件の反証である。

hostが既に安全に算出できる値をLLMに再計算させる処理は、今回の観測では誤差と追加costを増やす。次の設計では既存限定observerの確定factをauthoritativeとし、LLMが追加すべき意味判断を限定する必要がある。ただしfact一致や同一testだけでは同一intentを証明できず、hard unionにはできない。公開sourceの人工入力成功/失敗は独立意味品質の評価を代替しない。

## Next Steps
- 今回のcandidateを保存して終了し、同じ値計算taskのprompt/model sweepは行わない。局所化しても基本的な値とunknownの失敗が残ったため。
- 次のモデル計測より先に、hostのみで得られる確定factとChangeUnit/source spanの対応を監査し、LLMが必要な残余判断を明文化する。既存observerの再実装や値の再計算を研究成果としないため。
- 観測可能なpositive/negative/unknownを持つ未使用実履歴を確保し、host-only対照で解ける範囲と解けない範囲を分離する。その入力と事前gateが成立するまで、独立能力GO・B/C開始・production拡張を判断しない。
