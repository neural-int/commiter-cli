## 要約

A2のcoarse-first所有権は、使用済みD入力6/6で完全割当・再構築・正逆順staging・planning.Validateを通過した。前回停止した16 selected filesも成立し、構造能力を改善した。意味境界はfile fallbackのままなのでexact2/6（33.3%）で、80%の意味能力GOではない。

## 検証結果

- 同一input hashの固定6診断を評価。1/4/8/16fileの行operation数は6/22/46/141、Select resetは37、azuresql bindは4。全て元の256operation/fileと4096total、1MiB/20kline/file上限内。
- 16fileのbyte所有権・最終tree一致・正逆順512file-stage stepsが成立した。監査wall50.329秒。LLM calls/tokens0。
- cmp/compare_test.goは27行operationで完全再構築できた。任意inline refinementは元の256leaf上限でunit_budgetとなり、27行operationを保持した。上限拡大や変更file除外は行っていない。
- file粒度FM/FSは1file0/0、4file0/6、8file0/28、16file0/120、Select0/1、bind0/0。前回のUTF8 atom pair数と粒度が異なるため、その数値とは比較しない。exactは2/6、coverage6/6、fallback率100%、単独file commit率100%、compose0。
- 6研究回帰テストPASS、差分整合PASS。追加1テストは実際に停止した公開snapshotで、eager拒否・coarse所有権保持・refinement拒否時の非破壊fallbackを確認する。
- production、既定モデル、4file上限は変更していない。独立意味holdoutはまだ評価していない。Goalの最終品質ゲートも未実施。

## 考察

全ファイルを先に文字単位へ細分化することは、file fallbackのためには不要だった。行単位の所有権でも同じsnapshotを失わずに保持できる。A2は構造的予算失敗を解消する限定GOであり、ファイル内を単一目的とみなす意味仮定を追加していない。

同一行の独立目的を分割する際はbounded inline refinementが必要になる。refinement拒否は構造的所有権の破損とは異なり、既に検証済みcoarse fallbackへ戻れる。一方、mixed検出もcommit境界推定も未実装・未立証なので、fallbackを意味成功へ数えない。

## Next Steps

- Cの新しい独立mixed入力と分割推定を推論前に固定する。A2により意味判断前の所有権停止を避けられるため、混在目的の分類とCU帰属を独立して観測する。
- Bのscore閾値を変えず、A2+Cという明示的な別仮説を評価する。Bの順序感度を承継した全面統合は行わない。
- Dのnegative/shared-test/weak/unknown等は別の未使用履歴と固定合成入力で補う。使用済み6件を独立holdoutへ再分類しない。
- production導入はNO-GOを維持する。C/Dの意味能力・80%達否を確定してから必要な別実装Issueを判断する。
