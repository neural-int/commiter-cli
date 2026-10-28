## 要約

前回改善後のD-Tを保持したまま、共通prefixのcache評価を層別から一括へ変更した。変更は1行のみ。Metal Toolchain不足でSwift helperのbuildが失敗し、実モデル検証の開始条件を満たしていない。追加生成は0要求であり、性能改善・出力一致は未評価。

## 検証結果

- 専用worktree: issue-180-batch-eval/commiter-cli。baselineは198d2c2dのiteration03 D-T。通常経路への未commit採用実装は所在不明のため、保存済み検証経路で比較を準備した。
- source snapshotとの差は`for entry in cache { eval(entry.state) }`から`eval(cache)`への置換のみ。前回の不要logits評価省略、chunk512、同期と解放頻度、KVコピー、promptを維持。
- 固定依存revision c6446cf7bfb7cea76408013b614d4b2c530eaa03、fixture C01/C02のHEAD/index/対象ファイルの一致を確認。
- Go全test/vet/build成功。先行一括コマンドの終了状態が不明だったため、vet/buildは単独で終了コード0を確認した。
- Swift release buildは依存MLXのMetalファイルcompileで`cannot execute tool metal due to missing Metal Toolchain`となり失敗。初回失敗logを保存。依存追加・テスト無効化・ソース条件変更による回避は未実施。
- 候補helper未生成。Swiftテストと実モデル検証は未実施。生成要求数0。

## 考察

ソース変更の実装と比較条件案は用意したが、変更後helperをbuildできていないため速度・メモリ・出力品質の結論は出せない。前回結果を今回結果として流用しない。

## Next Steps

Metal Toolchain導入の承認後にbuildとSwift既存テストを行う。全gate成功後、実行hashを固定してC01/C02 D-T各1回・最大6要求を実行する。baselineは再推論せず、追加調整・retry/repair/judgeは行わない。今回の対象は①のみとし②③は変更しない。
