# Expression ablation preregistration

未使用3syntheticを固定: 局所単一定義のassertion、兄弟call引数の独立diagnostic、再代入後の独立diagnostic。既知statement-guardrails2件は回帰として別集計。A-onlyとexpression条件は同一container/metadataで、観測配列だけが異なる。初期定義のみの構文観測で、unknown/空callsは無関係の証拠にしない。source/test graphは両条件で同一とし、式依存の増分だけ比較。goldは評価側のみ。

10比較行/最大10call、context16384/output1536/timeout120、既存model/helper/schema/system固定、retry/repairなし。fresh exact改善、FM/FS非悪化と既知回帰非悪化を採用候補条件とする。ただし全B Goはlegacy/new-file/rename/history/memory/cacheも必要。2file synthetic成功をmulti-file全体Goへ転用しない。
