# Adaptive plannerの検証契約と責務境界

この文書はproduction導入の仕様確定ではなく、#156検証候補の成立範囲を記録する。結果の正本は各iterationの事前登録JSON・raw結果・Issue報告である。

|段階|責務|観測済み境界|採用判断|
|---|---|---|---|
|A / A2|source snapshotから変更所有権を確定し、unknownをfile blockへ収束|line/inlineは編集操作であり目的ではない。A2はline所有権を先に検証し、必要時だけ元の256leaf上限内でrefineする|構造的限定GO。意味品質のGOではない|
|B|最大8個のsoft groupを順位づけ、全memberのcoverageとmarginを確認|3というscoreだけではmergeしない。partial/mixed/unknown・同率競合はfallback。推移閉包なし|順序依存とFS残存でNO-GO。閾値変更なし|
|C|mixed推定、bounded refinement、全CUの目的帰属、部分stage|混在推定とstage可能性は独立。goldは評価器専用。モデルがmixedとしたfileだけrefine|現行検証器は同じfileの複数commit割当を拒否。raw splitをproduction計画へ偽装しない|
|D|成立段階だけを固定して独立入力で比較|A2を最終候補。B/Cの失敗を無条件統合しない。coverageとexactを分ける|80%達否はiteration-7結果で決定。production変更は別承認が必要|

## 所有権と安全性

selected filesは1〜16、regular mode100644/100755のみ。file IDとpathの一意性、相対path、選択範囲、byte source spans、before payload hash、欠落/重複のない割当、空集合と全集合の再構築を確認する。ファイル名やsymbolを意味goldに使わない。

Aの元inline抽出器は1MiB/20kline、256CU/file、4096inline bytes、4096total CU。A2は同じ行抽出器の256operation/fileを先に適用する。inline refinementも元の上限を維持し、拒否時は既に検証済みcoarse所有権を保持する。行数/byte上限まで満たせない場合は明示的に停止する。今回whole-file巨大blobへ無制限退避する実装は追加していない。

temp-index replayは使い捨てGit repositoryと独立GIT_INDEX_FILEだけを使う。各commitのsource subsetから目標treeを作り、差分をcached applyして中間treeと最終treeを比較する。正順・逆順の両方を監査する。元のworktree/indexへの変更や、公開repoのコード・test実行は行わない。途中のsourceがcompile/test可能であるという主張は含まない。

rename、symlink、submodule、mode-only、競合は未サポート。構造不正をsemantic unknownとして継続させない。モデルの非completed応答や不正なdraftは採用せず、事前に定義したA2 fallbackを新たに検証する。最終fallbackそのものがinvalidなら停止する。

## 意味分類とmetadata

`single`・`mixed`・`unknown`はモデルの仮説。モデルの出力を意図の事実とみなさず、独立goldとの不一致を測定する。unknownの場合は同じfileの変更を脱落させず、`purpose_status=unknown`、`mixed_intent_risk=unresolved`、`fallback=true`とする。通常時の確認をユーザーへ追加しない。

A2のsummaryは「目的未確定の変更をファイル単位で保存」。通常の単一目的だという表示を避ける。意味が観測不能なだけで`compose`を付けない。confirmed mixedを分離不能なときだけcompose候補という契約は維持するが、現行経路で実行できる能力はNO-GOである。混合目的のbullet本文や目的名をモデルから得ていない場合は創作しない。

互換性の観測は以下の通り。

- `planning.allowedTypes`はcomposeを含まず、実検証で`invalid_type`となる。
- `planning.completeAssignment`は同一file IDの重複割当を拒否し、`NewConstraints`も「do not split a file」と明示する。raw C splitは`invalid_assignment`となる。CU IDをfile IDと偽って検証を迂回しない。
- PR PolicyのConventional Commit regexにもcomposeは含まれない。独自typeのPRタイトルは受理されない。
- commitlint設定は見つかっていない。release notesはPR分類に依存するため、未知typeの互換性が成立したとは扱わない。

将来の実装候補には、元file IDとCU ownershipを明示するplan schema、authoritative validation、repo-scoped Git safety、metadata/type・changelog・releaseの一貫した契約が必要になる。このIssue内ではこれらのproduction変更を行わない。rawの意味分割が正解でも互換性がなければ最終exactの成功には算入しない。

## 評価上の制限

使用済み#151 fixtureやiteration-4の再診断は未知holdoutではない。公開履歴の同じcomponent/projectionが複数ケースに含まれる場合、その相関を記録する。独立作者の母集団統計に読み替えない。公開履歴の重ね合わせは自然な同時変更ではなく、選択diffのcontrolled superpositionである。

goldの識別不能・multiple-validと、モデル失敗・資源停止を別記録にする。前者は事前監査の別集計、後者はprimaryに含まれる場合にexact失敗として数える。pair数は行operation/UTF8 atom/fileの粒度を明示し、異なる粒度の合計を同じFM/FSとして比較しない。token nullを0という観測値に変換しない。監査wallはruntime latencyではない。
