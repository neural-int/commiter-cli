# Iteration 1: ハーネス選定と取得前監査

本iterationは段階1の構造preflightと、段階2の推論前に必要な資源・取得条件の監査を行う。モデル推論は0call。preflightのPASSをモデル資格、意味品質、4file拡張の達成としない。

## 固定対象と責務境界

- goal: `neural-int/goals/goal.md@ad92785ee8b4f4d04ded9c5755c4450cf02281ca`（blob SHA）。Skill使用無し。
- base: `f275d953f0c15452e9281d1c678cb9bb20adcbbb`。
- 参照: #162 `b8a7fe14a798ff36680efffdbd879c9a02cb37f2`。GitHub branch headとローカルrefの一致を確認。過去結果・未完了項目を変更しない。
- #162の`sourcefacts/main.go`と`validate/main.go`をbyte一致で再利用。`reused-source.json`にhashと由来を保存。既存authoritative `planning.Validate()`は変更しない。
- #162の局所merge/keep_separate/deferとsource-only入力規範を再利用。ただしGemma固定呼び出し、対象pair以外をsingletonにする計画生成、テスト結果を入力に加える方式は主比較から除外。
- #162の複数許容分割、順序付き適用/revert、単独revertの別評価を評価規範として再利用。既存の依存抽出はsame-package Goのtop-level新functionと新unqualified callのみ。method/alias/任意言語の網羅性を主張しない。
- #160 A2は最終比較でLLM0callのため主ハーネスから除外。#149 H23等は過去構成の再現対照として参照し、主比較へ無条件に移植しない。
- production/default model/4file制約/SRS/stagingは変更しない。metadata生成は評価対象外。研究用の固定chore/summaryでvalidatorの構造資格だけを確認する。

## 二つの実験入口

`tools/benchmark163/harness.py`の`messages/schema/request/classify`を共通adapterとする。モデル固有chat templateの適用は既存Swift runtimeに任せ、同一system/user source情報を渡す。入力は原diff、before/after、必要な未変更コード、確認可能なsource facts。gold・テスト実測・変更内容の要約は入力に入れない。モデル選択は固定2pinのみ。

局所入口はEA/EBの2fileを判断する。出力は`decision: merge | keep_separate | defer`と`evidence_refs`。mergeには両IDが必要。keep_separateに明示的なAPI依存があればhostが確認して順序を付ける。deferは未解決として保持し、fallbackが安全な計画でも元応答の意味成功に加算しない。対象2file判断は全8/16file能力とは別。

全体入口は全8または16fileの原diffから直接`decision: partition | defer`、`groups`、`evidence_refs`を返す。groupは非空、全selected file IDをexactly-onceで割当て、適用順を提案する。根拠参照は全fileを含む必要がある。deferは空groups。対象pair・goldに近いgroupの事前指定、pair scoreの推移統合、4file window連結を利用しない。現在のfile-only contractを維持し、mixed-fileの妥当なfile partition不存在を制限として事前区分する。

構造受理後に、既存validator・使い捨てGit treeのbyte再構築・確認可能な依存順を検証する。別の評価層で許容partition・対応の誤分割・独立変更の誤統合・順序付き適用/revert・任意group単独revertを測る。これらの意味/動作評価は本iterationのpreflightでは未実施。

## 段階2の予算と進行条件

初回はGemma-4-E4B-it固定4bitとQwen2.5-Coder-3B-Instruct固定4bitの2モデルのみ。共通生成条件はneutral grammar、context16384、output1536、temperature0/top_p1/top_k0/seed144、native thought0、retry0、per-call30秒。Qwen2.5のtemplateは独立thinking切替を持たない。Gemmaはenable_thinking=falseを既存runtimeへ渡す。templateを同一文字列へ改変せず、差分を保存する。

局所資格用は未使用の自己作成10caseを2提示順で各モデル20call、合計40call上限。対応実装/assertion2、独立変更2、弱いテストでは片側誤りを検出できない補償変更2、新API依存2、複数妥当1、根拠不足1を用意する。whole1800秒（推論・事前監査・結果検証）、必須結果監査400秒を確保。近似変種・family相関、過去fixtureとの重複、識別可能性を推論前に監査する。file-only/static-new-api baselineは同じsource情報を使用し、test-assistedは追加情報方式として主比較と分ける。

各モデルの局所Gateは20/20 completed/schema/source-reference-valid、識別可能6caseで12/12許容判断、API依存2caseで4/4妥当なmergeまたは確認済み順序split、複数妥当caseの両順で許容分割、根拠不足caseの両順でdefer、独立変更誤統合0、必要対応/補償変更誤分割0、10/10提示順一致、対応caseでstatic baseline改善、全資源/時間条件成立。これは全体試験開始条件だけでありproduction資格ではない。

上記は入口・予算・規範の固定であり、局所モデル比較事前登録の完成ではない。具体的source/gold/許容partition/input/prompt/schema/template/model/helper/binary hash、実token長、資源負荷条件を追加commit/pushするまで推論禁止。モデル取得も容量・条件への承認前には行わない。結果取得後のgold/prompt/閾値変更は禁止し、変更は別契約/結果集合とする。

## 資源計測の主判定

主指標をhelper processの`proc_pid_rusage(RUSAGE_INFO_V4).ri_phys_footprint`の採取最大値とし、5GBは5,000,000,000 bytesとする。3GB未満は失格にしない。100ms間隔で起動→ロード→prefill→decode→終了を採取し、可能なin-process peak値が得られれば別記する。サンプリング間の未観測ピークは限界として明記。RSSとMLX active/peak/cache、Metal割当は別列で保持し、重複量を足し合わせない。MLX以外の部分量や保存容量だけでGOとしない。

`vm_stat`、`vm.swapusage`、`memory_pressure -Q`を試行前/中/終了後に記録。開発アプリを停止せず現在の共存負荷を記録し、モデル切替時の負荷差も残す。swap増加、高pressure、5GB超過、実input+output超過、メトリクス不足は資源資格を与えない。ロード/prefill/decode/解放の時間・最初の生成tokenまでのTTFTとtotal latencyを区別し、未観測値はnullとする。現在のhelperにはTTFT/MLX割当の応答項目がなく、計測adapterの準備とhash固定が推論前に必要。

## 段階3の独立契約

局所Gate成立した構成だけ、未使用8file/16file各4case・2提示順・最大16call/model、per-call30秒、whole1800秒、retry0、context16384/output1536で進める。局所corpusを全体資格入力へ再分類しない。実fixture/gold/input長、採用独立性、確定したhelper/負荷条件は局所結果後に別の事前登録としてcommit/pushする。成立する前に複雑な全体plannerや入力corpusを実装しない。

全体Gateは16/16 completed/完全割当/validator/byte再構築/確認済み依存順、16/16許容partition、negative誤統合0、必要対応誤分割0、8/8提示順一致、事前資源条件成立。file-only/static baselineとの差、保留、tokens/calls/latency、順序付き適用/revertと単独revertを別指標で記録する。未使用repository/変更family等の独立性が成立しない入力は採用holdoutと呼ばない。NO-GO時はgated-out原因・未立証能力・再開条件を保存し、4file拡張達成とは扱わない。

## 構造preflightと再現

初回25checkを保存し、timeout/context超過の独立分類を追加した最終27checkを実施。不正JSON/重複JSON key、schema、不正/不足/重複file ID、根拠ID、非completed、未解決、既存validatorの拒否、Go AST source、forward/reverseのGit tree byte一致、hash不一致のsource拒否を記録。元worktree/indexでのGit mutationはない。合成応答・自己作成sourceのみでモデル0call、外部repoのsource/test実行無し。

```sh
GOCACHE=/tmp/issue163-go-cache GOPROXY=off GOSUMDB=off go build -o /tmp/issue163-validate ./tools/benchmark163/validate
GOCACHE=/tmp/issue163-go-cache GOPROXY=off GOSUMDB=off go build -o /tmp/issue163-sourcefacts ./tools/benchmark163/sourcefacts
PYTHONDONTWRITEBYTECODE=1 python3 tools/benchmark163/harness.py --validator /tmp/issue163-validate --facts /tmp/issue163-sourcefacts --output /tmp/issue163-preflight.json
```

outputは排他的作成。過去成果物を上書きしない。raw応答と停止理由を保持し、timeout/context超過/helper停止/invalid JSON/schema/assignment/根拠不正/未解決/計画監査失敗を別分類とする。未観測tokensはnull。source/templateの完全性は、推論前の新しい固定入力監査が必要。
