# H182-018 Sparkプロンプト段階比較

## Goal / Contract

ユーザー承認の順序で2セルを直列実行する。対照は保存済みH182-017で、対照を再試行しない。

| 段階 | 条件 | 変更 | 比較対象 |
| --- | --- | --- | --- |
| 1 | Thinking OFF / grammarあり | Categoryに既存挙動の修復と新規コード追加の区別を追加。Textに主目的・自然な日本語・テスト値を安易に転載しない指示を追加 | H182-017 OFF / grammarあり |
| 2 | Thinking Low / grammarあり | 共通構造化指示のthink freelyを、簡潔な根拠確認と判断後の回答遷移を促す指示へ変更 | H182-017 Low / grammarあり |

段階2は元のCategory/Textへ戻す。段階1の変更を引き継がない。段階1のGrouping指示は原本と一致することを静的検査済み。段階2のplanner promptは原本と一致し、adapterが追加するThinking指示だけを変更する。フィールド/型/必須/Schema、構文制約と停止制御は不変。

## Fixed conditions

同一Spark weight revision・固定fixture・32K入力/2048出力（Thinking含む）・native KV・sampling1/.95/topk無効・penalty1/0/0・seed144、各1回・時間制限なし。最大2セル/6phase。基準の正解groupはモデルへ注入しない。既存の段階別時間・CPU/GPU/メモリ/swap/thermal/KV配列計測を維持する。Thinking予算分離・強制境界・予算増加なし。

## Execution / Gates

専用Go overlayをコンパイルし、既存Spark境界/Schema等4unit成功。プロンプトの変更範囲の非推論検査に成功。変更したadapterは有効なwire messagesをprivate保存する。オーケストレータが段階1の正常測定・monitor正常・adapter例外なしを確認してから段階2を開始する。分類や最終validatorの不受理は測定結果であり、それだけで独立した段階2を禁止しない。インフラ異常なら後続を開始しない。登録済み段階の再起動を拒否。

今回のアダプターの差は出力契約の文面とwireログ保存のみ。モデル実装、生成器、トークン予算、自然終了判定、grammarは変更しない。原本・失敗・旧結果を保持する。追加推論/修復/採用判断を自動で行わない。
