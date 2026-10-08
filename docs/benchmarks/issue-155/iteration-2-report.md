## 要約

#155のcontext選択方法を推論前に固定し、使用済み診断4件・開発用2件・実履歴候補2件で監査した。実履歴8092acd2は全snapshot約38.5KBから根拠付きcontext15402bytesになり、今回固定した16KiBのcontext上限内に収まった。c643154bは39409bytesで上限超過。上限に合わせた間引きや再試行は行っていない。モデルcallは0、attribution能力GOは未判定。

専用worktree `issue-155-context-holdout-audit`、branch `codex/issue-155-context-holdout-audit`、開始点 `af8b9c9553dd6638af1936a5c75179e821c7df3a`。

## 検証結果

選択規則は、構文上変更されたfunction/type/binding宣言、そこから直接呼ばれる同一packageの一意なplain function宣言を1段、選択sourceのimport。gold・fixture名・期待groupは選択に使わない。同じfile/kind/symbol/textの断片だけを重複排除し、各versionのsource span/hash/evidence IDを保持する。type定義もsourceとして取得する。call名は構文観測であり型解決や意味的な関係の証明ではない。

| 対象 | 固定contextのJSON bytes | 16KiB内 |
|---|---:|---|
| 使用済み6file behavior | 7391 | yes |
| 使用済みFee / Wrap | 4554 | yes |
| 使用済みsame-call diagnostic | 1812 | yes |
| 使用済みshared test | 3596 | yes |
| 開発用same-line | 2561 | yes |
| 開発用testなし | 651 | yes |
| 実履歴8092acd2 | 15402 | yes |
| 実履歴c643154b | 39409 | no |

8件すべてでcontextのsource位置/hashを元snapshotと検査し、file順反転でも同一出力を確認した。実履歴8092acd2には変更されたGraphStatistics型、graphStatisticsのbefore/after、追加testとその直接calleeのBuildGraphが含まれる。実履歴c643154bは変更されたExtract等と直接calleeを含み、固定context上限を超えた。

新しいcontext上限16KiBは、二versionのコード断片と根拠参照を含むattribution入力の監査用に、観測前に固定した値。#153の旧8192byte message guardと同じ費用条件とは扱わない。この測定はcontext部分のみで、ChangeUnit payload・system prompt・schema・chat templateを含む推論messageの上限やtoken fitを証明していない。能力比較では同じ観測情報と予算をbaselineにも適用し、旧#153結果との直接的な費用比較は行わない。

既存#149/#151/#152/#153のdocs/toolsで変更testのsymbol参照を検索した。既知診断4件のtest symbolは既存資料に見つかった。8092acd2のTestGraphStatisticsCountEdgesByKindは当該検索範囲で0件。c643154bの追加test symbolも0件だが、commit自体には前回監査で既存資料の参照があった。symbol参照0から未使用を認定していない。全件holdout_certified=false、semantic_quality=null。

`go test ./tools/benchmark155`（3テスト）、`go vet ./tools/benchmark155`、observer binary build、Python contextの2テスト、`git diff --check`は成功。contextの一段制限、曖昧な同名calleeの非選択、source改ざん拒否、type sourceとcall名の取得を確認。repository対象コード・testの実行、Git staging、モデル推論は行っていない。追加依存なし。

## 考察

今回の選択はsource上の比較に必要な候補を保存する処理であり、Program Slicingや意味的なunionではない。一段の関数選択でテストから実装までの経路が必ず閉じるとは限らない。未取得calleeや型・bindingが必要なら、入力識別性監査で不足として扱う。Goのbuiltinや型変換も現観測のnot_in_selected_snapshotsに含まれるため、この一覧を一律にsemantic unknownや失敗として扱わない。支持する関係ごとに必要な根拠を監査する。

8092acd2は入力候補として前進したが、変更testにあるerr検査を新しいstatistics behaviorの契約と同一視できない。任意の失敗条件をBehavioral Contractへ昇格させず、どの条件が実装変更によって変化する観測なのかをgold側で明示する必要がある。

現在の実履歴候補だけでは、独立holdout上のpositive/negative/unknown、cross-boundary、shared-test等の同時成立を判定できない。使用済み診断の成功で代替しない。実repository由来でも、保存された過去入力との非重複を確認するまでfreshと呼ばない。context上限内という観測だけではmodel入力適格性や意味品質は未証明。

## Next Steps

- 8092acd2のsource根拠と過去の保存モデル入力を照合し、評価対象となる観測条件を事前に固定する。symbol参照0だけでは入力の独立性と正解識別性が未証明のため。
- positiveに加え、独立変更・共有test・cross-boundary・unknownを含む実repository由来の評価対象を確保する。一種類の成功や使用済み診断だけでは#155の能力gateを満たせないため。
- ChangeUnit/anchorの入力契約、根拠付きrole出力とhost verifier、schemaを含むmessage/token budget、数値gate、モデル/helper/profileを固定する。同情報baselineと限定推論を一度だけ比較できる状態を作るため。
- c643154bは現context方法の上限超過として保持する。goldに合わせた選択範囲の追加調整や上限緩和で成功扱いしないため。
