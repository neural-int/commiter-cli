# Category/Textの現行入力スキーマ更新

## Goal / Decision

ユーザー指定の13項目候補をCategory/Textの送信投影に採用。条件付き省略で通常更新は5項目。breaking_evidence_candidates[].factとevidence_pool_limitationを送信から除去。Issue183の新しい履歴共通スキーマはまだ接続しない。

## Implementation

internal/planning/metadata_input.goで送信型を定義し、hostのGit識別情報/Factは維持。metadataGroupsに確定typeを反映しTextへ渡す。元入力を変更しない。Grouping入力と各出力契約は維持。benchmark146のアダプターも同じ投影を使い、完全diffを持つテンプレートに追従。過去原本と測定値は変更しない。

## Validation

cmd/internal/tools全パッケージのgo test成功。関連race、vet、CLI build、diff check成功。通常更新/rename/削除/モード/追加実行ファイル/symlink/gitlink/言語/binary/opaque/vendor/不正入力/実送信フィールド除去/元入力保持/Textのtype引継ぎを確認。

最初の全体テストではdocs内の原本のpackage衝突、benchmark146の追従不足、並行負荷下の既存PoC受信上限fixtureのタイムアウトが発生。benchmark146は追従修正、PoC fixtureは変更せず最終の全実行コード検証で成功。docs原本の複数package同居は既存のまま保持し、go test ./...のsetup失敗は残る。モデル精度や速度の改善は今回未測定。

## Publication

Issue183コメント https://github.com/neural-int/commiter-cli/issues/183#issuecomment-6368406614 へ変更後を現行スキーマとして投稿し、再取得一致を確認。ローカル実装済みでcommit/push/main反映は行っていない。
