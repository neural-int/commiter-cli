# Harness 004と通常3-Phaseの整合性監査

確認日: 2026-11-03。調査対象はPR #185と同じHEAD `2e2e9533622ee073a37cb1da85db10f6b0b966d7`、および今回のfinal-only修正。公開manifest、実装、保存済みfinal-only fixtureのフィールド構造を照合した。生の会話・差分・Thinking・個人パスは転載していない。

## 結論

004のメッセージ配置、固定指示、実改行表示、同一helper、型別KV再利用は接続されている。しかし、通常CLIから供給する履歴・ブロック照合・コマンド文脈・ファイル投影は、検証fixtureと同じではない。旧Plannerとの互換性を残した部分もある。したがって「004の検証済み入力契約全体を本番へ移した」とは扱えない。

manifest revision 2にも旧codex_contextを維持する例外が明記されている。その例外は実装済み部分の限界の記録であり、今回のユーザー方針である004への完全な統一を満たす根拠にはならない。

## 今回修正した事項

**A01: 最終応答より依頼を優先する旧規則。**

- `internal/codexcontext/collect.go`は依頼本文があるTurnだけを採用し、8KiBを超えると最終応答を捨てて依頼だけを残していた。
- `internal/planning/codex_context.go`も依頼を必須とし、最終応答を先に捨てていた。
- 004の表示投影は非空のuser_requestsを保持していた。

取得用RecordのUserRequestを廃止し、userMessage本文を選別用データへ取り込まないよう変更した。安全な非空のfinal_answerだけを採用し、最終応答がない・危険・予算超過の場合は記録全体を除外する。依頼へのフォールバックと本文途中の切り詰めはしない。Go/Pythonの004表示もuser_requestsを常に省く。保存fixtureの原本は変更しない。

これは履歴本文の方針変更であり、後述の取得経路・スキーマ・実token計数まで接続した変更ではない。

## 通常経路で解消が必要な不一致

| ID | 現在の実装・場所 | 004／今回の目的に対する影響 | 必要な対応 |
|---|---|---|---|
| A02 | `internal/cli/planning.go`はoptionalCodexContext→ThreePhaseGenerator.CodexContext。`internal/cli/codex_context.go`は旧Collectorを呼ぶ | 内容照合済みagent_history_contextsへ接続していない。現在のチェックアウトにinternal/historyattribution自体がない | 別作業ツリーにある採用済み取得・照合・投影実装を確認して取り込み、通常入口へ接続する。フィールド名だけの置換で代用しない |
| A03 | Collectorはcompleted fileChangeのパスだけを関連付け、diff本文を保持・照合しない。move_pathは保守的に除外 | change_blocks、diff_range、候補、matched/ambiguous/unmatched/error、理由・省略理由を生成できない。rename照合にも対応しない | 固定Git差分と履歴ブロックを照合する正規パイプラインを使う。パス一致をmatchedへ昇格しない |
| A04 | CollectorのitemにcommandExecutionのコマンド本文がなく、switchにも分岐がない | Groupingのagent_command_execution_contextsが供給されない。シェル操作の作業文脈が検証入力と違う | 安全性検査・旧新パスへの言及判定・出典重複排除を実装した既存方式へ接続する。編集成功やブロック一致の証拠にはしない |
| A05 | CollectorのStatusはCLIのmetricsにのみ渡る。モデルには旧paths/final_response配列 | context_refによる本文辞書、取得完全性、Groupingの投影状態、ブロック単位のhistory_incomplete等がない。履歴なしと取得失敗を入力から区別できない | 確定したagent_history_*契約へ移行する。Category/Textに取得・投影のトップレベル状態を復活させず、影響するブロック理由で表す |
| A06 | `internal/codexcontext/types.go`: 5秒、受信4MiB、保持8KiB、ページ20件。collect.goは直列取得。CLIはHEADコミット時刻以降を探索し、他worktreeの会話も候補にする | 既に検証した100件・4並列・現在worktree限定の取得方式とは別経路。モデルの64K予算より前に履歴が欠落し得る | 採用済み取得方式の各制御を照合して移行する。5秒／8KiBをモデル予算の代替にしない。受信量等の資源上限は無条件に撤廃せず、正規経路の保護へ置換する |
| A07 | `internal/config/config.go`のthree-phase検証はSHA固定Gemmaだけを許可。Swiftの004とmanifestはQwenも許可 | 実モデル疎通済みQwenを通常CLIでは設定できない | 同一の対応モデル一覧を設定・セットアップ・helperで使用する。未検証モデルまで無条件に解放しない |
| A08 | `internal/planning/grouping_input.go`／contextinput.Fileを再利用。filesはpath:... ID、statusはGitコード、不要なfalse/null/mode等を含む。ブロックなし・metadata-only時raw_diff省略 | 004 fixtureのF1等のID、正規状態名、条件付きフィールド、必須raw_diff、全change_blocksと一致しない。入力量・配置が異なる | ファイル単位の許可フィールドを明示した投影へ統一し、ホストのID・安全性情報は内部保持する。ID形式等は意味的不具合と実測入力との差を区別して検査する |
| A09 | BreakingEvidence.FactがJSONへ出る。Category/Textはmetadataを共有し、Textにもbreaking_evidence_candidatesを渡す | 削除対象のfactが残る。分類済みTextにCategory用の判断材料が残る | host側の証拠検証情報とLLM向け投影を分離し、Categoryの許可項目だけを渡す。Textは確定type/breakingと必要な根拠に限定する |
| A10 | ファイル配列は_test.goを後ろへ置いてから自然順。candidateTestOnlyも_test.goだけで判定 | 全ファイルを同じ順序規則で表示する方針と差がある。Goのテストだけ分類Schemaをtestへ限定する規則も、Python等では動かない | 言語をまたぐ入力配置・分類規則として必要か明示する。暗黙のGo専用処理を004へ持ち込まない |

A06の「他worktree」は、会話の探索範囲の差である。現Collectorも編集パス自体は選択worktree内に制限しており、他worktreeの編集が無条件に混ざるという意味ではない。

## 旧Planner互換処理として整理する箇所

| ID | 残っている処理 | 整理方針 |
|---|---|---|
| L01 | `internal/config/config.go`はllm.plannerがない既存設定をsingle-pass/Ollamaへ暗黙に戻す | 004を通常経路とする方針と直接衝突する。旧経路へ戻す互換推論を削除し、設定移行か明確なエラーにする |
| L02 | configはsingle-passを許可し、`internal/cli/planning.go`には旧Generator分岐がある。エラー文も「別モデルや大きい変更ではsingle-pass」を案内 | 廃止予定ではなく今回削除する方針なら、設定受付・分岐・案内・関連ドキュメントと旧専用テストをまとめて移行する |
| L03 | 通常3-Phaseの前にもInitialSystemMessage→旧Renderer→contextinput.Prepare→旧constraints検査が走る | 実際にLLMへ渡さない旧単一Planner用のprompt/schemaが予算を消費し、004へ変換する前に拒否できる。relation情報も作成して旧準備処理に通すが004では使わない | snapshot/安全性/完全割当の検査を保持し、3-Phase専用の入力準備へ置換する。旧promptを成立させるためだけの処理を削除する |
| L04 | helperはinput_layoutなしの旧bounded経路、旧512/768出力、旧Thinking指定、Schema追記処理も維持 | 004の通常要求とは別契約。現在004明示時は多くが回避されるが、旧呼び出しも可能 | 通常3-Phaseは004契約を必須とする。過去比較用モードを残すなら研究専用入口へ隔離し、履歴的な成果物を削除しない |

L03の圧縮処理は、現在の3-PhaseではRequireUncompressedで止められている。したがって、現在の3-Phaseが黙ってdiffを要約しているという指摘ではない。問題は不要な旧入力を組み立て、バイト数基準で事前拒否し得ることである。

## 直近相談したが未実装の変更

| ID | 現在 | 相談中の変更 |
|---|---|---|
| N01 | 最初のGroupingで16K/32K/64Kを選択してサイクル内固定 | 全3フェーズ64K固定。CLI準備・設定・doctor/probeも一貫させる |
| N02 | Goのcontextinput.SelectContext、通常3-PhaseのrequiredSize、履歴候補のfitがUTF-8バイト数をtoken数とみなす | 同じhelper/tokenizerでテンプレート適用後の実token数を計測し、採用判定と生成前検査に使う。条件だけを削除して履歴の採否をなくさない |
| N03 | 004ではlegacy保護を強制し、設定context>16Kで同期・解放ON。共通prefix準備にも無条件の毎chunk同期・解放 | 実入力12,000 tokensを暫定境界にする相談案。入力全体を判定対象とし、通常prefillとprefix準備で同じ方針を使う。境界の安全性・高速化が実証されたとは扱わない |

N01〜N03は元の004 revision 2が満たしていなかったバグではなく、その後に相談した新しい実行条件である。今回のfinal-only修正には混ぜていない。

## 維持する仕様と安全性

- 4ファイル上限・120秒制限は直前のユーザー指示で維持した設計例外であり、旧Plannerを削除するという理由だけで今回解除しない。summary48文字とscope16文字も維持する。
- JSON Schema制約、scope禁止語、出力4096、native KV、Thinking OFF、固定sampling/seed、同一helper、Grouped/Textの共通prefixと独立コピー、Category独立prefillは現在の004接続と整合している。
- 機密情報除外、選択ファイルの完全割当、重複ID拒否、Git snapshot再検証、未対応型のKVフォールバック、通信フレームのバイト上限は削除対象ではない。
- final-onlyは自然言語の履歴本文を最終応答に限定する方針。ブロック照合の出典・不確実性や、別枠のGrouping用コマンド文脈を削除する根拠にはしない。
- 旧出力の失敗ログや歴史的なmanifest／fixtureを、現行仕様に合わないという理由で書き換えない。現行入口と過去検証の資産を区別する。

## マージ前に揃える順序

1. 本書をもとに004の入力契約と通常実行条件を一つの現行契約へまとめる。final-onlyは今回反映した。
2. 採用済み履歴取得・照合・コマンド文脈・投影を実際の通常CLIへ接続し、保存fixtureと同じ構造を作れるようにする。
3. 旧設定のsingle-pass復帰、旧Generator、Gemma専用制約、不要な旧prompt準備を置換する。
4. 64K固定・実token計数・12,000境界を接続する。通常prefillと共通prefix準備の両方を確認する。
5. 固定fixtureで、元データ→投影→実際のhelper受信本文を比較する。JSON成立・全フェーズ疎通・validator・意味品質を別々に評価する。4ファイル制限を維持する通常入口と、7/9/11の研究経路を混同しない。

今回の作業は静的監査とfinal-onlyの修正・回帰確認。その他の一覧項目の実装、モデル再推論、push、main mergeを行ったとは扱わない。


### 12K境界作業の追記

上記は監査時点のスナップショット。後続のユーザー指示によりN01〜N03を[004 revision 3](actual-token-context.md)で実装する。3-Phaseでの旧Prepareのバイト判定は迂回し、履歴のfitと各最終メッセージを実tokenで判定する。L03の旧Renderer・制約形式そのものの整理、A02以降の取得・投影統一、旧Planner廃止はIssue #187の別スコープに残る。
