## 要約
Iteration 7の別repository監査でも現host-fact方式は自然履歴を扱えなかった。iancoleman/strcaseの固定3履歴、9追加・変更table行について自然callが取得できた11状態はknown0/unknown11。変更前対応行なし7は別区分。前回と合わせ2repository・6履歴で自然call29状態のknown0。現host-fact候補の追加モデル計測を終了し、#155の独立能力GOは未証明のまま保持する。

## 検証結果
- 新専用worktree/branch: codex/issue-155-second-corpus-audit。
- snake.go履歴6件からsnake_test.goも変更する先頭3件をsource確認前に選択。固定規則・全6SHAを保存し成功で再選択しない。
- 192a02172ed4cef346e7f4d476182ea129101b97: 2追加/変更row、natural call3状態unknown、before対応行なし1、unit_budget拒否。
- 6fab6530048ce8268098c5ef39c193028c5947ec: 6追加row、after call6状態unknown、before対応行なし6、attribution入力受理。
- a6b8dcde35569b34185fc011be4f0f41cf393f3d: 1expected変更row、before/after call2状態unknown、attribution入力受理。
- before/afterでignore引数の型がuint8からstringへ変わる履歴は、実testの版ごとの変換を反映。after stringをbefore byte APIへ機械的に渡す比較にはしていない。
- ソースはsnake.go/testと直接参照するacronyms.goに限定。全repositoryの再現ではない。他の変更ファイルもmanifestに記録し、完全commit観測と主張しない。
- 元table行のbyte span/textを検証。known0、意味gold/attribution/partitionはnull。変更前の自然table行欠落をknownやunknownの値推定へ置き換えない。
- Python host検証3tests pass、モデルcall0、任意source実行0、新依存0。source、hash、入力、版別state、rejectを保存。

## 考察
現observerはscalar/限定intrinsicに閉じている一方、自然な文字列変換はloop、index、global map等を含む。bytes履歴のuint64/複数return/formatだけの局所問題ではない。2repositoryの固定小標本でknownがないことは観測coverage不足の証拠であり、全repositoryの不可能性やLLM一般の能力限界の証明ではない。

現在の前提でモデル計測を追加しても、hostが検証できるpositive/negativeの能力gateを満たせない。成功する人工例を追加すること、observer対応機能だけを自然holdoutから選ぶこと、unknownを正解対応として数えることはしない。また、source interpreterの拡張だけでは変更目的への意味判断を解決したことにならない。

## Next Steps
- 現host-fact候補の推論を終了し、再開条件を確定する。必要なのは型・複数return・loop/global状態等を含む自然sourceのauthoritative観測、変更assertionとの対応、独立目的negative/cross-directory positiveの評価入力である。
- その能力を新たに実装する前に、#155 Aの最小能力検証として必要な範囲と費用、#154 B/Cへ越境しない境界を明記する。条件がないまま汎用Go interpreter/実行sandboxを開発しない。
- 別の判断責務仮説がなければ、同じfact taskのprompt/model/fixture追加は行わずno-candidateと再開前提を保持する。B/C、production、旧#150完了は未達として維持する。
