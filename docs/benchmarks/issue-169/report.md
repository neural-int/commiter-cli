# Issue #169 有限A/B/C比較の結果

Gate 1は **NO-GO**。B/Cは構造化EvidenceのAより具体的な変更目的coverageを改善したが、purpose list exact、過剰分割、grounding/hallucinationの絶対基準を満たさなかった。C対Bの事前登録した増分価値も未達。Phase 2の直接Grouping、file partitionのFM/FS、fallback、Metadata、productionは未評価として停止する。研究記録の完了であり、製品採用成功ではない。

## 観測結果

固定Gemma E4B 4bit、同revision/seed144/native thought512/output768、同じsystem/schemaとGit変更を使用した。known3＋holdout24の計27生成、retry/repair/第二LLM検証/prompt調整0。全27callでcompleted/schema valid、Git保持、preflight/実生成token数の一致を確認した。v1自己参照pinの停止はhelper起動前で、生成やprocess attemptの失敗に数えない。

holdoutは新規合成4fixture・7gold目的、通常/逆順の8観測/arm。提示順を独立8fixtureとして扱わず、以下の目的分母14は7gold×2順の反復観測である。両順ともexactだったfixtureはB/CともH01だけで1/4。

| 指標（holdout） | A: Structural | B: Raw Diff | C: Raw Diff＋所属宣言 |
|---|---:|---:|---:|
| 具体的目的coverage | 0/14 | 13/14 | 13/14 |
| purpose list exact | 0/8 | 2/8 | 2/8 |
| target対応coverage（具体的変更説明とは別） | 13/14 | 13/14 | 14/14 |
| purpose over-merge / over-split | 0 / 7 | 0 / 5 | 0 / 4 |
| grounded説明 | 17/20 | 16/18 | 14/18 |
| hallucination説明 | 3 | 2 | 4 |
| semantic対応の順序差fixture | 1/4 | 2/4 | 2/4 |
| input / output tokens | 7,630 / 4,862 | 6,214 / 4,954 | 6,654 / 4,899 |
| helper wall p50 / p95 秒 | 48.817 / 51.770 | 48.383 / 55.856 | 45.859 / 51.969 |
| helper peak RSS 最大 GiB | 3.438 | 3.336 | 3.300 |
| 前後swap増分 最大MiB | 0.00 | 283.06 | 238.75 |
| pressure 最大 | 1 | 1 | 1 |
| 絶対意味品質 / 資源Gate | FAIL / PASS | FAIL / FAIL | FAIL / PASS |

- H01: B/Cは負値の絶対値→0のclampを両順で一つの目的として説明。Aは一般的な対象説明と実装/testの過剰分割。
- H02: B通常順はJoin separator変更を欠落。B/CのByteCount cast変更の説明には4096をpage sizeまたはrecord/block sizeと呼ぶ未根拠の意味付けがある。
- H03: B/Cは秒→millisecondのproducer/consumerを同目的へまとめず、別purposeに列挙。C通常順は>=2→>=4を<=2→<=4と逆向きに説明し、逆順では正しく説明した。
- H04: B/Cにも実装/testや同じSelect変更の重複列挙が残る。C逆順は旧PageStart入力indexをzero-indexingと解釈する説明があり、旧式(page-1)*size/旧testとは一致しない。

purpose over-splitは上記の目的列挙の過剰分割であり、未実施のfile partition False Splitではない。追加descriptionの判定には解釈が含まれる。例えばBの「zero-based offset」は旧式がone-based入力をzero offsetへ変換する説明として扱い、Cの旧「zero-indexing」は旧page入力規約の説明として区別した。各原文・gold対応・理由はholdout-evaluation.jsonで確認できる。独立人手評定ではない。

補助的に、groundingの曖昧な説明をすべて許容しても、目的数と一対一対応だけから求めるexact上限はB3/8、C4/8で、80%基準に届かない。元の採点・出力・Gateは変更していない。過剰分割とCの明示的な比較演算子誤りも独立した停止根拠である。

## B/Cの絶対基準と増分価値

B/Cともcoverageは13/14（92.86%）で90%基準に届いたが、exact2/8（25%）は80%未達。grounding95%、hallucination0、over-split0にも未達。BはH03逆順の前後swap増分283.06MiBが256MiB基準を超えた。Cは観測された資源基準を満たすが意味基準に届かない。swap/pressureはhostの前後値で、helperや共存アプリへの因果帰属はしていない。

C−Bはcovered gold0、exact cell0、over-split1件減、hallucination2件増。input tokensは+7.08%、wall p95比0.9304で費用の許容値以内だが、事前要求のcovered gold>=1かつexact>=1の増分改善はない。**B/CともNO-GO、C追加の採用根拠不足**。小標本でCの効果不存在を証明したとは解釈しない。

Bのinput tokensはAより18.56%少なかった。この小規模Go入力では構造化ノード列挙の方が長く、Raw Diffが常に入力を増量するとはいえない。一方、S16（16file×150変更行）のPurpose Discovery入力はA165,017/B47,146/C47,481 tokensで、全armが16K予算で不達。省略/縮小を行わず、生成・時間0・意味成功へ換算しない。S16は独立品質holdoutではない。

## 資源と安全性の適用範囲

Mac15,12 / arm64 / 16GiBで、各callは新helper process、既存local model cacheを使う。真のstorage coldや一定の共存アプリ負荷を再現した測定ではない。helper RSSはprocess peak、pressure/swapは前後snapshotで、短いpeakや他アプリの負荷を分離できない。全計27生成のinput22,528/output16,297 tokens、helper wall合計1,188.544秒。output数はnative thought/channelを含む計測helperの値。

mainのHEAD/index/.gitignore/statusを保持し、元fixture6repositoryの全Git/worktree receiptも終了後一致した。製品mode排他契約、Git staging/commit mutation、既定4file上限、#168 Closed/Gate 1 NO-GOは変更していない。自作fixtureの初期baseline commitと研究履歴commit以外に実ユーザー変更へのGit mutationはない。

公開source mappingのsnapshot.rootに一時絶対パスが残っていたため、生成とintegrity-audit後に公開projectionだけをplaceholderへ修正した。取得原本はlocal archiveへ保持し、変更前/後のhashと6fieldsのみの修正をpublication-audit.jsonへ記録した。初期pathはusername/秘密情報を含まない一時identifierで、Git履歴は保持する。実験時sourceは07f9c433、全pinは生成直前/各call/生成終了時に一致。現在のprepare toolはSnapshotをcopyして公開版だけ匿名化する。モデルrequest/gold/測定receipt/事前Gateは変更していない。

## Next Steps

- 今回の有限候補を停止し、Phase 2/製品接続へ進まない。絶対品質と増分基準が未達のため。
- 10達成条件を観測・条件分岐・未評価理由へ照合し、チェックの意味を研究記録完了としてIssueへ記載する。導入成功と混同しないため。
- Verification完了後にrepository test/vet/build、release notes、研究契約検査、branch差分整合を実行し、結果を専用branchへ保存する。意味品質とソフトウェア品質を別に確認するため。
- 将来の製品採用は別Issueで扱う。本Issueのsmall synthetic結果から16file約80%や別作者への一般化を主張しない。
