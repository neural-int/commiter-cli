# Joint impact soft-feature ablation

既知4反例/回帰と未使用3synthetic transfer casesを固定。未使用はtable-driven共有assertion・相殺・comment absenceの異なるsymbol/値/文言であり、real repository holdoutとは呼ばない。

A-onlyはinline unit＋selected before/after＋AST annotation。impact conditionはgoldなしの単独/pair test結果と未変更test sourceを追加。predicateでpartitionを強制しない。同一schema/system/model/helperのまま14call、context16384/output1536/120sec、repair/retryなし。従来65unit拒否は未解決として保存する。

独立3件でexact増加、aggregate FM/FS減少、各反例でFM/FS悪化なし、completeな受理を追加価値の条件とする。未使用3件のGo probeを先に実行し、その結果を入力として固定してからmodel比較する。要求/IntentEdits/Goldをモデルへ渡さない。モデル結果でfixture/feature/promptを変更しない。B全達成条件のnew file/rename/memory/cacheは依然別途必要。
