# Issue #166 出力形式バイアスの探索的診断

## 要約

**enum順を逆転しても、方向欄を削除しても、全16応答がcompensationのまま維持された。両介入のlabel効果は `no material change`。** ラベル一致は各2/16、独立識別は各0/6。方向欄削除は意味識別を改善せず、引用と理由に副作用を伴った。固定32新規生成・retry0でこの調査を終了する。

## 検証結果

| 指標 | B1（既存） | T1 enum逆順 | T2 方向欄削除 |
|---|---:|---:|---:|
| compensation件数 | 16/16 | 16/16 | 16/16 |
| relation一致 | 2 | 2 | 2 |
| 独立の正しい識別 | 0 | 0 | 0 |
| 構造valid | 16 | 16 | 16 |
| 全引用実在の応答 | 7 | 7 | 3 |
| 実在引用claim / 全claim | 10/22 | 9/21 | 20/42 |
| 不要direction違反 | 16 | 16 | 評価対象外 |
| input tokens | 16044 | 16044 | 15276 |
| output tokens | 1681 | 1586 | 2445 |
| TTFT中央値 秒 | 6.421 | 5.112 | 5.227 |
| process wall中央値 秒 | 10.397 | 8.256 | 10.097 |

ラベル一致2件はboolean codecの通常/逆順だけ。全条件のlabel提示順安定性は8/8だが、一律誤分類の安定性を含む。B1/T1の期待direction一致は2/16（APIのEB→EA）だが、返したlabelはそこでもcompensationなので出力規約に対する不要direction違反は各16/16。T2ではdirection欄の存在は0/16であり、違反消失をスコア改善と数えない。

T1はB1比でreasonが5応答、evidenceが2応答変化。T2はreasonが12応答、evidenceが14応答変化。T2のinputは各48token減、全体768token減だが、outputは1681→2445tokenへ増加した。全引用実在の応答は7/16→3/16へ減少した。一方、引用claim総数が22→42へ増えているため、実在claimの10→20という増分を意味理解の向上と扱わない。

| case | 期待relation | B1 全引用実在 通常/逆順 | T1 同 | T2 同 |
|---|---|---|---|---|
| nil-slice-corresponding-assertion | corresponding | ×/× | ×/× | ×/× |
| shared-bound-independent-ui-and-batch | independent | ○/× | ○/× | ○/× |
| same-package-independent-time-and-permission | independent | ○/× | ○/× | ×/× |
| shared-import-independent-search-and-sort | independent | ○/× | ○/× | ×/× |
| boolean-polarity-compensation | compensation | ×/× | ×/× | ×/× |
| new-round-provider-eb | dependency | ○/○ | ○/○ | ×/× |
| cache-expiry-multiple-boundaries | multiple_defensible | ○/× | ○/× | ×/○ |
| unknown-admission-contract | insufficient | ○/× | ○/× | ×/○ |

全caseのB1/T1/T2・通常/逆順で返したlabelはcompensation。完全回答、引用source/version/line/quote、個別方向、hashは `case-evidence.json` と全rawに保持する。引用の空白や引用符を正規化して成功へ変更しない。

全48reasonを同じsourceに照合したCodex監査では、B1 supported2/partial5/unsupported9、T1 supported2/partial1/unsupported13、T2 supported0/partial10/unsupported6。これは独立人間レビューではない。partialはsourceで支持された下位記述と誤った共同契約主張が混在する分類であり、relationが正しいという評価ではない。T2のBatchSizeに+2を加えたという2件の理由はsourceと矛盾する。APIのTotalをRoundToTenで置換したとの説明も不正確で、実際は既存Totalの呼び出し先が変わる。codecの両関数return valuesをswapしたとの説明は、int対応の反転とbool条件変更の差を曖昧にしている。

生成32callは全てexit0/completed/構造valid、model/入力tokens/hash一致。runは307.944秒、初回中断と測定修復を含む記録準備は201.152秒。記録phase合計509.095秒は1800秒以内で、契約作成前の作業や終端記録・最終品質ゲートまで含む全作業wall timeではない。準備timerはbaseline-reference保存以降。準備の33tokenizer callと生成32callを分けて計上する。

生成footprint最大2,794,293,576bytes（約2.79GB）、lifetime観測最大も同値、全call5GB以内。全32callにpressure warningを観測し、正常pressure資格は成立しない。swap増加call0、system usedは4020.06M→4012.06M。global値をmodel単独へ帰属しない。RSS/MLX/Metalは加算しない。独立Metalはnull、sampling終了間際のpeakや共存開発アプリは限界として保存する。API/model料金0、download0、新依存0、電力金額未計測。

準備の初回tokenizer-only callは資源assertionで停止し、raw保存前に終了したため原因と観測値は未確定。値を復元していない。測定器自己検査後のtokenizer-only診断は観測保存と正常終了を確認し、最初のtoken監査へ再利用した。残り31件と合わせてtokenizer33call。結果に合わせた生成条件/gold/閾値の変更と生成retryは0。この欠測は資源監査に残る。

network拒否sandbox/offline envで全helperを実行。自己作成source以外のbenchmark code/外部repo/新fixtureの実行なし、クラウド推論/実diff外部送信/新モデル取得0。研究専用docs/tools以外の変更0。main HEADと既存.gitignore差分を保持し、production/default/4file/依存を変更していない。

## 考察

H1について、この固定schemaのenum逆順だけでcompensation集中が変わる仮説は支持されなかった。schemaはsystem埋込とgrammar引数の二重表現で両方を変えているが、自然文のラベル説明順は固定である。その説明順の効果、別の順序、decoderの内部選択、学習済み形式への偏り、source解釈は未識別。

H2について、方向欄と整合に必要な指示2断片の削除だけで関係labelを救済する仮説は支持されなかった。欄の消失は構造上期待通りだが、独立変更をcompensationにする誤りは継続した。必要な指示修正・長さ・引用数の変化を伴うため、schema負荷とprompt表面/長さを個別に同定できない。理由の記述量や部分的なsource認識の変化を、正しい関係識別と同一視しない。

全比較はhistorical B1と既知8caseの探索的診断。共存負荷/cache/実行時期の差、自然文の固定説明順、両schema表現、seed1種類、他の順序/介入未試験が残る。速度の差を一般的効率改善と断定しない。TTLは複数許容のレビュー規範、Admissionは外部制約不足であり、作者意図の真値を推定していない。内部推論の唯一原因、あらゆるschema変更の無効、一般モデル能力の限界を断定しない。

## Next Steps

- 本固定診断で調査を終了し、追加生成・prompt調整・case追加・新モデル・C対策へ拡張しない。2介入の出力分布不変と副作用を最大32call内で記録でき、追加研究はこのIssueの範囲外のため。
- 本結果を保存してIssueを終了する。全6成果条件と本文チェック、後続の最終品質ゲートが検証済みであり、本固定診断の追加生成は不要のため。
- 最終commit/push、GitHub成果物の一致、Issue closureを確認する。localの成果だけでは外部への保存と終了状態を証明できないため。

生成前登録 `1217e86b4a7ae99a58f81a69c4642da1944403d6`、生成時HEAD `e1727ac512d60228a788ffbdd6dcb3d4e3da1c51`。再監査は同digestのローカル資産で `PYTHONDONTWRITEBYTECODE=1 python3 tools/benchmark166/audit_results.py`（既存派生ファイルはexclusive-createにより上書きせず、別copyで比較する）。原raw・登録・goldは変更しない。#163/#164/#165のNO-GO、本番資格、4file上限を保持する。

## 最終品質ゲートと保存

Issue本文の6チェックを確認した後、Go test20package、Go vet/build、CLI build、既存release-notes Python tests25件、同digest helperの既存Swift Testing32件（8+24）、入力差分/5構造拒否/不存在引用拒否、旧構造preflight、resource lifecycle、Python構文、diff検査がPASS。全48rawから再集計したsummary/case-evidence/result-auditは保存済み派生ファイルとbyte一致。生成の追加0、unit tests追加0、不要テストなし。

SwiftPMは外側network拒否sandbox内で自身のsandbox適用が拒否され、テスト開始前に一度失敗した。初回ログを保存し、SwiftPMの内側sandbox適用だけを省いた再検査では、外側deny networkを維持して32件PASS。skip-buildと自動依存解決禁止を維持し、テスト/validatorを無効化していない。helper/model/source digestも一致。`final-quality-gates.json` に初回失敗と最終PASSを併記する。

GitHub上の結果commitについてreport/raw/case別証拠/集計/reason/登録/完了監査/資源/安全の9artifactを取得し、localとのbyte一致を検証した。最終品質証跡は後続commitへ保存し、remote headとtreeも最終監査する。成果の保存とIssue closureをもってこの診断を終了し、production採用や4file拡張は行わない。
