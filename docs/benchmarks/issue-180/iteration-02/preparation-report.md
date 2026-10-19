## 要約

追加検証の事前固定。採用済みの言語名直接指定・変更内容と裏付けのある目的を述べるText指示を、B/C/D共通で適用した。Category/Textの両方thinkingあり(T)、両方通常(N)を比較する。Grouping設定は維持する。

## 検証結果

生成前のtoken監査はT/N×C01/C02すべてで、共通Raw Diff/履歴を含むprefixの一致を確認した。Dも登録した条件内で実行する。モデル保持/生成cacheの処理はbenchmark専用経路で、production既定は変えていない。

- native予算: Grouping512、TはCategory/Text各384、Nは両方0。最終生成の総token上限は従来の768/512/768で、thinkingもその総数に含まれるためTでは最終JSONに使える余地が減り得る。長さ失敗も保存して救済しない。
- 順序: C01 B-T→C-T→D-T→B-N→C-N→D-N、その後C02の同順。各1回、最大12cycle/36要求、timeoutなし、retry/repair/judgeなし。失敗cycleは止め、残る登録cycleだけ続ける。
- 履歴あり、同じfixture/Gold/固定グループ。実Groupingも保存するが後段比較は同じ参照partitionで行う。既存Goldは人手確定ではない探索的参照。
- DはCategoryで共通prefixのcacheを作成・保存し、Category生成に入る前のコピーだけをTextへ使う。Textではtoken列一致を再確認する。prefix処理・コピー費用もwall timeに含める。runtime prompt時間だけではDの共通prefix計算費用が抜けるためprefixPrefillSecondsと合わせて報告する。
- Go test/vet/build成功。Swift build初回の変数宣言エラーと修正履歴を保存し、修正後release buildとMetal smoke成功。Swift既存testの完了後に生成を開始する。

## 考察

同一mode内B→Cがモデル保持、C→DがKV追加効果。T対Nは思考設定/思考予算の差も含む。既存mixed-modeのA/B/Cとpromptが異なり、旧結果から単一変更の因果効果を主張しない。各1回・固定順序なので一般的な非劣性や速度保証は行わない。

## Next Steps

固定source/helper/binary/overlay/Goldと実行前Git状態をexecution-contract.jsonへ保存した。Swift testが通過したら登録12cycleのみ実行し、全失敗を保持して品質・時間・メモリ・cache再利用を同じ形式で報告する。ソース変更や追加試行は行わない。
