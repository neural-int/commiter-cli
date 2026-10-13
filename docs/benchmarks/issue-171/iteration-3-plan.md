# Iteration 3 実行方針

ユーザーの明示指示に従い、システム全体のswap前後差を停止条件から除外する。commiter/helperのRSSとMLX peak allocationを観測する。システムswapはIssue上の観測項目としてraw記録に残すが、モデル単独のメモリ消費・意味品質やこのrunの受理条件には使わない。旧run1/run2の結果・停止・旧契約は変更しない。

新run3は同じ96入力、Gold、共通schema、4条件、通常/逆順、helper/checkpoint、seed/temperature/decoder/context/output、timeout120秒、retry0、品質Exact/FM/FS/順序/完全割当Gateを維持する。新model・新Planner・追加stage・新依存はない。停止はtimeout、missing/critical pressure>=4、helper RSS>8GiB、MLX allocation>8GiB、Git/hash driftとする。モデルprocess memoryの観測欠落もfail closed。RSSとMLX peakは異なる量として報告する。

この同一比較matrixのrun3は最大96callで終了し、追加のrun4やprompt/Gold/threshold調整は自動実施しない。run1/run2は各1call、run3最大96callで全体最大98call。既知F01（A通常順を2回観測）は診断として明記し、未生成confirmed F02〜F11は別集計を示す。全confirmed11の固定品質Gateを変更しない。F12は事前未確定として主評価から除外する。

helper/validator/model/check_modelは再利用し、resource stopping/acceptanceの判定だけを別研究runner/evaluatorに分離する。旧runner/evaluatorは凍結hashどおり保持。新判定は実run2の+4183.5MiB swapだけでは拒否せず、helper allocation上限・pressure・timeoutでは拒否する回帰テストで確認する。新実験前にrequest/Gold旧pinと新code/契約をcommit/pushし、Issueへ全文方針を報告する。
