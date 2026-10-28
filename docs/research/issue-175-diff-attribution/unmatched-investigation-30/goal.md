# 未一致4ブロックの原因調査（2026-10-25）

## Goal / Decision

fixture-preinference-e2e-29の未一致4ブロックは、すべてCodexの作業である。fileChangeに保存された中間編集の後に、commandExecutionのsedで修正されたため、fileChangeだけを採用する現行取得経路では最終編集が欠けた。Gitブロックの分割漏れでも時間切れでもなく、採用する編集記録の種類によるカバレッジ不足と確認した。取得completeは対応APIページの取得完了であり、全方式のファイル編集を捕捉した保証ではない。

## Evidence / Flow

実App Serverを同一fixture cwdで再照会。完了fileChangeは4操作あり、E2Eで保持した4操作と一致する。SiteFooter.jsxの5ブロック、SiteFooter.test.jsxの最初の5ブロック、NavigationHrefLink.jsxの3ブロック、SiteFooter.test.jsxの最後の1ブロックとして記録されていた。

NavigationHrefLinkの最初のfileChangeは引数にaria-labelを追加し、JSXにaria-label={aria-label}を追加した。後続commandExecutionのsedが引数をariaLabelへ、JSXをaria-label={ariaLabel}へ修正した。同じコマンドがSiteFooterのcontactリンクの属性をaria-labelからariaLabelへ変更した。現在Git差分はこれらの修正後の内容である。シェルコマンド全体はstatus=failed/exitCode=1だが、ファイル内容はsedの置換後と一致する。コマンド全体の終了コードを個々の編集成否と同一視しない。

| ファイル・ブロック | fileChangeの追加行 | 現在Gitの追加行 | 判定 |
|---|---|---|---|
| NavigationHrefLink B1 | aria-label, | ariaLabel, | unmatched |
| NavigationHrefLink B2 | aria-label={aria-label} | aria-label={ariaLabel} | unmatched |
| NavigationHrefLink B3 | aria-label={aria-label} | aria-label={ariaLabel} | unmatched |
| SiteFooter B2 | aria-label={contact文言} | ariaLabel={contact文言} | unmatched |
| SiteFooter B1/B3/B4/B5 | Gitの変更行と同じ | 同左 | matched 4件 |
| SiteFooter.test B1〜B6 | Gitの変更行と同じ | 同左 | matched 6件 |

この表は識別子の違いを示すため共通のインデントや文言を省略した。受信原本・完全diffはprivate tmpに保持。

## Implementation

history.go/readHistoryTurnはuserMessage、agentMessage、fileChangeを処理するがcommandExecutionを編集証拠へ変換しない。よってsed修正は受信していても照合操作にはならない。model.go/equalBlockは変更方向と行本文の完全一致、overlapも同じ変更行の存在で判定する。未一致4件は単一行の追加なので、識別子が変わると完全一致も部分一致も成立しない。現在ブロックは14件すべて入力に含まれており、ブロック欠落ではない。

## Implication / Next

Codexが編集したという前提は正しい。しかし全編集がfileChangeになるという前提は誤り。現方式はシェル経由編集に非対応で、今回のunmatchedを履歴取得完了だけから無関係と解釈してはいけない。対策は、制御された正常系E2Eでは全編集をfileChangeで記録する一方、実運用としてcommandExecution経由編集の観測・対応不能理由を設計すること。一般的なシェルを文字列だけで解析してmatchedへ昇格することや、aria-labelとariaLabelを同一視する正規化は行わない。

## Delivery

調査は読み取り専用、通常コード・fixture・Git状態は変更せず、追加推論なし。原因特定後の修正は本調査では実施していない。会話本文・出典ID・個人絶対パスは報告に含めない。commit/push/main反映なし。
