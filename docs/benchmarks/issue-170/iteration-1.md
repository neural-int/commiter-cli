## 要約
Iteration 1: モデル生成0回で、A/B/Cの全文request、Raw Diffのpaired input、gold、有限Gate、runtime hashを事前登録した。小さい評価入力は全条件で受理された。16file stressは全条件でcontext超過のため未受理であり、意味品質の評価には使わない。

## 検証結果
- 専用worktree/branch `codex/issue-170-prompts` を#169最終headから作成。変更は `tools/benchmark170/` と `docs/benchmarks/issue-170/` のみ。
- Aは#169 H03-normal-B-discoveryの実送信system判断指示bytesを維持。B/Cは本Issue提示文と改行込みで一致。全条件へ同じ安全suffix・JSON Schema指示を付加した。
- 78 frozen requestsの監査で、同一fixture/順序/phaseのrequestはsystem以外が一致。Raw DiffはGit snapshotの完全U3差分とhash一致し、切り詰め0。
- #169 H01-H04はknown回帰専用。新規N01-N04は今回作成した合成holdoutで、実装/test/consumer、producer/consumer/test、同package/helperの独立目的、同fileの独立目的を含む。目的goldは1/1/2/2。N03/N04は過剰統合の評価対象。
- native tokenize-onlyは生成0回。holdout discoveryはA 851–1103、B 860–1112、C 818–1070 input tokens。S16はA47146、B47155、C47113で、context16384を超過し全条件未受理。
- runtimeは#169と同じhelper/model/tokenizer/source hash、Gemma revision475b9088d29754a3379866cf5aeb6b41acd313c2、seed144、temperature0、top-p1、top-k0、context16384、output768、native thought512、bounded-grouping。
- 入力同一性、hash改変拒否、Gate1 NO-GO時のPhase2起動拒否の3契約検証がpass。
- 過去source監査の初版はファイル列挙0件だったため、生成0回のまま訂正して初版も保存。#168/#169 sourceの完全一致なし、#167はGo generator内の新規target symbol不在を確認し、expanded sourceの完全一致は未検査と記録した。prompt/gold/閾値/requestは変更していない。
- 事前登録成果物: [preregistration.md](https://github.com/neural-int/commiter-cli/blob/codex/issue-170-prompts/docs/benchmarks/issue-170/preregistration.md)、[preregistration.json](https://github.com/neural-int/commiter-cli/blob/codex/issue-170-prompts/docs/benchmarks/issue-170/preregistration.json)。

## 考察
比較はPrompt全体の効果を調べるもので、個別文の単独因果効果を識別しない。normal/reverseはpaired観測であり独立標本ではない。holdoutは未生成の自作合成4件であり、設計者独立性・実repositoryの未知精度は立証できない。Aの表示上の改行は#169実使用では文間spaceだったため、実送信bytesを優先した。

## Next Steps
- known normal順12callを一度だけ実行し、凍結gold/Raw Diffとのoffline照合を保存する。回帰結果を未知精度の採用根拠にしないため。
- prompt/gold/Gateを変えずholdout24callを実行する。A/B/Cの実質改善、絶対品質、順序差、資源基準をpaired比較するため。
- Gate1 GOの場合だけGrouping24callを追加する。NO-GOならPhase2未評価として有限研究を終了するため。最大生成60call、retry/repair/追加modelは0。
