## 要約

可視source/test対応の未使用3syntheticでも固定score+partition候補はdirectを上回らず、同一fileケースにFM回帰。候補No-Go。全scoreが+2で、solverはその一意最適解を再現した。診断oracleは3/3だがproduction/D根拠として採用しない。

## 検証結果

| 条件 | complete | exact | FM | FS |
| --- | --- | --- | --- | --- |
| A-only direct | 3/3 | 1/3 | 48 | 0 |
| score+partition | 3/3 | 0/3 | 49 | 0 |

5file/8unitのdirect対応とshared依存は両方式FM24で全merge。同一file/2unitはdirect exactだがscore+partition FM1。全6call completed/accepted。input9985/output1185tokens、累積147.021秒。条件9ff29f5、入力検証1703832で計測前固定。全intent部分状態36はGo成功（同一file4状態はtestなしcompileのみ）。scorer/system/schema/solver/予算を初回から変更しない。

signed score誤りは24/24/1pair。全scoreが+2で入力順反転のsolver結果は同一。一意最適値56/56/2。goldから作った診断専用oracle scoreはsolverで3/3 exact。oracleはruntimeに渡さず観測結果を置換しない。

## 考察

今回の直接call対応はA-only内に見えるので、前回のhidden routingだけでは誤りを説明できない。solverのobjectiveに対する最適化は正しく、誤ったpair評価を最適化して全mergeした。ここまでのfixed candidateで責務分離による改善は未立証。構造validity、oracle回復、completeはsemantic正しさの証拠ではない。

同作者syntheticで同時係数変更が多く、独立要件goldの実repository一般化を証明しない。全pair positivityはscorerの弁別不足として観測できるが、完全なrepresentation/evidence/partitioning妥当性と実holdoutを確認しておらず、D必要性を断定しない。B No-Go、8unit外拒否、production4file制限を保持する。

## Next Steps

- 新規専用worktreeでscore contract自体のpositive/negative/unknown controlsを事前固定して検証する。全pair +2が意味判断の失敗か、採点interfaceを理解できない失敗かを切り分けるため。
- 元scorer prompt/schema/model/予算を変更せず、明示的なpurpose記述を持つ独立診断入力を使う。これはruntime入力拡張やfixtureへのgold追加ではなくscorer能力診断として別集計し、品質成功へ転用しない。
- 弁別が成立しない場合は現行scorer候補の終了条件を記録し、別architectureの必要性とD開始条件の未達を分ける。optimizer/閾値だけを既知goldへ合わせない。
