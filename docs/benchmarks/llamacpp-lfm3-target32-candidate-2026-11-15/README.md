# LFM3 Textの暫定採用候補

2026-11-15のユーザー指定により、固定入力Text試験の「②＋32文字目安」を現状の候補として採用した。実験IDは05_local_target32。profile.jsonに実際のsystem、要約指示、Schema、sampling、入力配置と元要求のdigestを保存した。

## 採用条件

- LFM3-1.7B Q8_0、llama.cpp、Thinking OFF
- グループ直下に対応するファイルの完全な差分を配置し、共有側のfilesは空配列にする。履歴は共有メッセージに保持する
- 旧要約指示に次の1文だけを追加する
- Schemaのsummary上限48文字、scope上限16文字、context32768、総出力上限4096
- temperature0.1、top_k50、repeat_penalty1.1、seed144。その他のsampling値はprofile.jsonに収録

> Aim for 32 characters or fewer and state the change in one complete sentence.

元の1〜48文字という要求も残している。比較対象の②へこの追加文を除いて復元すると、要求オブジェクトが一致することを確認した。

## 測定結果と残課題

1固定入力・1seed・1回。Text単独で7.8846秒、150生成tokens、decode42.92tok/s。対象対応3/3、Schema・既存validator通過、文末の途切れ1/3。summary文字数45・48・47で、32文字目安の達成は0/3。熱状態5観測すべてnominal、低電力OFF。

G002は表示例の途中で途切れた。G003は空の山括弧をNoneとして扱う実差分に対し、山括弧へ正規化すると誤記した。G002/G003のscopeは部分的なパスで終わる。この候補のGrouping・Category生成、伝播E2E、3フェーズ合計時間は未測定。

今回の採用対象はこの研究プロファイル。通常のHarness004への組み込みとその動作検証は未実施。要約例や採点用の正解文をモデル入力に追加していない。
