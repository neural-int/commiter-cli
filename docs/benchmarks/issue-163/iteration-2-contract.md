# Iteration 2: 原sourceを共有するGemma/Coder局所比較

本契約は初回2モデルの未使用10case×2提示順の比較に限定する。生成前のnative template/token長監査を含む準備証跡を保存し、最終manifestをcommit/pushしてから生成する。モデル40call上限、retry0、per-call30秒、context16384/output1536、neutral grammar、temperature0/top_p1/top_k0/seed144、native thought0、whole1800秒、mandatory post-audit400秒予約。前iterationの局所Gateと全体資格を変更しない。

## 固定入力と評価規範

`iteration-2-preregistered.json`に原diff付きのprompt/schema、全source/gold、許容判断、source/input/prompt/hash、2モデルpin、実験helper/Metal/AST/validator hashを保存する。変更内容の事前要約、gold、テスト結果をモデルへ渡さない。両モデルに同じsystem/user情報を渡し、native templateによるtoken長差は`iteration-2-token-audit.json`へ記録する。出力metadataは未評価。

caseは対応実装/assertion2、独立変更2、弱いテストで誤りを検出できない補償変更2、新API依存2、複数妥当1、根拠不足1。ネットワーク拒否sandboxの自己作成Goのみでbefore/only-A/only-B/afterと計画の適用・順序付きrevert・単独revertを監査済み。弱いテストの全PASSとレビュー独立性を同一視しない。補償caseはsource-visibleなroundtrip invariantをjoint-review根拠とする。外部policy不足caseは政策実装・制約が与えられていないため、この診断ではdeferを期待する。作者の過去commit境界の再現ではない。

#162 Iteration2とのchanged before/after source hash重複は0。未推論の自己作成sourceであり、根拠監査後は将来の未使用holdoutへ再分類しない。対応実装/assertion・API依存は過去と同じ課題family、補償2caseはroundtrip familyで相関する。独立作者/repositoryの採用holdoutとは呼ばない。比較の差をコード学習量だけの因果効果としない。

機械的対照はfile-onlyとstatic-new-api-order。Go ASTが確認する新top-level functionと新unqualified callだけでprovider→consumerを順序化する。shared helper/packageは同目的の証拠ではない。独立/複数妥当はsource規範で評価し、テスト単独PASSをgoldの万能な証拠にしない。test-assistedはこの比較へ混ぜない。

## 実験helperと資源規範

#156の公開source archiveを再現した別のhelperに、Coder固定pin許可、観測wrapper、load/tokenize-only入口だけを追加した。mask/commit/copyの生成処理と予算/seed/temperatureは元のneutral profileと一致する。既存source/helperを上書きせず、既存依存とMetalを再利用した。archive省略のtest target不足、およびpreflight CodingKey置換のbuild失敗は、生成0callで修正し元provenanceとlog digestを保存した。

`token_audit.py`はmodelをロードしnative templateを適用して全token長と出力予約を確認し、generationへ進まない。空objectはpreflightの固定応答であり、モデル生成の正常完了やgrammar受理ではない。40回のload/tokenize/helper callと、40call上限の生成を分ける。ロードのみの観測をprefill/decode込みの資源資格にしない。

各生成は新しいhelper processで起動し、終了解放までtotalを測定する。`proc_pid_rusage(RUSAGE_INFO_V4).ri_phys_footprint`を100ms間隔で採取し、主指標は採取最大値<=5,000,000,000 bytes。RSS、API lifetime peak、MLX active/peak/cacheは別列とし足し合わせない。MLX内Metal allocatorの値とprocess footprintは重複し得る。Metal固有の別割当APIは未採取（null）。サンプリングと終端intervalの未観測ピークの限界を残す。

TTFTはrequest validation後のhelper handle開始からneutral grammar processorの最初の`didSample`に渡されたtokenを観測するまで。最初のdecoded chunkとは別値。load、prepare、runtimeが報告するprefill/first-token区間とdecode区間、外側process walltimeを別記する。helper外側startup時間をTTFTに加えない。未観測値はnull。

開発アプリを終了せず現在の共存負荷を保持する。`vm_stat`、swap、`memory_pressure -Q`を前後に記録し、`kern.memorystatus_vm_pressure_level`を前・実行中1秒間隔・後に採取する。dispatch level1がnormal、2 warning、4 critical。全採取値normal、swap増分<=0、footprint上限、TTFT可測を資源Gateとする。未知メトリクスは不合格。事前にもwarningを観測したため、結果でwarningが続いてもモデルが発生させたと帰属しない。現在の負荷で資源GOを与えない根拠と、モデルの増分を区別する。

圧力値の解釈はApple XNUの`sysctl_memorystatus_vm_pressure_level`とdispatchへの変換を確認した（https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/kern_memorystatus_notify.c）。保存容量1.75GBだけでruntime資格を認めない。

## Gate・費用・停止

各モデル20/20 completed/schema/evidence-ref-valid、20/20許容判断（根拠不足のdefer2件を含む）、独立変更の誤統合0、対応/補償の誤分割0、提示順一致10/10、確定案のauthoritative/byte/tree/ordered-test全PASS、対応assertion2case両順でstatic baseline改善、資源Gate、whole予算内で局所GOとする。deferを含むresponse-validと、確定partitionのcoverageは別に報告する。

初回source監査費用はログ作成〜manifest保存のwall spanを切り上げ+1秒とした保守的な65秒。native template監査時間を追加し、生成/post-auditのdeadlineからこれらを差し引く。buildとdownloadは別の準備費用。input/gold/閾値を結果後に変更しない。元応答の停止/timeout/JSON/schema/assignment/根拠不正/未解決を保存し、fallbackは意味成功へ加算しない。

局所GOが成立した構成だけ、別の未使用8/16file全体契約へ進める。未達ならgated-out原因と不足する独立入力・全体能力・再開条件を保存する。本iterationの2file判断とsource-only成功は8/16file拡張、metadata、production資格ではない。
