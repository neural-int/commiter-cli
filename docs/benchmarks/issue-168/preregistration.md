# #168 Phase 1 事前登録

モデル推論前の固定契約。目的先行以外の入力変更・入力削減A/Bの再試行・製品変更は行わない。

- Gemma E4B 4bit revision `475b9088d29754a3379866cf5aeb6b41acd313c2`、#167 measured helperを固定。temperature 0 / top-p 1 / top-k 0 / seed 144、16K、native thought最大512、出力1024、120秒/call。decoder/model追加なし。`bounded-grouping`は既存helperの推論profile名であり今回の目的発見schemaと区別する。
- 既知K1/K2、未使用H1〜H6。各通常/逆順1回、最大16 calls。チューニング0回、プロンプトvariant1個、retry0。known評価を保存してからholdoutへ進む。追加LLM検証0。baseline requestも同時に固定するがGate 1前は実行0。
- H1〜H3は新規合成入力、H4は既存templateの新規値、H5/H6は公開commiter-cli履歴由来。同一作者であり別作者holdoutとは主張しない。H4のtemplate依存を分けて報告する。gold・commit message・fixture名・cohortはモデルへ渡さない。
- H6は生成前の既存baselineで16K上限を超えて受入不可。2call未到達、分母から削除しない。受入可能7fixture / 14callのみ実行予定。gold/threshold/入力を都合よく変更しない。失敗と未到達を成功/時間0として数えない。
- projectionは#167から同じ除去関数を再利用。9 Git内部fieldsとEvidenceの4位置fieldsのみ除去。双方の`files` bytes/digest一致、baselineのsystem/schemaを製品generatorからcapture。Purpose Discoveryではtask/system/schema/出力budgetが変わるアーキテクチャ差を明示する。Git snapshot・Prepared原本・source mapping・製品validator/stagingは変更しない。
- 評価単位はgold purpose。観測できる振る舞いまたは変更されたcontractを具体的に記述した目的だけを対応させる。名前を列挙しただけ、広いテーマだけはcoverageに数えない。複数goldを1目的にまとめる場合はover-merge、1goldの重複目的はover-split、未根拠の作用・旧値・作者要求はungrounded/hallucinationとする。式・値が入力で観測不能なら識別不能と記録しgoldを緩和しない。正常JSONだけを意味成功に数えない。
- オフライン評価はLLMを使わず各出力のgold対応・grounding・理由を保存。通常/逆順を独立fixture件数として一般化しない。coverage・exact・overmerge・oversplit・groundingを別集計する。
- Gate 1: 全計画入力の受入、返却schema100%、holdout coverage>=90%、exact>=80%、overmerge0、oversplit0、grounded>=95%、hallucination0、全Git保持。H6を含む未到達はGate未達。p95はnearest rankで120秒以下、helper RSS<=6GiB、全call前後swap増分<=256MiB、pressure<4、全観測が取得できること。thresholdは結果後に変更しない。
- hard stopはcall timeout120秒、call後pressure>=4・swap増分>1024MiB・資源観測不能・Git不一致。timeoutはprocess groupを終了しreceiptを保存。前後pressure/swapだけの観測なので短いpeakの未観測と他アプリ共存を限界として残す。後者をモデル単独原因と断定しない。
- Gate 1未達ならPhase 2/3、FM/FS paired比較、mixed fallback安全性、Metadata、production、16file性能を未評価として停止する。到達していないstageの成功を偽らない。
