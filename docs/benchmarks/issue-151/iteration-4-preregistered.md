# Iteration 4: bounded UTF-8 inline edit atoms

## 仮説

line-unitのreplacementに限って、UTF-8 codepointのedit atomsへ分解すると、同一lineの別操作をsource byte spanを保持して表現できる。これはintent推定ではなく、intentが選択できる基盤の検証。言語・file名・goldに依存しないdiff構造を使用する。過去のline baselineは変更しない。

line editを抽出した後、非empty before/afterのreplacementのみUTF-8 decodeし、SequenceMatcherのnon-equal opcodeを位置対応の1codepoint editへ分解する。挿入・削除の余剰codepointは独立atoms。双方のoffsetをUTF-8 byteへ戻す。同じ場所への複数insertはnew byte順で保持する。非UTF-8は既存atomic unitを維持する。empty側のwhole-line insert/deleteは既存unitを維持する。

fileの1MiB / 20k line / 256unit上限を維持し、refinement対象の1line 4096byte超過はinline budgetでfail closed。256unitを超える場合はatomic fallbackやtruncationではなく停止する。意味的に不適切なatomの中間適用を自動で採用するとは主張しない。

## 固定入力とgate

iteration3の12件を使用済み回帰セットとする。新しい6件: Unicode / nested settings / inline add-delete / CRLF / no newline / contiguous independent labels。全件別要求のsynthetic。real repository holdoutではない。`inline_fixtures/manifest.json`が測定前のbefore/after/gold/全subset期待snapshotを固定する。要求とgoldを抽出器へ渡さない。

lineとinlineの両方式を同一の固定6件で比較する。旧12件もinlineで評価する。両セット全件でgold表現可能、coverage/reconstruction/determinism/全intent subset・順序のstage一致の場合、今回のbounded representation能力gateはGo。失敗ケースは除外しない。旧#149 14件＋旧synthetic10件のcoverage/stagingもinlineで確認する。これは親の既知#149 semantic改善条件とは別であり、親gate達成へ読み替えない。

feasibility oracleの16unit/file上限を維持する。超過は検査不能としてgateを通さず保存する。新6件の測定後に抽出器をgoldに合わせて変更しない。

## 検証と指標

未知/重複ID、stale snapshot、overlapは既存reconstruct contractで拒否。新方式のUTF-8 boundary、同位置insertの順序、refinement byte予算とunit予算、非UTF8のbyte保存を追加testで確認する。単一実行のunit数・overhead・latencyを記録する。LLM呼出0、exact/FM/FS/unresolvedはN/A。細粒度化によるfragmentationは費用として保持し、semantic品質やproduction採用を宣言しない。

測定前new fixture manifest SHA256: `3174e13813e9b0e99f3fc062ef898c9de014b2f0d946d97434188ddb50fa018f`

## 測定開始前の評価器補足

まだ候補抽出・計測を実行していない時点で、行内の複数文字insert/deleteを扱うatom方式では従来16unitの総当たりoracleが指数的になると確認した。goldや抽出器予算を変更せず、byte prefixのdynamic programmingで最大2個の一致subsetを検出するoracleを追加する。256unit/file、4096探索stateを上限として失敗時はgateを通さない。これは評価器の検査方法の変更であり、モデルbudget増加・production budget増加ではない。旧exhaustive方式は維持し、小規模入力で両方式の一致を検証する。今回の比較はline/inline両方で同じbounded-dpを使用する。上記16unitの検査条件は今回のDP比較に限り、この補足で置換する。
