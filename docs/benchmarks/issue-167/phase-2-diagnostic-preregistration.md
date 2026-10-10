# 主計測の失敗に対する追加診断

主計測は `f7bcbb16` のバイナリと事前登録の36 runを継続する。主計測の prompt・生成設定・fixture・反復条件は変更しない。主計測の artifact は書き換えず、詳細失敗 codes が保存されていなかった限界を残す。

## 観測と仮説

最初の1〜3fileの18 run は metadata の call が completed でも完全 plan を返さなかった。既存 #146 の記録には bounded-text が48文字を超え、最終 metadata 検査で停止したケースがある。今回も長さ超過か、scope・schema・language 等の違反かをまだ特定できない。生成内容を保存せずに違反 codes と数値を調べる。

## 追加診断の条件

- 主計測が terminal になってから実行する。推論を並列に走らせない。
- 主計測と同じ pinned model/helper、production generator、fixture内容、category/text/schema、context/output budget、120秒を使う。再生成 feedback、修復、fallback は追加しない。
- 1〜4の4 workloadを各1反復、Three-phase → File-first の順で実行する。これは主計測の失敗分類と、metadata 前の partition を確認する最小のケース数であり、新しい latency sample の3反復へ混ぜない。
- decoded group数・欠落/未知件数・scope/summary 最大文字数・許可済み violation codes を保存する。型付き decode 成功は duplicate/値/validator の全検査成功と同一ではない。
- metadata に渡った complete membership の ID だけを記録する。File-first は Phase 1 の complete preview groupsを使う。参照 purpose との provisional exact/FM/FS を最終 plan のスコアと分ける。
- 5/8/9/16 の File-first provisional partition は決定的 singleton である。独立した構造確認と静的参照比較を行い、LLM の意味成功率に数えない。完成 planが得られない場合は、最終意味品質・commit mutation・中間 state・revert・独立 holdout は到達不能として残す。
- 長さ超過等を確認しても、切り詰め・目的の捏造・validator 緩和・新しい decoder で通さない。安全な既存 fallback がなければ metadata 停止が正しい結果である。
