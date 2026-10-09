## 要約

Stage Bの固定候補は**NO-GO**。独立repository由来の2workloadを正逆順に評価した4callはcompleted・構造valid4/4だが、exactは1/4。positiveの順序逆転で単一目的をmixedと誤判定し、file fallbackへ退避した。全体groupのcoverage/margin gateを省くhighest-score対照ではnegative逆順でREADMEを誤吸収しFM648が発生した。閾値を変更しない。

## 検証結果

- 未使用BurntSushi/tomlの9fe9ada4（ParseError.Unwrap実装とerrors.Unwrap/Is/As test）をpositiveとした。b66ba2a0（READMEのUnmarshalTOML説明の語句訂正）は独立目的。両履歴のdisjoint file差分を重ねたnegativeは制御合成workloadで、同一時点の自然repository diffではない。
- gold根拠は変更されたUnwrapのエラー返却と対応assertion、別のREADMEの説明訂正である。単なる同一commitという理由だけでgoldにしていない。source/patch、commit SHA、input/prompt/source hash、schema、予算を結果前に保存した。コミットメッセージやgoldはモデル入力に含まない。
- model Qwen3-8B 4bit revision545dc4251c05440727734bcd94334791f6ab0192、helper SHA bc61461f860957780394650f8b46fab2b737f9931917d0194a0f736e671b41a4、neutral grammar、context16K/output1536/native thought0、temperature0/topP1/topK0/seed144、4call/480秒・120秒/call・retry0。
- candidate上限8。変更pathのstem/directoryと全体groupをsoft proposalにのみ使用。ordinal0〜3、score3・全member single/coverage all・member証拠参照・競合候補との差1以上を固定採用条件とした。推移mergeなし、一部CUの関連性からfile全体を吸収する条件なし。
- positive正順 exact1、FM0、FS0、1commit。positive逆順 exact0、FM0、FS272、2commit。negative正逆ともexact0、FM0、FS272、3commit。A baselineは全4観測exact0、FM0、FS272/観測。
- highest-score対照はexact3/4だがnegative逆順FM648。候補scoreそのものは同一目的の根拠にはならなかった。
- 4callのinput tokens合計5414/output667。wallは19.243/15.515/26.515/26.876秒（モデルload・host stage監査を含む）。fallback commit率0/1/1/1。正逆stageと最終tree一致4/4、欠落・重複0。compose0。
- additional corpus preflight: toml 6a671b23は既存256CU/file上限でreject。上限緩和は行わない。9fe9ada4は72CU、c6d720d8は140CU、sqlx421d1cdbは271CU/2fileで対応可能。upstream test同期61fileの選択16fileは126CUだが、複数の独立syntax目的を含みunique intent goldが曖昧なためDのprimaryには自動採用しない。
- neutral grammarは既存固定helperのsourceでwhitespace penaltyなし、非負ordinalenumを用いた。実prompt+outputのcontext適合はhelperが確認。今回schemaで推論前のtokenizer-only合法/不合法全経路監査は未実施で、その完全性は主張しない。順序感度は計測結果として保存した。

## 考察

Bのguardrailは誤った高scoreの全体統合を抑えたが、真の同目的もmixed/partialと判断するためFS改善が順序依存となった。negativeでgateを弱めればFMが発生する対照があり、事後のcoverage/margin緩和は採用根拠にならない。独立入力は2workloadだけであり、他の作者・repository・16fileへの一般化を示していない。

Aが構造fallbackを返す能力とBの意味GOは別である。本候補BはDの最終候補へ採用しない。意味不明時のfallbackは合法計画を保つが、unknownを正解やFM/FS0へ読み替えない。

## Next Steps

- CはBのNO-GOを保ったまま、Aに対する独立した部分stage能力/意味判定の監査のみを行う。mixed判定の順序感度が細分化の安全な開始条件になるか未証明であるため、B+Cの自動統合はしない。
- DにはAのみと現行4file three-phase、既存H23の固定対照を残す。NO-GO候補を採用せず到達可能な構成の限界を測るため。
- 16fileのunique goldには、独立repositoryの一貫したAPI/表記移行履歴を別に事前固定する。複数目的が妥当なupstream同期を都合のよいsingle goldにしないため。

production変更・既定model変更・4file上限変更・cloud送信・外部repo code実行は行っていない。
