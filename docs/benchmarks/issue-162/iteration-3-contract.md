# Iteration 3: テスト補助の観測限界と実測費用

未使用の自己作成Go8ケース、各4変更file内のEA/EBだけを判断する。テストなし2件、変更しない強い既存assertion2件、片側の誤りを見逃す弱い既存assertion4件。後者には連動する補償変更2件と独立変更2件を含める。Result(n)=nを維持する補償ペアは共同レビューを要求する規範を入力・結果より先に固定する。単なる同一symbolでmust_joinとはしない。

前回のtest-assisted方式を変更せず比較対象に残す。新しい保留方式は両片側stateがPASSでも独立性を確定せずdefer。deferは未解決であり、成功や安全な分割へ変換しない。両片側FAILはmerge、一方FAILは依存順splitを提案。どちらも本ケースの規範に照らして別に評価し、一般の目的推定成功とは呼ばない。runtime selectorにはgold・ケース分類・理由を渡さない。

before/only_A/only_B/both/full_afterの40probeを `go test -count=1 -json ./...` で実行し、exit/log/hash/実行test数/秒数を保存する。testなしPASSとtestありPASSを区別するが、test数を十分なcoverageの証拠にはしない。compile cacheは通常使用。source時間、probe時間、共有されるpartition audit時間を分離し、方法間で同じ測定を二重加算しない。production全経路のlatencyではない。

既存planning.Validate adapterとtemp Git tree byte検証、順次apply/逆順revert、各group単独revertの結果を保存。旧auditは通常のGo test cacheを使用するため、費用の新計測はcacheを無効化したprobeに限って解釈する。

採用Gateは8/8で分割確定・レビュー規範・構造/byte/ordered tests成立、誤分割/誤統合0、1200秒内。保留はGate成功ではない。全方式未達でも観測限界の切り分け結果を保存して研究完了とする。逐次構築・独立16file・production GOは出さない。モデル0call、retry0、各command30秒、新依存0、外部repoのcode/test実行0。既存結果・production・main作業内容は変更しない。

fixture endpointsに失敗があれば当該入力と失敗を保存して停止し、修正は再現原因特定後、再登録してから別実行する。事前登録をcommit/pushしてからfixture実行開始。
