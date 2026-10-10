# Issue #167 完了条件の検証監査

## 判定

検証の結論は **Gate 1: 対応する通常ファイルで成立、Gate 2: 未達、Phase 3: gated-out、production: NO-GO** である。以下の完了条件は、Issue が明記する「Gate 失敗時の理由記録」と「到達不能は未評価」を適用して照合する。チェックは検証・停止判断の記録を意味し、完全 plan・CLI・本番導入の成功を意味しない。

## 条件と根拠

| Issue の完了条件 | 検証結果と根拠 |
|---|---|
| 先行 #151/#157/#160 と再利用範囲 | decision 文書の先行資産監査。構造契約だけを Go collector / blob / tree に接続し、意味 NO-GO を保持 |
| Phase 1 の 1〜16、lossless / assignment / tree / temp-index / 原状態 | `phase-1-validation.json`、`internal/gitstate/file_first_test.go`。全16 file 数と混在 Git 状態・逆順・停止経路 |
| Gate 1 または停止理由 | 通常ファイル契約で pass。symlink/gitlink、変換/filter、snapshot drift、上限等の拒否を decision に明記 |
| Phase 2 接続、境界、費用、非回帰 | `FileFirstGenerator` と既存 metadata/validator の共有、1〜16 packet テスト。主計測36 runで完成 plan 0、Gate 2 fail。後続 packet 未到達の総費用は未評価 |
| Phase 3 opt-in / mutation / partial failure | Gate 2 未達により gated-out。CLI 接続・実 commit・partial failure は未実施 |
| 層別 coverage / 品質 / commit / 中間状態 / 資源 | 1/2/3/4/5/8/9/16 の主計測、8 fixture の構造確認。最終品質・実 commit 数・中間 build / revert・独立 holdout は未評価 |
| 同一1〜4 paired と事前条件 | `phase-2-preregistration.md` と主計測各3反復。snapshot/Prepared 一致、各 call の時間・TTFT・tokens・RSS/MLX、diff、pressure/swap を保存。失敗 run を除外しない。最終 semantic 品質は null、metadata 前 partition の追加診断は別標本 |
| 4→5 増分と方式差 | 同じ基底4 file bytesをテスト確認。反復ごとの差の p50/p95 は同方式 +18.49/+29.55秒、方式差込み −11.58/+11.21秒。静的 singleton は4 groupsから5 groups、5 file の参照同目的ペアは FS1。成功 plan / 実 commit 数は未評価 |
| 自動切替なし、default / 5+拒否維持 | CLI/executor の変更なし。`CandidateMaxFiles = 4` と既存 Three-phase の拒否・生成 profile の回帰テストを保持 |
| 過剰分割と事前 production 採否 | 構造参照8 fixtureで exact3/8、FM0、FS45。独立精度として扱わない。完全 plan と資源条件で Gate 2 fail、production GO なし。#160 の80%未達を変更しない |
| 1〜4非回帰、安全/local-only/Git | 既存 Three-phase 回帰テストと repository テスト。44推論 runで原状態保持。ローカル helperのみ、測定成果物に prompt/生成値を保存しない。実モデルの両方式失敗を意味品質の非回帰証明とは呼ばない |
| テストと品質 Gate | Phase 1/2で repository Go test/vet/build、release notes25テスト、diff整合が成功。検証結果確定後の最終品質 Gateも成功。Go testは22 package pass / 3 packageテストなし、vet/build、release notes25テスト、diff整合の結果を `final-validation.json` に記録 |
| 偽の成功にせず Issue 記録 | 主計測と追加診断を別成果物に保存、Issue Iteration 1〜4、decision と `verification-audit.json` で失敗・未評価・停止条件を記録 |

## 停止条件と残余リスク

追加診断8 runは全て `invalid_schema`。scope 最大18〜22文字、summary 最大49〜54文字という上限違反を観測した。主計測の33停止の詳細理由は保存されておらず、追加診断から遡及確定しない。有効な完全 metadata を保証する既存 fallback は確認できなかった。長さ制約を緩めたり、生成文を切り詰めたり、新しい推論方式を追加したりして通過扱いにしない。

Gate 2 の pass と将来の CLI 実装再開には、現在の規約に適合する完全 metadata、120秒・資源上限、成功した最大16 groupsの費用を新しい検証で示す必要がある。独立 holdout / 中間状態 / partial failure は、その後の Phase 3 の作業である。今回の検証終了から本番使用可能性を推定しない。

既存テストを置換・削除する機能変更はない。追加テストは所有権/状態保持、1〜16 packet の境界と予算、後続失敗時の部分 plan 禁止、測定取消時の子プロセス停止、生成値を含めない数値診断を検証する。単なる実装写しやモデル意味正解の代用として扱わない。
