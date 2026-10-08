## 要約
Iteration 17ではGo解析を追加せず、別repositoryの公開3履歴を内容確認前に固定して入力監査した。2件はunit budget拒否、1件は観測anchor0であり、中心仮説のpositive/negative/unknown比較セットは成立しなかった。現候補NO-GOを維持。モデルcall0。

## 検証結果
- 専用worktree: issue-155-central-gate、branch: codex/issue-155-central-gate。
- 事前選択commit bab6d4dc。google/uuidのuuid_test.go最新6件のmetadataから、過去にsource監査したcommit/parent3件を除き先頭3件を選択。検索対象で未使用だったが、repository全体の非使用や正式holdoutを証明したものではない。
- 0e97ed3b: uuid.goとuuid_test.go、版別source計66701bytes。既存prepareがunit_budgetで拒否。
- 16939daf: テスト追加のみ、source計45971bytes。32units、anchor0、Must/NewV7/SetRandのsource欠落。構造入力11905bytesだが意味goldの識別可能性を満たさない。
- a2b2b323: version7.goとuuid_test.go、source計51309bytes。unit_budget拒否。
- 公開コードは実行しない。Go抽出器・予算・既存promptは変更しない。モデルcall0、semantic品質は未計測。

## 考察
既存attribution promptは具体的観測への効果がない編集をindependentと扱う。このラベルだけでは、回帰テストを含むコミット目的の独立性を表さない。したがってその既存taskの再実行を#155 GO検証に転用できない。

今回の3履歴は必要な中心評価を提供しない。test-only履歴のanchor欠落をunknown成功と数えたり、拒否2件を除外して成功率を作ったりしない。diffの自然commitまとまりも独立目的goldの十分条件ではない。

## Next Steps
- 解析拡張、unit budget緩和、既存effect taskの再計測を行わない。現在の候補の能力証拠がないため。
- 再開には、既存抽出機能で扱える未使用のpositive/negative/shared-test/cross-boundary入力と、観測効果と目的独立性を区別したgold根拠・出力契約が必要。入力確保だけを繰り返す探索やGo対応追加を避け、この不足を現候補の停止理由とする。
- #155の限定GOを付けず、B/Cへ進まず、production4ファイル上限と旧#150未達を保持する。新しい責務仮説またはBの依存変更は別途判断が必要。
