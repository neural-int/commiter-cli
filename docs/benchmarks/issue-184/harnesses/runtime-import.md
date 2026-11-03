# H184-HARNESS-001のSwift実装取り込み

## 目的と出典

H184-HARNESS-002の実改行表示に加え、親001のscope制約と実行経路のソースも同じブランチに保持する。

取り込み元は`374f8f6da57adc84dc230f997afe070d55f160a6`。元コミットを丸ごとmergeせず、`mlx-helper`の差分10ファイルだけを移した。設定の新規採用や推論条件の変更ではないため、登録は002 revision 1のまま、manifestに取り込み元と各ソースのSHA-256を追記する。

## 取り込み範囲

| ファイル | 役割 |
| --- | --- |
| Sources/BoundedGeneration/ScopeValueConstraint.swift | 完成したscope文字列を検査し、カテゴリ名の単独利用を拒否 |
| Sources/BoundedGeneration/GrammarSampling.swift | JSON文法マスクとprefix検査を交差し、制約・状態コピー・除外数を保持 |
| Sources/commiter-mlx-helper/CommiterMLXHelper.swift | 研究用bounded-textへの接続、policy読込、除外数の計測、モデル設定、wire監査 |
| Sources/BoundedGeneration/NativeThoughtBudget.swift | 元の研究経路が使うThinking境界・任意予算・回答状態の管理 |
| Sources/BoundedGeneration/StructuredOutputPrompt.swift | 元の研究経路が使う構造化出力契約とSchema説明 |
| Sources/BoundedGeneration/OverlappedTokenProcessor.swift | 元の研究ランタイムに含まれる選択式のlookahead経路。002の既定はeagerのまま |
| Tests/BoundedGenerationTests/ScopeValueConstraintTests.swift | scope文字列検査、禁止値の閉じトークン除外、状態コピー |
| Tests/BoundedGenerationTests/NativeThoughtBudgetTests.swift | Thinkingと回答境界の前提テスト |
| Tests/BoundedGenerationTests/OutputConstraintTests.swift | 出力制約選択・Schema説明の前提テスト |
| Tests/BoundedGenerationTests/OverlappedTokenProcessorTests.swift | 元の選択式lookahead経路の前提テスト |

表のパスは`mlx-helper/`からの相対パス。全24個のSwiftパッケージ内の追跡ソース・テスト・設定が、取り込み元コミットとbyte単位で一致する。取り込み先に変更が必要だったのは10ファイル。

scope追加マスクは、研究設定が存在するbounded-textで`COMMITER_184_SCOPE_CONSTRAINT=1`の場合だけ有効。禁止語配列は`COMMITER_184_SCOPE_POLICY`から読む。`docs-api`やsummary内の`docs`は許可し、完成したscope値が禁止語へ正規化される場合を除外する。後処理での回答置換はしない。

## 検証

| 検証 | 結果 |
| --- | --- |
| Swiftパッケージ内24ファイルのソース・設定一致 | 元コミットおよび検証用キャッシュのworkspaceとbyte一致 |
| Releaseビルド | 成功、95.22秒、Apple Swift 6.4 / swiftbuild |
| Swiftテスト | 53件成功（BoundedGeneration 43件、Protocol 10件） |
| Python表示・投影テスト | 11件成功 |
| 再ビルドしたhelperのMetal最小動作 | metal_ok |
| Grouping/Category/Textの非推論wire監査 | 3件とも過去の採用B条件と完全一致、各generation_calls=0 |
| 追加モデル推論 | 0回 |

空き容量の制約から、移植先と全24ファイルが一致する元のworkspaceの既存ビルドキャッシュを再利用した。移植先の新しい空のビルドディレクトリで依存から全て再構築した検証ではない。Package.swiftとPackage.resolvedは変更しておらず、新規依存を導入していない。

初回のReleaseビルド自体は成功したが、GPUテストはdefault.metallibの探索失敗で停止した。MLX依存の探索実装で実行ファイル横のmlx.metallibが優先されることを確認し、測定時と同じSHA-256のMetalライブラリをテスト実行ファイル横へ配置した。その後、ビルド済みテストを`--skip-build`で再実行して全53件が成功した。ソース修正やテスト無効化では解決していない。

機械可読な結果と今回の再ビルドhelperのSHA-256は[runtime-verification.json](runtime-verification.json)に記録する。過去の測定バイナリSHAとは別に保持する。

## 再ビルドと適用範囲

依存は既存のPackage.resolvedに固定する。通常のテストと配布用Releaseビルドは次で実行できる。

```sh
bash .github/scripts/build-mlx-helper.sh
bash .github/scripts/test-mlx-helper.sh --triple arm64-apple-macosx14.0
```

Swiftテストで同じMetal探索失敗が出る環境では、検証済みのmlx.metallibをテスト実行ファイル横に配置してから`swift test --package-path mlx-helper -c release --disable-automatic-resolution --skip-build`を再実行する。配布用ビルドスクリプトはhelper横へMetalライブラリを配置する既存処理を維持している。

PR #185の再レビュー後は、上のテスト用スクリプトでビルド・Metal配置・テスト実行を順に行う。テスト無効化による回避はしない。[再レビュー対応](pr-185-review-fixes.md)に通常Plannerへの適用範囲と回帰検証を記録する。

モデル・実履歴・固定入力・Goの3phase評価器は別の資産であり、この取り込みで追加しない。完全な3phase推論の再実行や精度の再測定も今回の検証範囲には含まない。歴史的なhelper/Metalの測定SHAを、新しいビルド結果として上書きしない。
