## 要約

#155の初回入力監査を実施した。使用済み診断4件と開発用境界条件2件の全6件で、Go構文から取得した観測のbyte位置・source hash・再現性と、inline ChangeUnitの完全再構築を確認した。モデルcallは0。これはsource観測の成立範囲であり、attributionの能力GOでもproduction GOでもない。

新規専用worktree `issue-155-attribution-audit`、branch `codex/issue-155-attribution-audit`、開始点 `4abfd13b6d6917c1fcbf539265d42347cd3c249e`。旧#150の未達判定を変更しない。

## 検証結果

既存#149のassertionFactsはsource spanを保存せず、call比較・単純bool条件に限定され、複合OR条件は抽出されない。新しい監査処理はGo標準parserで、Test関数の明示的なFatal/Fatalf/Error/Errorf単独bodyのif条件と関数sourceを記録する。ORの各条件には元の全条件spanも保持し、単独条件の真偽がtest全体を決めるという推論は行わない。自由生成した振る舞い名、意図ラベル、コミット結合規則は含まない。

| 対象 | inline atom | 変更された条件：before / after |
|---|---:|---:|
| 既知6file behavior | 88 | 2 / 3 |
| Fee / Wrap | 10 | 2 / 2 |
| 同じcall・独立diagnostic変更 | 39 | 0 / 0 |
| shared testの別assertion | 4 | 2 / 2 |
| 同一行の2条件（開発用） | 2 | 2 / 2 |
| testなし（開発用） | 2 | 0 / 0 |

全6件で観測sourceのUTF-8 byte spanとhashを検査。観測処理の再実行およびfile順反転で同一JSONを確認。全inline atomを適用するとafter、無適用ではbefore、選択ID順反転でもafterを再構築できた。Git stagingや部分状態の実行は今回行っていない。semantic qualityは全件null。

`policy-fixtures.json`の一部に残る`model_unused_synthetic`を採用せず、既存4件をすべてused diagnosticに固定した。初回監査契約・fixture hash・observer binary/source hashは実行前に`iteration-1-audit-contract.json`へ保存。測定は`iteration-1-input-audit.json`。

実履歴候補をsource構文だけで監査した。対象repositoryのコードやtestは実行していない。

| 候補commit | 全snapshot入力bytes | 過去benchmarkのcommit文字列参照 | 変更された条件観測 |
|---|---:|---:|---:|
| 8092acd2886ba95ee8032518da6c1fcb67481e3b | 38517 | 0 | 2 |
| c643154b0498e5d441efdb864a6cee26e3deb2e0 | 59506 | 2 | 6 |

観測条件にはerr検査なども含まれる。観測数は有効なbehavior contract数でも正解relation数でもない。commit参照0は過去のモデル入力との非重複の証明ではない。両候補ともholdout認定・gold設定は未実施。

`go test ./tools/benchmark155`の2テスト、同packageの`go vet`とbinary build、`git diff --check`は成功。UTF-8位置、複合条件、変更されない条件、対応未サポート言語と重複ID拒否を検査した。全project最終gateはVerification未達のため実行していない。追加依存なし。

## 考察

shared test・同一行に複数条件が存在しても、sourceの観測位置を区別して保持できる。message変更のみの診断例には変更assertion anchorがない。この区別は将来のattribution入力に利用できるが、実装との意味的な対応付けを証明しない。

現observerは構文観測であり、型解決・callee解決・data/control dependence・条件の実行真偽を証明しない。nested callback、assertion helper、table expectation、未サポート言語などは成立範囲外。条件が観測できないことはsemantic independenceの証明ではなく、対応に必要な観測不足として扱う必要がある。function sourceの構文変化もそのままbehavior contractへ昇格させない。

実履歴の全snapshotは#153で使った8192byte guardを超える。入力を意味ラベルやgoldで間引かず、変更unitと構文上の周辺sourceを取り出す方法を先に固定し、必要な参照定義・期待値が欠ける対象はunknownまたは評価不適格とする。現在の候補は正解識別性・入力budgetとも未認定であり、まだ推論を行う段階ではない。

## Next Steps

- 決定的なsource context選択とanchor差分仕様を固定する。実履歴全体を無制限に投入せず、選択後のコードで対応関係を説明できるか監査するため。
- 実履歴候補の過去モデル入力への重複をhash・source照合で調べ、positive/negative/unknownとcross-boundaryを含む未使用holdoutを確保する。commit参照の有無だけでは独立性を保証できないため。
- attribution出力schema、ID/evidence検証、数値gate、モデル/helper/profile/予算とbaselineを推論前に登録する。anchor取得成功と意味判断成功を分離し、測定後の基準変更を防ぐため。
- 上記が成立した後にだけ単独能力比較を実施する。今回はBOUNDED ATTRIBUTION GO / NO-GOのいずれも未判定で、B/Cやproductionへ進めない。
