## 要約

新規専用worktreeでbounded UTF-8 inline edit atomsを評価した。未使用6件はline方式0/6、inline方式6/6でgold境界を表現できた。使用済み12件もinlineで12/12、#149回帰14件＋従来synthetic10件もcoverage/reconstruction/stagingを維持した。

ユーザーの明示判断により、親A gateは「既存#149は回帰検査として保持し、新規同一file/複数intent fixtureで構造制約の解消を判定する」へ更新する。今回の事前固定したbounded representation条件でAはGO。過去のA No-Go/未達記録、gold、B No-Goは変更しない。semantic grouping品質やproduction採用は未判定。

## 検証結果

- branch: `codex/issue-151-inline-operations`、worktree: `/Users/Natsuki/.codex/worktrees/issue-151-inline-operations/commiter-cli`。
- 新規fixture/事前条件固定commit: `4e88531`。候補・測定前DP評価器固定commit: `5097675`。
- 新規6件manifest: `3174e13813e9b0e99f3fc062ef898c9de014b2f0d946d97434188ddb50fa018f`。
- 新規6件: Unicode/nested/inline add-delete/CRLF/no newline/contiguous independent labels。before/afterと要求由来gold・全subset snapshotを抽出器から隔離して固定。
- file-only 0/6、line 0/6、inline 6/6。inlineで全24subset状態・12intent順序系列のGit index byte一致、disjoint complete assignment、source coverage、reconstruction、repeat extraction determinismを確認。
- 使用済み12件: inline 12/12、全44subset状態・22intent順序系列を確認。前回未達のsame-line 2件も表現可能。
- #149の14回帰＋従来synthetic10件: 24/24でcoverage/reconstruction/forward-reverse staging成功。138file / 831unit（line baselineは141unit）。
- 新規6件: line 6unit / 2,884byte / 抽出合計0.231ms、inline 42unit / 17,352byte / 0.551ms。inline add-delete 1件だけで27unit。
- 使用済み12件: inline 42unit / 17,936byte / 抽出合計1.199ms。単一synthetic実行の数値でありproduction latency推定ではない。
- モデルcall/input token/output token 0。semantic prediction exact/FM/FS/unresolvedはN/A。
- 既存7test＋新3testのPython10件成功。UTF8 byte boundary、同位置insert/delete、256unit/4096byteのfail closed、非UTF8 atomic byte保存、DPとexhaustiveの一致を検証。Python構文検査/diff check成功。

## 考察

inline方式は既存line方式の同一line制約を、fixture/path/gold依存ruleなしで解消した。新しい6件は初回評価前に固定し、結果を見て候補を調整していない。ただし同作者のsyntheticでありreal repository一般化を証明しない。

表現の原子単位はsemantic intentではなくUTF8 codepoint edit。どのatomsをまとめるべきかはB/Cの仕事である。atomの単独適用がsemanticに妥当と主張せず、gold intent集合にまとめたstage結果で評価した。

fragmentationはboundedだが、#149全体のunit数は141から831へ増えた。新規のinline add-deleteも27atomsとなる。unit ID/byte factをそのままモデルへ展開した場合のcostは未測定であり、BではA-only inputとの比較を固定して確認する。

file 1MiB / 20k line / 256unit、refinement line 4096byteという上限を保持する。超過はfail closedで、atomic fallbackによる合格化やsilent lossは行わない。非UTF8は既存atomic単位のまま保持するため、非UTF8の行内intent分離を証明したとは扱わない。

親gate変更はユーザーの今回の明示判断に基づく新しい評価契約である。旧#149のsemantic failureが改善したことへ読み替えず、過去の未達判断は当時の条件のまま残す。A GOはbounded representationを次段階へ渡せる判断であり、#150の完了やB/C gateの通過ではない。

## Next Steps

- 新規B専用worktreeで、inline A-onlyのinput costとrepository evidence追加を比較する。fragmentationが増えた基盤で独立改善があるかを確認するため。
- Bの前回No-Goを保持し、少なくとも未使用の評価要求とbounded evidence仮説を先に固定する。過去の失敗を新A GOで合格へ変更しないため。
- Bが独立改善gateを満たした場合だけCへ進む。production候補は現行baseline、semantic品質、holdout、complete partitionを別途検証するため。
