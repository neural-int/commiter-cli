# Iteration 3: Cの依存・部分stage能力監査

BはNO-GOとして凍結済み。CはBの意味判断を採用せず、A＋#151の既知stage能力とCの開始条件を独立監査する。責務分離の合理性は、部分stageの成立と目的の自動判定が別の能力であることにある。

- #151の既存18診断でgoldのsubsetを選べるかと正逆順temp-index tree equalityを確認する。gold oracleは評価器だけが使用し、plannerの出力とは呼ばない。
- refined CUをどの目的へ置くかをgoldから決めるruntimeは禁止。計画の分割は目的の根拠があり、stage/reconstructionが成立するときだけ。unknownはfile fallback、confirmed mixedで分離不能ならmixed metadataへ退避。構造不正は停止。
- B結果を使った開始条件監査は結果後の独立holdout実験ではない。mixed labelをもつfileが独立目的を含むと実装が断定する候補は採用しない。
- Bの失敗を理由に無制限のAST拡張・prompt sweep・model sweepを開始しない。モデルcall0、既存予算不変、production不変。
- Cの意味能力GOには独立mixed-file holdoutでFM改善・FS非悪化とmetadata整合が必要。この監査だけではGOにしない。
- DはC未GOでもAだけの最終監査を行う。本候補Cが意味未立証でも、親Issueの到達可能な構成の評価は可能である。
