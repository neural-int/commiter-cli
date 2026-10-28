## 要約

#180の検証専用実装を用意し、B/Cの一巡を開始する。モデル/tokenizerはCでGroupingから保持する。Dはchat template後の共通prefixが両fixtureとも先頭4tokenのみで、共通Raw Diff/履歴まで到達しないため不成立となった。thinking設定を変更せず、DのKV実装・生成は行わない。

## 検証結果

- 専用branch codex/issue-180-prefix-cacheを作成。#178の未commit入力契約と実測を引き継いで目的別にcommitし、元worktreeとmainの変更は保持した。
- helperに明示的なbenchmark専用経路を追加した。Bはphaseごとにモデルをload、Cは一回のcycle内でmodel/tokenizerを保持する。grammar/native状態・生成KVは各要求で新規作成する。load/input prepare/grammar setup/generation wall、依存runtimeのprompt/generation時間・token数、MLX allocator active/peakを記録する。
- productionの通常一要求protocolと120秒期限は維持した。検証binaryだけに適用するGo overlayで共有deadlineを外し、外側test runnerもtimeout=0とする。
- 実際のGit fixtureからproductionの収集・Raw Diff投影・準備を実行した。Grouping要求は#178の履歴あり要求と両ケースで一致し、BのCategory/Textの共通2messagesが一致することを機械確認した。schema/生成予算を保持した。
- 実token化ではC01=2610/2526tokens、C02=18360/18276tokens。共通先頭は両方4tokens `[2,105,9731,107]`、5番目はCategory=98（thinking marker）、Text=10896で異なった。共通system本文やRaw Diffより前の分岐であり、配置変更だけでは目的のKV再利用が成立しない。実token化にモデル生成は含まない。
- go test ./...、go vet ./...、go build ./...が成功。Swiftはprotocol10件＋bounded generation27件が成功。保持した同一helperへの逐次通信とcancel終了の回帰検証も成功した。
- 全Goテストの初回は過去保存したsource-snapshot配下のGo複製が独立packageとして探索され失敗した。内容を変更せず_source-snapshotへ移し、Goの探索対象から除外して再実行した。初回失敗logも保存した。

## 考察

モデル保持とKV再利用は成立条件が異なる。モデル保持Cはthinkingの違いに依存せず評価できるが、Dには共通事実入力までの一致が必要である。先頭4tokensだけのcacheを実装してRaw Diffの再計算を省けたと報告しない。

Aは#178 iteration03のC01-B84.51秒の完了とC02-B120.28秒の途中停止をそのまま参照する。今回のB/Cはtimeoutなし、後段のグループも固定するため、特にC02のA→Bの時間削減率や完全な品質比較は算出しない。

## Next Steps

- C01-B→C01-C→C02-B→C02-Cを各1回、最大12生成要求で固定し、順序・source/helper/Gold/入力hashを推論前に保存する。失敗phaseではcycleを止め、残りの登録済みcycleだけを実行する。retry/repair/judgeなし。
- 実際のGroupingは記録・妥当性確認した後、固定参照partitionを後段へ渡す。通常のGrouping成功と扱わず、B→Cでモデル保持の品質・時間・メモリを比較する。
- Dはnot_evaluated_prefix_gate_failedとして保存する。新しいthinking設定やtemplateの変更による救済は今回行わない。
