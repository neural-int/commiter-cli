# Issue #166 出力形式バイアスの探索的診断：生成前契約

既知8caseの B1 原source×通常/逆順16件に対し、T1/T2各16件、最大32新規生成、retryなしで終了する。#165 B1の16応答を保存済みbaselineとして再利用し、再生成しない。B2・A・C・新モデル・新case・prompt探索・production変更・4file拡張は対象外。#163/#164/#165のNO-GOと本番資格は変更しない。

基点は #165 の最終checkout `4b9ac67e1cf1317986b659935c3970192e83a41f`。既存model/helper/Metal/runtime/source/tokenizerの全digestを #165 manifest と比較し一致を要求する。Qwen2.5-Coder-3B-Instruct 4bit/group64、revision `3dd939c621c08e5753d5b89f35a2642cd83b98ca`、neutral JSON grammar、native template、temperature0/top_p1/top_k0/seed144/native thought0、context16384/output1536/input最大4096、fresh helper processを固定する。モデル取得・新依存・helper再buildは行わない。

T1は `relation.enum` を逆順の `insufficient,multiple_defensible,independent,dependency,compensation,corresponding` へ反転する。systemの自然文ラベル説明順・意味・方向指示、schemaの他項目、user全文は変更しない。schemaにはsystem文字列とhelperのgrammar引数の二重表現があり、両方の同一enumを変更する。これは自然文ラベル説明順の対照ではなく、埋込schema＋grammarを一体とした順序対照であり、両者の寄与は分離できない。

T2はB1を直接基点とし `provider`/`consumer` のpropertiesとrequired要素だけを除く。自然文から `(provider/consumer direction required)` と `For nondependency use provider=consumer=none. ` の2断片だけを除く。relationのenum順/説明、reason、evidence、userは保持する。方向の評価は対象外であり、欄の消失を意味精度の改善と数えない。必要なschema/指示変更とtoken/長さの変化は一体の介入で、schema負荷・指示表面・長さを個別には同定できない。

通常/逆順は #165 B1 のuserをbyte一致で保持する。source位置・機械facts・source入力hash・期待relation/direction・規範をそのまま再利用し、未使用holdoutとは呼ばない。TTLのmultiple_defensibleはレビュー許容規範、Admissionのinsufficientは外部制約不足の診断規範であり、作者意図の真値を主張しない。8case×2提示順についてT1/T2を交互に並べ、各提示で先行treatmentを交替する。historical baselineとの時間・共存負荷の違いは残り、同時期baselineは再生成しない。

生成前に完全messages/schema/request文字列・hash・gold・差分・identity・native tokenizer実測をcommit/pushする。32件のload/tokenize-only応答はpreflight_onlyで生成0/output tokens nullを検査し、生成32callと別計上する。準備の初回1callは資源assertionで停止したがrawの保存前に終了し、原因は未確定。計測器自己検査後の1回のtokenizer-only診断は完全記録して正常終了し、最初のT1 token監査へ再利用する。残り31件を含めtokenizer呼出しは33回、生成前準備の測定修復だけを認める。未保存の初回値を復元・推定せず、生成条件/閾値/gold変更0、生成retry0を保持する。remote headとmanifest byte一致を検証し、その登録証跡をcommit/pushしてclean treeから生成する。

準備＋生成＋機械監査は1800秒、post監査用200秒を予約、各call30秒。100msでphys_footprint/RSS/lifetime peak、約1秒ごとのpressure、前後system swap、TTFT/load/prefill/decode/MLXを保存する。前後pressure/swap/footprint計測不能、digest/model/native tokens/停止契約不一致、時間/呼出し上限違反、footprint>5,000,000,000bytes、critical pressure>=4、1call swap増分>256MiBまたは各phase開始から累積>512MiBで以降の生成を停止し、未実施も記録する。上限は30秒call間で判定するためcall中の即時killを保証しない。warning level2と小さいswap増分も記録し、資源資格PASSとは扱わない。global swap/pressureをモデル単独へ帰属しない。sampling終了間際のpeak・OS cache・共存負荷が限界。RSS/MLX/Metalを加算しない。独立Metal allocationはnull。

全helper呼出しはnetwork拒否sandboxとoffline env。自己作成fixtureのsourceを入力として使い、fixtureの新規実行は必要としない。外部repo code/test・クラウド推論・実diff外部送信・新モデルdownloadなし。GitHubへ公開するのは自己作成の研究コード/fixture/診断記録のみ。API/model料金0、電力金額未計測。

評価はB1/T1/T2のlabel分布・compensation集中・case別label一致・独立識別・不要direction違反・構造validity・引用実在・sourceでのreason支持・提示順安定・latency/input/output tokens・pressure/swapを別に集計する。reasonは全48応答を同じsourceに対しCodexが `supported / partially_supported / unsupported / unavailable` に照合し、引用実在だけで真実性を推定しない。独立人間レビュー済みとは呼ばない。

全16件が契約正常で完了し、T1でlabelがB1から変われば `schema-order effect supported`、T2で変われば `field-interference possible`。変化なしなら各介入は `no material change`。停止/契約不一致/未実施なら該当対照は `undetermined`。reason/引用だけの変化はlabel効果と区別する。内部原因を唯一に断定せず、分布変化やdirection省略を正しい意味区別や救済と同一視しない。

結果を見てtreatment/gold/rubricを変更せず、本固定診断で調査を終了する。結果と未識別をIssueの所定4節で報告、全6成果条件を証拠で照合してチェックし、その後にgoal.mdの最終品質ゲートを実行する。Go test/vet/build、CLI build、既存release-notes Python tests、同digest helperの既存Swift tests、入力差分/構造拒否/引用検査、resource lifecycle、Python構文、git diff検査を保存する。追加unit tests0、productionの機能更新/削除0のため既存テストは保持する。commit/push・remote成果物一致・Issue本文チェック/closureを確認後にGoal完了を判定する。
