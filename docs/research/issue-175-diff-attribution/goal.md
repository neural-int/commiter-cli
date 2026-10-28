# H175 / H182 保存diff帰属PoC（初回）

## Goal

保存済みApp Server fileChangeのdiff取得と、パス一致より保守的な内容照合の可能性を検証。移動先 https://github.com/neural-int/commiter-cli/issues/182#issuecomment-6332126533 を参照。本番Collector/Planner/投影schemaは変更していない。

## Result

CLI 0.155.1。通常checkoutの保存履歴から完了fileChange26件・changes30件を読み、30件すべてdiffあり・非空。update20件は@@を含む形式、add10件はそれ以外（生内容の可能性を残し、形式不正とは断定しない）。4MiB上限で途中停止、受信上限検出値4,194,305bytes、約2.95秒。1thread/60turn/1,995itemsまでの便宜標本であり、全体の存在率ではない。専用worktreeの同じフィルタは0thread。unavailableとno matching threadsを区別。delete/renameの実例は今回未取得。5秒と4MiBを緩和せず、本文・パス・source IDは集計に保存しない。

## Deterministic PoC

15人工ケース。before/afterの方向・path・worktree・completed状態を照合し、候補が1Turnだけならunique_content_correspondence、複数ならambiguous、なしならunmatched。パス一致候補の誤候補数合計6、強い内容対応の誤候補数合計0、強い対応として取り逃がした参照Turn数合計8。これは合成データ内の集計で実履歴の精度ではない。ケース別はsynthetic-results.json。

A→B→Cの合成、同一行の再編集、renameの追跡で取り逃がし/曖昧が残る。内容一致でも独立した手動の同一編集を区別できず、因果を証明しない。PoCは正規化済みbefore/afterへの照合であり、実diffのparserや操作合成を実装していない。取消し・ログなし・失敗・別worktree・手動混入は強い帰属を捏造しない。時刻や同一セッションだけでは採用しない。

## Gate / Next

取得可能性は部分標本で成立。ただし実Git差分との結合、依頼/最終回答への安全な参照投影、実履歴での誤帰属/取り逃がし、8KiB投影は未評価。専用worktreeは着手時に既存の追跡済み変更がなく、今回の研究用未追跡ファイルを過去履歴に帰属させない。Gateは未達のため内部Adapter/3phase schemaを設計・採用しない。過去Goldやfixtureは未変更。Hooks/DB/常駐/新規依存/追加LLMなし。

次は同意のある実変更snapshotと対応Turnを固定し、addの実形式とupdateのparserを検証してから、逐次編集の合成と取消しを実履歴で評価する必要がある。今回の人工結果だけでパス一致を置き換えない。

## Verification / Safety

Python4テスト、go test ./internal/codexcontext ./internal/planning、同範囲go vet、diff check成功。既存fallbackテストは維持し、本PoCはPlannerへ未接続なので取得失敗時の本番挙動は変更なし。HEAD/index/追跡済みworktreeは回帰テスト前後でhash一致。取得前snapshotを取っていないため取得全体の不変証明とはしない。主checkoutの無関係な.gitignore変更を維持。ライブ取得全体のpeak memoryは未測定、人工照合のtracemallocのみ記録。probeはread APIだけを呼ぶがApp Server内部のstate/cache書込みがないとは保証しない。履歴内容の外部送信なし。公式仕様参照: https://learn.chatgpt.com/docs/app-server 。
