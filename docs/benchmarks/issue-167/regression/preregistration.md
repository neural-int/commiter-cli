# #143成功 / #167失敗の共有経路診断

## 対象とPhase A

実行契約は [指定コメント](https://github.com/neural-int/commiter-cli/issues/167#issuecomment-6104456189) に従う。目的は過去の有効metadataと今回の失敗を再現可能な証拠で説明することであり、Gate 2を成功扱いに変更することではない。

#143の `TestGemmaThreePhaseDefaultPlanning` の第1サイクルに使った `sample.go` のreturn 1→2入力、固定configとCLI adapterはrevision `59391b6392d4118bfc333532cc9b1d66f1f175c4` から再取得できた。#167の負対照は `one-independent` の境界 `<`→`<=`、1 fileである。同一入力の両経路requestは、LLM 0のprotocol doubleで3 stageすべて同じbytes/hashだった。PreparedにはCLIのrelation contextと計測経路の差があるが、この1 fileの実requestには到達しなかった。mockの成功は実モデルpositive controlに数えない。

helper本体・Package.swift/Package.resolved・CLI adapter/config・Go validator・contextinput/syntaxは上記revisionと同じ。Three-phaseの変更はmetadata/invocationの共有関数への抽出。計測helperの差分は数値telemetryで、sampling/schema/decoderの変更は確認されなかった。cached13依存のHEADはpin一致、tracked差分なし。歴史的なhelperバイナリhash・実行時負荷・完全な当時のrequestログは残っていないため、当時の全環境を同一再現したとは主張しない。

静的監査では原因を確定できなかった。以下の小さな対照を実行する。新model/decoder/stage、修復、再試行、summary切詰め、validator緩和は行わない。

## Phase B — 最大4 cycles / 12 calls

| cell | 入力 | 実行経路 |
|---|---|---|
| A | #143の1file smoke | 製品CLI `generateCommitPlan` + 製品helper |
| B | 同じsmoke | #167 `prepare` / Three-phase / measuredBackend + 計測helper |
| C | #167の1file境界変更 | 製品CLI `generateCommitPlan` + 製品helper |
| D | 同じ境界変更 | #167 `prepare` / Three-phase / measuredBackend + 計測helper |

- A→B→C→Dを各1回、順次実行。最大3 stage/cell、共有120秒。host観測は180秒で上限を持つ。成否にかかわらず同じcellを再生成しない。
- 同じローカルcacheのGemma E4B 4bit、revision `475b9088d29754a3379866cf5aeb6b41acd313c2`、16K、temperature0 / top-p1 / top-k0 / seed144、native512/384/0、output768/512/768を固定。英語、1 file / 1 group、host evidenceなし、圧縮なし、feedbackなし。
- 各callはhelperを新プロセス起動。モデル常駐なし、filesystemは既使用cache。真のstorage coldではない。共存アプリ負荷は固定できないのでpressure/swap前後差を記録し、原因と断定しない。
- A/Cは既存の製品Go adapter/clientを実行し、ローカルproxyがrequestをそのまま製品helperへ転送する。proxyはJSON responseを変更せず、数値/hashだけ記録する。doctorの追加3段probeは省略する。B/Dは既存計測backendを使う。
- 入力・request/messages/schema・profile・モデル/予算のhash、group数/順序、stage stop、typed JSON/重複/欠落/未知の数、scope/summary最大文字数、Go validator、wall、RSS（計測helperはMLXpeak/token/TTFTも）、元Git状態の前後digestを保存。生成本文/思考/helper stderr/request本文は保存しない。
- JSON typed decodeをauthoritative validatorの代用にしない。失敗時のgroup数0は完全planが返らない結果であり、実commit数の評価ではない。前回36/8 runへ混ぜず、既知入力の有限診断とする。

## Phase Cと停止判断

- A/B成功・C/D失敗なら入力に依存するmetadata規約違反をstage別に説明し、helper差を示す根拠がなければPhase Cを省略する。
- A/C成功・B/D失敗等の実行経路差が疑われる場合だけ、同じシリアライズ済みrequestを両helperへ1回ずつ与えるPhase Cを事前登録する。今回の4-cycle枠には無断追加しない。
- 全失敗なら推論を増やさずpositive controlの環境/実行物を再監査する。その他はstage/stop/requestの一致・不一致から必要な単一条件を選ぶ。原因未特定は未特定と記録する。
- Gate 2未達、Phase 3 gated-out、production NO-GOを維持する。完全plan成立と元Git状態保持を個別に記録し、packet増加の成功率・120秒・FS評価をこの1file診断で代用しない。
