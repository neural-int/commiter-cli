# コピー先での実履歴付き推論前E2E（2026-10-25）

## Goal / Decision

commiter-test-my-siteで新しく行われたアクセシビリティ改善について、実App Server→固定Git差分→内容照合→Grouping/Category/Text入力→受信専用helper subprocessまで成功。依頼1件・最終応答1件の実本文と候補参照が全3フェーズへ到達した。モデル推論は0回。14ブロックすべての照合成功を意味しない。

## Conditions

対象はNavigationHrefLink.jsx、SiteFooter.jsx、SiteFooter.test.jsxの3ファイル。コピー前からあるホーム画面関連の差分は選択しなかった。Git HEADは38e2f52。現在ワークツリーの会話のみ、固定HEAD時刻〜実行開始時刻、100件ページ・最大4並列・受信上限・メモリ保護を維持。Go overlayによって取得の2箇所の5秒期限のみ無効化した。テスト自体の期限も無効。通常設定・元ソースは変更していない。

通常CLIのgenerateCommitPlan、実Git解析、実App Serverと実MLXクライアントのstdin通信を通した。モデルストアは検証用メタデータ、helperは受信専用プロセス。後続フェーズへ進むための返答は固定fixtureでありモデルの分類や文章品質を評価していない。Swift製モデルhelper内部・トップレベルCLIの全操作は対象外。

## Measurements

- 取得・照合: 0.194129秒
- 3フェーズを含む通常生成関数: 1.691185秒
- テスト全体: 2.41秒、PASS
- RPC: 6回、受信126,993 bytes
- acquisition: complete、reasons=[]、期限到達なし
- 保持編集操作: 4件、履歴context: 1件
- Git HEAD/branch/index/選択変更: 前後一致

| フェーズ | wire bytes | 本文bytes | context | matched / unmatched | 候補参照 |
|---|---:|---:|---:|---|---:|
| Grouping | 10,731 | 880 | 1 | 10 / 4 | 10 |
| Category | 10,454 | 880 | 1 | 10 / 4 | 10 |
| Text | 9,953 | 880 | 1 | 10 / 4 | 10 |

## Integrity / Limitations

各入力の実履歴本文がhost取得結果と完全に一致すること、同一context_refの定義が重複しないこと、候補参照先が存在すること、F1〜F3とraw_diffのSHA256が3フェーズで変わらないことを検査した。Category/Textで全体のacquisition/projectionを含めず、Textが固定Category応答のtype/breakingを保持することも確認。

未一致4ブロックはNavigationHrefLink.jsxのariaLabel引数追加と2箇所の属性渡し、およびSiteFooter.jsxのcontactリンクへのariaLabel追加。SiteFooterの残る4ブロックとテスト6ブロックはmatched。未一致の原因が履歴の段階的編集か内容照合の制約かは本実行だけでは確定していない。履歴が届くE2Eは成立したが、全ブロックの照合精度の検証は残る。

以前の実リポジトリでは約20秒で45MBを受信したが、今回のfixtureは別の履歴集合であり、速度改善率としては比較しない。

## Delivery

検証は読み取りのみ、Git書き込み・reset・commit・推論なし。受信原本はprivate tmpの0700ディレクトリ内に0600で保持。会話本文・出典IDを公開用報告に含めない。通常ソースは変更せず報告のみ専用作業ツリーへ追加。commit/push/main反映なし。
