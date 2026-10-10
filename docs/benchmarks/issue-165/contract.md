# Issue #165 失敗段階の探索的診断：生成前契約

## 目的と責務差

既知8ケースで、事実抽出A、意味関係B、境界判断Cの検証可能な出力を区別する。内部推論過程の位置を断定しない。#155は振る舞い契約への変更帰属能力、#162は根拠に基づくレビュー・revert妥当性の規範、#163は固定小型モデルの局所資格、#164は追加判断規則の効果の評価だった。今回はモデル改善、prompt最適化、方式採用、4file拡張を目的としない。#155/#163/#164のNO-GO、旧gold、production契約を保持する。

参照checkoutは#164の最終revision `f8d21f937f550acae1f6f334a547bce9fca99649`。旧helper基盤は#163の `3d514898144f00e37152148e3464f6fb44e71cc5`、helper元archiveは `82a179a11e0c75562fadd595f208584f7a43cdb0` の `docs/benchmarks/issue-156/measured-helper-source-archive.json` と#163の追加計測/生成前token監査の記録。実際のbinary、Metal library、Swift source/Package/既存test、モデル9file、Python/Go harnessのSHA-256をmanifestに保存・再確認する。

## source監査と規範

既知8caseの完全source/hash・機械source_facts・原gold_basisをそのまま保持する。case別監査は `source-audit.json`。各監査事実Fをsourceのversion/line/quoteへ結び、関係の期待値、方向、複数許容、外部情報不足を生成前に固定する。外部作者意図を事実に含めない。

nil/assertionは同じnilnessの実装とassertionの対応。shared helper・同package・shared importの3ケースは異なる観測出力で必要適応の契約なし。boolean codecは明示された両boolean roundtrip契約への補償。new APIはEBがprovider、EAがconsumerで、mergeとprovider-first splitを許容する。TTLは倍増という共通性があるが一意の作者意図を要求せず複数許容のレビュー規範。Admissionはclient実装とcross-resource制約が欠けるため、sourceだけで必要補償・独立性を確定できない。ここはdeferを要求する診断規範であり、実際の作者境界を当てるgoldではない。

AST baselineは旧抽出器を変更しない。package/関数定義/裸identifier call/新API edgeを保存する。literal、比較演算子、assertion条件、roundtrip意味は抽出しない。selector call（strings.Contains/ToLower/Compare等）は抽出されない。この既存範囲の欠落をLLMとの付加価値比較へ混ぜない。

## 条件と出力

- A：raw diff＋before/after＋unchanged context＋既存source_facts＋行番号表を入力し、短い事実claimと実source引用、不足情報を出力。境界を尋ねない。監査済み事実は入力しない。
- B1：Aと同じsource情報、独立した関係schemaで `corresponding / compensation / dependency / independent / multiple_defensible / insufficient`、provider/consumer、短いreasonと引用を出力する。commit境界を尋ねない。
- B2：B1と同じsourceに監査済み事実を加える。事実に関係ラベル、must_merge、commit境界、test実行結果を含めない。
- C1：#164 baselineのsystem/user/schemaとbyte一致。従来の `merge / keep_separate / defer` とEA/EB引用。行番号表やA/B回答は追加しない。
- C2：C1のsourceにB2と同じ監査済み事実を追加。Bの正解ラベルや生成回答は渡さない。

完全なprompt/schema/input/token数は `preregistered.json`。各case×通常/逆順×5条件で80生成call上限、retry0。A→B→Cの回答を次へ渡す連鎖は作らず、すべて独立したfresh helper processで測る。通常順はA,B1,B2,C1,C2、逆順はA,B2,B1,C2,C1。reverseは旧規約どおりchanged_files提示順のみを反転し、EA/EB IDとsource_facts順は保持する。

過去#164 policyの不適切deferは保存済みraw履歴とcase別に照合する。今回C1はbaselineでありpolicyの再生成ではない。policyを再現した同時期のA/B連鎖とは呼ばない。

## 予算と統制

Qwen2.5-Coder-3B-Instruct 4bit/group64、pin `3dd939c621c08e5753d5b89f35a2642cd83b98ca`、既存MLX helperをbyte一致で再利用する。新モデル取得/新依存/buildなし。native chat templateは固定tokenizer、neutral JSON grammar、temperature0/top_p1/top_k0/seed144/native thought0、context16384/output1536、input native tokens<=4096。source省略・timeout retry・fallbackなし。

80 load/tokenize-only callを別計測し、生成0を確認してから全条件をcommit/pushする。生成前remote headとmanifest byte一致を確認して証跡をcommit/pushし、生成を開始する。準備＋推論＋構造監査3600秒、post監査用400秒を予約、各helper call30秒上限。実験前契約の作成時間と最終品質ゲートはこの実験時間に含めない。出力上限/timeoutの失敗は未識別として保持し、成功への置換・事後除外をしない。

100msでphys_footprint/RSS/lifetime peak、前中後pressure/swap、helperのTTFT/load/prefill/decode/MLX allocationを記録する。5,000,000,000bytesを観測上限として別集計し、pressure正常/非増加swap/TTFT可測も保持する。RSS/MLX/Metalは重複するため加算しない。独立Metal allocationは既存計測で取得できずnull。終了直前の未観測ピーク、開発アプリ共存、OS cacheは限界として記録する。API/モデル金銭費用0、download0bytes、計測時の電力金額は未計測。

全helper/必要な自己作成fixtureの動作検査はsandbox-execのnetwork拒否環境。クラウド推論、外部への実diff送信、外部repository code/test実行なし。公開するのは既存自己作成fixture・診断コード・結果。モデル外へrepo全体や秘密情報は送らない。

## 評価・原因分類・終了

Aは生成前のF項目ごとに `present_correct / missing / wrong / partial` をsourceと照合し、余剰のunsupported claimも保存する。語一致を正解判定に使わず、必須before/after意味・根拠を同じrubricで人手監査する。引用の実在性と事実の真実性は別集計。正確でも関係判断に不十分な抽出を区別する。ASTにある情報の反復を追加理解と呼ばない。Admissionの外部不足も別に評価する。

Bは関係label、方向、引用実在性、reasonのsource支持を分ける。A/B/C間の単純な正答率比較を能力階層や同一推論の因果説明へ変換しない。Cは旧規範で誤統合/正しい分離/必要merge/誤分割/必要defer/不適切defer/確定coverage/提示順を集計。確定案は既存authoritative validator/完全割当/byte/tree/依存順apply/revertを検査するが、PASSを意味理解へ読み替えない。

ケース別に `fact-missing / fact-wrong / relation-wrong / boundary-wrong / unsupported-or-ambiguous / undetermined` を複数可で仮分類し、支持証拠と反証/未識別を記録する。C2が変化しても事実認識失敗だけへ因果帰属しない。Oracle factsには規範labelを含めないが、source意味の言語化・追加token・注意喚起による間接的な答えの手掛かりは残る。独立した長さ対照を含めていないためformat/length/salienceの因果分離はできない。B正解relationを渡す探索上界診断は実施しない。

最小固定80callで支持/反証/未識別を報告してこの診断を終了する。同じcaseを再調整したり新caseへ拡張したりしない。新規未使用caseは現診断で判別できない仮説を対象とする別の事前登録候補として最小限提案する。今回の8caseはholdout/採用Gateへ再分類しない。

成果物が成立したらIssueの8チェックを根拠と照合して更新し、goal.mdの最終品質ゲート（Go test/vet/build、CLI build、既存release-note Python tests、同digest固定helper Swift tests、構造拒否preflight、resource計測器lifecycle、evaluator構文、diff検査）を実行する。production更新/削除なしなので既存テストを保持し、不要なunit testを足さない。研究専用docs/tools以外の変更0、main既存差分保持、remote deliveryとIssue closureを確認して検証完了とする。改善/production採用は主張しない。
