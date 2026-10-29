# H184-001 Category履歴本文の英訳比較

## Goal

固定された実Git差分・実Grouping結果に対し、Categoryの履歴本文だけを英訳すると分類が改善するか確認する。App Server取得と実Git参照は推論経路から除外した。

## Conditions

C184-02: Qwen3.5-4B MLX 4bit revision `0e7ffd5c629ef7719d4cbc04069232580bfa9d9c`、Thinking OFF、JSON文法あり、Swift Release、native KV、32K/4096、temperature0.6/top_p0.95/top_k20、penalty1.0、seed144。時間制限なし、warmup/生成失敗の自動再試行/出力修復なし。

fixture H184-KIMDEL-FIXED-001（5ファイル、3目的、3履歴context、依頼3本文・最終応答3本文、Grouping用command14件）。出典はH182-024のQ1-R再構成版で、失われたH182-016 wireのbyte同一性を主張しない。systemは両群とも英語で不変。

jaでGroupingを1回新規推論し、実応答をenで共有。両群でCategory→Textを実推論し、各群の実CategoryをTextへ継承。正解注入なし。英訳はassistantが用意した全文で、人手承認済みという意味ではない。翻訳エンジンは使っていない。英訳したのはCategoryトップレベルの `agent_history_contexts[].user_requests[]` と `final_responses[]` の6本文だけ。Textの履歴本文は日本語を維持。

## Result / Decision

| 目的 | 正解type | ja | en |
| --- | --- | --- | --- |
| READMEの品質チェック文書 | docs | docs | docs |
| 期限日tooltip追加 | feat | feat | fix |
| Message-ID正規化の不具合修正 | fix | feat | feat |

Categoryはja 2/3、en 1/3。両群のbreaking_evidence_refは3件ともnoneで正解。共通Groupingは3/3（enは再利用で独立した再測定ではない）。両群ともTextまで到達したがinvalid_scopeで最終validator拒否。禁止scopeはja1件、en3件。日本語文字を含むsummaryは両群3/3だが、自然さ・意味全正解の保証ではない。enのメールsummaryに不自然な表現も残った。

このfixture・この英訳・各1回では分類改善を確認できず、今回の英訳入力を採用する根拠は得られなかった。一般に翻訳が有害、またはApple翻訳が無効とは断定しない。追加推論・モデル条件の変更は行わない。

## Measurements

| arm | phase | 入力tokens | 生成tokens | prefill秒 | decode秒 | decode tok/s | helper wall秒 | MLX peak GiB |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| ja | category | 4946 | 71 | 16.033 | 2.702 | 26.27 | 20.762 | 3.142 |
| ja | text | 4830 | 68 | 15.952 | 2.665 | 25.51 | 21.675 | 3.130 |
| en | category | 4847 | 71 | 15.781 | 2.621 | 27.09 | 20.142 | 3.134 |
| en | text | 4830 | 81 | 15.784 | 3.187 | 25.41 | 21.245 | 3.130 |

Category入力は4946→4847tokens（99tokens、約2.0%減）。単発・別時刻の測定であり、速度差を安定した改善と扱わない。helper wallは翻訳プロキシと非推論contract監査を含む。実行済みGを共有したためenの短い全体時間を独立3phase高速化として比較しない。主比較はCategoryの同一phase。

## Input / serialization audit

6本文以外のJSON値、system、Schema、raw_diff、ファイル／group／履歴ID、照合状態、生成設定は不変。英訳したJSON文字列6個を原文文字列へ戻すと、元のuser入力とバイト単位で一致することを確認した。code spanは内容と出現数、パス・日本語の表示文字列・数値を保持。各phaseの実要求、wire、応答、host/runtime計測は非公開原本に保存した。

## Invalid measurements and corrections

最初のenはJSONをPythonで再serializeしたため、本文以外の`\u003c`/`\u003e`等の表現も変わっていた。このenはCategory2/3だったが、比較条件不成立として採用結果から除外し原本保持。語彙・モデル・samplingを変えず、元のJSON外形を保った6文字列の直接置換へ修正し、enのC/Tだけを新規測定した。jaは有効なまま対照として再利用。

実モデル呼び出しは総計7回＝共通G1回＋有効C/T4回＋除外en C/T2回。有効な対比較は各群1回。初期の準備段階では数値表記チェック、過度に厳しいcode span語順チェック、ハッシュ検査のパス型エラーもあったが、いずれもモデル呼び出し前。失敗原本と修正記録を保持した。

## Resources / limitations

ja監視はG/C/T、修正後en監視はC/Tが対象。すべての監視サンプルはvalid、fatal0、同時helper最大1。RSS観測最大はja 2.311GiB、en 3.040GiB。MLX peakとRSSを加算しない。各armの監視範囲が異なるためメモリの単純な優劣比較はしない。CPU user/system・GPU範囲・thermal等は隣接JSONと非公開原本に保持。native KVの実dtype/bytesは未export。物理温度・電力・process別GPUは未取得。

単一fixture・単一seed・各1回。履歴翻訳のみで、diff内の日本語は原文。今回の翻訳は事前作成なので、翻訳モデルの待ち時間・意味保持率の一般評価は含まない。

## Recheck: input, output and runtime evidence

2026-10-29に保存済み入力・出力を再監査した。追加推論0回。fixture検証33ファイルと、モデル・helper・実行入力などの固定ハッシュ52件および修正proxyのハッシュ1件が一致した。実helperの非推論contract probeをCategory/Textの両群で4回実行し、保存wireと一致した。各probeのgeneration_callsは0。

修正後の対比較では、実Category要求の差はuser contentだけであり、内部の差は指定した履歴本文6文字列だけだった。6文字列を原文へ戻したuser contentはバイト単位で一致。system、Schema、group/file/context参照、raw_diff、その他設定は共通。依頼・最終応答を対読し、修正／追加の目的、否定条件、対象ファイル、数値、コード識別子の重大な脱落・反転は確認しなかった。表示文字列などの日本語リテラルは意図して原文を維持している。翻訳の意味保持について第三者の独立評価や定量的な保証を行ったものではない。

ただし元fixtureの履歴取得状態はpartial、理由はhistory_incompleteだった。9変更ブロックすべてがambiguousでhistory_incompleteを持ち、履歴候補への参照は計4件。ブロックの省略理由フィールドはなく、保持済み3contextの依頼3本文・最終応答3本文は各群のCategory入力にすべて残っていた。これは翻訳や32K入力予算による脱落ではなく、元の取得段階に残る制約である。完全な履歴照合を前提とした精度結果としては扱わない。

Releaseのbuild-request記録、使用binaryの固定ハッシュ、ケース設定、生成経路のソース、実応答メトリクスを再確認した。Thinking OFF、json_schema、native KV、32768/4096、temperature0.6/top_p0.95/top_k20、seed144、追加penalty処理なし（1.0相当）、eagerの条件に矛盾は見つからなかった。正確な入力tokensと4096の予約を合わせても32768以内で、出力上限到達・履歴の予算削減はなかった。全4件のCategory/Text応答はSchemaに適合しcompleted。進捗ログのthinking_tokensは全サンプル0。

所属ファイル集合から採点し直してもCategoryはja2/3、en1/3。Textへ継承された差は期限日tooltipグループのtype（feat→fix）だけで、Textの履歴本文は両群とも元の日本語だった。Textの不受理はSchemaではなくscopeの意味上の制約による。jaはG001=docs、enはG001=docs/G002=fix/G003=featが禁止語に該当し、ホストのinvalid_scope判定と一致した。Textの結果を「Text履歴の英訳効果」とは扱わない。

資源監視はja35件、修正en21件がvalidで、同時helper最大1、thermalは全件nominal、低電力モードfalse。両群ともBattery Powerで稼働していた。swap usedはja436.06–635.88M、修正en810.69Mで異なり、背景負荷とOSキャッシュも固定していない。速度・RSSの小差を翻訳による安定した改善または悪化と断定しない。native KVのdtype/bytes、全環境変数、生成時の全sampling値を独立した実行メタデータとして記録していない点は残る。

初回の無効en（JSONエスケープ変更あり）は採点対象へ混入していない。追加の推論、条件変更、原本上書き、Issue投稿は行っていない。機械検証の詳細はrecheck.jsonに保存した。

## Follow-up: incomplete history and safety false positives

「取得段階の制約」はApp Server自体の取得失敗を確定した表現ではなかった。元取得のRPC計測では12要求すべてfailed=false、本文取得は4Turnで各1要求、保持された編集操作は3件。保存済みResultは変換後のHistoryのみで、変換前History.ErrorのStage/CodeやDroppedの件数・対象を保存していない。

既存アーカイブでは対象5ファイルすべてにapply_patch呼び出しがある。実装のsafeTextを保存済み5ファイルの差分へ適用する非推論診断を実行すると、F002（widget_view.py）とF005（test_normalization.py）だけが拒否された。前者のself._due_label.setToolTip、後者のpytest.mark.parametrizeはドット区切り3要素を許すjwtPatternへ一致し、秘密情報として誤検出される。検査用コピー上でそれぞれの識別子だけを除くとsafeTextがtrueへ変わる。他3ファイルは原文のままtrueだった。原本・製品コードは変更していない。

拒否される2ファイルは、履歴候補のない5ブロック（UI実装4、メールテスト1）と一致する。読み取り実装は安全性検査で不合格のfileChangeを落とし、1件でもDroppedがあると履歴全体をpartial/history_incompleteとする。照合実装は全体がcomplete以外ならすべてのブロックへhistory_incompleteを付けるため、残るexact候補4参照もambiguousになる。9ブロックすべての差分や会話が欠落した、という意味ではない。

今回再現できたのはcommiter側の安全性判定の不具合であり、App Serverの限界と片付けるべきではない。ただし過去の除外イベント詳細が未保存のため、当日の全除外を特定の分岐・件数へ完全に対応付けた監査証跡まではない。固定fixtureの現在Git差分5ファイルと保持済み会話6本文は完全に入力されており、欠けているのは主に編集記録に基づくブロックと履歴の対応情報。両言語群は同じ制約を共有する。

非推論の診断テストは成功、モデル呼び出し0。JWT検出と除外診断の製品修正・履歴再取得・再推論は未実施。
