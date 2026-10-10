# Iteration 1: 固定ハーネスと取得前監査

## 要約

段階1の固定ハーネスと構造preflightを作成し、27checkすべてPASSした。M3/16GBの実機と初回2モデルの取得条件を監査した。Qwen2.5-Coder-3Bの重みは確認した既存cacheに存在せず、取得可否をユーザーへ確認中。モデル推論0call、資源/意味/全体資格は未評価。Issue/Goalは未完了。

## 検証結果

- 参照#162のGitHub/ローカルheadは`b8a7fe14a798ff36680efffdbd879c9a02cb37f2`。AST/authoritative validator adapterをbyte一致で再利用し、モデル固定呼び出しと局所singleton計画構築は再利用しなかった。
- 局所merge/keep_separate/deferと全体membership/保留を別のprompt/schema/出力監査にした。schema/assignment/根拠ID、非completed、timeout/context、不正JSON、byte/tree再構築、source改変拒否の27checkがPASS。合成応答・自己作成sourceのみで、モデルcalls0。
- `proc_pid_rusage(RUSAGE_INFO_V4)`によるfootprint/RSS/observed lifetime peakの採取器を作成。自己プロセスと無害な子プロセスの4sampleを採取し、lifecycle preflightがPASS。モデル資源資格やMLX/TTFTの計測成功とは別。
- 実機はApple M3、物理メモリ17,179,869,184 bytes。cache volumeの空きは30,840,972 KiB（約31.6GB）として`iteration-1-environment.json`へ保存した。
- Gemma固定revision`475b9088d29754a3379866cf5aeb6b41acd313c2`は既存cacheあり。Qwen2.5-Coder-3B固定revision`3dd939c621c08e5753d5b89f35a2642cd83b98ca`は4bit/group_size64、Qwen2Tokenizer。取得対象9fileで1,747,849,128 bytes（約1.75GB/1.63GiB）。weight SHA-256とtokenizer/configのblob/digestを保存した。
- 固定upstream LICENSEはQwen Research Licenseで研究・評価用途を許可し、商用利用には別ライセンスを要求する。量子化repoのcardはupstream licenseを参照し、repo内の独立LICENSEは無かった。今回の評価でweightsを配布しない。
- Swift runtimeにはqwen2登録があるが、現在の固定benchmark helperのbounded profile許可リストにはCoder pinがない。ロード/grammar/template実行は未確認。TTFT/MLX割当の応答項目も現helperには無い。
- `go test ./...`、`go vet ./...`、`go build ./...`、release note unittest25件はPASS。新依存・モデルweight取得無し。production/4file契約/mainの既存変更を変更していない。

## 考察

構造preflightは不正応答と破損sourceを拒否する証跡であり、分割の意味妥当性を証明しない。qwen2実装の存在も固定checkpointのロード、template、grammar、5GB以内の実行を証明しない。現在の不足はモデル重み・実験helper対応・計測項目であり、モデル能力のNO-GOと呼べない。

主比較にtest-assistedの実測結果を加えないことで情報差を除き、原diffと同じsource根拠を各モデルへ渡す。native template差は残してhash保存する。局所資格と全体資格を分け、未使用入力/source/gold/token長と実helper hashが揃うまで推論を開始しない。

## Next Steps

- Qwenの9file・約1.75GB取得へのユーザー回答を確認する。既存cacheを削除せず、研究・評価用途で固定revisionを取得するため。
- 実験用helperだけに固定Coder pin対応と必要な計測を追加してbuild/hashを固定する。既存helper/defaultを変更せず、同じ生成条件と資源計測を成立させるため。
- 未使用10case・2提示順のsource/gold/許容partition/family相関を監査し、実token長・資源負荷・全hashをcommit/pushする。未決定条件で比較せず、結果後の調整を防ぐため。
- 同情報の2モデル局所比較を実施し、各モデルの資源/局所Gateを判定する。成立構成だけ別の事前登録で8/16fileへ進め、未達なら原因と再開条件を保存するため。
