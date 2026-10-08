# B Iteration 3: inline ChangeUnitとbounded symbol-call evidence

## 仮説

file間のedgeだけでは同一file内の複数intentを区別できない。inline unitのAST symbolと、変更していないadapterを通る最大2hopの実call pathを結び付けると、A-onlyでは情報が不足するsource/testの対応に追加signalを得られる。call pathはsoft evidenceであり、shared calleeだけでintentをmergeしない。

Aはユーザー承認の新gateでGoした`codex/issue-151-inline-operations`のinline prototype。旧B No-Goを保存し、新A-onlyとrepository evidenceを新規専用worktreeで比較する。history sourceは今回追加しない。

## 固定入力

新5件は同じ作者が要求から作成したsynthetic。qualification2件（direct source/test・shared helper guard）、未使用independent3件（同一fileの独立3source symbolと別fileの3tests、未変更adapterによる異なる対応）。要求/SymbolGoldは評価側のみ。source/testの変更値は各fixture内で同じであり、数値だけでは対応を特定できない。baselineから未変更adapterを除くことを隠れた正解ruleとせず、A-only / repository比較の入力差として固定する。

#149の既知4class same-directory-independent-5 / cross-directory-single-intent-9 / multiple-intents-boundary-12 / shared-callee-independent-6も回帰比較する。既存exported goldは変更しない。

## 固定条件

- 同一モデル/helper/schema/system文面を両conditionで使用。旧B native0のschema-in-message契約を再利用。
- モデルQwen3-8B-4bit revision 545dc4251c05440727734bcd94334791f6ab0192、helper native0、context16384/output1536、1call/120sec、retry/repairなし。
- 9fixture × A-only/repository = 18call。fixture/conditionによりrouting/promptを変えない。
- repository graphはGo標準AST、最大64file/4MiB。unit上限64、原A抽出のfile/line/unit予算を維持する。call pathは最大2hop、出力256pairまでで超過はfail closed。
- absenceはunknown、missing/parse/ambiguous referenceからdefault mergeを作らない。unresolvedをschemaで保持しGit mutationしない。
- raw prompt/responseやmodel重みは保存しない。membership/stop/validation/token/call/costとsynthetic evidenceのみ保存。

## Gate

independent3件のexact増加かつaggregate FM/FS減少、qualificationと#149回帰でaggregate FM/FSを悪化させない、accepted completeな出力を得る場合に追加signalを支持する。独立改善なし・regressionの場合は今回仮説No-Go。全B完了にはnew file/rename/sparse-history、memory/cache/scaleの未達条件も別途確認が必要であり、この比較だけで#152全checkboxを達成としない。

測定開始前にgold isolation、symbolごとのassignment、common-helperがdefault mergeにならないこと、bounded paths/absenceとschema contractを検証する。入力とhelperのSHA256をpinsに保存する。semantic model結果を見てgold/graph/promptを調整しない。

## preflight不成立と正式測定の区別

モデル結果を見る前のpreflightでshared-callee回帰が64unitを超えた。入力・gold・モデルbudget・graphを変更せず、その2conditionをcalls=0 / preflight_rejectedとして正式結果へ保存し、Goのcomplete条件を未達とする。従って全18比較行のうちモデル呼出は最大16。shell列がpreflight失敗後も進んでしまった初回起動は3完了＋1中断で停止し、setup-resultsとして保存する。まだ未使用のindependent3件のgold/evidenceは変更せず、正式な条件固定とpins保存後に測定する。条件伝達やevidence設計の調整ではなく、host budget超過のfail-closed記録を修正する。以降はset -eで事前失敗から計測へ進ませない。
