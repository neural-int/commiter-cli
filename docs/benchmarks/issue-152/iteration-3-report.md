## 要約

新A（inline ChangeUnit）上で、最大2hopのsymbol-call evidenceをA-onlyと比較した。independent3件は両条件ともexact0/3で、repository追加はFM9→14、FS9→6となった。固定仮説はNO-GO。A GOは維持し、B全体は未達としてCへ進まない。

## 検証結果

- 新規専用worktree: `/Users/Natsuki/.codex/worktrees/issue-152-inline-evidence/commiter-cli`、branch `codex/issue-152-inline-evidence`。
- 新5fixture/事前条件固定commit `dd8b9a4`、正式評価器/pins固定commit `1e487dd`。入力SHA256はiteration-3-pins.jsonへ保存。
- Qualification2 / independent3 / #149 regression4、各A-only/repositoryで計18比較行。モデルcall16、host unit budget拒否2。正式16callは全てcompleted / accepted / completeでtimeoutなし。
- independent: A-only exact0/3・FM9・FS9、repository exact0/3・FM14・FS6。
- qualification: A-only exact0/2・FM6・FS6、repository exact0/2・FM0・FS6。
- #149 regression: A-only exact0/4・FM10・FS66・complete3/4、repository exact1/4・FM0・FS66・complete3/4。same-directoryはexact改善、cross-directory/multiple-intentのFSは同じ。shared-calleeは両方で65unitとなり64unit上限でcall前に拒否、gold品質指標はnull/N/A。
- 全9件の観測集計: A-only exact0/9・FM25・FS81、repository exact1/9・FM14・FS78。FM/FS合計は受理済みcaseのみであり、拒否caseを成功へ含めない。
- 正式input token合計28,119、output token1,712、model wall合計352.890sec。repositoryのinput15,233はA-only12,886より2,347token増加。
- 新5件のgraph/units/evidenceを再生成し、同一入力で完全一致。出力した計15unit relationは全て評価gold上で同intent、parse unavailable0。関係の正しさとmodelのpartition品質は別の結果である。
- 新5件でbaseline / 各single intent / 全変更の25状態をtemporary Go moduleで実行し、全てgo test成功。goldのsource/test対応に対して変更状態が成立することを確認した。
- Python schema/host/evidence contract6test、Go graph build/vet、diff check成功。
- 初回事前検査がunit budgetで失敗した後、shell列が計測へ進んでしまった。3call完了＋1call中断で自分のプロセスを停止し、setup-resultsに隔離。正式測定とは混ぜない。正式16＋setup4で研究上の起動callは20。setup完了分input 4505 / output 282、中断分telemetryは不明。以降set -eを使用し、host超過をcalls0/preflight_rejectedとして保存した。

## 考察

file-IDだけの関係では区別できない同一file内のsymbol関係を、unit-levelで追加できた。しかしstructural relationが正しく抽出された事実だけでは、汎用scorerが正しいsemantic partitionを返す保証にならなかった。independentの1件ではFM低下、別件ではall-mergeのFM12を観測した。今回のevidence設計/固定model/合成入力において改善が安定していない。

全体FM/FSの低下やsame-directory1件のexact成功で、independent regressionを相殺してGoとはしない。Gold/SymbolGold/要求は抽出器・graph・model requestへ渡していない。module `fixture`のGo synthetic namespaceを解決するprototypeであり、real repositoryの全import/type/method解決は未実装。fixture名/Goldによるroutingはない。

65unitによる拒否はbounded hostが作動した結果だが、対象回帰のcomplete assignmentが未達となる事実も示す。64→65へ上限を変えて今回結果を合格へ補正せず、次仮説ではrepresentationの費用と入力契約を設計し直す。モデルcontext/output budget増加を主解決策にしない。

new file / rename / sparse history / real-size memory/cacheはまだ未評価である。固定history sourceの前回No-Goと今回symbol-call No-Goを保持し、repository evidence一般やモデル一般が不可能とは結論しない。専用scorer DはCの前提が満たされておらず着手しない。

## Next Steps

- boundedなtest-impact / counterfactual evidenceを次の独立仮説として事前定義する。call接続だけでなく、単独変更でどのtestが失敗するかという観測が追加signalとなるかを検証するため。
- 新規専用worktreeと未使用fixtureで、shared dependency由来のfalse-positive・非behavior変更・absenceを含める。単なるcall接続とsemantic境界を混同しないため。
- 計測前にatom数/入力費用を検査し、未知・予算超過は明示してfail closedとする。今回の65unit拒否を捨てず、無制限な細分化やbudget増加を主対策にしないため。
- Bの独立改善と未評価conditionsが満たされるまでCへ進まない。Goal #150は未完了・継続対象とする。
