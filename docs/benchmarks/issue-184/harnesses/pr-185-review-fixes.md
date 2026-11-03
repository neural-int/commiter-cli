# PR #185 再レビューへの対応

## 対象と再現

レビュー対象は`cd22ff91`。修正開始時、リモートのブランチは`a246bd56`だったが、GitHubのPR APIと`refs/pull/185/head`は`cd22ff91`を返した。ブランチへのpushとPRへの反映を別に確認する必要がある。

MLX Release Buildの失敗ログは、テスト起動後の`Failed to load the default metallib`を示した。コンパイルエラーとしては扱わない。既存評価器は保存済みの3メッセージ要求を渡すと`invalid character '\n' in string`で停止した。bounded backendへ通常のテキストを渡すテストも、表示変換のJSON解析で失敗した。

修正前Swift helperに存在しないモデルディレクトリを渡した非推論テストでは、失敗応答から`generation_profile`が欠落した。単発経路では`internal_error`、セッション経路では同じ応答がProtocolエラーになった。

## 変更内容

| 指摘 | 対応 |
| --- | --- |
| Metalライブラリ不足 | CIで既存のXcodeビルドを先に実行。Swiftのテストを実行せずにビルドし、生成した`mlx.metallib`をテストProducts横へ配置してから全テストを実行するスクリプトを追加 |
| 通常Plannerのプロセス再利用 | 意図した変更として維持。以下の適用範囲を明記し、Plannerの正常終了・各フェーズ失敗・起動失敗・入力検証失敗に対するライフサイクル回帰テストを追加 |
| 評価器の入力形式 | canonicalの2メッセージとmixed-layoutの3メッセージに対応。後者は3番目の`Phase input:`本文だけを読み、表示用の文字列内LFをJSON用に戻す。共有辞書をフェーズ入力と誤認しない |
| 失敗応答のprofile欠落 | Swiftの生成失敗、ロード前の失敗、セッション外側の失敗で、デコード済みの要求profileを保持。Goのprofile一致検査は維持 |
| bounded入力互換性 | `llm.Options.LiteralNewlines`を追加。Plannerがcanonical JSONに対して明示した場合だけ表示変換する。profile指定だけでは通常テキストや既存メッセージを変更しない |

## 通常Plannerへの適用範囲

同一helperのモデル・文法準備をGrouping→Category→Textで再利用する変更は、ユーザーが採用したランタイム改善であり、研究用表示投影だけに限定しない。`BeginCycle`を実装するバックエンドでは、入力検証後に1回だけ開始し、正常終了でも途中失敗でも閉じる。対応しないバックエンドは従来の呼び出しを維持する。

各フェーズのmatcher・回答状態・通常KVは独立。順序、共有deadline、未完了時の停止、最終validatorは維持し、別のPlanner実行へセッションを持ち越さない。通常の`Client.Generate`は1要求1プロセスのまま。研究用の型別KV prefix再利用は、別途private planと研究設定を明示した場合だけ有効になる。

既存のセッション回帰テストは、1プロセスでの3要求、順序違反、キャンセル、不正フレーム、最終応答後のクラッシュを確認する。今回追加したテストは、Plannerが返されたcycle backendを使用すること、各終了経路でcloseすること、エラー時に部分計画を返さないことを確認する。

## 検証の区別

| 検証 | ローカル結果 |
| --- | --- |
| `go test ./...` | 成功。初回はローカルHTTP待受のsandbox制限で失敗し、制限外で実行。表示指定を同期した既存benchmark146も含めて通過 |
| `go vet ./...` | 成功 |
| `go build ./cmd/commiter` | 成功。生成物はprivateな一時領域へ保存 |
| Swift Releaseテスト | BoundedGeneration 46件・Protocol 10件、計56件成功 |
| 新しいMetal配置付きテストスクリプト | 成功。テストの事前ビルド、Metal配置、56件の実行、実Swift helperの単発／セッション失敗応答テストを通過 |
| Python表示・投影テスト | 15件成功 |
| `git diff --check` / shell構文確認 | 成功 |

ローカルSwift確認は既存の固定依存ビルドキャッシュと、同じ依存から生成済みのMetalライブラリを利用した。初回の`swift build --build-tests`はテスト用moduleの互換性エラーで停止したが、`swift test`は成功。新スクリプトではReleaseの事前ビルドにも`-enable-testing`を明示し、スクリプト全体の成功を確認した。GitHub CI環境での新しい実行結果とは区別する。

保存済みQwen/Gemmaのprefix再利用なし・あり計4条件を新評価器で再評価し、すべてCategory 3/3・最終host validator受理を確認した。追加モデル推論は行っていない。独立した固定フェーズ入力の評価であり、上流応答を伝播するE2Eの証明ではない。

003/004のmanifestにあるソース・バイナリSHAは過去の採用・計測時点を表すため、今回の修正後バイナリで上書きしない。今回の修正はcommitで追跡する。依存・sampling・prompt・予算・文法の条件は変更しない。

## Metal探索の追加修正

`92abd770`のGitHub CI（run `41074826240`）では、helperとMetalのビルドは成功したが、Swiftテストは再び`Failed to load the default metallib`で停止した。上記のローカル成功は、このCI構成での成功を証明していなかった。

固定依存のMLXは、`current_binary_dir()`内の`dladdr`でMLXコードを含むMach-Oイメージのディレクトリを取得し、そこから`mlx.metallib`等を探す。CIのSwiftPMテストではイメージが`.xctest/Contents/MacOS/`内にあるため、`--show-bin-path`のProductsルートにだけ配置しても届かない。ローカルの別ビルド方式での配置結果を、CIへ一般化したことが不足だった。

配置処理はProductsルートに加え、ビルド済みの各`.xctest/Contents/MacOS/`内の実行可能ファイル横へコピーする。各コピーを`cmp`で確認し、配置先をCIログへ出す。ランタイム・モデル設定・テスト内容・テスト有効／無効は変更しない。

既存SwiftPMテストbundleをprivateな一時領域へ複製した再現では、Productsルートだけの配置で同じMetalエラーとなり、同じbundleの`Contents/MacOS/`へ追加配置する変更だけで全53件が成功した。この53件は再現用の既存バイナリであり、現在のソースの56件とは区別する。配置順序・複数bundle・空白を含むパスを、標準Pythonのみの回帰テストでも確認する。新しいGitHub CIの最終結果はPRで確認する。

修正後の配置スクリプトを現在のソースで実行し、Protocol 10件・BoundedGeneration 46件の全56件と、実helperの失敗応答テストが成功した。Productsルートと2つのテストbundle内への配置ログも確認した。
