# キャッシュ型に基づくprefix再利用：実装・検証範囲

H184-HARNESS-003は再利用なしの採用済み基準として保持する。今回のコードはユーザー採用によりH184-HARNESS-004 / revision 1の研究用実装として登録。モデル名によるアルゴリズム分岐ではなく、ランタイムが生成した各層の具象型に対応するcopy()を利用する。

## 接続

- 同じhelper/modelの研究経路でCOMMITER_184_PREFIX_PLANを指定した場合だけ試行する。従来の--benchmark-cacheとは併用しない。
- planはprivateなJSON。tokens（チャットテンプレート後の監査済み整数配列）、profiles（先頭bounded-grouping、例Grouping/Textのみ）、model（検証対象ID）、runtimeRevision（固定mlx-swift-lm revision）、verifiedCacheTypes（検証済み型名）、validationRecord（モデル/ランタイムの実検証記録ID）を持つ。
- model/runtime/設定の識別と先頭tokens完全一致を確認。最初の対象phaseでruntimeのnewCacheから取得した型を全層確認する。
- 共通prefixを512tokensずつ評価し、固有部分へ進む前に全cacheをcopyして保存。最初のphaseも保存状態から独立forkし、後段も別forkを使用する。Categoryを対象profilesへ含めなければ独立prefillし、保存状態は上書きしない。
- Textはprefixを除く残りのtokensをgenerateへ渡す。位置はcacheが保持。モデルの再利用とprefix再利用は別の計測値。
- plan不備、未登録型、copy検査失敗、token/設定不一致は通常の全入力prefillへfallback。未知層だけを飛ばすことはしない。

## 型対応

具象型の一致でKVCacheSimple、RotatingKVCache、MambaCacheを扱う。新しいサブクラス、ArraysCache、合成cache、量子化cacheは今回未対応。元cacheとコピーの型・offset・metaState・state配列の数/shape/dtypeを検査し、既存runtime.copy()へ状態複製を委譲する。Mambaの2つの再帰状態、offset、padding等はruntime.copy()が保持する。投機生成やcache外に状態を持つモデルの対応は保証しない。

モデルIDは検証記録の適用範囲を制限するために照合する。cache操作そのものをモデル名で選択しない。verification recordは実測済みの記録を指定する運用契約であり、非空文字列だけで実モデルの妥当性が証明されるという意味ではない。現時点で有効化できる実モデルの検証済みplanは作成していない。

## テストと残課題

Swift Releaseの新規3テストで、Attention/回転型の更新再開同等性（容量を跨ぐケースを含む）、兄弟fork/保存状態の独立性、Mambaの状態/位置/metadata保持、未知型/未検証型/identity不一致/prefix不一致の拒否を確認。これらはcache単体のテストであり、実モデルのlogits・生成結果同等性を証明するものではない。

次は003の最終入力を再token監査し、Grouping/Text共通tokensを固定する。cacheなしとありの実モデルで生成結果・完走・Grouping/Category/Text・validator・メモリ・総時間/prefill時間を比較し、検証記録に基づいて有効化する。prefixPrefillSecondsとprefixSnapshotSecondsを通常prefill/decodeに加えて比較し、保存・コピーの費用も含める。

採用済みmanifestと測定バイナリは変更しない。追加依存なし。実モデル推論は今回実行していない。採用記録と検証の限界は004 manifestに記載。
