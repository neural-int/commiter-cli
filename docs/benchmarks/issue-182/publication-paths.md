# 公開用パスの整理

公開コピー内のホーム・一時ディレクトリを`<HOME>`、`<PRIVATE_TMP>`、`<SYSTEM_TMP>`へ置換した。これらは実在パスではなく、実行時には任意の保存先へ読み替える。実行契約中のSHAは元の測定時の値を保持し、SHA256SUMS.txtは置換後の公開コピーに合わせて再計算した。計測値・応答内容・採点は変更していない。置換前の資料は公開対象外に保持する。

研究用Pythonハーネスの一時領域は`COMMITER_BENCHMARK_ROOT`で指定でき、省略時はOSの一時ディレクトリを使用する。過去Gemmaモデルの場所は`COMMITER_182_GEMMA_MODEL`、参照fixtureの上書きは`COMMITER_182_REFERENCE_FIXTURES`を使用する。任意の追加履歴rootはopt-inテストの`COMMITER_ATTRIBUTION_ADDITIONAL_ROOT`で指定する。モデル・fixtureは引き続き元の固定revision／SHAを確認してから実行する。
