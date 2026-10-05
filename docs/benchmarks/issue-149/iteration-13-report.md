## 要約

H8 cross12のdraftを診断したところ、根拠coverage失敗だけでなくFS24のsemantic false splitが残っていた。引用の生成をhostへ移すだけの再試行は行わない。試験済み構成のproduction candidateは未選定で、探索余地が完全に尽きたとは立証していない。次の合理的なcapability検証として未取得8B checkpointを具体化したが、容量確保と取得確認が必要。

## 検証結果

| H8 cross12 diagnostic | final complete | final unresolved | final exact / FM / FS | draft complete | draft exact / FM / FS | calls | input / output tokens | wall s |
| --- | --- | --- | --- | --- | --- | ---: | --- | ---: |
| same input/schema/profile/seed | false | true | null / null / null | true | false / 0 / 24 | 1 | 4005 / 776 | 36.223 |

helper stopはcompleted、final gateは以前と同じmissing_member_evidence。metadataなし。draft診断で有効なID partitionがあってもcoverage gateを通らずfinal groupingを返せないことをtestで確認した。draftの契約文/group配列はartifactへ出力していない。元Iteration11結果は保持した。

変更したbenchmark packageのgo testはpass（31.062秒）、git diff --checkもpass。repo全体の最終品質ゲートはgoal.mdのVerificationが未達のため未実行。main checkoutの既存.gitignore変更はそのまま保持。production code/default/helperの変更なし、Skillなし、model downloadなし。

未導入capabilityの候補はmlx-community/Qwen3-8B-4bit、revision 545dc4251c05440727734bcd94334791f6ab0192、4bit。公開registryの固定revision metadataから対象9filesの合計4,623,782,544bytes（4.306GiB）を確認した。local HF snapshotはtokenizer/configだけでweightsなし。既存Swift runtime registryにはqwen3があり、新規dependency追加は想定しないが、実modelのロード・適合性は未検証。dfの現在空きは4.7GiB、容量99%。容量を取得量ちょうどまで使い切る条件では実行しない。

公開model files: https://huggingface.co/mlx-community/Qwen3-8B-4bit/tree/545dc4251c05440727734bcd94334791f6ab0192

## 考察

構造処理の役割を変更しても、観測的に共通な期限契約を複数source/test pairへ分割する問題は残った。独立定数を別groupにする根拠と、異なるentityにまたがる同じ契約をまとめる根拠の両立が現model/task構成の主要bottleneck。異なるIR・global/local役割・generation contract・cached routingを検証したが、prompt/順序の再調整だけで安全条件を同時達成する証拠は得られていない。

8B checkpointで改善するかは仮説であり、parameter数だけで優位性は保証しない。既存の3〜4B級cached routesとは別capabilityを最小資格試験で調べる余地は残るため、「全architectureが不可能」「合理的探索余地が完全に尽きた」とは結論しない。ユーザーの明示したdiff/codeのみの根拠範囲を維持し、不可観測な利用者意図の追加やgold変更で成功扱いしない。

現在の容量で追加取得を進めるには、余裕ある保存領域または容量確保が必要。既存modelやユーザーファイルを勝手に削除しない。全checkboxを埋めるための探索完了扱い、条件緩和、production上限拡大は行わない。

## Next Steps

- 追加checkpoint取得の確認と、少なくとも10GiBの空きまたは別のcache保存先を得る。4.306GiBのmodelに加えてOS/build/temporary領域の余裕を残すため。既存ファイル削除は別の明示指示なしに行わない。
- 条件が整ったら、固定revisionをgrouping-onlyのexperimental helperへ割り当て、同じweak16/cross12を各1回資格試験する。元Gemma metadata、16K/call120秒/output1536/native0、no fallback、no cloud、repair0を維持し、capability差だけの改善根拠を分離するため。
- 両方通過時のみ未推論fresh protocol8、全レンジ、旧regression、順序、final metadata/Validateへ進む。試験済み構成の失敗を無制限の同条件反復へ戻さないため。
- 容量・取得条件が整わない場合は追加capability検証を保留し、Issue/Goalを未完了のまま保持する。未検証の合理的hypothesisがある状態を探索完了へ読み替えないため。
