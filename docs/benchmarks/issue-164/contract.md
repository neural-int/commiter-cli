# Issue #164: 追加したコミット分割判断規則の事前登録

未使用の自己作成8case×2提示順にbaseline/policyを対で適用する。最大32生成call、retry0。既知#163の10case×2提示順はpolicyのみ最大20生成callを追加し、固定raw baselineと探索的に比較する。全生成最大52call。Gemma・別モデル・全体membershipは対象外。未使用比較を先に実施し、既知診断の観測によるprompt調整をしない。

## 再利用と比較変数

参照revisionは `3d514898144f00e37152148e3464f6fb44e71cc5`。`tools/benchmark163/{harness,local_trial,resource}.py`、AST sourcefacts、既存 `planning.Validate()` adapter、実験helper、native template、neutral grammarをbyte一致で再利用する。モデルはQwen2.5-Coder-3B-Instruct 4bit/group64、revision `3dd939c621c08e5753d5b89f35a2642cd83b98ca`。モデル9file・helper・Metal・adapter・全sourceのSHA-256をmanifestへ保存する。既存モデルとhelperを利用し、ダウンロード・新依存・production変更は行わない。

baselineは#163の完全なLOCAL_PROMPT。既に実装/test対応、独立性、shared helper、defer等を含むため、規則の完全な不在対存在を比較する実験とは呼ばない。policyはbaselineへ6規則と判断順序を追加する。policyの完全な英語文字列・全system/user/schema・SHA-256・入力source・before/after・raw diff・gold・facts・native token数は `preregistered.json` に保持する。モデルへgold、変更の事前要約、観測test結果は渡さない。schemaは変更せず、出力はdecisionと変更source IDのみ。

比較の差はsystem promptに追加した判断規則とそのtoken費用。固定同一source/schema/runtime/decoder/native templateで、temperature0/top_p1/top_k0/seed144、native thought0、context16384/output1536、各call30秒。input native token上限4096で入力を省略しない。load/tokenize-only監査52helper callは生成0callとして別記する。全準備監査・52生成・結果監査に1800秒、post-audit400秒予約。build/downloadを行わない。各提示内でAB/BAを交互に実行し、両armを非並列で測定する。

## 未使用性・規範・source監査

8caseは直接対応1、独立変更3（shared helper・同package・shared import）、補償1、新API依存1、複数許容1、根拠不足1。事前根拠はmanifestのcase別gold_basisに固定する。各sourceはこの事前登録で新規作成し、生成履歴はない。#162/#163の変更sourceのbefore/afterとの完全hash一致を検査する。ただし自作・課題familyの相関があり、独立作者/repositoryの採用holdoutとは呼ばない。近似familyの相関をmanifestへ明記する。

直接対応はnil sliceとassertion、補償はboolean codecの極性と明示されたroundtrip契約。両partial stateのtest失敗は必要対応の補助根拠になる。独立変更は異なる観測出力と適応不要を根拠にし、testなし・別file・単独test PASSだけから独立性を推定しない。API caseはprovider EB→consumer EA順の分離と統合を許容する。複数許容caseは作者意図を一意推定しない。根拠不足caseは外部Admission制約が入力に無いためdeferを要求する診断規範であり、未知の作者境界の正解を主張しない。

自己作成Goのみ、ネットワーク拒否sandboxでbefore/only-A/only-B/after、merge/ordered-splitのauthoritative validator、完全割当、byte/tree再構築、順序付きapply/revert、単独revertを保存する。構造・test PASSと意味妥当性は別列。既知の不正JSON/schema/source ID/完全割当/再構築拒否を#163の27checkで再検証する。モデル実行もネットワーク拒否sandbox・offline環境で行う。外部repositoryコード/testは実行しない。

## 事前Gate

未使用16応答/armだけで次段の局所試験GO/NO-GOを判定する。以下の全条件を必要とする。

- 両arm16/16 backend completedかつschema/source-ref valid。期待deferのcompleted応答は有効だが確定coverageへ加えない。
- policy許容判断16/16。独立変更誤統合0/6かつbaselineより厳密に減少。必要対応/補償の誤分割0/4かつbaseline以下。
- 根拠不足defer2/2、他の14応答defer0。policyの確定coverageはbaseline以上。全部分離・全部deferは不合格。
- policy提示順一致8/8。両armの確定案すべてauthoritative/完全割当/byte/tree/順序付きtest PASS。単独revertは依存順付きrevertと分ける。
- 両armの各process採取最大phys_footprint<=5,000,000,000 bytes、TTFT可測、前/中/後pressure全1、swap増分<=0。未知メトリクスは不合格。RSS/observed lifetime peak/MLX/Metalは重複するため加算しない。100ms sampling、終了間際の未観測ピーク、現在の開発アプリ共存負荷の限界を残す。保存容量と実行メモリは区別する。
- policy process walltime中央値<=baselineの2倍、total input+output tokens<=baselineの3倍、native input<=4096、全費用1800秒以内。
- model/helper/source digest一致、generation token長が事前監査と一致。retry/fallbackなし、失敗/未解決/rawを除外しない。

既知10caseの結果を採用Gateや未使用評価に混ぜない。歴史baselineは同時期・同負荷でなく、sandboxのネットワーク拒否追加もあるため、費用差の因果比較は未使用paired armだけで行う。小標本なのでGOでも局所の改善傾向と次段試験のみ。NO-GOなら失敗分類と再開条件を記録し、同じpromptの改変反復・事後gold/閾値変更で救済しない。

## 完了条件と境界

事前登録をcommit/pushしremote一致を確認してから生成を開始する。全8成果条件を証跡付きでIssueへ反映し、その後goal.mdの最終test/vet/build・既存release-notes tests・helper Swift tests・preflight・diff検査を通す。不要テスト・自明テスト増加と範囲を監査する。各iterationは要約/検証結果/考察/Next Stepsで報告する。#163のNO-GO、gold、production/default model/4file上限、staging、metadata、既存main差分を保持する。16file/80% exact/production品質、連鎖統合そのものの能力はこの2file診断では立証しない。
